package main

import (
	"runtime/volatile"
	"time"
	"unsafe"
)

//go:linkname ticks runtime.machineTicks
func ticks() uint64

const (
	TIMER_BASE      = 0x40054000        // Base address of the RP2040 timer
	TIMER_TIMERAWL  = TIMER_BASE + 0x08 // Low 32 bits of the raw timer register
	TIMER_TIMERAWH  = TIMER_BASE + 0x0C // High 32 bits of the raw timer register
	SYSTEM_CLOCK_HZ = 125_000_000       // Default system clock (125 MHz)
)

// readRawTimer returns the current system clock timer value in nanoseconds.
func readRawTimer() uint64 {
	low := (*volatile.Register32)(unsafe.Pointer(uintptr(TIMER_TIMERAWL)))
	high := (*volatile.Register32)(unsafe.Pointer(uintptr(TIMER_TIMERAWH)))
	return (uint64(high.Get()) << 32) | uint64(low.Get())
}

// delayNanoseconds creates a delay with nanosecond precision.
func delayNanoseconds(ns time.Duration) {
	start := readRawTimer()
	targetCycles := uint64(ns*SYSTEM_CLOCK_HZ) / 1_000_000_000 // Convert ns to clock cycles
	for {
		elapsed := readRawTimer() - start
		if elapsed >= targetCycles {
			break
		}
	}
}

// delayNanoseconds creates a precise delay for the specified nanoseconds using CPU cycles.
func delayNanoseconds1(ns time.Duration) {
	cycles := uint64(ns*125) / 11_000

	// Prevent preemption by disabling garbage collection temporarily
	// runtime.LockOSThread()
	var dummy volatile.Register32

	// Wait for the required number of cycles
	// machine.DisableInterrupts()
	for i := uint64(0); i < cycles; i++ {
		dummy.Get() // Volatile operation to avoid optimization
	}
	// machine.EnableInterrupts()

	// Re-enable preemption
	// runtime.UnlockOSThread()
}

// delayNanoseconds creates a precise delay for the specified nanoseconds using CPU cycles.
func delayNanoseconds2(ns time.Duration) {
	cycles := uint64(ns / 1_000)

	// Wait for the required number of cycles
	for i := ticks(); ticks()-i < cycles; {
	}
}
