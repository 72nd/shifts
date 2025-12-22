package web

import (
	"net/http"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/validation"
)

// handleCreateShift creates a new shift
// @Summary Create shift
// @Description Creates a new shift for a shift type. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param shift body model.ShiftCreateRequest true "Shift details"
// @Success 200 {object} model.ShiftResponse "Created shift"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or type not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts [post]
func (s *Server) handleCreateShift(w http.ResponseWriter, r *http.Request) {
	var shift db.NewShift
	if !s.parseJSON(w, r, &shift) {
		return
	}

	// Get plan first (needed for validation)
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Validate and sanitize input (including plan date range check)
	if result := validation.ValidateNewShiftWithPlanDates(&shift, plan.StartDate, plan.EndDate); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Get the shift type to verify it belongs to this plan
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify shift type",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Check if the type exists in this plan
	typeExists := false
	for _, t := range types {
		if t.ID == shift.TypeID {
			typeExists = true
			break
		}
	}

	if !typeExists {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift type not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Create the shift
	createdShift, err := s.db.CreateShift(shift)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to create shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Track metrics
	shiftsCreatedTotal.WithLabelValues(plan.Token).Inc()

	// Convert to response DTO
	response := ShiftToResponse(createdShift)
	s.respondJSON(w, response)
}

// handleUpdateShift updates an existing shift
// @Summary Update shift
// @Description Updates an existing shift. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param shiftId path int true "Shift ID"
// @Param X-Admin-Token header string true "Admin token"
// @Param shift body ShiftUpdateRequest true "Updated shift details"
// @Success 204 "Shift updated successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/{shiftId} [put]
func (s *Server) handleUpdateShift(w http.ResponseWriter, r *http.Request) {
	// Get plan and shift ID
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	shiftID, ok := s.getShiftID(w, r)
	if !ok {
		return
	}

	// Verify shift belongs to plan
	if !s.verifyShiftOwnership(w, shiftID, plan.ID) {
		return
	}

	// Parse and validate input
	var req model.ShiftUpdateRequest
	if !s.parseJSON(w, r, &req) {
		return
	}

	// Validate including plan date range check
	if result := validation.ValidateShiftUpdateWithPlanDates(&req, plan.StartDate, plan.EndDate); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Convert to db.Shift for the update
	shift := db.Shift{
		TypeID:    req.TypeID,
		StartTime: req.StartTime,
		Duration:  req.Duration,
		Required:  req.Required,
		CommentDE: req.CommentDE,
		CommentEN: req.CommentEN,
	}

	// Update the shift
	if err := s.db.UpdateShift(shiftID, shift); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.respondNoContent(w)
}

// handleDeleteShift deletes an existing shift
// @Summary Delete shift
// @Description Deletes a shift and all its participants. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param shiftId path int true "Shift ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Shift deleted successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/{shiftId} [delete]
func (s *Server) handleDeleteShift(w http.ResponseWriter, r *http.Request) {
	// Get plan and shift ID
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	shiftID, ok := s.getShiftID(w, r)
	if !ok {
		return
	}

	// Verify shift belongs to plan
	if !s.verifyShiftOwnership(w, shiftID, plan.ID) {
		return
	}

	// Delete the shift
	if err := s.db.DeleteShift(shiftID); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to delete shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Track metrics
	shiftsDeletedTotal.WithLabelValues(plan.Token).Inc()

	s.respondNoContent(w)
}

// handleLockShift locks a shift
// @Summary Lock shift
// @Description Locks a shift, preventing non-admin users from joining or leaving. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param shiftId path int true "Shift ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Shift locked successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/{shiftId}/lock [post]
func (s *Server) handleLockShift(w http.ResponseWriter, r *http.Request) {
	// Get plan and shift ID
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	shiftID, ok := s.getShiftID(w, r)
	if !ok {
		return
	}

	// Verify shift belongs to plan
	if !s.verifyShiftOwnership(w, shiftID, plan.ID) {
		return
	}

	// Lock the shift
	if err := s.db.LockShift(shiftID); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to lock shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.respondNoContent(w)
}

// handleUnlockShift unlocks a shift
// @Summary Unlock shift
// @Description Unlocks a shift, allowing all users to join or leave. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param shiftId path int true "Shift ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Shift unlocked successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/{shiftId}/unlock [post]
func (s *Server) handleUnlockShift(w http.ResponseWriter, r *http.Request) {
	// Get plan and shift ID
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	shiftID, ok := s.getShiftID(w, r)
	if !ok {
		return
	}

	// Verify shift belongs to plan
	if !s.verifyShiftOwnership(w, shiftID, plan.ID) {
		return
	}

	// Unlock the shift
	if err := s.db.UnlockShift(shiftID); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to unlock shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.respondNoContent(w)
}

// handleBatchCreateShifts creates multiple shifts in a single operation
// @Summary Batch create shifts
// @Description Creates multiple shifts in a single transaction. All shifts are validated first, then created. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param request body model.BatchShiftCreateRequest true "Batch create request with array of shifts"
// @Success 200 {object} model.BatchCreateResponse "Batch create result with created shifts and any failures"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/batch [post]
func (s *Server) handleBatchCreateShifts(w http.ResponseWriter, r *http.Request) {
	// Get plan
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Parse request
	var req model.BatchShiftCreateRequest
	if !s.parseJSON(w, r, &req) {
		return
	}

	// Validate request (including plan date range check)
	if result := validation.ValidateBatchShiftCreateRequestWithPlanDates(&req, plan.StartDate, plan.EndDate); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Map to DB models
	newShifts := BatchShiftCreateRequestToNewShifts(req.Shifts)

	// Call DB layer
	result, err := s.db.CreateShiftsBatch(newShifts, plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to create shifts",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Track metrics for successful creations
	shiftsCreatedTotal.WithLabelValues(plan.Token).Add(float64(len(result.Success)))

	// Map response
	response := BatchCreateResultToResponse(result)
	s.respondJSON(w, response)
}

// handleBatchUpdateShifts updates multiple shifts in a single operation
// @Summary Batch update shifts
// @Description Updates multiple shifts in a single transaction. All updates are validated first, then applied. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param request body model.BatchShiftUpdateRequest true "Batch update request with array of shift updates"
// @Success 200 {object} model.BatchUpdateResponse "Batch update result with updated shift IDs and any failures"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/batch [put]
func (s *Server) handleBatchUpdateShifts(w http.ResponseWriter, r *http.Request) {
	// Get plan
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Parse request
	var req model.BatchShiftUpdateRequest
	if !s.parseJSON(w, r, &req) {
		return
	}

	// Validate request
	if result := validation.ValidateBatchShiftUpdateRequest(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Map to DB models
	updates := BatchShiftUpdateRequestToBatchUpdates(req.Shifts)

	// Call DB layer
	result, err := s.db.UpdateShiftsBatch(updates, plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update shifts",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Map response
	response := BatchUpdateResultToResponse(result)
	s.respondJSON(w, response)
}

// handleBatchDeleteShifts deletes multiple shifts in a single operation
// @Summary Batch delete shifts
// @Description Deletes multiple shifts in a single transaction. All shift IDs are validated first, then deleted. Requires admin authentication.
// @Tags Shifts
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param request body model.BatchShiftDeleteRequest true "Batch delete request with array of shift IDs"
// @Success 200 {object} model.BatchDeleteResponse "Batch delete result with deleted shift IDs and any failures"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/shifts/batch [delete]
func (s *Server) handleBatchDeleteShifts(w http.ResponseWriter, r *http.Request) {
	// Get plan
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Parse request
	var req model.BatchShiftDeleteRequest
	if !s.parseJSON(w, r, &req) {
		return
	}

	// Validate request
	if result := validation.ValidateBatchShiftDeleteRequest(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Call DB layer
	result, err := s.db.DeleteShiftsBatch(req.ShiftIDs, plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to delete shifts",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Track metrics for successful deletions
	shiftsDeletedTotal.WithLabelValues(plan.Token).Add(float64(len(result.Success)))

	// Map response
	response := BatchDeleteResultToResponse(result)
	s.respondJSON(w, response)
}
