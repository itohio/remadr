package main

import (
	// "fmt"

	"fmt"
	"image/color"
	"machine"
	"time"

	"github.com/itohio/remadr/config"
	"github.com/itohio/remadr/dev"
	ui "github.com/itohio/tinygui"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"

	_ "unsafe"
)

//go:generate tinygo flash -target=pico

var (
	WIDTH  int16 = 120
	HEIGHT int16 = 8
)
var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 0}

	encoder *encoders.QuadratureDevice
)

func main() {
	cS := time.Now().UnixMicro()
	dur := time.Millisecond * 2
	waitMilli := dev.BenchmarkWait(dur, 100)
	dev.SetWaitCalibration(dur, waitMilli)
	println(fmt.Sprintf("Calibration: %v / %v; %v = %v", dev.WaitCalibrationK, dev.WaitCalibrationM, dur, waitMilli))

	var waitMilliTuned int64
	for i := 0; i < 100; i++ {
		t1 := time.Now().UnixMicro()
		dev.WaitCalibrated(time.Millisecond)
		t2 := time.Now().UnixMicro()
		waitMilliTuned += t2 - t1
	}
	overallCalibration := time.Duration(time.Now().UnixMicro() - cS)

	println(fmt.Sprintf("Calibration: %v / %v took %v and 1ms = %v, originally %v", dev.WaitCalibrationK, dev.WaitCalibrationM, overallCalibration, waitMilliTuned/100, waitMilli))

	machine.InitADC()
	config.VoltageA.Configure(machine.ADCConfig{})
	config.VoltageB.Configure(machine.ADCConfig{})

	config.TEST1.Configure(machine.PinConfig{Mode: machine.PinOutput})
	config.TEST2.Configure(machine.PinConfig{Mode: machine.PinOutput})
	config.TEST3.Configure(machine.PinConfig{Mode: machine.PinInput})
	config.TEST4.Configure(machine.PinConfig{Mode: machine.PinOutput})

	//runTest01()

	encoder = encoders.NewQuadratureViaInterrupt(config.ButtonA, config.ButtonB)
	encoder.Configure(encoders.QuadratureConfig{Precision: 1})
	config.Button.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	config.SenseA.Configure(machine.PinConfig{Mode: machine.PinInput})
	config.SenseB.Configure(machine.PinConfig{Mode: machine.PinInput})
	config.ChronoA.Configure(machine.PinConfig{Mode: machine.PinInput})
	config.ChronoB.Configure(machine.PinConfig{Mode: machine.PinInput})

	machine.I2C0.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	// the delay is needed for display start from a cold reboot, not sure why
	time.Sleep(time.Second)
	display := ssd1306.NewI2C(machine.I2C0)
	cfg := ssd1306.Config{Width: 128, Height: 64, Address: 0x3C, VccState: ssd1306.SWITCHCAPVCC}
	display.Configure(cfg)
	display.ClearDisplay()

	var dashboard *ui.ContainerBase[ui.Widget]
	pulseWidth.Store(int64(defaultPulseWidth))

	dashboard = ui.NewContainer[ui.Widget](
		uint16(WIDTH), 0, ui.LayoutVList(1),
		NewLabel(uint16(WIDTH), 9, func() string {
			return fmt.Sprintf("%d %01v", time.Millisecond, overallCalibration)
		}, white),
		NewLabel(uint16(WIDTH), 9, func() string {
			return fmt.Sprintf("%d (%01v)", waitMilli, time.Duration(waitMilliTuned))
		}, white),
		NewLabel(uint16(WIDTH), 9, func() string {
			return fmt.Sprintf("%d", dev.WaitCalibrationK)
		}, white),
		NewLabel(uint16(WIDTH), 9, func() string {
			return fmt.Sprintf("%d", dev.WaitCalibrationM)
		}, white),
		NewLabel(uint16(WIDTH), 9, func() string {
			pw := time.Duration(waitMilli)
			pw1 := time.Duration(waitMilliTuned / 10)
			return fmt.Sprintf("%01v %01v", pw, pw1)
		}, white),
	)
	dW, dH := dashboard.Size()
	ctx := ui.NewRandomContext(&display, time.Second*10, dW, dH)

	machine.Watchdog.Configure(machine.WatchdogConfig{
		TimeoutMillis: 3000,
	})
	machine.Watchdog.Start()

	go runUI(runButtons(), dashboard)
	runInterruptTest()
	// go runTest1()
	// go runTest2()

	// Drawing and moving display around
	ticker := time.NewTicker(time.Millisecond * 50)
	lastReset := time.Now()
	for range ticker.C {
		if time.Since(lastReset) > time.Second*210 {
			display.Configure(cfg)
			time.Sleep(time.Millisecond * 100)
			lastReset = time.Now()
		}
		display.ClearBuffer()
		dashboard.Draw(&ctx)
		display.Display()
		machine.Watchdog.Update()
	}
}

type InterruptTest struct {
	ch         chan time.Duration
	start, end time.Duration
}

//go:noinline
func (it *InterruptTest) handlePinInterrupt(p machine.Pin) {
	config.TEST1.High() // -> noinline: 4-10us
	if p.Get() {
		config.TEST1.High() // -> closure: 12-14us;  method: 6-10us; noinline: 4-10us
		config.TEST4.High() // -> ~12-14us
		// start = time.Now()  // 18uS
		it.start = dev.Now() // ~2us
		config.TEST4.Low()
		return
	}
	config.TEST1.Low() // -> closure: ~5us;   method: 4-5us
	it.end = dev.Now()
	config.TEST4.High() // -> 12us
	select {
	case it.ch <- it.end - it.start: // ~4-5us
	// case ch <- end.Sub(start): // ~4, <5us
	default:
	}
	config.TEST4.Low() // -> ~15us
}

func runInterruptTest() {
	test := InterruptTest{
		ch: make(chan time.Duration, 1),
	}
	config.TEST3.SetInterrupt(machine.PinToggle, test.handlePinInterrupt)

	go func() {
		for d := range test.ch {
			config.TEST2.High()                     // -> 50us up to 1.5ms!!!
			println(fmt.Sprintf("Duration: %v", d)) // 150us
			config.TEST2.Low()
		}
	}()
}

func runTest00() {
	for {
		config.TEST2.High()
		dev.Wait(1000 / 37)
		config.TEST2.Low()
		dev.Wait(1000 / 37)
	}
}
func runTest0() {
	for {
		config.TEST2.High()
		dev.Wait(time.Microsecond * 10)
		config.TEST2.Low()
		dev.Wait(time.Microsecond * 10)
	}
}
func runTest01() {
	for {
		config.TEST2.High()
		dev.WaitCalibrated(time.Microsecond * 10)
		config.TEST2.Low()
		dev.WaitCalibrated(time.Microsecond * 30)
	}
}

func runTest1() {
	pt := dev.NewPulseTrain(
		time.Microsecond*40, // pause
		time.Microsecond*10,
		time.Microsecond*30, //
		time.Microsecond*20,
		time.Microsecond*20, //
		time.Microsecond*40,
		time.Microsecond*10, //
		time.Microsecond*80,
		time.Microsecond*10, //
		time.Microsecond*100,
	)

	t := time.NewTicker(time.Millisecond * 50)
	for range t.C {
		pt.Run(config.TEST1)
	}
}

func runTest2() {
	for {
		config.TEST2.High()
		time.Sleep(time.Microsecond * 5)
		config.TEST2.Low()
		time.Sleep(time.Microsecond * 5)
	}
}
