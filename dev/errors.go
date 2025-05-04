package dev

// error definitions
type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrDriverBusy         = Error("driver already active")
	ErrInvalidStage       = Error("invalid stage index")
	ErrInvalidPulseTrain  = Error("invalid pulse train")
	ErrPulseWidthExceeded = Error("pulse width exceeds maximum allowed")
	ErrInvalidPinMode     = Error("invalid pin mode")
	ErrInvalidPinChange   = Error("invalid pin change mode")
	ErrSensePin           = Error("invalid sense pin value")
)
