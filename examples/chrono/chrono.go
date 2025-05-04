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

	chrono     *dev.Chronograph
	count      int
	speed      float64
	lenA, lenB float64
)

func main() {
	println("Hello!")
	machine.InitADC()

	chrono = dev.NewChronograph(
		// config.SenseA, config.SenseB,
		config.ChronoA, config.ChronoB,
		78.39,
		nil,
	)
	chrono.SetCallback(func(f float64) {
		println("Shot!")
		speed = f
		dA, dB := chrono.Durations()
		lenA = speed * float64(dA) / 1000
		lenB = speed * float64(dB) / 1000
		count++
	})
	chrono.Configure(machine.PinInput, machine.PinFalling)

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
		widget.NewLabel(uint16(WIDTH), 12, &freemono.Regular9pt7b, func() string {
			return fmt.Sprintf("%v %v", count, chrono.IsValid())
		}, white),
		widget.NewLabel(uint16(WIDTH), 11, &freemono.Regular9pt7b, func() string {
			return fmt.Sprintf("%0.3f m/s", speed)
		}, white),
		widget.NewLabel(uint16(WIDTH), 11, &freemono.Regular9pt7b, func() string {
			return fmt.Sprintf("%0.5f", lenA)
		}, white),
		widget.NewLabel(uint16(WIDTH), 11, &freemono.Regular9pt7b, func() string {
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
