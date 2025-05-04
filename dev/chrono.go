package dev

import (
	"machine"
	"runtime/volatile"
	"time"
)

// Chronograph measures the speed of a projectile passing through two sensors
type Chronograph struct {
	a, b machine.Pin // Input pins for sensors
	pc   machine.PinChange

	distanceMM             float64 // Distance between sensors in millimeters
	timeStartA, timeStartB uint64
	timeEndA, timeEndB     uint64

	callback func(float64) // Callback function to report speed
	val      bool
}

// NewChronograph creates a new chronograph
func NewChronograph(
	a, b machine.Pin,
	distanceMM float64,
	callback func(float64),
) *Chronograph {
	ret := &Chronograph{
		a:          a,
		b:          b,
		distanceMM: distanceMM,
		callback:   callback,
	}

	return ret
}

func (c *Chronograph) Configure(mode machine.PinMode, pc machine.PinChange) error {
	if mode != machine.PinInput && mode != machine.PinInputPulldown && mode != machine.PinInputPullup {
		return ErrInvalidPinMode
	}
	if pc == machine.PinToggle {
		return ErrInvalidPinChange
	}

	// Set pins as inputs
	c.a.Configure(machine.PinConfig{Mode: mode})
	c.b.Configure(machine.PinConfig{Mode: mode})

	c.val = pc == machine.PinRising
	if c.a.Get() == c.val || c.b.Get() == c.val {
		return ErrSensePin
	}

	c.a.SetInterrupt(machine.PinToggle, c.handleA)
	c.b.SetInterrupt(machine.PinToggle, c.handleB)
	return nil
}

//go:noinline
func (c *Chronograph) handleA(pin machine.Pin) {
	if !(volatile.LoadUint64(&c.timeStartA) == 0 || volatile.LoadUint64(&c.timeEndA) == 0) {
		return
	}
	if pin.Get() == c.val {
		volatile.StoreUint64(&c.timeStartA, ticks())
	} else {
		volatile.StoreUint64(&c.timeEndA, ticks())
	}
	c.processEvent()
}

//go:noinline
func (c *Chronograph) handleB(pin machine.Pin) {
	if !(volatile.LoadUint64(&c.timeStartB) == 0 || volatile.LoadUint64(&c.timeEndB) == 0) {
		return
	}
	if pin.Get() == c.val {
		volatile.StoreUint64(&c.timeStartB, ticks())
	} else {
		volatile.StoreUint64(&c.timeEndB, ticks())
	}
	c.processEvent()
}

func (c *Chronograph) SetCallback(f func(float64)) {
	c.callback = f
}

// Reset resets the chronograph and prepares it for a new measurement
func (c *Chronograph) Reset() {
	volatile.StoreUint64(&c.timeStartA, 0)
	volatile.StoreUint64(&c.timeEndA, 0)
	volatile.StoreUint64(&c.timeStartB, 0)
	volatile.StoreUint64(&c.timeEndB, 0)
}

// processEvent calculates speed when both sensors have been triggered
func (c *Chronograph) processEvent() {
	// Check if both sensors have finished being triggered
	if !(volatile.LoadUint64(&c.timeEndA) == 0 || volatile.LoadUint64(&c.timeEndB) == 0) {
		if c.callback == nil {
			return
		}
		c.callback(c.Speed())
	}
}

// Speed returns the last measured speed
func (c *Chronograph) Speed() float64 {
	timeDiff := time.Duration(ticksToNanoseconds(volatile.LoadUint64(&c.timeStartB) - volatile.LoadUint64(&c.timeStartA)))
	return (c.distanceMM / timeDiff.Seconds()) * 0.001
}

func (c *Chronograph) Durations() (time.Duration, time.Duration) {
	return time.Duration(ticksToNanoseconds(volatile.LoadUint64(&c.timeEndA) - volatile.LoadUint64(&c.timeStartA))), time.Duration(ticksToNanoseconds(volatile.LoadUint64(&c.timeEndB) - volatile.LoadUint64(&c.timeStartB)))
}

func (c *Chronograph) IsValid() bool {
	return ((volatile.LoadUint64(&c.timeStartA) != 0) && (volatile.LoadUint64(&c.timeEndA) != 0) && (volatile.LoadUint64(&c.timeStartB) != 0) && (volatile.LoadUint64(&c.timeEndB) != 0)) ||
		((volatile.LoadUint64(&c.timeStartA) == 0) && (volatile.LoadUint64(&c.timeEndA) == 0) && (volatile.LoadUint64(&c.timeStartB) == 0) && (volatile.LoadUint64(&c.timeEndB) == 0))
}
