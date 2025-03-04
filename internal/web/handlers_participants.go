package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) handleAddParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["shiftId"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to verify shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.logger.Error("failed to verify shift ownership", "error", err)
		http.Error(w, "Failed to verify shift ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	var data struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validiere und sanitisiere den Namen
	name := SanitizeText(data.Name)
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if len(name) > 20 {
		http.Error(w, "Name cannot exceed 20 characters", http.StatusBadRequest)
		return
	}

	participant, err := s.db.AddParticipant(shiftID, name)
	if err != nil {
		s.logger.Error("failed to add participant", "error", err)
		http.Error(w, "Failed to add participant", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participant)
}

func (s *Server) handleRemoveParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID, err := strconv.ParseInt(vars["shiftId"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Verify the shift belongs to the plan
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to verify shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Verify the shift belongs to this plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, plan.ID)
	if err != nil {
		s.logger.Error("failed to verify shift ownership", "error", err)
		http.Error(w, "Failed to verify shift ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	// Get the participant from the database first to get their name
	participants, err := s.db.GetParticipantsByShift(shiftID)
	if err != nil {
		s.logger.Error("failed to get participants", "error", err)
		http.Error(w, "Failed to get participants", http.StatusInternalServerError)
		return
	}

	participantID, err := strconv.ParseInt(vars["participantId"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid participant ID", http.StatusBadRequest)
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
		http.Error(w, "Participant not found", http.StatusNotFound)
		return
	}

	// Remove the participant using their name
	err = s.db.RemoveParticipant(shiftID, participantName)
	if err != nil {
		s.logger.Error("failed to remove participant", "error", err)
		http.Error(w, "Failed to remove participant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
