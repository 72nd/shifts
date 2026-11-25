package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/validation"
	"github.com/gorilla/mux"
)

// getPlanByToken retrieves a plan by token from the URL path.
// Returns the plan and true if successful, or writes an error response and returns nil, false.
func (s *Server) getPlanByToken(w http.ResponseWriter, r *http.Request) (*db.ShiftPlan, bool) {
	vars := mux.Vars(r)
	token := vars["token"]

	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift plan",
			http.StatusInternalServerError,
			err,
		))
		return nil, false
	}

	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return nil, false
	}

	return plan, true
}

// getShiftID parses the shift ID from the URL path.
// Returns the ID and true if successful, or writes an error response and returns 0, false.
func (s *Server) getShiftID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	vars := mux.Vars(r)
	shiftIDStr := vars["shiftId"]

	id, err := strconv.ParseInt(shiftIDStr, 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift ID",
			http.StatusBadRequest,
			err,
		))
		return 0, false
	}

	return id, true
}

// getTypeID parses the type ID from the URL path.
// Returns the ID and true if successful, or writes an error response and returns 0, false.
func (s *Server) getTypeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	vars := mux.Vars(r)
	typeIDStr := vars["typeId"]

	id, err := strconv.ParseInt(typeIDStr, 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift type ID",
			http.StatusBadRequest,
			err,
		))
		return 0, false
	}

	return id, true
}

// verifyShiftOwnership checks if a shift belongs to a plan.
// Returns true if the shift belongs to the plan, or writes an error response and returns false.
func (s *Server) verifyShiftOwnership(w http.ResponseWriter, shiftID, planID int64) bool {
	belongs, err := s.db.VerifyShiftBelongsToPlan(shiftID, planID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify shift ownership",
			http.StatusInternalServerError,
			err,
		))
		return false
	}

	if !belongs {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift not found",
			http.StatusNotFound,
			nil,
		))
		return false
	}

	return true
}

// getShiftByID retrieves a shift by ID and verifies it exists.
// Returns the shift and true if successful, or writes an error response and returns nil, false.
func (s *Server) getShiftByID(w http.ResponseWriter, shiftID int64) (*db.Shift, bool) {
	shift, err := s.db.GetShiftByID(shiftID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift",
			http.StatusInternalServerError,
			err,
		))
		return nil, false
	}

	if shift == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift not found",
			http.StatusNotFound,
			nil,
		))
		return nil, false
	}

	return shift, true
}

// verifyAdminForLockedShift checks admin access when a shift is locked.
// Returns true if access is granted, or writes an error response and returns false.
// If the shift is not locked, always returns true.
func (s *Server) verifyAdminForLockedShift(w http.ResponseWriter, r *http.Request, shift *db.Shift, plan *db.ShiftPlan) bool {
	if !shift.IsLocked {
		return true
	}

	adminToken := r.Header.Get("X-Admin-Token")
	if adminToken == "" {
		s.RespondWithError(w, NewAppError(
			ErrUnauthorized,
			"Shift is locked - admin access required",
			http.StatusForbidden,
			nil,
		))
		return false
	}

	isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify admin token",
			http.StatusInternalServerError,
			err,
		))
		return false
	}

	if !isValid {
		s.RespondWithError(w, NewAppError(
			ErrUnauthorized,
			"Shift is locked - admin access required",
			http.StatusForbidden,
			nil,
		))
		return false
	}

	return true
}

// getAdminToken extracts the admin token from the request header.
// Returns the token and true if present, or writes an error response and returns empty string, false.
func (s *Server) getAdminToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	adminToken := r.Header.Get("X-Admin-Token")
	if adminToken == "" {
		s.RespondWithError(w, NewAppError(
			ErrUnauthorized,
			"Admin token required",
			http.StatusUnauthorized,
			nil,
		))
		return "", false
	}
	return adminToken, true
}

// verifyAdminToken verifies the admin token from the request header.
// Returns true if valid, or writes an error response and returns false.
func (s *Server) verifyAdminToken(w http.ResponseWriter, r *http.Request, plan *db.ShiftPlan) bool {
	adminToken, ok := s.getAdminToken(w, r)
	if !ok {
		return false
	}

	isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to verify admin token",
			http.StatusInternalServerError,
			err,
		))
		return false
	}

	if !isValid {
		s.RespondWithError(w, NewAppError(
			ErrInvalidPassword,
			"Invalid admin token",
			http.StatusUnauthorized,
			nil,
		))
		return false
	}

	return true
}

// parseJSON decodes JSON from the request body into the target struct.
// Returns true if successful, or writes an error response and returns false.
func (s *Server) parseJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return false
	}
	return true
}

// respondValidationError writes a validation error response.
func (s *Server) respondValidationError(w http.ResponseWriter, result *validation.Result) {
	s.RespondWithError(w, NewAppError(
		ErrInvalidInput,
		result.FirstError(),
		http.StatusBadRequest,
		nil,
	))
}

// respondJSON writes a JSON response with the given data.
func (s *Server) respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// respondNoContent writes a 204 No Content response.
func (s *Server) respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// respondOK writes a 200 OK response.
func (s *Server) respondOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// isAdminRequest reports whether the request carries a valid admin token for the plan.
// A missing or wrong token is not an error here; the caller decides what a non-admin may see.
func (s *Server) isAdminRequest(r *http.Request, plan *db.ShiftPlan) bool {
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		return false
	}
	ok, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(token))
	return err == nil && ok
}

// requireAdminForbidden checks the admin token and answers 403 with message if it is missing or wrong.
func (s *Server) requireAdminForbidden(w http.ResponseWriter, r *http.Request, plan *db.ShiftPlan, message string) bool {
	if s.isAdminRequest(r, plan) {
		return true
	}
	s.RespondWithError(w, NewAppError(ErrUnauthorized, message, http.StatusForbidden, nil))
	return false
}

// hideDeclined removes declined participations from the types if the viewer must not see them:
// the plan hides them (show_declined = false) and the request is not from an admin.
// This is the single place where the visibility rule is applied.
func (s *Server) hideDeclined(r *http.Request, plan *db.ShiftPlan, types []db.ShiftType) {
	if plan.ShowDeclined || s.isAdminRequest(r, plan) {
		return
	}
	for i := range types {
		for j := range types[i].Shifts {
			kept := types[i].Shifts[j].Participants[:0]
			for _, p := range types[i].Shifts[j].Participants {
				if p.Status != db.StatusDeclined {
					kept = append(kept, p)
				}
			}
			types[i].Shifts[j].Participants = kept
		}
	}
}
