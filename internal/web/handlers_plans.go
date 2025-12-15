package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/export"
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

// handleCreateShiftPlan creates a new shift plan
// @Summary Create a new shift plan
// @Description Creates a new shift plan with the provided details. Returns the plan token, URL, and admin password.
// @Tags Plans
// @Accept json
// @Produce json
// @Param plan body db.NewShiftPlan true "Shift plan details"
// @Success 200 {object} map[string]interface{} "Created plan with token, URL, and password"
// @Failure 400 {string} string "Invalid input"
// @Failure 500 {string} string "Internal server error"
// @Router /plans [post]
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

// handleGetShiftPlan retrieves a shift plan by token
// @Summary Get shift plan details
// @Description Retrieves the details of a shift plan by its token
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Success 200 {object} db.ShiftPlan "Shift plan details"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token} [get]
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
		s.logger.Error("failed to get shift plan", "error", err)
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	if plan == nil {
		data := map[string]interface{}{
			"error": fmt.Sprintf("Shift plan with ID '%s' does not exist", token),
		}
		s.templates.render(w, "404.html", data)
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

// handleGetShiftPlanStats retrieves statistics for a shift plan
// @Summary Get shift plan statistics
// @Description Retrieves comprehensive statistics for a shift plan including overview, helper stats, type stats, and daily stats
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Success 200 {object} db.Stats "Plan statistics"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/stats [get]
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

// handleUpdateShiftTypeOrder updates the sort order of shift types
// @Summary Update shift type order
// @Description Updates the display order of shift types for a plan
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body object true "Type IDs in desired order" SchemaExample({"type_ids": [1, 2, 3]})
// @Success 200 {string} string "Order updated successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/types/order [put]
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

// handleUpdateShiftPlan updates an existing shift plan
// @Summary Update shift plan
// @Description Updates the details of an existing shift plan. Requires admin authentication.
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param plan body db.NewShiftPlan true "Updated plan details"
// @Success 200 {object} db.ShiftPlan "Updated shift plan"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token} [put]
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

// handleExportPlan handles the export of a shift plan
// @Summary Export shift plan
// @Description Exports a shift plan in various formats (JSON, PDF, etc.) with optional filtering. Requires admin authentication.
// @Tags Plans
// @Accept json
// @Produce application/json,application/pdf,image/png
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param format query string false "Export format (json, pdf, timetable, report, poster)" default(json)
// @Param day query string false "Filter by day (YYYY-MM-DD)"
// @Param status query string false "Filter by status (past, current, future)"
// @Param search query string false "Search query"
// @Param type_ids query string false "Comma-separated list of type IDs to filter"
// @Success 200 {file} file "Exported file"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/export [get]
func (s *Server) handleExportPlan(w http.ResponseWriter, r *http.Request) {
	// Get plan token from URL
	vars := mux.Vars(r)
	token := vars["token"]
	if token == "" {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Missing plan token",
			http.StatusBadRequest,
			nil,
		))
		return
	}

	// Get admin token from header
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

	// Get export format from query params (default to JSON)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Parse filter parameters from query string
	filter := s.parseExportFilters(r)

	// Export plan using the export service with the new Export method
	data, contentType, filename, err := s.exportService.Export(token, hashPassword(adminToken), format, filter)
	if err != nil {
		var appErr *AppError
		switch e := err.(type) {
		case *export.Error:
			appErr = NewAppError(string(e.Type), e.Message, e.StatusCode(), e)
		default:
			appErr = NewAppError(ErrInternalError, "Export failed", http.StatusInternalServerError, err)
		}
		s.RespondWithError(w, appErr)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Write the data
	if _, err := w.Write(data); err != nil {
		s.logger.Error("failed to write export data", "error", err)
	}
}

// parseExportFilters parses export filter parameters from the request
func (s *Server) parseExportFilters(r *http.Request) *export.ExportFilter {
	query := r.URL.Query()

	filter := &export.ExportFilter{
		Day:         query.Get("day"),
		Status:      query.Get("status"),
		SearchQuery: query.Get("search"),
		TypeIDs:     []int64{},
	}

	// Parse type IDs (comma-separated list)
	if typeIDsStr := query.Get("type_ids"); typeIDsStr != "" {
		typeIDsParts := splitAndTrim(typeIDsStr, ",")
		for _, idStr := range typeIDsParts {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				filter.TypeIDs = append(filter.TypeIDs, id)
			}
		}
	}

	return filter
}

// splitAndTrim splits a string by separator and trims whitespace
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// handleDeleteShiftPlan handles the deletion of a shift plan
// @Summary Delete shift plan
// @Description Deletes a shift plan and all its associated data (shifts, types, participants). Requires admin authentication.
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Plan deleted successfully"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token} [delete]
func (s *Server) handleDeleteShiftPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	// Get admin token from header
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

	// Hash the admin token for verification
	hashedPassword := hashPassword(adminToken)

	// Delete the plan
	if err := s.db.DeleteShiftPlan(token, hashedPassword); err != nil {
		var status int
		var errType string
		var message string

		switch err.Error() {
		case "shift plan not found":
			status = http.StatusNotFound
			errType = ErrResourceNotFound
			message = "Shift plan not found"
		case "invalid password":
			status = http.StatusUnauthorized
			errType = ErrInvalidPassword
			message = "Invalid password"
		default:
			status = http.StatusInternalServerError
			errType = ErrDatabaseError
			message = "Failed to delete shift plan"
		}

		s.RespondWithError(w, NewAppError(
			errType,
			message,
			status,
			err,
		))
		return
	}

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}
