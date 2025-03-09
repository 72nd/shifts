package export

import "fmt"

// ErrorType represents different types of export errors
type ErrorType int

const (
	ErrLoadingPlan ErrorType = iota
	ErrLoadingShiftTypes
	ErrLoadingShifts
	ErrLoadingParticipants
	ErrInvalidToken
	ErrUnauthorized
	ErrInvalidInput
)

// Error represents an export-specific error
type Error struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewError creates a new export error
func NewError(errType ErrorType, message string, err error) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}
