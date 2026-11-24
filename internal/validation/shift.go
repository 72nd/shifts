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

// ValidateNewShiftWithPlanDates validates a NewShift struct for creation,
// including validation that the shift's logical day (see day boundary) lies
// within the plan's date range (both ends inclusive).
func ValidateNewShiftWithPlanDates(s *db.NewShift, planStart, planEnd model.Date, boundary string) *Result {
	r := ValidateNewShift(s)
	checkShiftInPlan(r, "start_time", s.StartTime, planStart, planEnd, boundary)
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

// ValidateShiftUpdateWithPlanDates validates a shift update request,
// including validation that the shift's logical day (see day boundary) lies
// within the plan's date range (both ends inclusive).
func ValidateShiftUpdateWithPlanDates(s *model.ShiftUpdateRequest, planStart, planEnd model.Date, boundary string) *Result {
	r := ValidateShiftUpdate(s)
	checkShiftInPlan(r, "start_time", s.StartTime, planStart, planEnd, boundary)
	return r
}

// checkShiftInPlan adds an error to r if the logical day of start (start time
// with the day boundary applied) lies outside [planStart, planEnd]. Zero values
// are skipped (reported as required elsewhere). A shift that ends after the
// plan's last day is fine (night shift on the last day).
func checkShiftInPlan(r *Result, field string, start model.DateTime, planStart, planEnd model.Date, boundary string) {
	if start.IsZero() || planStart.IsZero() || planEnd.IsZero() {
		return
	}
	// The boundary was validated when the plan was saved; fall back to 0 if not.
	boundaryMinutes, _ := model.ParseBoundary(boundary)
	day := start.LogicalDay(boundaryMinutes)
	if day.Before(planStart.Time) {
		r.Add(field, CodeInvalidValue, "Shift start time is before the plan's start date")
	} else if day.After(planEnd.Time) {
		r.Add(field, CodeInvalidValue, "Shift start time is after the plan's end date")
	}
}

// ValidateParticipantName validates and sanitizes a participant name.
func ValidateParticipantName(name string) (string, *Result) {
	r := NewResult()

	name = ValidateRequiredMaxLength(name, "name", "Name", MaxNameLength, r)

	return name, r
}

// =============================================================================
// Batch Shift Validation
// =============================================================================

// ValidateBatchShiftCreateRequest validates a batch shift create request.
func ValidateBatchShiftCreateRequest(req *model.BatchShiftCreateRequest) *Result {
	r := NewResult()

	// Check array is not empty
	if len(req.Shifts) == 0 {
		r.AddRequired("shifts", "Shifts array")
		return r
	}

	// Check array doesn't exceed max size (100 items)
	if len(req.Shifts) > 100 {
		r.AddInvalid("shifts", "Batch operations are limited to 100 items")
		return r
	}

	// Validate each shift individually
	for i := range req.Shifts {
		shift := &req.Shifts[i]

		// Create a db.NewShift for validation
		newShift := db.NewShift{
			TypeID:    shift.TypeID,
			StartTime: shift.StartTime,
			Duration:  shift.Duration,
			Required:  shift.Required,
			CommentDE: shift.CommentDE,
			CommentEN: shift.CommentEN,
		}

		itemResult := ValidateNewShift(&newShift)

		// Merge errors with index prefix
		if itemResult.HasErrors() {
			r.MergeWithPrefix(itemResult, "shifts["+itoa(i)+"].")
		}

		// Copy sanitized values back
		shift.CommentDE = newShift.CommentDE
		shift.CommentEN = newShift.CommentEN
	}

	return r
}

// ValidateBatchShiftCreateRequestWithPlanDates validates a batch shift create request,
// including validation that all shifts fall within the plan's date range.
func ValidateBatchShiftCreateRequestWithPlanDates(req *model.BatchShiftCreateRequest, planStart, planEnd model.Date, boundary string) *Result {
	r := NewResult()

	// Check array is not empty
	if len(req.Shifts) == 0 {
		r.AddRequired("shifts", "Shifts array")
		return r
	}

	// Check array doesn't exceed max size (100 items)
	if len(req.Shifts) > 100 {
		r.AddInvalid("shifts", "Batch operations are limited to 100 items")
		return r
	}

	// Validate each shift individually
	for i := range req.Shifts {
		shift := &req.Shifts[i]

		// Create a db.NewShift for validation
		newShift := db.NewShift{
			TypeID:    shift.TypeID,
			StartTime: shift.StartTime,
			Duration:  shift.Duration,
			Required:  shift.Required,
			CommentDE: shift.CommentDE,
			CommentEN: shift.CommentEN,
		}

		// Validate with plan date range
		itemResult := ValidateNewShiftWithPlanDates(&newShift, planStart, planEnd, boundary)

		// Merge errors with index prefix
		if itemResult.HasErrors() {
			r.MergeWithPrefix(itemResult, "shifts["+itoa(i)+"].")
		}

		// Copy sanitized values back
		shift.CommentDE = newShift.CommentDE
		shift.CommentEN = newShift.CommentEN
	}

	return r
}

// ValidateBatchShiftUpdateRequest validates a batch shift update request.
func ValidateBatchShiftUpdateRequest(req *model.BatchShiftUpdateRequest) *Result {
	r := NewResult()

	// Check array is not empty
	if len(req.Shifts) == 0 {
		r.AddRequired("shifts", "Shifts array")
		return r
	}

	// Check array doesn't exceed max size (100 items)
	if len(req.Shifts) > 100 {
		r.AddInvalid("shifts", "Batch operations are limited to 100 items")
		return r
	}

	// Validate each update item individually
	for i := range req.Shifts {
		item := &req.Shifts[i]
		prefix := "shifts[" + itoa(i) + "]."

		// Validate ID is positive (always required)
		if item.ID <= 0 {
			r.Add(prefix+"id", CodeInvalidValue, "Shift ID must be a positive number")
		}

		// Validate TypeID is positive (if provided)
		if item.TypeID != nil && *item.TypeID <= 0 {
			r.Add(prefix+"type_id", CodeInvalidValue, "Type ID must be a positive number")
		}

		// Validate StartTime is not empty (if provided)
		if item.StartTime != nil && item.StartTime.IsZero() {
			r.Add(prefix+"start_time", CodeInvalidValue, "Start time cannot be empty")
		}

		// Validate Duration is positive (if provided)
		if item.Duration != nil && *item.Duration <= 0 {
			r.Add(prefix+"duration", CodeInvalidValue, "Duration must be a positive number of minutes")
		}

		// Validate Required is -1 or positive (if provided)
		if item.Required != nil && *item.Required != -1 && *item.Required < 1 {
			r.Add(prefix+"required", CodeInvalidValue, "Required participants must be either -1 for unlimited or a positive number")
		}

		// Sanitize comments if provided
		if item.CommentDE != nil {
			sanitized := ValidateOptional(*item.CommentDE)
			item.CommentDE = &sanitized
		}
		if item.CommentEN != nil {
			sanitized := ValidateOptional(*item.CommentEN)
			item.CommentEN = &sanitized
		}
	}

	return r
}

// ValidateBatchShiftUpdateRequestWithPlanDates validates a batch shift update
// request and checks, for every item that sets start_time, that the logical day
// lies within the plan's date range. The check needs only the start time (not
// the duration), so no existing shift has to be loaded.
func ValidateBatchShiftUpdateRequestWithPlanDates(req *model.BatchShiftUpdateRequest, planStart, planEnd model.Date, boundary string) *Result {
	r := ValidateBatchShiftUpdateRequest(req)
	for i := range req.Shifts {
		item := &req.Shifts[i]
		if item.StartTime != nil {
			checkShiftInPlan(r, "shifts["+itoa(i)+"].start_time", *item.StartTime, planStart, planEnd, boundary)
		}
	}
	return r
}

// ValidateBatchShiftDeleteRequest validates a batch shift delete request.
func ValidateBatchShiftDeleteRequest(req *model.BatchShiftDeleteRequest) *Result {
	r := NewResult()

	// Check array is not empty
	if len(req.ShiftIDs) == 0 {
		r.AddRequired("shift_ids", "Shift IDs array")
		return r
	}

	// Check array doesn't exceed max size (100 items)
	if len(req.ShiftIDs) > 100 {
		r.AddInvalid("shift_ids", "Batch operations are limited to 100 items")
		return r
	}

	// Validate all shift IDs are positive
	for i, id := range req.ShiftIDs {
		if id <= 0 {
			r.Add("shift_ids["+itoa(i)+"]", CodeInvalidValue, "Shift ID must be a positive number")
		}
	}

	return r
}
