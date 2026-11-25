package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
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
// @Param data body AddParticipantRequest true "Participant name and optional person details"
// @Success 200 {object} ParticipantResponse "Added participant"
// @Failure 400 {string} string "Invalid input"
// @Failure 403 {string} string "Shift is locked - admin access required"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 409 {string} string "Shift full, duplicate name, or the person was declined for this shift"
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
	var data model.AddParticipantRequest
	if !s.parseJSONTimes(w, r, &data) {
		return
	}

	// Validate and sanitize the name
	name, result := validation.ValidateParticipantName(data.Name)
	if result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	personInput, ok := s.preparePersonInput(w, plan, name, data.Person)
	if !ok {
		return
	}

	// Add participant
	participant, err := s.db.AddParticipant(shiftID, name, personInput)
	if err != nil {
		if errors.Is(err, db.ErrParticipantDeclined) {
			participantJoinsTotal.WithLabelValues(plan.Token, "failure").Inc()
			participantJoinsFailedTotal.WithLabelValues(plan.Token, "declined").Inc()

			s.RespondWithError(w, NewAppError(
				ErrParticipantDeclined,
				"You are already listed as not selected for this shift. Please contact the coordination team.",
				http.StatusConflict,
				err,
			))
			return
		}

		// Handle specific errors
		if errors.Is(err, db.ErrShiftFull) {
			// Track failed join due to capacity
			participantJoinsTotal.WithLabelValues(plan.Token, "failure").Inc()
			participantJoinsFailedTotal.WithLabelValues(plan.Token, "capacity").Inc()

			s.RespondWithError(w, NewAppError(
				ErrShiftFull,
				"Shift is already full",
				http.StatusConflict,
				err,
			))
			return
		}

		if errors.Is(err, db.ErrParticipantExists) {
			// Track failed join due to duplicate name
			participantJoinsTotal.WithLabelValues(plan.Token, "failure").Inc()
			participantJoinsFailedTotal.WithLabelValues(plan.Token, "duplicate").Inc()

			s.RespondWithError(w, NewAppError(
				ErrDuplicateName,
				"Name already taken",
				http.StatusConflict,
				err,
			))
			return
		}

		// Track other failures
		participantJoinsTotal.WithLabelValues(plan.Token, "failure").Inc()
		participantJoinsFailedTotal.WithLabelValues(plan.Token, "error").Inc()

		s.RespondWithError(w, NewAppError(
			ErrInternalError,
			"Failed to add participant",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Track successful join
	participantJoinsTotal.WithLabelValues(plan.Token, "success").Inc()
	s.notifyPlanChanged(plan.ID, scopeParticipants)

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
	var participantStatus string
	for _, p := range participants {
		if p.ID == participantID {
			participantName = p.Name
			participantStatus = p.Status
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

	// Declined participations are fairness information: only the admin may remove them
	if participantStatus == db.StatusDeclined &&
		!s.requireAdminForbidden(w, r, plan, "Declined participations can only be removed by the admin") {
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

	// Track metrics
	participantRemovalsTotal.WithLabelValues(plan.Token).Inc()
	s.notifyPlanChanged(plan.ID, scopeParticipants)

	s.respondNoContent(w)
}

// preparePersonInput validates the optional person details of a sign-up and checks that all
// required participant fields have a value afterwards (taking the person's stored values into
// account). It returns the input for AddParticipant (nil if there is nothing to store), or writes
// a 400 response and returns false.
func (s *Server) preparePersonInput(w http.ResponseWriter, plan *db.ShiftPlan, name string, in *model.PersonInput) (*db.PersonInput, bool) {
	// Without coordination the person fields are not asked: sent data is ignored, nothing is required.
	if !plan.CoordinationEnabled {
		return nil, true
	}

	r := validation.NewResult()
	var result *db.PersonInput

	existing, err := s.db.GetPersonByName(plan.ID, name)
	if err != nil {
		s.RespondWithError(w, NewAppError(ErrDatabaseError, "Failed to get person", http.StatusInternalServerError, err))
		return nil, false
	}
	merged := map[string]any{}
	var from, until = (*model.DateTime)(nil), (*model.DateTime)(nil)
	if existing != nil {
		for k, v := range existing.Values {
			merged[k] = v
		}
		from, until = existing.AvailableFrom, existing.AvailableUntil
	}

	if in != nil {
		result = &db.PersonInput{}
		if in.AvailableFrom != nil || in.AvailableUntil != nil {
			if plan.AttendanceMode != "participant" {
				r.AddInvalid("available_from", "Attendance is not asked from participants in this plan")
			}
			if in.AvailableFrom != nil {
				from = in.AvailableFrom
				result.AvailableFrom = in.AvailableFrom
			}
			if in.AvailableUntil != nil {
				until = in.AvailableUntil
				result.AvailableUntil = in.AvailableUntil
			}
			validation.ValidateAttendance(from, until, r)
		}
		values := validation.ValidateFieldValues(plan.Fields, in.Values, db.EditableParticipant, true, r)
		result.Values = values
		for k, v := range values {
			if v == nil {
				delete(merged, k)
			} else {
				merged[k] = v
			}
		}
	}
	validation.CheckRequiredFields(plan.Fields, merged, r)

	if r.HasErrors() {
		s.respondValidationError(w, r)
		return nil, false
	}
	return result, true
}
