package dev

import (
	"machine"
	"sync/atomic"
	"time"
)

type PulseTrain struct {
	durations []time.Duration
	r         uint32
}

func NewPulseTrain(d ...time.Duration) PulseTrain {
	return PulseTrain{durations: d}
}

func (pt PulseTrain) Run(p machine.Pin) (triggerStart time.Duration) {
	atomic.StoreUint32(&pt.r, 1)
	for i, t := range pt.durations {
		if atomic.LoadUint32(&pt.r) == 0 {
			return triggerStart
		}
		if i%2 == 0 {
			WaitCalibrated(t)
			continue
		}
		if i == 1 {
			triggerStart = Now()
		}
		if atomic.LoadUint32(&pt.r) == 0 {
			return triggerStart
		}
		p.High()
		WaitCalibrated(t)
		p.Low()
	}
	return triggerStart
}

func (pt PulseTrain) Abort(p machine.Pin) {
	p.Low()
	atomic.StoreUint32(&pt.r, 1)
}

func (pt PulseTrain) Durations() []time.Duration {
	return pt.durations
}
