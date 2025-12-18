package validation

import (
	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// ValidateNewShift validates a NewShift struct for creation.
// It sanitizes text fields in place and returns validation errors.
func ValidateNewShift(s *db.NewShift) *Result {
	r := NewResult()

	// TypeID is required and must be positive
	if s.TypeID <= 0 {
		r.AddInvalid("type_id", "Type ID must be a positive number")
	}

	// StartTime must not be zero
	if s.StartTime.IsZero() {
		r.AddRequired("start_time", "Start time")
	}

	// Duration must be positive
	if s.Duration <= 0 {
		r.AddInvalid("duration", "Duration must be a positive number of minutes")
	}

	// Required must be -1 (unlimited) or positive
	if s.Required != -1 && s.Required < 1 {
		r.AddInvalid("required", "Required participants must be either -1 for unlimited or a positive number")
	}

	// Comments are optional, just sanitize them
	s.CommentDE = ValidateOptional(s.CommentDE)
	s.CommentEN = ValidateOptional(s.CommentEN)

	return r
}

// ValidateShiftUpdate validates a shift update request.
func ValidateShiftUpdate(s *model.ShiftUpdateRequest) *Result {
	r := NewResult()

	// TypeID is required and must be positive
	if s.TypeID <= 0 {
		r.AddInvalid("type_id", "Type ID must be a positive number")
	}

	// StartTime must not be zero
	if s.StartTime.IsZero() {
		r.AddRequired("start_time", "Start time")
	}

	// Duration must be positive
	if s.Duration <= 0 {
		r.AddInvalid("duration", "Duration must be a positive number of minutes")
	}

	// Required must be -1 (unlimited) or positive
	if s.Required != -1 && s.Required < 1 {
		r.AddInvalid("required", "Required participants must be either -1 for unlimited or a positive number")
	}

	// Comments are optional, just sanitize them
	s.CommentDE = ValidateOptional(s.CommentDE)
	s.CommentEN = ValidateOptional(s.CommentEN)

	return r
}

// ValidateParticipantName validates and sanitizes a participant name.
func ValidateParticipantName(name string) (string, *Result) {
	r := NewResult()

	name = ValidateRequiredMaxLength(name, "name", "Name", MaxNameLength, r)

	return name, r
}
