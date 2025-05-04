package dev

import (
	"device"
	"time"
	_ "unsafe"
)

var (
	// Wait calibration constant. Actual `nop` loop value is duration * K / M. Default value for rp2040.
	WaitCalibrationK time.Duration = 80339
	// Wait calibration constant. Actual `nop` loop value is duration * K / M. Default value for rp2040.
	WaitCalibrationM time.Duration = 1000000
)

//go:linkname ticks runtime.ticks
func ticks() uint64

//go:linkname ticksToNanoseconds runtime.ticksToNanoseconds
func ticksToNanoseconds(ticks uint64) int64

//go:inline
func Now() time.Duration {
	return time.Duration(ticksToNanoseconds(ticks()))
}

// Helper to compute the greatest common divisor (GCD)
func Gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Helper to compute the least common multiple (LCM)
func lcm(a, b int64) int64 {
	return a * b / Gcd(a, b)
}

// Compute k and m such that A * k = B * m
func scaleConstants(a, b int64) (int64, int64) {
	l := lcm(a, b)
	return l / a, l / b
}

func SetWaitCalibration(wanted, actual time.Duration) {
	k, m := scaleConstants(int64(actual), int64(wanted))
	WaitCalibrationK = time.Duration(k)
	WaitCalibrationM = time.Duration(m)
}

func CalibrateWait(d time.Duration, n int) {
	actual := BenchmarkWait(d, n)
	SetWaitCalibration(d, actual)
}

func BenchmarkWait(d time.Duration, n int) time.Duration {
	t1 := ticks()
	for i := 0; i < n; i++ {
		Wait(d)
	}

	return time.Duration(ticksToNanoseconds(ticks()-t1)) / time.Duration(n)
}

//go:inline
func Wait(wait time.Duration) {
	for ; wait > 0; wait-- {
		device.Asm(`nop`)
	}
}

func WaitTicks(wait time.Duration) {
	t1 := Now()
	for Now()-t1 < wait {
		device.Asm(`nop`)
	}
}

//go:inline
func WaitCalibrated(wait time.Duration) {
	Wait((wait * WaitCalibrationK) / WaitCalibrationM)
}
