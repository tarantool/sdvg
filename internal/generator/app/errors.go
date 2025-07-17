package app

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// SignalError type of event that is generated when an OS signal is received.
type SignalError struct {
	signal os.Signal
}

// Error function returns text of error.
func (e *SignalError) Error() string {
	return fmt.Sprintf("%v signal", e.signal)
}

// NewSignalError function creates SignalError object.
func NewSignalError(signal os.Signal) *SignalError {
	return &SignalError{
		signal: signal,
	}
}
