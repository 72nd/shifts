package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleGetShiftTypes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "get_shift_types_request", "token", token)

	// Get plan
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

	// Get types
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logError(r, "get_shift_types", err, http.StatusInternalServerError, "plan_id", plan.ID)
		http.Error(w, "Failed to get shift types", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "get_shift_types", http.StatusOK, "plan_id", plan.ID, "count", len(types))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

// handleCreateShiftType handles the creation of a new shift type
func (s *Server) handleCreateShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "create_shift_type_request", "token", token)

	// Get plan
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

	// Parse request body
	var shiftType db.NewShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set plan ID
	shiftType.PlanID = plan.ID

	// Create shift type
	createdType, err := s.db.CreateShiftType(shiftType)
	if err != nil {
		s.logError(r, "create_shift_type", err, http.StatusInternalServerError,
			"plan_id", plan.ID, "title_en", shiftType.TitleEN)
		http.Error(w, "Failed to create shift type", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "create_shift_type", http.StatusCreated,
		"plan_id", plan.ID, "type_id", createdType.ID, "title_en", createdType.TitleEN)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdType)
}

// handleUpdateShiftType handles updating an existing shift type
func (s *Server) handleUpdateShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["typeId"], 10, 64)
	if err != nil {
		s.logError(r, "parse_type_id", err, http.StatusBadRequest, "type_id", vars["typeId"])
		http.Error(w, "Invalid type ID", http.StatusBadRequest)
		return
	}

	s.logRequest(r, "update_shift_type_request", "token", token, "type_id", typeID)

	// Get plan
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

	// Verify type belongs to plan
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logError(r, "get_shift_types", err, http.StatusInternalServerError, "plan_id", plan.ID)
		http.Error(w, "Failed to verify type ownership", http.StatusInternalServerError)
		return
	}

	typeBelongsToPlan := false
	for _, t := range types {
		if t.ID == typeID {
			typeBelongsToPlan = true
			break
		}
	}

	if !typeBelongsToPlan {
		s.logError(r, "type_not_found", fmt.Errorf("type does not belong to plan"), http.StatusNotFound,
			"type_id", typeID, "plan_id", plan.ID)
		http.Error(w, "Type not found", http.StatusNotFound)
		return
	}

	// Parse request body
	var shiftType db.ShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update shift type
	if err := s.db.UpdateShiftType(typeID, shiftType); err != nil {
		s.logError(r, "update_shift_type", err, http.StatusInternalServerError,
			"type_id", typeID, "title_en", shiftType.TitleEN)
		http.Error(w, "Failed to update shift type", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "update_shift_type", http.StatusNoContent,
		"type_id", typeID, "plan_id", plan.ID, "title_en", shiftType.TitleEN)

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteShiftType handles deleting an existing shift type
func (s *Server) handleDeleteShiftType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["typeId"], 10, 64)
	if err != nil {
		s.logError(r, "parse_type_id", err, http.StatusBadRequest, "type_id", vars["typeId"])
		http.Error(w, "Invalid type ID", http.StatusBadRequest)
		return
	}

	s.logRequest(r, "delete_shift_type_request", "token", token, "type_id", typeID)

	// Get plan
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

	// Verify type belongs to plan
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.logError(r, "get_shift_types", err, http.StatusInternalServerError, "plan_id", plan.ID)
		http.Error(w, "Failed to verify type ownership", http.StatusInternalServerError)
		return
	}

	typeBelongsToPlan := false
	for _, t := range types {
		if t.ID == typeID {
			typeBelongsToPlan = true
			break
		}
	}

	if !typeBelongsToPlan {
		s.logError(r, "type_not_found", fmt.Errorf("type does not belong to plan"), http.StatusNotFound,
			"type_id", typeID, "plan_id", plan.ID)
		http.Error(w, "Type not found", http.StatusNotFound)
		return
	}

	// Delete shift type
	if err := s.db.DeleteShiftType(plan.ID, typeID); err != nil {
		s.logError(r, "delete_shift_type", err, http.StatusInternalServerError, "type_id", typeID)
		http.Error(w, "Failed to delete shift type", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "delete_shift_type", http.StatusNoContent, "type_id", typeID, "plan_id", plan.ID)

	w.WriteHeader(http.StatusNoContent)
}
