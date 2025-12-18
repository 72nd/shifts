package web

import (
	"net/http"

	"github.com/72nd/shifts/internal/db"
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
// @Param shift body db.NewShift true "Shift details"
// @Success 200 {object} db.Shift "Created shift"
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

	// Validate and sanitize input
	if result := validation.ValidateNewShift(&shift); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Get plan
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
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

	s.respondJSON(w, createdShift)
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
// @Param shift body db.Shift true "Updated shift details"
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
	var shiftUpdate validation.ShiftUpdate
	if !s.parseJSON(w, r, &shiftUpdate) {
		return
	}

	if result := validation.ValidateShiftUpdate(&shiftUpdate); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Convert to db.Shift for the update
	shift := db.Shift{
		TypeID:    shiftUpdate.TypeID,
		StartTime: shiftUpdate.StartTime,
		Duration:  shiftUpdate.Duration,
		Required:  shiftUpdate.Required,
		CommentDE: shiftUpdate.CommentDE,
		CommentEN: shiftUpdate.CommentEN,
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
