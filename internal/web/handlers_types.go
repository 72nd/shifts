package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/gorilla/mux"
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
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

	// Parse request body
	var shiftType db.NewShiftType
	if err := json.NewDecoder(r.Body).Decode(&shiftType); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Validiere und sanitisiere die Titel
	shiftType.TitleEN = SanitizeText(shiftType.TitleEN)
	if shiftType.TitleEN == "" {
		s.RespondWithError(w, NewAppError(
			ErrMissingField,
			"English title is required",
			http.StatusBadRequest,
			nil,
		))
		return
	}
	if len(shiftType.TitleEN) > 150 {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"English title cannot exceed 150 characters",
			http.StatusBadRequest,
			nil,
		))
		return
	}

	if plan.IsMultilingual {
		shiftType.TitleDE = SanitizeText(shiftType.TitleDE)
		if shiftType.TitleDE == "" {
			s.RespondWithError(w, NewAppError(
				ErrMissingField,
				"German title is required for multilingual plans",
				http.StatusBadRequest,
				nil,
			))
			return
		}
		if len(shiftType.TitleDE) > 150 {
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
	if shiftType.DescriptionEN != "" {
		shiftType.DescriptionEN = SanitizeText(shiftType.DescriptionEN)
	}
	if shiftType.DescriptionDE != "" {
		shiftType.DescriptionDE = SanitizeText(shiftType.DescriptionDE)
	}

	// Sanitisiere die Orte (optional)
	if shiftType.LocationEN != "" {
		shiftType.LocationEN = SanitizeText(shiftType.LocationEN)
	}
	if shiftType.LocationDE != "" {
		shiftType.LocationDE = SanitizeText(shiftType.LocationDE)
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

	// Return created shift type
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdType)
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
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["typeId"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift type ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Get plan first to verify it exists
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

	// Parse request body into a temporary struct that matches the frontend field names
	var requestData struct {
		TitleDE       string `json:"title_de"`
		TitleEN       string `json:"title_en"`
		DescriptionDE string `json:"description_de"`
		DescriptionEN string `json:"description_en"`
		LocationDE    string `json:"location_de"`
		LocationEN    string `json:"location_en"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid request body",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Validiere und sanitisiere die Titel
	requestData.TitleEN = SanitizeText(requestData.TitleEN)
	if requestData.TitleEN == "" {
		s.RespondWithError(w, NewAppError(
			ErrMissingField,
			"English title is required",
			http.StatusBadRequest,
			nil,
		))
		return
	}
	if len(requestData.TitleEN) > 150 {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"English title cannot exceed 150 characters",
			http.StatusBadRequest,
			nil,
		))
		return
	}

	if plan.IsMultilingual {
		requestData.TitleDE = SanitizeText(requestData.TitleDE)
		if requestData.TitleDE == "" {
			s.RespondWithError(w, NewAppError(
				ErrMissingField,
				"German title is required for multilingual plans",
				http.StatusBadRequest,
				nil,
			))
			return
		}
		if len(requestData.TitleDE) > 150 {
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
	if requestData.DescriptionEN != "" {
		requestData.DescriptionEN = SanitizeText(requestData.DescriptionEN)
	}
	if requestData.DescriptionDE != "" {
		requestData.DescriptionDE = SanitizeText(requestData.DescriptionDE)
	}

	// Sanitisiere die Orte (optional)
	if requestData.LocationEN != "" {
		requestData.LocationEN = SanitizeText(requestData.LocationEN)
	}
	if requestData.LocationDE != "" {
		requestData.LocationDE = SanitizeText(requestData.LocationDE)
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

	w.WriteHeader(http.StatusOK)
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
	vars := mux.Vars(r)
	token := vars["token"]
	typeID, err := strconv.ParseInt(vars["typeId"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(
			ErrInvalidInput,
			"Invalid shift type ID",
			http.StatusBadRequest,
			err,
		))
		return
	}

	// Get plan first to verify it exists
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

	w.WriteHeader(http.StatusOK)
}
