package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/validation"
	"github.com/gorilla/mux"
)

// handleAddParticipant adds a participant to a shift
// @Summary Add participant to shift
// @Description Adds a participant to a shift. Admin token required if shift is locked.
// @Tags Participants
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param shiftId path int true "Shift ID"
// @Param X-Admin-Token header string false "Admin token (required if shift is locked)"
// @Param data body object true "Participant name" SchemaExample({"name": "John Doe"})
// @Success 200 {object} db.Participant "Added participant"
// @Failure 400 {string} string "Invalid input"
// @Failure 403 {string} string "Shift is locked - admin access required"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 409 {string} string "Shift full or duplicate name"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/shifts/{shiftId}/participants [post]
func (s *Server) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
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

	// Get shift to check if locked
	shift, ok := s.getShiftByID(w, shiftID)
	if !ok {
		return
	}

	// If shift is locked, verify admin access
	if !s.verifyAdminForLockedShift(w, r, shift, plan) {
		return
	}

	// Parse request body
	var data struct {
		Name string `json:"name"`
	}
	if !s.parseJSON(w, r, &data) {
		return
	}

	// Validate and sanitize the name
	name, result := validation.ValidateParticipantName(data.Name)
	if result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Add participant
	participant, err := s.db.AddParticipant(shiftID, name)
	if err != nil {
		// Handle specific errors
		if errors.Is(err, db.ErrShiftFull) {
			s.RespondWithError(w, NewAppError(
				ErrShiftFull,
				"Shift is already full",
				http.StatusConflict,
				err,
			))
			return
		}

		if errors.Is(err, db.ErrParticipantExists) {
			s.RespondWithError(w, NewAppError(
				ErrDuplicateName,
				"Name already taken",
				http.StatusConflict,
				err,
			))
			return
		}

		s.RespondWithError(w, NewAppError(
			ErrInternalError,
			"Failed to add participant",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.respondJSON(w, participant)
}

// handleRemoveParticipant removes a participant from a shift
// @Summary Remove participant from shift
// @Description Removes a participant from a shift. Admin token required if shift is locked.
// @Tags Participants
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param shiftId path int true "Shift ID"
// @Param participantId path int true "Participant ID"
// @Param X-Admin-Token header string false "Admin token (required if shift is locked)"
// @Success 204 "Participant removed successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 403 {string} string "Shift is locked - admin access required"
// @Failure 404 {string} string "Plan, shift, or participant not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/shifts/{shiftId}/participants/{participantId} [delete]
func (s *Server) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
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

	// Get shift to check if locked
	shift, ok := s.getShiftByID(w, shiftID)
	if !ok {
		return
	}

	// If shift is locked, verify admin access
	if !s.verifyAdminForLockedShift(w, r, shift, plan) {
		return
	}

	// Parse participant ID
	vars := mux.Vars(r)
	participantID, err := strconv.ParseInt(vars["participantId"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid participant ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Get participants to find the name
	participants, err := s.db.GetParticipantsByShift(shiftID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get participants",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Find the participant with the matching ID
	var participantName string
	for _, p := range participants {
		if p.ID == participantID {
			participantName = p.Name
			break
		}
	}

	if participantName == "" {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Participant not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Remove the participant
	if err := s.db.RemoveParticipant(shiftID, participantName); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInternalError,
			"Failed to remove participant",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.respondNoContent(w)
}
