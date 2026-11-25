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

	// ErrSchemaTooNew is returned when the database was migrated by a newer version of the software
	ErrSchemaTooNew = errors.New("database schema is newer than this version of the software")

	// ErrInvalidRequired is returned when the required field has an invalid value
	ErrInvalidRequired = errors.New("required participants must be either -1 for unlimited or a positive number")

	// ErrParticipantDeclined is returned when a person tries to sign up for a shift where they are already listed as declined
	ErrParticipantDeclined = errors.New("participant is listed as declined in this shift")

	// ErrPersonExists is returned when renaming a person to a name that is already taken (without merging)
	ErrPersonExists = errors.New("a person with this name already exists")

	// ErrPersonNotFound is returned when a person does not exist in the plan
	ErrPersonNotFound = errors.New("person not found")

	// ErrFieldNotFound is returned when a person field does not exist in the plan
	ErrFieldNotFound = errors.New("field not found")
)
