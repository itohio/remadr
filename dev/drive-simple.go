package dev

import (
	"machine"
	"time"
)

// SimpleStage represents a single stage of the mass driver.
// This stage simply triggers the coil after `delay` time of the first detection of the projectile.
type SimpleStage struct {
	*ShapedPulseStage
}

func NewSimpleStage(trigger, sense machine.Pin, delay, duration time.Duration) *SimpleStage {
	ret := &SimpleStage{
		ShapedPulseStage: NewShapedPulseStage(trigger, sense, delay, duration),
	}
	return ret
}

func (s *SimpleStage) SetDelay(d time.Duration) {
	s.ShapedPulseStage.Shape()[0] = d
}

func (s *SimpleStage) SetDuration(d time.Duration) {
	s.ShapedPulseStage.Shape()[1] = d
}
