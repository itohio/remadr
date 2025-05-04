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
	"github.com/itohio/tinygui/widget"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont/proggy"
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
	dev.CalibrateWait(time.Microsecond*2000, 50)
	machine.InitADC()
	encoder = encoders.NewQuadratureViaInterrupt(config.ButtonA, config.ButtonB)
	encoder.Configure(encoders.QuadratureConfig{Precision: 1})
	config.Button.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	println(fmt.Sprintf("Calibration %v / %v", dev.WaitCalibrationK, dev.WaitCalibrationM))

	configureVoltage()
	configureDriver(1)
	configureChrono()

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
		widget.NewLabelArray(uint16(WIDTH), 9, &proggy.TinySZ8pt7b, white,
			func() string {
				return fmt.Sprintf("%v %v", config.SenseA.Get(), config.SenseB.Get())
			},
			func() string {
				return fmt.Sprintf("%v %v", config.ChronoA.Get(), config.ChronoB.Get())
			},
			func() string {
				return fmt.Sprintf("%0.2f", voltages[0])
			},
			func() string {
				return fmt.Sprintf("%0.2f", voltages[1])
			},
		)...,
	)
	dW, dH := dashboard.Size()
	ctx := ui.NewRandomContext(&display, time.Second*1, dW, dH)

	machine.Watchdog.Configure(machine.WatchdogConfig{
		TimeoutMillis: 3000,
	})
	machine.Watchdog.Start()

	go runUI(runButtons(), dashboard)

	// Drawing and moving display around
	ticker := time.NewTicker(time.Millisecond * 50)
	lastReset := time.Now()
	for range ticker.C {
		if time.Since(lastReset) > time.Second*10 {
			display.Configure(cfg)
			time.Sleep(time.Millisecond * 100)
			lastReset = time.Now()
		}
		display.ClearBuffer()
		dashboard.Draw(&ctx)
		display.Display()

		meter.ReadVoltages(voltages[:])
	}
}
