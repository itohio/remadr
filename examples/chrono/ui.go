package main

import (
	"machine"
	"time"

	"github.com/itohio/remadr/config"
	ui "github.com/itohio/tinygui"
)

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
			switch {
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
				chrono.Reset()
			case ui.ESC:
			case ui.IDLE:
			}
		}
	}
}
