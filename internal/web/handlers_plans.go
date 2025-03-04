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
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validiere und sanitisiere die Titel
	plan.TitleEN = SanitizeText(plan.TitleEN)
	if plan.TitleEN == "" {
		http.Error(w, "English title is required", http.StatusBadRequest)
		return
	}
	if len(plan.TitleEN) > 150 {
		http.Error(w, "English title cannot exceed 150 characters", http.StatusBadRequest)
		return
	}

	if plan.IsMultilingual && plan.TitleDE != nil {
		*plan.TitleDE = SanitizeText(*plan.TitleDE)
		if *plan.TitleDE == "" {
			http.Error(w, "German title is required for multilingual plans", http.StatusBadRequest)
			return
		}
		if len(*plan.TitleDE) > 150 {
			http.Error(w, "German title cannot exceed 150 characters", http.StatusBadRequest)
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
		s.logger.Error("failed to create plan", "error", err)
		http.Error(w, "Failed to create plan", http.StatusInternalServerError)
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
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
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
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
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
		s.logger.Error("failed to get plan", "error", err)
		http.Error(w, "Plan not found", http.StatusNotFound)
		return
	}
	if plan == nil {
		http.Error(w, "Plan not found", http.StatusNotFound)
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
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}

	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Get statistics
	stats, err := s.db.GetShiftPlanStats(plan.ID)
	if err != nil {
		s.logger.Error("failed to get shift plan statistics", "error", err)
		http.Error(w, "Failed to get statistics", http.StatusInternalServerError)
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
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Parse request body
	var data struct {
		TypeIDs []int64 `json:"type_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update type order
	if err := s.db.UpdateShiftTypeOrder(plan.ID, data.TypeIDs); err != nil {
		s.logger.Error("failed to update shift type order", "error", err)
		http.Error(w, "Failed to update shift type order", http.StatusInternalServerError)
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
		s.logger.Error("failed to get shift plan", "error", err)
		http.Error(w, "Failed to get shift plan", http.StatusInternalServerError)
		return
	}
	if plan == nil {
		http.NotFound(w, r)
		return
	}

	// Verify admin token
	adminToken := r.Header.Get("X-Admin-Token")
	if adminToken == "" {
		http.Error(w, "Admin token required", http.StatusUnauthorized)
		return
	}

	isValid, err := s.db.ValidateShiftPlanPassword(plan.ID, hashPassword(adminToken))
	if err != nil {
		s.logger.Error("failed to validate admin token", "error", err)
		http.Error(w, "Failed to validate admin token", http.StatusInternalServerError)
		return
	}
	if !isValid {
		http.Error(w, "Invalid admin token", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var updateData db.NewShiftPlan
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		s.logger.Error("failed to decode request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update plan
	updatedPlan, err := s.db.UpdateShiftPlan(token, updateData)
	if err != nil {
		s.logger.Error("failed to update plan", "error", err)
		http.Error(w, "Failed to update plan", http.StatusInternalServerError)
		return
	}

	// Return updated plan data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedPlan)
}
