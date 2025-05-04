package main

import (
	"machine"
	"sync/atomic"
	"time"

	"github.com/itohio/remadr/config"
	ui "github.com/itohio/tinygui"
)

var (
	lastEncoderValue int
	encoderValue     int
	encoderDelta     int
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
				println("up")
				pulseIncrease(1)
			case ui.DOWN:
				println("down")
				pulseDecrease(1)
			case ui.LONG_UP:
				println("long up")
			case ui.LONG_DOWN:
				println("long down")
			case ui.RESET:
				println("reset")
				machine.CPUReset()
			case ui.ENTER:
				println("enter")
				config.TriggerA.High()
				time.Sleep(time.Duration(pulseWidth.Load()))
				config.TriggerA.Low()
			case ui.ESC:
				println("esc")
			case ui.IDLE:
			}
		}
	}
}

func pulseIncrease(d int) {
	pw := time.Duration(pulseWidth.Load())
	l := pow10(log10(pw) - 1)

	pulseWidth.Store(int64(pw + time.Duration(d)*l))
}

func pulseDecrease(d int) {
	pw := time.Duration(pulseWidth.Load())
	l := pow10(log10(pw) - 1)

	pw -= time.Duration(d) * l

	if pw < minPulseWidth {
		pw = minPulseWidth
	}

	pulseWidth.Store(int64(pw))
}

var pulseWidth atomic.Int64

const (
	defaultPulseWidth = time.Microsecond * 100
	minPulseWidth     = time.Microsecond * 10
)

// pow10 computes 10^exp, where exp is given as a time.Duration.
// It treats the duration (in nanoseconds) as the integer exponent.
// If exp is negative, it returns 0 since 10^negative is undefined for integers.
func pow10(exp time.Duration) time.Duration {
	if exp < 0 {
		return 0
	}

	result := time.Duration(1)
	for i := time.Duration(0); i < exp; i++ {
		result *= 10
	}
	return result
}

// log10 calculates the integer base-10 logarithm of n, where n is a time.Duration.
// It returns the largest time.Duration `x` such that 10^x <= n.
// If n <= 0, it returns an error.
func log10(n time.Duration) time.Duration {
	if n <= 0 {
		return 0
	}

	log := time.Duration(0)
	for n >= 10 {
		n /= 10
		log++
	}
	return log
}
