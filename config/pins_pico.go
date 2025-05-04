//go:build rp2040

package config

import "machine"

var (
	TriggerA = machine.GP14
	SenseA   = machine.GP18
	VoltageA = machine.ADC{Pin: machine.ADC0}

	TriggerB = machine.GP15
	SenseB   = machine.GP19
	VoltageB = machine.ADC{Pin: machine.ADC1}

	ChronoA = machine.GP20
	ChronoB = machine.GP21

	TEST1 = machine.GP10
	TEST2 = machine.GP11
	TEST3 = machine.GP12
	TEST4 = machine.GP13

	Button  = machine.GP28
	ButtonA = machine.GP7
	ButtonB = machine.GP6
)

const (
	WaitCalibrationK = 80339
	WaitCalibrationM = 1000000
)
