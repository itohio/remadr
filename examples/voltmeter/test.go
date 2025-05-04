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
	"tinygo.org/x/drivers/ssd1306"
	"tinygo.org/x/tinyfont/freemono"
)

//go:generate tinygo flash -target=pico

var (
	WIDTH  int16 = 120
	HEIGHT int16 = 8
)
var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 0}
)

func main() {
	machine.InitADC()

	meter, _ := dev.NewVoltageMeter(3.3, 12, 100,
		[]machine.ADC{config.VoltageA, config.VoltageB},
		[]dev.Approximator[float32]{
			dev.NewOptoisolatorCTRModel[float32](2094, 10, 7911, 19),
			dev.NewOptoisolatorCTRModel[float32](2470, 10, 8360, 19),
		})

	machine.I2C0.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	// the delay is needed for display start from a cold reboot, not sure why
	time.Sleep(time.Second)
	display := ssd1306.NewI2C(machine.I2C0)
	cfg := ssd1306.Config{Width: 128, Height: 64, Address: 0x3C, VccState: ssd1306.SWITCHCAPVCC}
	display.Configure(cfg)
	display.ClearDisplay()

	var dashboard *ui.ContainerBase[ui.Widget]

	var (
		raw   []uint32  = make([]uint32, 2)
		volts []float32 = make([]float32, 2)
	)
	dashboard = ui.NewContainer[ui.Widget](
		uint16(WIDTH), 0, ui.LayoutVList(1),
		widget.NewLabelArray(uint16(WIDTH), 11, &freemono.Regular9pt7b, white,
			func() string {
				return fmt.Sprintf("%v", raw[0])
			},
			func() string {
				return fmt.Sprintf("%v", raw[1])
			},
			func() string {
				return fmt.Sprintf("%0.2f", volts[0])
			},
			func() string {
				return fmt.Sprintf("%0.2f", volts[1])
			})...,
	)
	dW, dH := dashboard.Size()
	ctx := ui.NewRandomContext(&display, time.Second*10, dW, dH)

	machine.Watchdog.Configure(machine.WatchdogConfig{
		TimeoutMillis: 3000,
	})
	machine.Watchdog.Start()

	meter.Configure()

	// Drawing and moving display around
	ticker := time.NewTicker(time.Millisecond * 50)
	lastReset := time.Now()
	for range ticker.C {
		volts = meter.ReadVoltages(volts)
		for i, r := range meter.Raw() {
			raw[i] = r / 100
		}

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
