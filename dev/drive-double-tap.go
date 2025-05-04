package dev

import (
	"machine"
	"time"
)

// DoubleTapStage represents a single stage of the mass driver.
// This stage triggers the coil two times after `delay` time of the first detection of the projectile.
type DoubleTapStage struct {
	*ShapedPulseStage
}

func NewDoubleTapStage(trigger, sense machine.Pin, firstDelay, firstPulse, secondDelay, secondPulse time.Duration) *DoubleTapStage {
	ret := &DoubleTapStage{
		ShapedPulseStage: NewShapedPulseStage(trigger, sense, firstDelay, firstPulse, secondDelay, secondPulse),
	}
	return ret
}

func (s *DoubleTapStage) SetDelay(d time.Duration) {
	s.ShapedPulseStage.Shape()[0] = d
}

func (s *DoubleTapStage) SetDuration(d time.Duration) {
	s.ShapedPulseStage.Shape()[1] = d
}
