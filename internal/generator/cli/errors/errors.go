package errors

type UsageError struct {
	err string
}

func (e *UsageError) Error() string {
	return e.err
}

func NewUsageError(err error) error {
	return &UsageError{
		err: err.Error(),
	}
}
