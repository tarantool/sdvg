package common

type ContextCancelError struct{}

func (e *ContextCancelError) Error() string {
	return "context canceled"
}
