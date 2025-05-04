package dev

import (
	"errors"
	"machine"
)

var (
	ErrInvalidIndex            = errors.New("index")
	ErrPinApproximatorMismatch = errors.New("pin count")
)

type Approximator[T number] interface {
	Convert(uint16) T
}

// VoltageMeter reads voltage from multiple ADC pins
type VoltageMeter struct {
	adcs          []machine.ADC // ADC pins
	raw           []uint32
	readings      []float32
	approximators []Approximator[float32] // Linear approximators for each pin
	resolution    uint32                  // ADC resolution in bits
	reference     float32                 // Reference voltage (usually 3.3V or 5V)
	samples       uint8                   // Number of samples to average per reading
}

// NewVoltageMeter creates a new voltage meter with the specified pins and approximators
func NewVoltageMeter(reference float32, resolution uint8, samples uint8, adcPins []machine.ADC, approximators []Approximator[float32]) (*VoltageMeter, error) {
	// Check that the number of pins and approximators match
	if len(adcPins) != len(approximators) {
		return nil, ErrPinApproximatorMismatch
	}

	// Create and configure the voltage meter
	vm := &VoltageMeter{
		adcs:          adcPins,
		raw:           make([]uint32, len(adcPins)),
		approximators: approximators,
		resolution:    uint32(1<<resolution) - 1, // 2^resolution - 1
		reference:     reference,
		samples:       samples,
	}

	return vm, nil
}

func (vm *VoltageMeter) Configure() {
	for i := range vm.adcs {
		vm.adcs[i].Configure(machine.ADCConfig{})
	}
}

func (vm *VoltageMeter) readRaw(buf []uint32, N int) []uint32 {
	for i := range buf {
		buf[i] = 0
	}
	for ; N > 0; N-- {
		for i := range buf {
			buf[i] += uint32(vm.adcs[i].Get())
		}
	}

	return buf
}

func (vm *VoltageMeter) ReadRaw(buf []uint32, N int) []uint32 {
	if len(buf) > len(vm.adcs) {
		buf = buf[:len(vm.adcs)]
	}
	if buf == nil {
		buf = vm.raw
	}

	return vm.readRaw(buf, N)
}

func (vm *VoltageMeter) readVoltages(buf []float32) []float32 {
	vm.raw = vm.readRaw(vm.raw, int(vm.samples))
	for i, raw := range vm.raw {
		adc := raw / uint32(vm.samples)
		// Convert to voltage using the corresponding approximator
		buf[i] = vm.approximators[i].Convert(uint16(adc))
	}

	return buf
}

// ReadVoltages reads and converts the voltage from all ADC pins
func (vm *VoltageMeter) ReadVoltages(buf []float32) []float32 {
	if len(buf) > len(vm.adcs) {
		buf = buf[:len(vm.adcs)]
	}
	if buf == nil {
		buf = vm.readings
	}

	return vm.readVoltages(buf)
}

func (vm *VoltageMeter) Voltages() []float32 {
	return vm.readVoltages(vm.readings)
}

func (vm *VoltageMeter) Raw() []uint32 {
	return vm.raw
}
