package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleCreateShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "create_shift_request", "token", token)

	var shift db.NewShift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logError(r, "get_shift_plan", err, http.StatusInternalServerError, "token", token)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		s.logError(r, "plan_not_found", fmt.Errorf("plan not found"), http.StatusNotFound, "token", token)
		http.NotFound(w, r)
		return
	}

	// Get the shift type to verify it belongs to this plan
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logError(r, "get_shift_types", err, http.StatusInternalServerError, "plan_id", plan.ID)
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
		s.logError(r, "type_not_found", fmt.Errorf("shift type not found"), http.StatusNotFound,
			"type_id", shift.TypeID, "plan_id", plan.ID)
		http.Error(w, "Shift type not found", http.StatusNotFound)
		return
	}

	// Create the shift
	createdShift, err := s.db.CreateShift(shift)
	if err != nil {
		s.logError(r, "create_shift", err, http.StatusInternalServerError,
			"type_id", shift.TypeID, "start_time", shift.StartTime)
		http.Error(w, "Failed to create shift", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "create_shift", http.StatusCreated,
		"shift_id", createdShift.ID, "type_id", createdShift.TypeID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdShift)
}

func (s *Server) handleUpdateShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftId := vars["shiftId"]

	// Parse shift ID
	id, err := strconv.ParseInt(shiftId, 10, 64)
	if err != nil {
		s.logError(r, "parse_shift_id", err, http.StatusBadRequest, "shift_id", shiftId)
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	s.logRequest(r, "update_shift_request", "token", token, "shift_id", id)

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logError(r, "get_shift_plan", err, http.StatusInternalServerError, "token", token)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		s.logError(r, "plan_not_found", fmt.Errorf("plan not found"), http.StatusNotFound, "token", token)
		http.NotFound(w, r)
		return
	}

	// Verify shift belongs to plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(id, plan.ID)
	if err != nil {
		s.logError(r, "verify_shift_belongs_to_plan", err, http.StatusInternalServerError,
			"shift_id", id, "plan_id", plan.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !belongs {
		s.logError(r, "shift_not_found", fmt.Errorf("shift does not belong to plan"), http.StatusNotFound,
			"shift_id", id, "plan_id", plan.ID)
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	var shift db.Shift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the shift
	if err := s.db.UpdateShift(id, shift); err != nil {
		s.logError(r, "update_shift", err, http.StatusInternalServerError,
			"shift_id", id, "type_id", shift.TypeID)
		http.Error(w, "Failed to update shift", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "update_shift", http.StatusNoContent, "shift_id", id, "type_id", shift.TypeID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	shiftID := vars["shiftId"]

	// Log incoming request
	s.logRequest(r, "delete_shift_request", "token", token, "shift_id", shiftID)

	// Parse shift ID
	id, err := strconv.ParseInt(shiftID, 10, 64)
	if err != nil {
		s.logError(r, "parse_shift_id", err, http.StatusBadRequest, "shift_id", shiftID)
		http.Error(w, "Invalid shift ID", http.StatusBadRequest)
		return
	}

	// Verify admin token
	adminToken := r.Header.Get("X-Admin-Token")
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.logError(r, "get_shift_plan", err, http.StatusInternalServerError, "token", token)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		s.logError(r, "plan_not_found", fmt.Errorf("plan not found"), http.StatusNotFound, "token", token)
		http.NotFound(w, r)
		return
	}

	// Hash the admin token before comparing
	hashedToken := hashPassword(adminToken)

	// Verify admin token
	if hashedToken != plan.AdminPassword {
		s.logError(r, "unauthorized", fmt.Errorf("invalid admin token"), http.StatusUnauthorized, "token", token)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if shift belongs to plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(id, plan.ID)
	if err != nil {
		s.logError(r, "verify_shift_belongs_to_plan", err, http.StatusInternalServerError,
			"shift_id", id, "plan_id", plan.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !belongs {
		s.logError(r, "shift_not_found", fmt.Errorf("shift does not belong to plan"), http.StatusNotFound,
			"shift_id", id, "plan_id", plan.ID)
		http.Error(w, "Shift not found", http.StatusNotFound)
		return
	}

	// Get the shift type ID
	typeID, err := s.db.GetShiftTypeID(id)
	if err != nil {
		s.logError(r, "get_shift_type_id", err, http.StatusInternalServerError, "shift_id", id)
		http.Error(w, "Failed to delete shift", http.StatusInternalServerError)
		return
	}

	// Delete the shift
	err = s.db.DeleteShift(id, typeID)
	if err != nil {
		s.logError(r, "delete_shift", err, http.StatusInternalServerError,
			"shift_id", id, "type_id", typeID)
		http.Error(w, "Failed to delete shift", http.StatusInternalServerError)
		return
	}

	// Log successful deletion
	s.logSuccess(r, "delete_shift", http.StatusNoContent, "shift_id", id, "type_id", typeID, "plan_id", plan.ID)
	w.WriteHeader(http.StatusNoContent)
}
