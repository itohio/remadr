package main

import (
	"fmt"
	"io"
	"machine"
	"strconv"
	"strings"
	"time"

	"github.com/itohio/remadr/config"
	"github.com/itohio/remadr/dev"
	ui "github.com/itohio/tinygui"
)

var (
	meter  *dev.VoltageMeter
	chrono *dev.Chronograph
	driver *dev.MassDriver

	voltages        [2]float32
	voltagesPreShot [2]float32

	shotCh  = make(chan shot, 1)
	driveCh = make(chan drive, 1)
	chErr   = make(chan error, 1)
)

type shot struct {
	speed  float64
	dA, dB time.Duration
}

type drive struct {
	interStage time.Duration
	dA, dB     time.Duration
}

func configureVoltage() {
	var err error
	meter, err = dev.NewVoltageMeter(3.3, 12, 100,
		[]machine.ADC{config.VoltageA, config.VoltageB},
		[]dev.Approximator[float32]{
			dev.NewOptoisolatorCTRModel[float32](2094, 10, 7911, 19),
			dev.NewOptoisolatorCTRModel[float32](2470, 10, 8360, 19),
		})

	if err != nil {
		println("Voltmeter failed: " + err.Error())
	}

	meter.Configure()
}

func configureChrono() {
	chrono = dev.NewChronograph(
		// config.SenseA, config.SenseB,
		config.ChronoA, config.ChronoB,
		78.39,
		nil,
	)

	chrono.SetCallback(func(f float64) {
		dA, dB := chrono.Durations()
		select {
		case shotCh <- shot{speed: f, dA: dA, dB: dB}:
		default:
			println("SHOT !")
		}
	})
	chrono.Configure(machine.PinInput, machine.PinFalling)
}

func configureDriver(numStages int) {
	stages := make([]dev.Stage, numStages)
	if numStages == 0 {
		panic("too few stages")
	}

	if numStages >= 1 {
		stages[0] = dev.NewDoubleTapStage(config.TriggerA, config.SenseA,
			time.Microsecond*10,
			time.Microsecond*700,
			time.Microsecond*2500,
			time.Microsecond*1000,
		)
	}
	if numStages >= 2 {
		stages[1] = dev.NewSimpleStage(config.TriggerB, config.SenseB,
			time.Microsecond*2000,
			time.Microsecond*1000,
		)
	}
	if numStages > 2 {
		panic("too many stages")
	}

	driver = dev.NewMassDriver(
		func(i int8, err error) {
			chErr <- err
		},
		func(u []time.Duration) {
			stage1, _ := driver.GetStage(0)
			dt1 := stage1.(dev.DwellTimer)
			if numStages == 1 {
				driveCh <- drive{
					dA: dt1.DwellTime(),
				}
				return
			}
			stage2, _ := driver.GetStage(1)
			dt2 := stage2.(dev.DwellTimer)
			driveCh <- drive{
				interStage: u[1] - u[0],
				dA:         dt1.DwellTime(),
				dB:         dt2.DwellTime(),
			}
		},
		stages...,
	)
	driver.Configure(machine.PinInput, machine.PinFalling)

}

type whatCmd int

const (
	READ_VOLTAGE whatCmd = iota
	SET_STAGE
	DRIVE
	TEST
)

type cmdCfg struct {
	what  whatCmd
	param string
}

func configureMux(uart io.Reader) chan cmdCfg {
	cmd := make(chan cmdCfg, 3)
	submit := func(c whatCmd, data []byte) {
		select {
		case cmd <- cmdCfg{what: c, param: string(data)}:
		default:
		}
	}

	mux := ui.NewCommandStreamMux(uart, map[string]func([]byte){
		"?":     func(b []byte) { submit(READ_VOLTAGE, b) },
		"state": func(b []byte) { submit(READ_VOLTAGE, b) },
		"s":     func(b []byte) { submit(SET_STAGE, b) },
		"d":     func(b []byte) { submit(DRIVE, b) },
		"drive": func(b []byte) { submit(DRIVE, b) },
		"t":     func(b []byte) { submit(TEST, b) },
		"test":  func(b []byte) { submit(TEST, b) },
	})

	go mux.Run()

	return cmd
}

func configureSerial() {
	uart := machine.Serial
	uart.Configure(machine.UARTConfig{TX: machine.UART0_TX_PIN, RX: machine.UART0_RX_PIN})
	cmdCh := configureMux(ui.NewSerialReader(uart))

	go func() {
		shotCount := 0
		driveCount := 0
		for {
			select {
			case err := <-chErr:
				println("DRIVE ! " + err.Error())
			case c := <-cmdCh:
				handleCmd(c)
			case shot := <-shotCh:
				println(fmt.Sprintf("SHOT %d %f %v %v", shotCount, shot.speed, shot.dA, shot.dB))
				shotCount++
			case drive := <-driveCh:
				println(fmt.Sprintf("DRIVE %d %v %v %v", driveCount, drive.dA, drive.dB, drive.interStage))
				driveCount++
			}
		}
	}()
}

func handleCmd(c cmdCfg) {
	switch c.what {
	case READ_VOLTAGE:
		meter.ReadVoltages(voltagesPreShot[:])
		println(fmt.Sprintf("STATE %d %f %f %v %v %v %v", STAGES, voltagesPreShot[0], voltagesPreShot[1], config.SenseA, config.SenseB, config.ChronoA, config.ChronoB))
	case SET_STAGE:
		params := strings.Split(c.param, ",")
		if len(params) < 2 {
			println("STAGE ! Invalid count")
			return
		}
		id, _ := strconv.ParseUint(params[0], 10, 32)
		if id > 1 {
			println("STAGE ! Invalid index")
			return
		}
		shape := make([]time.Duration, len(params)-1)
		for i, param := range params[1:] {
			d, err := time.ParseDuration(param)
			if err != nil {
				println("STAGE ! " + err.Error())
				return
			}
			shape[i] = d
		}
		s, err := driver.GetStage(int(id))
		if err != nil {
			println("STAGE ! " + err.Error())
			return
		}
		err = s.(*dev.ShapedPulseStage).SetShape(shape)
		if err != nil {
			println("STAGE ! " + err.Error())
			return
		}
	case DRIVE:
		meter.ReadVoltages(voltagesPreShot[:])
		chrono.Reset()
		driver.Reset()
		driver.Arm()
	case TEST:
		println("TEST " + c.param)
	}
}
