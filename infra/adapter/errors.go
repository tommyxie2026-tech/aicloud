package adapter

// ErrorCode identifies normalized backend adapter error categories.
type ErrorCode string

const (
	ErrValidationError      ErrorCode = "ValidationError"
	ErrBackendUnavailable  ErrorCode = "BackendUnavailable"
	ErrBackendConflict     ErrorCode = "BackendConflict"
	ErrUnauthorized        ErrorCode = "Unauthorized"
	ErrRateLimited         ErrorCode = "RateLimited"
	ErrReconcileInProgress ErrorCode = "ReconcileInProgress"
	ErrUnknownBackendError ErrorCode = "UnknownBackendError"
)

// AdapterError is the normalized error returned by backend adapters.
type AdapterError struct {
	Code      ErrorCode
	Message   string
	Retryable bool
}

func NewAdapterError(code ErrorCode, message string, retryable bool) *AdapterError {
	return &AdapterError{Code: code, Message: message, Retryable: retryable}
}

func (e *AdapterError) Error() string {
	return string(e.Code) + ": " + e.Message
}

func ValidationError(message string) *AdapterError {
	return NewAdapterError(ErrValidationError, message, false)
}

func BackendUnavailable(message string) *AdapterError {
	return NewAdapterError(ErrBackendUnavailable, message, true)
}

func BackendConflict(message string) *AdapterError {
	return NewAdapterError(ErrBackendConflict, message, true)
}

func Unauthorized(message string) *AdapterError {
	return NewAdapterError(ErrUnauthorized, message, false)
}

func RateLimited(message string) *AdapterError {
	return NewAdapterError(ErrRateLimited, message, true)
}

func ReconcileInProgress(message string) *AdapterError {
	return NewAdapterError(ErrReconcileInProgress, message, true)
}

func UnknownBackendError(message string) *AdapterError {
	return NewAdapterError(ErrUnknownBackendError, message, true)
}
