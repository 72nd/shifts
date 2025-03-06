package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.templates.render(w, "index.html", nil)
}

func (s *Server) handleNewShiftPlan(w http.ResponseWriter, r *http.Request) {
	s.templates.render(w, "new.html", nil)
}

func (s *Server) handleShiftPlanSuccess(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	password := r.URL.Query().Get("password")

	if url == "" || password == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]string{
		"url":      url,
		"password": password,
	}

	s.templates.render(w, "success.html", data)
}

func (s *Server) handleCreateShiftPlan(w http.ResponseWriter, r *http.Request) {
	var plan db.NewShiftPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Validiere und sanitisiere die Titel
	plan.TitleEN = SanitizeText(plan.TitleEN)
	if plan.TitleEN == "" {
		s.RespondWithError(w, NewAppError(
			ErrMissingField,
			"English title is required",
			http.StatusBadRequest,
			nil,
		))
		return
	}
	if len(plan.TitleEN) > 150 {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"English title cannot exceed 150 characters",
			http.StatusBadRequest,
			nil,
		))
		return
	}

	if plan.IsMultilingual && plan.TitleDE != nil {
		*plan.TitleDE = SanitizeText(*plan.TitleDE)
		if *plan.TitleDE == "" {
			s.RespondWithError(w, NewAppError(
				ErrMissingField,
				"German title is required for multilingual plans",
				http.StatusBadRequest,
				nil,
			))
			return
		}
		if len(*plan.TitleDE) > 150 {
			s.RespondWithError(w, NewAppError(
				ErrInvalidInput,
				"German title cannot exceed 150 characters",
				http.StatusBadRequest,
				nil,
			))
			return
		}
	}

	// Sanitisiere die Beschreibungen (optional)
	if plan.DescriptionEN != "" {
		plan.DescriptionEN = SanitizeText(plan.DescriptionEN)
	}
	if plan.DescriptionDE != nil && *plan.DescriptionDE != "" {
		*plan.DescriptionDE = SanitizeText(*plan.DescriptionDE)
	}

	// Create plan
	createdPlan, err := s.db.CreateShiftPlan(plan)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to create plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Return plan data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":    createdPlan.Token,
		"url":      fmt.Sprintf("/plans/%s", createdPlan.Token),
		"password": createdPlan.AdminPassword,
	})
}

func (s *Server) handleGetShiftPlan(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

func (s *Server) handleViewShiftPlan(w http.ResponseWriter, r *http.Request) {
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

	s.templates.render(w, "view.html", plan)
}

func (s *Server) handleShiftPlanStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Check if plan exists
	plan, err := s.db.GetShiftPlanByToken(token)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	s.templates.render(w, "stats.html", map[string]interface{}{
		"plan": plan,
	})
}

func (s *Server) handleGetShiftPlanStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan first to verify it exists and get its ID
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

	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Get statistics
	stats, err := s.db.GetShiftPlanStats(plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get statistics",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleUpdateShiftTypeOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan to verify ownership
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
	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Parse request body
	var data struct {
		TypeIDs []int64 `json:"type_ids"`
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

	// Update type order
	if err := s.db.UpdateShiftTypeOrder(plan.ID, data.TypeIDs); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update shift type order",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUpdateShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get plan to verify ownership
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
	if plan == nil {
		s.RespondWithError(w, NewAppError(
			ErrResourceNotFound,
			"Shift plan not found",
			http.StatusNotFound,
			nil,
		))
		return
	}

	// Verify admin token
	adminToken := r.Header.Get("X-Admin-Token")
	if adminToken == "" {
		s.RespondWithError(w, NewAppError(
			ErrUnauthorized,
			"Admin token required",
			http.StatusUnauthorized,
			nil,
		))
		return
	}

	isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to validate admin token",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if !isValid {
		s.RespondWithError(w, NewAppError(
			ErrInvalidPassword,
			"Invalid admin token",
			http.StatusUnauthorized,
			nil,
		))
		return
	}

	// Parse request body
	var updateData db.NewShiftPlan
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Update plan
	updatedPlan, err := s.db.UpdateShiftPlan(token, updateData)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	// Return updated plan data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedPlan)
}
