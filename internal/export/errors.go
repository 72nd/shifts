package export

import (
	"fmt"
	"net/http"
)

// ErrorType represents different types of export errors
type ErrorType string

const (
	ErrLoadingPlan         ErrorType = "export_loading_plan"
	ErrLoadingShiftTypes   ErrorType = "export_loading_shift_types"
	ErrLoadingShifts       ErrorType = "export_loading_shifts"
	ErrLoadingParticipants ErrorType = "export_loading_participants"
	ErrInvalidToken        ErrorType = "export_invalid_token"
	ErrUnauthorized        ErrorType = "export_unauthorized"
	ErrInvalidInput        ErrorType = "export_invalid_input"
)

// Error represents an export-specific error
type Error struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e Error) StatusCode() int {
	switch e.Type {
	case ErrInvalidToken, ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrInvalidInput:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
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
