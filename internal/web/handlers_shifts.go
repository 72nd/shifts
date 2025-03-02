package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleCreateShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var shift db.NewShift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	// Get the shift type to verify it belongs to this plan
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logger.Error("failed to get shift types", "error", err)
		http.Error(w, "Failed to verify shift type", http.StatusInternalServerError)
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
		http.Error(w, "Shift type not found", http.StatusNotFound)
		return
	}

	// Create the shift
	createdShift, err := s.db.CreateShift(shift)
	if err != nil {
		s.logger.Error("failed to create shift", "error", err)
		http.Error(w, "Failed to create shift", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdShift)
}

func (s *Server) handleUpdateShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftId := vars["shiftId"]

	// Parse shift ID
	id, err := strconv.ParseInt(shiftId, 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	// Verify shift belongs to plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(id, plan.ID)
	if err != nil {
		s.logger.Error("failed to verify shift ownership", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	var shift db.Shift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the shift
	if err := s.db.UpdateShift(id, shift); err != nil {
		s.logger.Error("failed to update shift", "error", err)
		http.Error(w, "Failed to update shift", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftId := vars["shiftId"]

	// Parse shift ID
	id, err := strconv.ParseInt(shiftId, 10, 64)
	if err != nil {
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	// Verify admin token
	adminToken := r.Header.Get("X-Admin-Token")
	isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
	if err != nil {
		s.logger.Error("failed to verify admin token", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !isValid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify shift belongs to plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(id, plan.ID)
	if err != nil {
		s.logger.Error("failed to verify shift ownership", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	// Get the shift type ID
	typeID, err := s.db.GetShiftTypeID(id)
	if err != nil {
		s.logger.Error("failed to get shift type ID", "error", err)
		http.Error(w, "Failed to delete shift", http.StatusInternalServerError)
		return
	}

	// Delete the shift
	if err := s.db.DeleteShift(id, typeID); err != nil {
		s.logger.Error("failed to delete shift", "error", err)
		http.Error(w, "Failed to delete shift", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
