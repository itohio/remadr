package dev

import (
	"machine"
	"runtime/volatile"
	"time"
)

// ShapedPulseStage represents a single stage of the mass driver.
// This stage simply triggers the coil after `delay` time of the first detection of the projectile.
type ShapedPulseStage struct {
	trigger      machine.Pin   // Output pin to trigger the actuator
	sense        machine.Pin   // Input pin to sense projectile
	shape        PulseTrain    // Delay after sensing before triggering
	senseStart   uint64        // Timestamp when projectile was sensed entering the sensor
	senseEnd     uint64        // Timestamp when projectile was sensed leaving the sensor
	triggerStart time.Duration // Timestamp when actuator was triggered
	triggerEnd   time.Duration // Timestamp when actuator was turned off
	state        State
	t            chan struct{}
	val          bool
}

func NewShapedPulseStage(trigger, sense machine.Pin, shape ...time.Duration) *ShapedPulseStage {
	ret := &ShapedPulseStage{
		trigger: trigger,
		sense:   sense,
		state:   Idle,
		t:       make(chan struct{}, 1),
	}

	if err := ret.SetShape(shape); err != nil {
		panic(err)
	}

	return ret
}

func (s *ShapedPulseStage) Configure(mode machine.PinMode, pc machine.PinChange, onComplete func(time.Duration)) error {
	if mode != machine.PinInput && mode != machine.PinInputPulldown && mode != machine.PinInputPullup {
		return ErrInvalidPinMode
	}
	if pc == machine.PinToggle {
		return ErrInvalidPinChange
	}

	s.sense.Configure(machine.PinConfig{Mode: mode})

	s.trigger.Configure(machine.PinConfig{Mode: machine.PinOutput})
	s.trigger.Low()

	// TODO: check if the stage is functioning properly

	go s.handlePulse(onComplete)

	// Setup interrupt handler with stage index
	s.val = pc == machine.PinRising
	s.sense.SetInterrupt(machine.PinToggle, s.handleInterrupt)

	return nil
}

//go:noinline
func (s *ShapedPulseStage) handleInterrupt(pin machine.Pin) {
	state := s.getState()
	if state != Armed && state != Active && state != Done {
		return
	}
	if !(volatile.LoadUint64(&s.senseStart) == 0 || volatile.LoadUint64(&s.senseEnd) == 0) {
		return
	}
	if pin.Get() == s.val {
		volatile.StoreUint64(&s.senseStart, ticks())
		s.t <- struct{}{}
	} else {
		volatile.StoreUint64(&s.senseEnd, ticks())
		s.abortPulse()
	}
}

func (s *ShapedPulseStage) Arm() error {
	s.setState(Armed)
	if s.sense.Get() == s.val {
		volatile.StoreUint64(&s.senseStart, ticks())
		s.t <- struct{}{}
	}
	return nil
}

func (s *ShapedPulseStage) Reset() {
	select {
	case <-s.t:
	default:
	}
	s.setState(Idle)
	volatile.StoreUint64(&s.senseStart, 0)
	volatile.StoreUint64(&s.senseEnd, 0)
	s.triggerStart = 0
	s.triggerEnd = 0
}

func (s *ShapedPulseStage) handlePulse(onComplete func(time.Duration)) {
	for range s.t {
		if s.getState() != Armed {
			continue
		}
		s.setState(Active)

		s.triggerStart = s.shape.Run(s.trigger)
		s.triggerEnd = Now()
		if onComplete != nil {
			onComplete(time.Duration(ticksToNanoseconds(volatile.LoadUint64(&s.senseStart))))
		}
		s.setState(Done)
	}
}

func (s *ShapedPulseStage) abortPulse() {
	s.shape.Abort(s.trigger)
}

func (s *ShapedPulseStage) Shape() []time.Duration {
	return s.shape.Durations()
}

func (s *ShapedPulseStage) SetShape(shape []time.Duration) error {
	if len(shape) < 2 || len(shape)%2 != 0 {
		return ErrInvalidPulseTrain
	}
	s.shape = NewPulseTrain(shape...)
	return nil
}

func (s *ShapedPulseStage) setState(state State) {
	volatile.StoreUint8((*uint8)(&s.state), uint8(state))
}
func (s *ShapedPulseStage) getState() State {
	return State(volatile.LoadUint8((*uint8)(&s.state)))
}

// DwellTime calculate the time projectile spend occluding the sensor.
// Knowing projectile length it is possible to calculate the average projectile velocity.
func (s *ShapedPulseStage) DwellTime() time.Duration {
	return time.Duration(ticksToNanoseconds(volatile.LoadUint64(&s.senseEnd) - volatile.LoadUint64(&s.senseStart)))
}

// ActiveTime calculates how long the stage activation took.
func (s *ShapedPulseStage) ActiveTime() time.Duration {
	return s.triggerEnd - s.triggerStart
}
