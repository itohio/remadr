package dev

import "math"

// QuadraticApproximator converts raw ADC values to voltage using quadratic approximation
// This is useful for modeling non-linear relationships like optoisolator CTR curves
// The formula is: y = ax² + bx + c
type QuadraticApproximator[T number] struct {
	a T // Coefficient of x²
	b T // Coefficient of x
	c T // Constant term
}

// NewQuadraticApproximatorFromPoints creates a new approximator from three calibration points
// (x1, y1), (x2, y2), and (x3, y3) where:
// - x values are raw ADC readings
// - y values are the corresponding voltage/current measurements
func NewQuadraticApproximatorFromPoints[T number](
	x1 uint16, y1 T,
	x2 uint16, y2 T,
	x3 uint16, y3 T) QuadraticApproximator[T] {

	// Convert uint16 to T for calculations
	tx1, tx2, tx3 := T(x1), T(x2), T(x3)

	// Calculate the coefficients using the three-point formula
	// This solves the system of equations:
	// y1 = a*x1² + b*x1 + c
	// y2 = a*x2² + b*x2 + c
	// y3 = a*x3² + b*x3 + c

	// Calculate denominator for Cramer's rule
	denominator := (tx1 - tx2) * (tx1 - tx3) * (tx2 - tx3)

	// Calculate coefficient a
	a := (tx3*(y2-y1) + tx2*(y1-y3) + tx1*(y3-y2)) / denominator

	// Calculate coefficient b
	b := (tx3*tx3*(y1-y2) + tx2*tx2*(y3-y1) + tx1*tx1*(y2-y3)) / denominator

	// Calculate coefficient c
	c := (tx2*tx3*(tx2-tx3)*y1 + tx1*tx3*(tx3-tx1)*y2 + tx1*tx2*(tx1-tx2)*y3) / denominator

	return QuadraticApproximator[T]{
		a: a,
		b: b,
		c: c,
	}
}

// NewQuadraticApproximator creates a new approximator directly using calculated coefficients
func NewQuadraticApproximator[T number](a, b, c T) QuadraticApproximator[T] {
	return QuadraticApproximator[T]{
		a: a,
		b: b,
		c: c,
	}
}

// Convert transforms a raw ADC value to a voltage
func (qa QuadraticApproximator[T]) Convert(adcValue uint16) T {
	x := T(adcValue)
	return qa.a*x*x + qa.b*x + qa.c
}

// ConvertInverse finds the approximate ADC value that would produce the target output
// This is useful for finding what input is needed to achieve a specific output level
func (qa QuadraticApproximator[T]) ConvertInverse(targetValue T) uint16 {
	// Using quadratic formula: x = (-b ± sqrt(b² - 4ac)) / 2a
	// Where we're solving for: a*x² + b*x + (c - targetValue) = 0

	a := qa.a
	b := qa.b
	c := qa.c - targetValue

	// Calculate discriminant
	discriminant := b*b - 4*a*c

	// Handle cases where there's no real solution
	if discriminant < 0 {
		// Return closest possible value as fallback
		if c < 0 {
			return math.MaxUint16 // Target is above our range
		}
		return 0 // Target is below our range
	}

	// Calculate the two possible solutions
	sqrtDiscriminant := T(math.Sqrt(float64(discriminant)))

	// We usually want the positive solution for physical measurements
	x1 := (-b + sqrtDiscriminant) / (2 * a)
	x2 := (-b - sqrtDiscriminant) / (2 * a)

	// Choose the solution that makes more physical sense
	// For optoisolators, usually the smaller positive value is what we want
	var result T
	if x1 >= 0 && (x2 < 0 || x1 < x2) {
		result = x1
	} else {
		result = x2
	}

	// Ensure result is in valid range and convert to uint16
	if result < 0 {
		return 0
	}
	if result > T(math.MaxUint16) {
		return math.MaxUint16
	}

	return uint16(result + 0.5) // Round to nearest integer
}
