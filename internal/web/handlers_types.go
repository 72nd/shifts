package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleGetShiftTypes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan first to verify it exists and get its ID
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Get shift types with their shifts
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logger.Error("failed to get shift types", "error", err)
		http.Error(w, "Failed to get shift types", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

// handleCreateShiftType handles the creation of a new shift type
func (s *Server) handleCreateShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan first to verify it exists and get its ID
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Parse request body
	var shiftType db.NewShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set plan ID
	shiftType.PlanID = plan.ID

	// Create shift type
	createdType, err := s.db.CreateShiftType(shiftType)
	if err != nil {
		s.logger.Error("failed to create shift type", "error", err)
		http.Error(w, "Failed to create shift type", http.StatusInternalServerError)
		return
	}

	// Return created shift type
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdType)
}

// handleUpdateShiftType handles updating an existing shift type
func (s *Server) handleUpdateShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift type ID", http.StatusBadRequest)
		return
	}

	// Get plan first to verify it exists
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Parse request body
	var shiftType db.ShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the ID matches
	shiftType.ID = typeID
	shiftType.PlanID = plan.ID

	// Update shift type
	if err := s.db.UpdateShiftType(typeID, shiftType); err != nil {
		s.logger.Error("failed to update shift type", "error", err)
		http.Error(w, "Failed to update shift type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteShiftType handles deleting an existing shift type
func (s *Server) handleDeleteShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift type ID", http.StatusBadRequest)
		return
	}

	// Get plan first to verify it exists
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Delete the shift type
	if err := s.db.DeleteShiftType(plan.ID, typeID); err != nil {
		s.logger.Error("failed to delete shift type", "error", err)
		http.Error(w, "failed to delete shift type", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
