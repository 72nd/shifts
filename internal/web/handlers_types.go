package web

import (
	"net/http"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/validation"
)

// handleGetShiftTypes retrieves all shift types for a plan
// @Summary Get shift types
// @Description Retrieves all shift types for a plan, including their associated shifts
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Success 200 {array} model.ShiftTypeResponse "List of shift types"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/types [get]
func (s *Server) handleGetShiftTypes(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	if s.notModified(w, r, plan.ID) {
		return
	}

	// Get shift types with their shifts
	types, err := s.db.GetShiftTypesByPlan(plan.ID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift types",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.hideDeclined(r, plan, types)

	// Convert to response DTOs
	response := ShiftTypesToResponse(types)
	s.respondJSON(w, response)
}

// handleCreateShiftType handles the creation of a new shift type
// @Summary Create shift type
// @Description Creates a new shift type for a plan. Requires admin authentication.
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param shiftType body model.CreateShiftTypeRequest true "Shift type details"
// @Success 200 {object} model.ShiftTypeResponse "Created shift type"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/types [post]
func (s *Server) handleCreateShiftType(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	// Parse request body
	var req model.CreateShiftTypeRequest
	if !s.parseJSON(w, r, &req) {
		return
	}

	// Validate and sanitize input
	if result := validation.ValidateCreateShiftTypeRequest(&req, plan.IsMultilingual); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	if result := validateTypeCoordination(req.LoadFactor, req.Popularity, req.ExpMinCount, req.ExpMinLevel, plan); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Convert to db model
	shiftType := CreateShiftTypeRequestToNew(&req, plan.ID)

	// Create shift type
	createdType, err := s.db.CreateShiftType(shiftType)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to create shift type",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.notifyPlanChanged(plan.ID, scopeTypes)

	// Convert to response DTO
	response := ShiftTypeToResponse(createdType)
	s.respondJSON(w, response)
}

// handleUpdateShiftType handles updating an existing shift type
// @Summary Update shift type
// @Description Updates an existing shift type. Requires admin authentication.
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param typeId path int true "Shift type ID"
// @Param X-Admin-Token header string true "Admin token"
// @Param shiftType body ShiftTypeUpdateRequest true "Updated shift type details"
// @Success 200 "Shift type updated successfully"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or type not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/types/{typeId} [put]
func (s *Server) handleUpdateShiftType(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	typeID, ok := s.getTypeID(w, r)
	if !ok {
		return
	}

	// Parse request body
	var req model.ShiftTypeUpdateRequest
	if !s.parseJSON(w, r, &req) {
		return
	}

	// Validate and sanitize input
	if result := validation.ValidateShiftTypeUpdate(&req, plan.IsMultilingual); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	if result := validateTypeCoordination(req.LoadFactor, req.Popularity, req.ExpMinCount, req.ExpMinLevel, plan); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Coordination settings that are not sent stay as they are
	existing, err := s.db.GetShiftTypeByID(plan.ID, typeID)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to get shift type",
			http.StatusInternalServerError,
			err,
		))
		return
	}
	if existing == nil {
		s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Shift type not found", http.StatusNotFound, nil))
		return
	}

	// Convert to ShiftType
	shiftType := db.ShiftType{
		ID:            typeID,
		PlanID:        plan.ID,
		TitleDE:       req.TitleDE,
		TitleEN:       req.TitleEN,
		DescriptionDE: req.DescriptionDE,
		DescriptionEN: req.DescriptionEN,
		LocationDE:    req.LocationDE,
		LocationEN:    req.LocationEN,
		LoadFactor:    existing.LoadFactor,
		Popularity:    existing.Popularity,
		ExpMinCount:   existing.ExpMinCount,
		ExpMinLevel:   existing.ExpMinLevel,
	}
	if req.LoadFactor != nil {
		shiftType.LoadFactor = *req.LoadFactor
	}
	if req.Popularity != nil {
		shiftType.Popularity = *req.Popularity
	}
	if req.ExpMinCount != nil {
		shiftType.ExpMinCount = *req.ExpMinCount
	}
	if req.ExpMinLevel != nil {
		if *req.ExpMinLevel == "" {
			shiftType.ExpMinLevel = nil
		} else {
			shiftType.ExpMinLevel = req.ExpMinLevel
		}
	}

	// Update shift type
	if err := s.db.UpdateShiftType(typeID, shiftType); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to update shift type",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.notifyPlanChanged(plan.ID, scopeTypes)
	s.respondOK(w)
}

// handleDeleteShiftType handles deleting an existing shift type
// @Summary Delete shift type
// @Description Deletes a shift type and all its associated shifts. Requires admin authentication.
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param typeId path int true "Shift type ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 200 "Shift type deleted successfully"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or type not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/types/{typeId} [delete]
func (s *Server) handleDeleteShiftType(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}

	typeID, ok := s.getTypeID(w, r)
	if !ok {
		return
	}

	// Delete the shift type
	if err := s.db.DeleteShiftType(plan.ID, typeID); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrDatabaseError,
			"Failed to delete shift type",
			http.StatusInternalServerError,
			err,
		))
		return
	}

	s.notifyPlanChanged(plan.ID, scopeTypes)
	s.respondOK(w)
}

// validateTypeCoordination validates the coordination settings of a shift type request.
// A type has no place count, so the experience need is only bounded by a fixed maximum.
func validateTypeCoordination(loadFactor *float64, popularity *string, count *int, level *string, plan *db.ShiftPlan) *validation.Result {
	r := validation.NewResult()
	if loadFactor != nil {
		validation.ValidateLoadFactor(*loadFactor, "load_factor", r)
	}
	if popularity != nil && !validation.ValidPopularity(*popularity) {
		r.AddInvalid("popularity", "Popularity must be auto, popular, very_popular or normal")
	}
	validation.ValidateExperienceNeed(count, level, 0, plan, r)
	return r
}
