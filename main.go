package main

import (
	// "fmt"
	"fmt"
	"image/color"
	"machine"
	"time"

	"github.com/itohio/remadr/config"
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
)

func main() {
	machine.InitADC()
	voltage.Configure(machine.ADCConfig{})

	encoder = encoders.NewQuadratureViaInterrupt(config.ButtonA, config.ButtonB)
	encoder.Configure(encoders.QuadratureConfig{Precision: 1})

	config.SenseA.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	config.SenseB.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	config.Button.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

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
		NewLabel(uint16(WIDTH), 20, func() string {
			pw := time.Duration(pulseWidth.Load())
			return fmt.Sprintf("%01v %v", pw, config.SenseA.Get())
		}, white),
		// NewLabel(uint16(WIDTH), 20, func() string {
		// 	return fmt.Sprintf("%v %v %v", encoderValue, lastEncoderValue, encoderDelta)
		// }, white),
		NewLabel(uint16(WIDTH), 20, func() string {
			pw := time.Duration(pulseWidth.Load())
			energy := avgVoltage * avgVoltage * float32(pw.Seconds()) / resistance
			return fmt.Sprintf("%0.01f J", energy)
		}, white),
		NewLabel(uint16(WIDTH), 20, func() string {
			v := voltageDivider * 3.2 * float32(voltage.Get()) / 0xFFFF
			avgVoltage = .7*avgVoltage + 0.3*v
			return fmt.Sprintf("%0.02f V", avgVoltage)
		}, white),
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
		config.TEST.High()
		dashboard.Draw(&ctx)
		display.Display()
		config.TEST.Low()
	}
}
