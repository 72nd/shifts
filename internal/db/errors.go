package db

import "errors"

// Sentinel errors for the database layer.
// Use errors.Is() to check for these errors in handlers.
var (
	// ErrShiftFull is returned when trying to add a participant to a full shift
	ErrShiftFull = errors.New("shift is already full")

	// ErrParticipantExists is returned when a participant with the same name already exists in a shift
	ErrParticipantExists = errors.New("participant already exists in this shift")

	// ErrNotFound is returned when a requested resource does not exist
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidPassword is returned when password validation fails
	ErrInvalidPassword = errors.New("invalid password")

	// ErrNoPermission is returned when the operation is not allowed for the given resource
	ErrNoPermission = errors.New("no permission for this resource")

	// ErrInvalidRequired is returned when the required field has an invalid value
	ErrInvalidRequired = errors.New("required participants must be either -1 for unlimited or a positive number")
)
