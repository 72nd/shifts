package web

import (
	"net/http"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/validation"
)

// handleGetShiftTypes retrieves all shift types for a plan
// @Summary Get shift types
// @Description Retrieves all shift types for a plan, including their associated shifts
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Success 200 {array} db.ShiftType "List of shift types"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Router /plans/{token}/types [get]
func (s *Server) handleGetShiftTypes(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
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

	s.respondJSON(w, types)
}

// handleCreateShiftType handles the creation of a new shift type
// @Summary Create shift type
// @Description Creates a new shift type for a plan. Requires admin authentication.
// @Tags Types
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param shiftType body db.NewShiftType true "Shift type details"
// @Success 200 {object} db.ShiftType "Created shift type"
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
	var shiftType db.NewShiftType
	if !s.parseJSON(w, r, &shiftType) {
		return
	}

	// Validate and sanitize input
	if result := validation.ValidateNewShiftType(&shiftType, plan.IsMultilingual); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Set plan ID
	shiftType.PlanID = plan.ID

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

	s.respondJSON(w, createdType)
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
// @Param shiftType body object true "Updated shift type details" SchemaExample({"title_en": "Kitchen Duty", "title_de": "Küchendienst", "description_en": "Help in kitchen", "description_de": "Hilfe in Küche", "location_en": "Main Kitchen", "location_de": "Hauptküche"})
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
	var requestData validation.ShiftTypeUpdate
	if !s.parseJSON(w, r, &requestData) {
		return
	}

	// Validate and sanitize input
	if result := validation.ValidateShiftTypeUpdate(&requestData, plan.IsMultilingual); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}

	// Convert to ShiftType
	shiftType := db.ShiftType{
		ID:            typeID,
		PlanID:        plan.ID,
		TitleDE:       requestData.TitleDE,
		TitleEN:       requestData.TitleEN,
		DescriptionDE: requestData.DescriptionDE,
		DescriptionEN: requestData.DescriptionEN,
		LocationDE:    requestData.LocationDE,
		LocationEN:    requestData.LocationEN,
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

	s.respondOK(w)
}
