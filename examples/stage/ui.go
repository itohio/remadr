package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers"

	"github.com/itohio/remadr/config"
	ui "github.com/itohio/tinygui"
)

var (
	lastEncoderValue int
	encoderValue     int
	encoderDelta     int
)

func hLine(d drivers.Displayer, x, y, w int16, c color.RGBA) {
	for w > 0 {
		d.SetPixel(x, y, c)
		x++
		w--
	}
}
func vLine(d drivers.Displayer, x, y, h int16, c color.RGBA) {
	for h > 0 {
		d.SetPixel(x, y, c)
		y++
		h--
	}
}

func button(p machine.Pin) time.Duration {
	now := time.Now()
	time.Sleep(time.Millisecond * 10)
	for !p.Get() {
		time.Sleep(time.Millisecond)
		machine.Watchdog.Update()
	}
	return time.Since(now)
}

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
				d := button(config.Button)
				if d > time.Second {
					if d > time.Second*5 {
						cmd(ui.RESET)
					}
					cmd(ui.ESC)
				} else {
					cmd(ui.ENTER)
				}
			}
			time.Sleep(time.Millisecond * 10)
			if N == 0 {
				cmd(ui.IDLE)
			}
			N = (N + 1) % 100
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
				machine.CPUReset()
			case ui.ENTER:
				println("enter")
				chrono.Reset()
				println("stageA.Reset")
				stageA.Reset()
				println("stageB.Reset")
				stageB.Reset()
				println("stageA.Arm")
				stageA.Arm()
				println("stageB.Arm")
				stageB.Arm()
				println("Enter.Done")
			case ui.ESC:
			case ui.IDLE:
			}
		}
	}
}
