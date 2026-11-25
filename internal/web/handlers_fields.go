package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/validation"
	"github.com/gorilla/mux"
)

// getFieldID parses the field ID from the URL path.
func (s *Server) getFieldID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)["fieldId"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(ErrInvalidInput, "Invalid field ID", http.StatusBadRequest, err))
		return 0, false
	}
	return id, true
}

// fieldError maps database errors of the field operations to a response.
func (s *Server) fieldError(w http.ResponseWriter, err error, message string) {
	if errors.Is(err, db.ErrFieldNotFound) {
		s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Field not found", http.StatusNotFound, err))
		return
	}
	s.RespondWithError(w, NewAppError(ErrDatabaseError, message, http.StatusInternalServerError, err))
}

// handleCreateField creates a person field
// @Summary Create person field
// @Description Creates a custom person field (choice, number, boolean or text) for the plan. Requires admin authentication.
// @Tags Fields
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param field body model.PlanFieldRequest true "Field definition"
// @Success 200 {object} model.PlanFieldResponse "Created field"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/fields [post]
func (s *Server) handleCreateField(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.PlanFieldRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	if result := validation.ValidatePlanField(&req, plan.IsMultilingual, true); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	field, err := s.db.CreatePlanField(plan.ID, PlanFieldRequestToNew(&req))
	if err != nil {
		s.fieldError(w, err, "Failed to create field")
		return
	}
	s.notifyPlanChanged(plan.ID, scopePlan)
	s.respondJSON(w, PlanFieldToResponse(field))
}

// handleUpdateField updates a person field
// @Summary Update person field
// @Description Updates labels, options, editor and required flag of a person field. The kind cannot be changed. Requires admin authentication.
// @Tags Fields
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param fieldId path int true "Field ID"
// @Param X-Admin-Token header string true "Admin token"
// @Param field body model.PlanFieldRequest true "Field definition"
// @Success 200 {object} model.PlanFieldResponse "Updated field"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or field not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/fields/{fieldId} [put]
func (s *Server) handleUpdateField(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	fieldID, ok := s.getFieldID(w, r)
	if !ok {
		return
	}
	current := validation.FindField(plan.Fields, fieldID)
	if current == nil {
		s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Field not found", http.StatusNotFound, nil))
		return
	}
	var req model.PlanFieldRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	req.Kind = current.Kind // the kind cannot change
	if result := validation.ValidatePlanField(&req, plan.IsMultilingual, false); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	field, err := s.db.UpdatePlanField(plan.ID, fieldID, PlanFieldRequestToNew(&req))
	if err != nil {
		s.fieldError(w, err, "Failed to update field")
		return
	}
	s.notifyPlanChanged(plan.ID, scopePlan)
	s.respondJSON(w, PlanFieldToResponse(field))
}

// handleDeleteField deletes a person field
// @Summary Delete person field
// @Description Deletes a person field. If it was the experience field, the experience settings and all experience needs of types and shifts are cleared. Requires admin authentication.
// @Tags Fields
// @Param token path string true "Plan token"
// @Param fieldId path int true "Field ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Field deleted"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or field not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/fields/{fieldId} [delete]
func (s *Server) handleDeleteField(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	fieldID, ok := s.getFieldID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeletePlanField(plan.ID, fieldID); err != nil {
		s.fieldError(w, err, "Failed to delete field")
		return
	}
	s.notifyPlanChanged(plan.ID, scopePlan)
	s.respondNoContent(w)
}

// handleUpdateFieldOrder sets the order of the person fields
// @Summary Update person field order
// @Description Updates the display order of the person fields. Requires admin authentication.
// @Tags Fields
// @Accept json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body model.FieldOrderRequest true "Field IDs in desired order"
// @Success 200 {string} string "Order updated"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or field not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/fields/order [put]
func (s *Server) handleUpdateFieldOrder(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.FieldOrderRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	if err := s.db.UpdatePlanFieldOrder(plan.ID, req.FieldIDs); err != nil {
		s.fieldError(w, err, "Failed to update field order")
		return
	}
	s.notifyPlanChanged(plan.ID, scopePlan)
	s.respondOK(w)
}
