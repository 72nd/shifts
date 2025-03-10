package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleCreateShift(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var shift db.NewShift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift plan",
			http.StatusInternalServerError,
			err,
		))
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
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Verify shift belongs to plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(id, plan.ID)
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

	var shift db.Shift
	if err := json.NewDecoder(r.Body).Decode(&shift); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Update the shift
	if err := s.db.UpdateShift(id, shift); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update shift",
			http.StatusInternalServerError,
			err,
		))
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
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Get plan ID from token
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Verify admin token
	adminToken := r.Header.Get("X-Admin-Token")
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
			"Unauthorized",
			http.StatusUnauthorized,
			nil,
		))
		return
	}

	// Verify shift belongs to plan
	belongs, err := s.db.VerifyShiftBelongsToPlan(id, plan.ID)
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

	// Delete the shift
	if err := s.db.DeleteShift(id, plan.ID); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to delete shift",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
