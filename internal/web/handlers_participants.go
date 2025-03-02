package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["shiftId"], 10, 64)
	if err != nil {
		s.logError(r, "parse_shift_id", err, http.StatusBadRequest, "shift_id", vars["shiftId"])
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Log incoming request
	s.logRequest(r, "add_participant_request", "token", token, "shift_id", shiftID)

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logError(r, "get_shift_plan", err, http.StatusInternalServerError, "token", token)
		http.Error(w, "Failed to verify shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		s.logError(r, "plan_not_found", fmt.Errorf("plan not found"), http.StatusNotFound, "token", token)
		http.NotFound(w, r)
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.logError(r, "verify_shift_belongs_to_plan", err, http.StatusInternalServerError,
			"shift_id", shiftID, "plan_id", plan.ID)
		http.Error(w, "Failed to verify shift ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		s.logError(r, "shift_not_found", fmt.Errorf("shift does not belong to plan"), http.StatusNotFound,
			"shift_id", shiftID, "plan_id", plan.ID)
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	var data struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Add participant
	participant, err := s.db.AddParticipant(shiftID, data.Name)
	if err != nil {
		s.logError(r, "add_participant", err, http.StatusInternalServerError,
			"shift_id", shiftID, "name", data.Name)
		http.Error(w, "Failed to add participant", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "add_participant", http.StatusCreated,
		"shift_id", shiftID, "participant_id", participant.ID, "name", data.Name)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(participant)
}

func (s *Server) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["shiftId"], 10, 64)
	if err != nil {
		s.logError(r, "parse_shift_id", err, http.StatusBadRequest, "shift_id", vars["shiftId"])
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}
	participantID, err := strconv.ParseInt(vars["participantId"], 10, 64)
	if err != nil {
		s.logError(r, "parse_participant_id", err, http.StatusBadRequest, "participant_id", vars["participantId"])
		http.Error(w, "Invalid participant ID", http.StatusBadRequest)
		return
	}

	// Log incoming request
	s.logRequest(r, "remove_participant_request", "token", token, "shift_id", shiftID, "participant_id", participantID)

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logError(r, "get_shift_plan", err, http.StatusInternalServerError, "token", token)
		http.Error(w, "Failed to verify shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		s.logError(r, "plan_not_found", fmt.Errorf("plan not found"), http.StatusNotFound, "token", token)
		http.NotFound(w, r)
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.logError(r, "verify_shift_belongs_to_plan", err, http.StatusInternalServerError,
			"shift_id", shiftID, "plan_id", plan.ID)
		http.Error(w, "Failed to verify shift ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		s.logError(r, "shift_not_found", fmt.Errorf("shift does not belong to plan"), http.StatusNotFound,
			"shift_id", shiftID, "plan_id", plan.ID)
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	// Verify the participant belongs to this shift
	participants, err := s.db.GetParticipantsByShift(shiftID)
	if err != nil {
		s.logError(r, "get_participants", err, http.StatusInternalServerError, "shift_id", shiftID)
		http.Error(w, "Failed to verify participant", http.StatusInternalServerError)
		return
	}

	// Find the participant with the matching ID
	participantExists := false
	var participantName string
	for _, p := range participants {
		if p.ID == participantID {
			participantExists = true
			participantName = p.Name
			break
		}
	}

	if !participantExists {
		s.logError(r, "participant_not_found", fmt.Errorf("participant not found"), http.StatusNotFound,
			"shift_id", shiftID, "participant_id", participantID)
		http.Error(w, "Participant not found", http.StatusNotFound)
		return
	}

	// Remove participant
	if err := s.db.RemoveParticipant(shiftID, participantName); err != nil {
		s.logError(r, "remove_participant", err, http.StatusInternalServerError,
			"shift_id", shiftID, "participant_id", participantID, "name", participantName)
		http.Error(w, "Failed to remove participant", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "remove_participant", http.StatusNoContent,
		"shift_id", shiftID, "participant_id", participantID)

	w.WriteHeader(http.StatusNoContent)
}
