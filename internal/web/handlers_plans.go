package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/export"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/validation"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.templates.render(w, "index.html", nil)
}

// newShiftPlanPageData is passed to new.html (wrapped as template "Data").
type newShiftPlanPageData struct {
	// DefaultTimeZone is the IANA name preselected for new plans.
	DefaultTimeZone string
}

func (s *Server) handleNewShiftPlan(w http.ResponseWriter, r *http.Request) {
	s.templates.render(w, "new.html", newShiftPlanPageData{DefaultTimeZone: s.defaultZone})
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
// @Param plan body model.CreateShiftPlanRequest true "Shift plan details"
// @Success 200 {object} model.CreateShiftPlanResponse "Created plan with token, URL, and password"
// @Failure 400 {string} string "Invalid input"
// @Failure 500 {string} string "Internal server error"
// @Router /plans [post]
func (s *Server) handleCreateShiftPlan(w http.ResponseWriter, r *http.Request) {
	var req model.CreateShiftPlanRequest
	if !s.parseJSONTimes(w, r, &req) {
		return
	}

	// Validate and sanitize input
	if result := validation.ValidateCreateShiftPlanRequest(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Convert to db model
	plan := CreateShiftPlanRequestToNew(&req)

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

	// Track metrics
	shiftPlansCreatedTotal.Inc()

	// Return response DTO
	response := model.CreateShiftPlanResponse{
		Token:    createdPlan.Token,
		URL:      fmt.Sprintf("/plans/%s", createdPlan.Token),
		Password: createdPlan.AdminPassword,
	}

	s.respondJSON(w, response)
}

// handleGetShiftPlan retrieves a shift plan by token
// @Summary Get shift plan details
// @Description Retrieves the details of a shift plan by its token
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Success 200 {object} model.ShiftPlanResponse "Shift plan details"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token} [get]
func (s *Server) handleGetShiftPlan(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	if s.notModified(w, r, plan.ID) {
		return
	}

	// Convert to response DTO
	response := ShiftPlanToResponse(plan)
	s.respondJSON(w, response)
}

// viewShiftPlanPageData is passed to view.html (wrapped as template "Data").
type viewShiftPlanPageData struct {
	Plan        *db.ShiftPlan
	DefaultView string
}

func (s *Server) handleViewShiftPlan(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	s.templates.render(w, "view.html", viewShiftPlanPageData{
		Plan:        plan,
		DefaultView: s.defaultView,
	})
}

func (s *Server) handleShiftPlanStats(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	s.templates.render(w, "stats.html", map[string]interface{}{
		"plan": plan,
	})
}

// handleGetShiftPlanStats retrieves statistics for a shift plan
// @Summary Get shift plan statistics
// @Description Summary of statistics for a shift plan (overview, helper stats, type stats, daily stats), meant for API clients. The web UI computes its statistics itself from the plan data and does not use this endpoint. Unlimited shifts (required -1) do not count into required totals or open positions; overfilled shifts are capped at required for percentages.
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Success 200 {object} model.StatsResponse "Plan statistics"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/stats [get]
func (s *Server) handleGetShiftPlanStats(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
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

	// Convert to response DTO
	response := StatsToResponse(stats)
	s.respondJSON(w, response)
}

// handleUpdateShiftTypeOrder updates the sort order of shift types
// @Summary Update shift type order
// @Description Updates the display order of shift types for a plan
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body ShiftTypeOrderRequest true "Type IDs in desired order"
// @Success 200 {string} string "Order updated successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/types/order [put]
func (s *Server) handleUpdateShiftTypeOrder(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Parse request body
	var data struct {
		TypeIDs []int64 `json:"type_ids"`
	}
	if !s.parseJSON(w, r, &data) {
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

	s.notifyPlanChanged(plan.ID, scopeTypes)
	s.respondOK(w)
}

// handleUpdateShiftPlan updates an existing shift plan
// @Summary Update shift plan
// @Description Updates the details of an existing shift plan. Requires admin authentication.
// @Tags Plans
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param plan body model.ShiftPlanUpdateRequest true "Updated plan details"
// @Success 200 {object} model.ShiftPlanResponse "Updated shift plan"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token} [put]
func (s *Server) handleUpdateShiftPlan(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Verify admin token
	if !s.verifyAdminToken(w, r, plan) {
		return
	}

	// Parse request body
	var req model.ShiftPlanUpdateRequest
	if !s.parseJSONTimes(w, r, &req) {
		return
	}

	// Validate and sanitize input
	if result := validation.ValidateShiftPlanUpdate(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	if req.AttendanceMode != "" {
		if result := validation.ValidateCoordinationSettings(&req, plan.Fields); result.HasErrors() {
			s.respondValidationError(w, result)
			return
		}
	}

	// Convert to db model
	updateData := UpdateShiftPlanRequestToNew(&req)

	// Update plan
	updatedPlan, err := s.db.UpdateShiftPlan(plan.Token, updateData)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update plan",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.notifyPlanChanged(plan.ID, scopePlan)

	// Convert to response DTO
	updatedPlan.Fields = plan.Fields
	response := ShiftPlanToResponse(updatedPlan)
	s.respondJSON(w, response)
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
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Get admin token from header
	adminToken, ok := s.getAdminToken(w, r)
	if !ok {
		return
	}

	// Get export format from query params (default to JSON)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Parse filter parameters from query string
	filter := s.parseExportFilters(r)

	// Track export duration
	start := time.Now()

	// Export plan using the export service
	data, contentType, filename, err := s.exportService.Export(plan.Token, hashPassword(adminToken), format, filter)

	// Record export duration and result
	duration := time.Since(start).Seconds()
	exportDuration.WithLabelValues(format).Observe(duration)

	if err != nil {
		// Track export failure
		exportRequestsTotal.WithLabelValues(format, "failure").Inc()

		// Categorize error type
		var errorType string
		switch e := err.(type) {
		case *export.Error:
			errorType = string(e.Type)
		default:
			errorType = "internal_error"
		}
		exportErrorsTotal.WithLabelValues(format, errorType).Inc()

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

	// Track successful export
	exportRequestsTotal.WithLabelValues(format, "success").Inc()

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
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Get admin token from header
	adminToken, ok := s.getAdminToken(w, r)
	if !ok {
		return
	}

	// Delete the plan (handles auth internally)
	if err := s.db.DeleteShiftPlan(plan.Token, hashPassword(adminToken)); err != nil {
		var status int
		var errType string
		var message string

		switch {
		case errors.Is(err, db.ErrNotFound):
			status = http.StatusNotFound
			errType = ErrResourceNotFound
			message = "Shift plan not found"
		case errors.Is(err, db.ErrInvalidPassword):
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

	s.notifyPlanChanged(plan.ID, scopeDeleted)

	// Track metrics
	shiftPlansDeletedTotal.Inc()

	s.respondNoContent(w)
}
