package dev

import (
	"fmt"
	"math"
	"time"
)

// CircuitParameters holds the LCR circuit parameters
type CircuitParameters struct {
	L  float64 // Inductance in Henries
	C  float64 // Capacitance in Farads
	R  float64 // Resistance in Ohms
	V0 float64 // Initial voltage on capacitor in Volts
}

// NaturalFrequency computes the undamped natural angular frequency ω₀ = 1 / √(LC)
func (cp CircuitParameters) NaturalFrequency() float64 {
	return 1 / math.Sqrt(cp.L*cp.C)
}

// DampingFactor computes the damping factor α = R / (2L)
func (cp CircuitParameters) DampingFactor() float64 {
	return cp.R / (2 * cp.L)
}

// DampedFrequency computes the damped angular frequency ω_d = √(ω₀² - α²)
func (cp CircuitParameters) DampedFrequency() (float64, error) {
	omega0 := cp.NaturalFrequency()
	alpha := cp.DampingFactor()
	if alpha >= omega0 {
		return 0, fmt.Errorf("circuit is not underdamped: α ≥ ω₀")
	}
	return math.Sqrt(omega0*omega0 - alpha*alpha), nil
}

// Period returns the oscillation period T = 2π / ω_d for underdamped circuits
func (cp CircuitParameters) Period() (time.Duration, error) {
	omegaD, err := cp.DampedFrequency()
	if err != nil {
		return 0, err
	}
	T := 2 * math.Pi / omegaD
	return time.Duration(T * float64(time.Second)), nil
}

// TimeConstant computes the time constant τ = 1 / α
func (cp CircuitParameters) TimeConstant() float64 {
	return 1 / cp.DampingFactor()
}

// PeakCurrent computes the peak current I_peak = V₀ / (ω_d * L)
func (cp CircuitParameters) PeakCurrent() (float64, error) {
	omegaD, err := cp.DampedFrequency()
	if err != nil {
		return 0, err
	}
	return cp.V0 / (omegaD * cp.L), nil
}

// PeakMagneticEnergy computes the peak magnetic energy E_peak = ½ * L * I_peak²
func (cp CircuitParameters) PeakMagneticEnergy() (float64, error) {
	Ipeak, err := cp.PeakCurrent()
	if err != nil {
		return 0, err
	}
	return 0.5 * cp.L * Ipeak * Ipeak, nil
}

// Instantaneous current at time t (underdamped)
func (cp CircuitParameters) CurrentAt(t time.Duration) (float64, error) {
	alpha := cp.DampingFactor()
	omegaD, err := cp.DampedFrequency()
	if err != nil {
		return 0, err
	}
	if math.IsNaN(omegaD) {
		return 0, nil
	}
	seconds := t.Seconds()

	A := 0.0
	B := (cp.V0 / cp.L) / omegaD

	current := math.Exp(-alpha*seconds) * (A*math.Cos(omegaD*seconds) + B*math.Sin(omegaD*seconds))
	return current, nil
}

// Instantaneous magnetic energy at time t = 0.5 * L * i(t)^2
func (cp CircuitParameters) MagneticEnergyAt(t time.Duration) (float64, error) {
	i, err := cp.CurrentAt(t)
	return 0.5 * cp.L * i * i, err
}

// VoltageAt returns the voltage across the capacitor at time t (in seconds) for underdamped RLC response
func (cp CircuitParameters) VoltageAt(t float64) (float64, error) {
	alpha := cp.DampingFactor()
	omegaD, err := cp.DampedFrequency()
	if err != nil {
		return 0, err
	}

	voltage := cp.V0 * math.Exp(-alpha*t) * (math.Cos(omegaD*t) + (alpha/omegaD)*math.Sin(omegaD*t))
	return voltage, nil
}

// TotalMagneticEnergy computes the total magnetic energy in the inductor at time T (no integration needed).
// Works for underdamped LCR circuit (R < 2*sqrt(L/C)).
func TotalMagneticEnergy(params CircuitParameters, T float64) (float64, error) {
	// Check underdamped condition
	criticalR := 2 * math.Sqrt(params.L/params.C)
	if params.R >= criticalR {
		return 0, fmt.Errorf("circuit is not underdamped: R = %.6f, critical R = %.6f", params.R, criticalR)
	}

	// Derived quantities
	alpha := params.R / (2 * params.L)
	omega0 := 1 / math.Sqrt(params.L*params.C)
	omegaD := math.Sqrt(omega0*omega0 - alpha*alpha)

	// Initial conditions: i(0) = 0, di/dt(0) = V0 / L
	// A := 0.0
	B := (params.V0 / params.L) / omegaD

	// i(t) = e^(-αt) * (A cos(ω_d t) + B sin(ω_d t)) → since A = 0, i(t) = B * e^(-αt) * sin(ω_d t)
	i := B * math.Exp(-alpha*T) * math.Sin(omegaD*T)

	// Magnetic energy: E = ½ * L * i(t)^2
	energy := 0.5 * params.L * i * i
	return energy, nil
}

// Calculate the total magnetic energy stored in the inductor from t=0 to t=T
// for an underdamped LCR circuit
func TotalMagneticEnergyIntegrated(params CircuitParameters, T float64) (float64, error) {
	// Verify this is an underdamped system
	criticalR := 2 * math.Sqrt(params.L/params.C)
	if params.R >= criticalR {
		return 0, fmt.Errorf("circuit is not underdamped: R = %.6f, critical R = %.6f", params.R, criticalR)
	}

	// Calculate relevant parameters
	alpha := params.R / (2 * params.L)               // Damping factor
	omega0 := 1 / math.Sqrt(params.L*params.C)       // Natural frequency
	omegaD := math.Sqrt(omega0*omega0 - alpha*alpha) // Damped frequency

	// Initial conditions: i(0) = 0, di/dt(0) = V0/L
	i0 := 0.0
	di0dt := params.V0 / params.L

	// Coefficients for current equation
	A := i0
	B := (di0dt + alpha*i0) / omegaD

	// Calculate total energy through numerical integration
	return calculateTotalEnergyWithSimpson(params.L, alpha, omegaD, A, B, T)
}

// Use Simpson's rule for numerical integration to calculate total energy
func calculateTotalEnergyWithSimpson(L, alpha, omegaD, A, B, T float64) (float64, error) {
	if T <= 0 {
		return 0, fmt.Errorf("integration time T must be positive")
	}

	// Number of intervals (must be even)
	const n = 10000
	if n%2 != 0 {
		return 0, fmt.Errorf("number of intervals must be even")
	}

	h := T / float64(n)
	sum := 0.0

	// Apply Simpson's 1/3 rule
	for i := 0; i <= n; i++ {
		t := float64(i) * h
		energy := magneticEnergy(L, alpha, omegaD, A, B, t)

		// Apply appropriate weight based on position
		if i == 0 || i == n {
			sum += energy
		} else if i%2 == 0 {
			sum += 2 * energy
		} else {
			sum += 4 * energy
		}
	}

	return sum * h / 3, nil
}

// Calculate instantaneous magnetic energy at time t
func magneticEnergy(L, alpha, omegaD, A, B, t float64) float64 {
	// Calculate current at time t
	current := math.Exp(-alpha*t) * (A*math.Cos(omegaD*t) + B*math.Sin(omegaD*t))

	// Magnetic energy is (1/2)L*i²
	return 0.5 * L * current * current
}
