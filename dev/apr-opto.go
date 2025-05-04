package dev

// OptoisolatorCTRModel specifically models the current transfer ratio curve
// of an optoisolator from 0 to 1-2mA used to measure high voltage.
//
// Optoisolators are current devices, however, we measure that current by measuring the
// voltage on the emitter resistor.
type OptoisolatorCTRModel[T number] struct {
	QuadraticApproximator[T]
	maxVoltage T // Maximum voltage in V that the model is valid for
}

// NewOptoisolatorCTRModel creates a model specifically for optoisolator CTR curves
// Typical calibration points:
// - (0, 0): Zero input current produces zero output
// - (midADC, midCurrentVoltage): A point in the middle of the curve (typically non-linear)
// - (maxADC, maxCurrentVoltage): Maximum working current point
func NewOptoisolatorCTRModel[T number](
	midADC uint16, midVoltage T,
	maxADC uint16, maxVoltage T) OptoisolatorCTRModel[T] {

	// Create a quadratic approximator with (0,0) as the first point
	approximator := NewQuadraticApproximatorFromPoints(
		0, T(0), // Zero point
		midADC, midVoltage, // Mid-range point
		maxADC, maxVoltage, // Maximum point
	)

	return OptoisolatorCTRModel[T]{
		QuadraticApproximator: approximator,
		maxVoltage:            maxVoltage,
	}
}

// ConvertInverse estimates what ADC value would produce the target voltage
func (model OptoisolatorCTRModel[T]) ConvertInverse(targetVoltage T) uint16 {
	// Constrain to valid range
	if targetVoltage <= 0 {
		return 0
	}
	if targetVoltage > model.maxVoltage {
		targetVoltage = model.maxVoltage
	}

	return model.QuadraticApproximator.ConvertInverse(targetVoltage)
}
