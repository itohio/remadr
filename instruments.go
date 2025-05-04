package main

import (
	"fmt"
	"machine"
	"time"

	"github.com/itohio/remadr/config"
	"github.com/itohio/remadr/dev"
)

var (
	meter  *dev.VoltageMeter
	chrono *dev.Chronograph
	driver *dev.MassDriver

	voltages        [2]float32
	voltagesPreShot [2]float32
)

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

	type shot struct {
		speed  float64
		dA, dB time.Duration
	}
	shotCh := make(chan shot, 1)

	chrono.SetCallback(func(f float64) {
		dA, dB := chrono.Durations()
		select {
		case shotCh <- shot{speed: f, dA: dA, dB: dB}:
		default:
			println("SHOT !")
		}
	})
	chrono.Configure(machine.PinInput, machine.PinFalling)

	go func() {
		count := 0
		for shot := range shotCh {
			println(fmt.Sprintf("SHOT %d %f %v %v", count, shot.speed, shot.dA, shot.dB))
			count++
		}
	}()
}

func configureDriver(numStages int) {
	type shot struct {
		interStage time.Duration
		dA, dB     time.Duration
	}
	shotCh := make(chan shot, 1)
	chErr := make(chan error, 1)

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
		stages[0] = dev.NewSimpleStage(config.TriggerB, config.SenseB,
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
			shotCh <- shot{
				interStage: u[1] - u[0],
			}
		},
		stages...,
	)
	driver.Configure(machine.PinInput, machine.PinFalling)

	go func() {
		count := 0
		for {
			select {
			case err := <-chErr:
				println("DRIVE ! " + err.Error())
			case shot := <-shotCh:
				println(fmt.Sprintf("DRIVE %d %v between triggers", count, shot.interStage))
				count++
			}
		}
	}()

}
