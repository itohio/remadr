package dev

type number interface {
	float32 | float64
}

// LinearApproximator converts raw ADC values to voltage using linear approximation
type LinearApproximator[T number] struct {
	// y = mx + b
	m T // Slope
	b T // Y-intercept
}

// NewLinearApproximatorFromPoints creates a new approximator from two calibration points
// (adcValue1, voltage1) and (adcValue2, voltage2)
func NewLinearApproximatorFromPoints[T number](adcValue1 uint16, voltage1 T, adcValue2 uint16, voltage2 T) LinearApproximator[T] {
	// Calculate slope (m)
	m := (voltage2 - voltage1) / T(adcValue2-adcValue1)

	// Calculate y-intercept (b)
	b := voltage1 - m*T(adcValue1)

	return LinearApproximator[T]{
		m: m,
		b: b,
	}
}

// NewLinearApproximator creates a new approximator directly using slope and intercept
func NewLinearApproximator[T number](slope T, intercept T) LinearApproximator[T] {
	return LinearApproximator[T]{
		m: slope,
		b: intercept,
	}
}

// Convert transforms a raw ADC value to a voltage
func (la LinearApproximator[T]) Convert(adcValue uint16) T {
	return la.m*T(adcValue) + la.b
}
