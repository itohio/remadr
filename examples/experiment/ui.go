package main

import (
	"machine"
	"time"

	"github.com/itohio/remadr/config"
	ui "github.com/itohio/tinygui"
)

var (
	lastEncoderValue int
	encoderValue     int
	encoderDelta     int
)

func runButtons() chan ui.UserCommand {
	command := make(chan ui.UserCommand)

	cmd := func(c ui.UserCommand) {
		select {
		case command <- c:
		default:
		}
	}

	go func() {
		var (
			N int
		)
		for {
			lastEncoderValue = encoderValue
			encoderValue = encoder.Position()
			encoderDelta = (encoderValue - lastEncoderValue) / 2

			switch {
			case encoderDelta > 1:
				cmd(ui.LONG_UP)
			case encoderDelta < -1:
				cmd(ui.LONG_DOWN)
			case encoderDelta > 0:
				cmd(ui.UP)
			case encoderDelta < 0:
				cmd(ui.DOWN)
			case !config.Button.Get():
				d := ui.PeekButton(config.Button)
				if d > time.Second {
					if d > time.Second*5 {
						cmd(ui.RESET)
					}
					cmd(ui.ESC)
				} else {
					cmd(ui.ENTER)
				}
			}
			time.Sleep(time.Millisecond)
			if N == 0 {
				cmd(ui.IDLE)
			}
			N = (N + 1) % 10
		}
	}()

	return command
}

func runUI(cmd chan ui.UserCommand, w *ui.ContainerBase[ui.Widget]) {
	for {
		select {
		case c := <-cmd:
			machine.Watchdog.Update()
			if w.Interact(c) {
				continue
			}

			switch c {
			case ui.UP:
			case ui.DOWN:
			case ui.LONG_UP:
			case ui.LONG_DOWN:
			case ui.RESET:
			case ui.ENTER:
				meter.ReadVoltages(voltagesPreShot[:])
				chrono.Reset()
				driver.Reset()
				driver.Arm()
			case ui.ESC:
			case ui.IDLE:
			}
		}
	}
}
