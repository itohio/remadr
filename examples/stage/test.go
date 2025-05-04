package main

import (
	// "fmt"
	"fmt"
	"image/color"
	"machine"
	"time"

	"github.com/itohio/remadr/config"
	"github.com/itohio/remadr/dev"
	mdui "github.com/itohio/remadr/ui"
	ui "github.com/itohio/tinygui"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
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
	chrono  *dev.Chronograph
	stageA  *dev.ShapedPulseStage
	stageB  *dev.ShapedPulseStage
)

func main() {
	machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})

	config.TEST1.Configure(machine.PinConfig{Mode: machine.PinOutput})
	config.TEST2.Configure(machine.PinConfig{Mode: machine.PinOutput})

	println("Hello!")
	dev.CalibrateWait(time.Microsecond*2000, 50)
	machine.InitADC()
	println(fmt.Sprintf("Calibration %v / %v", dev.WaitCalibrationK, dev.WaitCalibrationM))

	stageA = dev.NewShapedPulseStage(config.TriggerA, config.SenseA,
		time.Microsecond*10,
		time.Microsecond*300,

		time.Microsecond*200,
		time.Microsecond*200,

		time.Microsecond*1800,
		time.Microsecond*200,

		time.Microsecond*200,
		time.Microsecond*200,

		time.Microsecond*200,
		time.Microsecond*200,

		time.Microsecond*100,
		time.Microsecond*800,
	)
	stageA.Configure(machine.PinInput, machine.PinFalling, func(when time.Duration) { println("triggered A! " + when.String()) })
	stageB = dev.NewShapedPulseStage(config.TriggerB, config.SenseB,
		time.Microsecond*2000,
		time.Microsecond*1000,
	)
	stageB.Configure(machine.PinInput, machine.PinFalling, func(when time.Duration) { println("triggered B!" + when.String()) })

	var (
		count      int
		speed      float64
		lenA, lenB float64
	)

	chrono = dev.NewChronograph(
		config.ChronoA, config.ChronoB,
		78.39,
		nil,
	)
	shotCh := make(chan struct{}, 1)
	chrono.SetCallback(func(f float64) {
		speed = f
		dA, dB := chrono.Durations()
		lenA = speed * float64(dA) / 1000
		lenB = speed * float64(dB) / 1000
		count++
		shotCh <- struct{}{}
		println("shot!")
	})
	chrono.Configure(machine.PinInput, machine.PinFalling)

	go func() {
		for range shotCh {
			d1, d2 := chrono.Durations()
			println(fmt.Sprintf("SHOT #%d:\n V = %f m/s (%f)\n v1 = %f m/s\n v2 = %f m/s\n  dwellA = %v\n  dwellB = %v", count,
				chrono.Speed(), speed, .02/d1.Seconds(), .02/d2.Seconds(),
				stageA.DwellTime(), stageB.DwellTime(),
			))
		}
	}()

	encoder = encoders.NewQuadratureViaInterrupt(config.ButtonA, config.ButtonB)
	encoder.Configure(encoders.QuadratureConfig{Precision: 1})
	config.Button.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	machine.I2C0.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	// the delay is needed for display start from a cold reboot, not sure why
	time.Sleep(time.Second)
	display := ssd1306.NewI2C(machine.I2C0)
	cfg := ssd1306.Config{Width: 128, Height: 64, Address: 0x3C, VccState: ssd1306.SWITCHCAPVCC}
	display.Configure(cfg)
	display.ClearDisplay()

	var dashboard *ui.ContainerBase[ui.Widget]

	dashboard = ui.NewContainer[ui.Widget](
		uint16(WIDTH), 0, ui.LayoutVList(1),
		mdui.NewLabel(uint16(WIDTH), 12, func() string {
			return fmt.Sprintf("%v %v", count, chrono.IsValid())
		}, white),
		mdui.NewLabel(uint16(WIDTH), 11, func() string {
			return fmt.Sprintf("%0.3f m/s", speed)
		}, white),
		mdui.NewLabel(uint16(WIDTH), 11, func() string {
			return fmt.Sprintf("%0.5f", lenA)
		}, white),
		mdui.NewLabel(uint16(WIDTH), 11, func() string {
			return fmt.Sprintf("%0.5f", lenB)
		}, white),
	)
	dW, dH := dashboard.Size()
	ctx := ui.NewRandomContext(&display, time.Second*10, dW, dH)

	machine.Watchdog.Configure(machine.WatchdogConfig{
		TimeoutMillis: 3000,
	})
	machine.Watchdog.Start()

	// Drawing and moving display around
	ticker := time.NewTicker(time.Millisecond * 50)
	lastReset := time.Now()

	go runUI(runButtons(), dashboard)

	println("Start loop!")
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
