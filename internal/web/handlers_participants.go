package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
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
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["shiftId"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify shift plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify shift ownership",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if !belongs {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Check if shift is locked
	shift, err := s.db.GetShiftByID(shiftID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if shift == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// If shift is locked, only admins can add participants
	if shift.IsLocked {
		adminToken := r.Header.Get("X-Admin-Token")
		if adminToken == "" {
			s.RespondWithError(w, NewAppError(
				ErrUnauthorized,
				"Shift is locked - admin access required",
				http.StatusForbidden,
				nil,
			))
			return
		}

		isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
		if err != nil {
			s.RespondWithError(w, NewAppError(
				ErrDatabaseError,
				"Failed to verify admin token",
				http.StatusInternalServerError,
				err,
			))
			return
		}
		if !isValid {
			s.RespondWithError(w, NewAppError(
				ErrUnauthorized,
				"Shift is locked - admin access required",
				http.StatusForbidden,
				nil,
			))
			return
		}
	}

	var data struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Validiere und sanitisiere den Namen
	name := SanitizeText(data.Name)
	if name == "" {
		s.RespondWithError(w, NewAppError(
			ErrMissingField,
			"Name is required",
			http.StatusBadRequest,
			nil,
		))
		return
	}
	if len(name) > 20 {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Name cannot exceed 20 characters",
			http.StatusBadRequest,
			nil,
		))
		return
	}

	var participant *db.Participant
	participant, err = s.db.AddParticipant(shiftID, name)
	if err != nil {
		// Spezifische Fehlerbehandlung für volle Schichten
		if errors.Is(err, db.ErrShiftFull) {
			s.RespondWithError(w, NewAppError(
				ErrShiftFull,
				"Shift is already full",
				http.StatusConflict,
				err,
			))
			return
		}

		// Spezifische Fehlerbehandlung für doppelte Namen
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participant)
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
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["shiftId"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify shift plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify shift ownership",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if !belongs {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Check if shift is locked
	shift, err := s.db.GetShiftByID(shiftID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if shift == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// If shift is locked, only admins can remove participants
	if shift.IsLocked {
		adminToken := r.Header.Get("X-Admin-Token")
		if adminToken == "" {
			s.RespondWithError(w, NewAppError(
				ErrUnauthorized,
				"Shift is locked - admin access required",
				http.StatusForbidden,
				nil,
			))
			return
		}

		isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
		if err != nil {
			s.RespondWithError(w, NewAppError(
				ErrDatabaseError,
				"Failed to verify admin token",
				http.StatusInternalServerError,
				err,
			))
			return
		}
		if !isValid {
			s.RespondWithError(w, NewAppError(
				ErrUnauthorized,
				"Shift is locked - admin access required",
				http.StatusForbidden,
				nil,
			))
			return
		}
	}

	// Get the participant from the database first to get their name
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

	// Remove the participant using their name
	err = s.db.RemoveParticipant(shiftID, participantName)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInternalError,
			"Failed to remove participant",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
