package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/72nd/schipla/internal/db"
	"github.com/gorilla/mux"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r, "index_page_request")
	s.templates.render(w, "index.html", nil)
}

func (s *Server) handleNewShiftPlan(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r, "new_shift_plan_page_request")
	s.templates.render(w, "new.html", nil)
}

func (s *Server) handleShiftPlanSuccess(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	password := r.URL.Query().Get("password")

	s.logRequest(r, "shift_plan_success_page_request", "url_param", url)

	if url == "" || password == "" {
		s.logError(r, "missing_parameters", fmt.Errorf("missing url or password"), http.StatusBadRequest)
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
	s.logRequest(r, "create_shift_plan_request")

	var plan db.NewShiftPlan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create plan
	createdPlan, err := s.db.CreateShiftPlan(plan)
	if err != nil {
		s.logError(r, "create_shift_plan", err, http.StatusInternalServerError,
			"title_en", plan.TitleEN, "is_multilingual", plan.IsMultilingual)
		http.Error(w, "Failed to create plan", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "create_shift_plan", http.StatusCreated,
		"plan_id", createdPlan.ID, "title_en", createdPlan.TitleEN, "token", createdPlan.Token)

	// Return plan data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":    createdPlan.Token,
		"url":      fmt.Sprintf("/plans/%s", createdPlan.Token),
		"password": plan.Password,
	})
}

func (s *Server) handleGetShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "get_shift_plan_request", "token", token)

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

	// Get shifts
	shifts, err := s.db.GetShiftsByPlan(plan.ID)
	if err != nil {
		s.logError(r, "get_shifts", err, http.StatusInternalServerError, "plan_id", plan.ID)
		http.Error(w, "Failed to get shifts", http.StatusInternalServerError)
		return
	}

	// Get participants for each shift
	for i, shift := range shifts {
		participants, err := s.db.GetParticipantsByShift(shift.ID)
		if err != nil {
			s.logError(r, "get_participants", err, http.StatusInternalServerError, "shift_id", shift.ID)
			http.Error(w, "Failed to get participants", http.StatusInternalServerError)
			return
		}
		shifts[i].Participants = participants
	}

	// Combine data
	data := map[string]interface{}{
		"plan":   plan,
		"types":  types,
		"shifts": shifts,
	}

	// Log success
	s.logSuccess(r, "get_shift_plan", http.StatusOK, "plan_id", plan.ID, "token", token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleViewShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "view_shift_plan_page_request", "token", token)

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

	s.templates.render(w, "view.html", map[string]string{
		"token": token,
		"title": plan.TitleEN,
	})
}

func (s *Server) handleUpdateShiftTypeOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "update_shift_type_order_request", "token", token)

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
	var typeIDs []int64
	if err := json.NewDecoder(r.Body).Decode(&typeIDs); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update order
	if err := s.db.UpdateShiftTypeOrder(plan.ID, typeIDs); err != nil {
		s.logError(r, "update_shift_type_order", err, http.StatusInternalServerError,
			"plan_id", plan.ID, "type_ids", typeIDs)
		http.Error(w, "Failed to update shift type order", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "update_shift_type_order", http.StatusNoContent, "plan_id", plan.ID)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	s.logRequest(r, "update_shift_plan_request", "token", token)

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

	// Verify admin token
	adminToken := r.Header.Get("X-Admin-Token")
	hashedToken := hashPassword(adminToken)
	if hashedToken != plan.AdminPassword {
		s.logError(r, "unauthorized", fmt.Errorf("invalid admin token"), http.StatusUnauthorized, "token", token)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var updateData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		s.logError(r, "decode_request_body", err, http.StatusBadRequest)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Konvertiere in NewShiftPlan-Struktur
	newPlan := db.NewShiftPlan{
		TitleEN:       updateData.Title,
		DescriptionEN: updateData.Description,
		// Übernehme die bestehenden Werte für die anderen Felder
		StartDate:      plan.StartDate,
		EndDate:        plan.EndDate,
		IsMultilingual: plan.IsMultilingual,
		Password:       "", // Passwort wird nicht aktualisiert
	}

	// Update plan
	updatedPlan, err := s.db.UpdateShiftPlan(token, newPlan)
	if err != nil {
		s.logError(r, "update_shift_plan", err, http.StatusInternalServerError,
			"plan_id", plan.ID, "title_en", newPlan.TitleEN)
		http.Error(w, "Failed to update plan", http.StatusInternalServerError)
		return
	}

	// Log success
	s.logSuccess(r, "update_shift_plan", http.StatusNoContent,
		"plan_id", plan.ID, "title_en", updatedPlan.TitleEN)

	w.WriteHeader(http.StatusNoContent)
}
