package domain

import "fmt"

// Error code constants used in ValidationError.
const (
	ErrCodeRequired     = "required"
	ErrCodeInvalid      = "invalid"
	ErrCodeNotFound     = "not_found"
	ErrCodeInvalidState = "invalid_state"
)

// ValidationError is a structured error for input validation failures.
type ValidationError struct {
	Field   string
	Code    string
	Message string
}

// Error implements the error interface.
// Format: "{field}: {code}: {message}"
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Field, e.Code, e.Message)
}
