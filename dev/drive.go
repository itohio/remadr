package dev

import (
	"iter"
	"machine"
	"sync/atomic"
	"time"
)

// Constants for the mass driver
const (
	maxPulseWidth = 2000 // Maximum pulse width in microseconds
	maxStages     = 8    // Maximum number of stages supported

	Idle   State = iota // Driver is idle
	Armed               // Driver is armed and waiting for projectile
	Active              // Driver is actively accelerating projectile
	Done
	Failed // Driver is actively accelerating projectile
)

type State uint8

type Stage interface {
	Arm() error
	Reset()
	Configure(mode machine.PinMode, pc machine.PinChange, afterTrigerred func(time.Duration)) error
}

type DwellTimer interface {
	DwellTime() time.Duration
}

type ActiveTimer interface {
	ActiveTime() time.Duration
}

type Shaper interface {
	Shape() []time.Duration
	SetShape(shape []time.Duration) error
}

// MassDriver manages the multi-stage acceleration system
type MassDriver struct {
	stages        []Stage // Array of stages
	timestamps    []time.Duration
	currentStage  atomic.Int32          // Index of current active stage
	state         atomic.Int32          // Current state of the driver
	errorCallback func(int8, error)     // Callback for error reporting (stage, message)
	doneCallback  func([]time.Duration) // Callback when sequence completes
}

// NewMassDriver creates a new mass driver with the specified stages
func NewMassDriver(errorCb func(int8, error), doneCb func([]time.Duration), stages ...Stage) *MassDriver {
	// Check if we exceed maximum stages
	if len(stages) > maxStages {
		panic("Too many stages specified")
	}
	if len(stages) == 0 {
		panic("Too few stages specified")
	}

	ret := &MassDriver{
		stages:        stages,
		errorCallback: errorCb,
		doneCallback:  doneCb,
		timestamps:    make([]time.Duration, len(stages)),
	}
	ret.setState(Idle, -1)

	return ret
}

func (c *MassDriver) Configure(mode machine.PinMode, pc machine.PinChange) error {
	// Configure all stage pins and interrupts
	for i := range c.stages {
		if err := c.stages[i].Configure(mode, pc, func(when time.Duration) {
			c.stageTriggered(int8(i))
			c.timestamps[i] = when
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *MassDriver) stageTriggered(stageIdx int8) {
	state, stage := d.getState()

	if stage != stageIdx {
		d.setState(Failed, stageIdx)
		return
	}
	if stage == 0 && state != Armed {
		d.setState(Failed, stageIdx)
		return
	} else if stage == 0 {
		state = Active
	}
	if state != Active {
		d.setState(Failed, stageIdx)
		return
	}
	stage = stageIdx + 1
	if stage < int8(len(d.stages)) {
		if err := d.stages[stage].Arm(); err != nil {
			state = Failed
		}
	} else {
		state = Done
	}

	d.setState(state, stage)
}

func (d *MassDriver) Reset() {
	d.setState(Idle, -1)
	// Reset all timestamps
	for i := range d.stages {
		d.stages[i].Reset()
		d.timestamps[0] = 0
	}
}

// Arm prepares the mass driver to start accelerating a projectile
func (d *MassDriver) Arm() error {
	// Check if already armed or active
	if d.State() != Idle {
		if d.errorCallback != nil {
			d.errorCallback(-1, ErrDriverBusy)
		}
		return ErrDriverBusy
	}

	// Reset all timestamps
	for i := range d.stages {
		d.stages[i].Reset()
	}

	// Arm the first stage
	d.setState(Armed, 0)
	if err := d.stages[0].Arm(); err != nil {
		if d.errorCallback != nil {
			d.errorCallback(0, err)
		}
		return err
	}

	return nil
}

// Abort immediately stops all actuators and resets the driver
func (d *MassDriver) Abort() {
	d.setState(Idle, -1)

	// Turn off all actuators
	for i := range d.stages {
		d.stages[i].Reset()
	}
}

func (d *MassDriver) setState(s State, idx int8) {
	d.state.Store(int32(s))
	d.currentStage.Store(int32(idx))

	if s == Done && d.doneCallback != nil {
		d.doneCallback(d.timestamps)
	}
}
func (d *MassDriver) getState() (State, int8) {
	return State(d.state.Load()),
		int8(d.currentStage.Load())
}

// State returns the current state of the driver
func (d *MassDriver) State() State {
	return State(d.state.Load())
}

// CurrentStage returns the index of the current active stage
func (d *MassDriver) CurrentStage() int8 {
	return int8(d.currentStage.Load())
}

func (d *MassDriver) GetStage(idx int) (Stage, error) {
	if idx >= len(d.stages) || idx < 0 {
		return nil, ErrInvalidStage
	}
	return d.stages[idx], nil
}

func (d *MassDriver) Stages() iter.Seq2[int, Stage] {
	return func(yield func(int, Stage) bool) {
		for i, stage := range d.stages {
			if !yield(i, stage) {
				return
			}
		}
	}
}
