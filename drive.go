package main

import (
	"sync/atomic"
	"time"

	"github.com/itohio/remadr/config"
)

var (
	capacitance float32 = (10000 + 4*1500) * 1e-6
	resistance  float32 = 1
	pulseWidth  atomic.Int64

	avgVoltage float32
)

const (
	defaultPulseWidth = time.Millisecond * 1
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

func trigger() {
}

func pinBenchmark() {
	for {
		config.TEST.High()
		delayNanoseconds2(time.Microsecond * 10)
		config.TEST.Low()
		delayNanoseconds2(time.Microsecond * 60)
	}
}
