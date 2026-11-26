package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/validation"
)

// itoa64 formats an ID as a decimal string.
func itoa64(id int64) string { return strconv.FormatInt(id, 10) }

// handleCoordinationPage renders the coordination page (admin only; the page checks the login in the browser).
func (s *Server) handleCoordinationPage(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	s.templates.render(w, "coordinate.html", map[string]interface{}{
		"plan": plan,
	})
}

// handleApplyCoordination applies participation status changes atomically
// @Summary Apply coordination operations
// @Description Applies up to 2000 operations in one transaction (all or nothing): set the status of a person in a shift to signed_up, assigned or declined, or remove the participation. A missing participation is created with added_by_admin set (except for remove). There is no capacity check. Optionally locks shifts afterwards. Requires admin authentication.
// @Tags Coordination
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body model.CoordinationApplyRequest true "Operations"
// @Success 200 {object} model.CoordinationApplyResponse "Applied operations and created participations"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or shift not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/coordination/apply [post]
func (s *Server) handleApplyCoordination(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.CoordinationApplyRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	if result := validation.ValidateCoordinationApply(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	ops := make([]db.CoordinationOp, len(req.Ops))
	for i, op := range req.Ops {
		ops[i] = db.CoordinationOp{ShiftID: op.ShiftID, Name: op.Name, Status: op.Status}
	}
	result, err := s.db.ApplyCoordinationOps(plan.ID, ops, req.Lock)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Shift not found", http.StatusNotFound, err))
			return
		}
		s.RespondWithError(w, NewAppError(ErrDatabaseError, "Failed to apply operations", http.StatusInternalServerError, err))
		return
	}
	s.notifyPlanChanged(plan.ID, scopeParticipants)
	if len(req.Lock) > 0 {
		s.notifyPlanChanged(plan.ID, scopeShifts)
	}

	response := model.CoordinationApplyResponse{Applied: result.Applied, Created: make([]model.CoordinationCreated, len(result.Created))}
	for i, c := range result.Created {
		response.Created[i] = model.CoordinationCreated{ShiftID: c.ShiftID, ParticipantID: c.ParticipantID, Name: c.Name}
	}
	s.respondJSON(w, response)
}

// draftError maps database errors of the lab variants to HTTP responses.
func (s *Server) draftError(w http.ResponseWriter, err error, message string) {
	switch {
	case errors.Is(err, db.ErrNotFound):
		s.RespondWithError(w, NewAppError(ErrResourceNotFound, "Variant not found", http.StatusNotFound, err))
	case errors.Is(err, db.ErrLimitReached):
		s.RespondWithError(w, NewAppError(ErrLimitReached, fmt.Sprintf("At most %d variants per plan", db.MaxDraftsPerPlan), http.StatusConflict, err))
	default:
		s.RespondWithError(w, NewAppError(ErrDatabaseError, message, http.StatusInternalServerError, err))
	}
}

// getDraftID parses the variant ID from the URL path.
func (s *Server) getDraftID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		s.RespondWithError(w, NewAppError(ErrInvalidInput, "Invalid variant ID", http.StatusBadRequest, err))
		return 0, false
	}
	return id, true
}

// handleListDrafts lists the saved lab variants of a plan
// @Summary List coordination variants
// @Description Returns the saved variants of the coordination lab without their operations (with the operation count). Variants are private working states of the admins; they are not exported and not announced to other clients. Requires admin authentication.
// @Tags Coordination
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Success 200 {array} model.CoordinationDraftSummary "Variants, most recently changed first"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/coordination/drafts [get]
func (s *Server) handleListDrafts(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	drafts, err := s.db.ListDrafts(plan.ID)
	if err != nil {
		s.draftError(w, err, "Failed to list variants")
		return
	}
	s.respondJSON(w, drafts)
}

// handleCreateDraft saves a new lab variant
// @Summary Create coordination variant
// @Description Saves a variant of the coordination lab (name 1 to 80 characters, up to 2000 operations). At most 50 variants per plan. The operations are validated like those of the apply endpoint but not applied. Requires admin authentication.
// @Tags Coordination
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body model.CoordinationDraftRequest true "Variant"
// @Success 200 {object} model.CoordinationDraft "Created variant"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan not found"
// @Failure 409 {string} string "Limit of 50 variants reached"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/coordination/drafts [post]
func (s *Server) handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	var req model.CoordinationDraftRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	if result := validation.ValidateCoordinationDraft(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	draft, err := s.db.CreateDraft(plan.ID, req.Name, req.Ops)
	if err != nil {
		s.draftError(w, err, "Failed to create variant")
		return
	}
	s.respondJSON(w, draft)
}

// handleGetDraft returns one lab variant
// @Summary Get coordination variant
// @Description Returns a saved variant including its operations. Requires admin authentication.
// @Tags Coordination
// @Produce json
// @Param token path string true "Plan token"
// @Param id path int true "Variant ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 200 {object} model.CoordinationDraft "Variant"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or variant not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/coordination/drafts/{id} [get]
func (s *Server) handleGetDraft(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	id, ok := s.getDraftID(w, r)
	if !ok {
		return
	}
	draft, err := s.db.GetDraft(plan.ID, id)
	if err != nil {
		s.draftError(w, err, "Failed to read variant")
		return
	}
	s.respondJSON(w, draft)
}

// handleUpdateDraft replaces a lab variant
// @Summary Update coordination variant
// @Description Replaces name and operations of a variant. With applied set to true, applied_at is set to the current time (the variant is only marked, its operations are not applied). Concurrent writes: the last write wins. Requires admin authentication.
// @Tags Coordination
// @Accept json
// @Produce json
// @Param token path string true "Plan token"
// @Param id path int true "Variant ID"
// @Param X-Admin-Token header string true "Admin token"
// @Param data body model.CoordinationDraftRequest true "Variant"
// @Success 200 {object} model.CoordinationDraft "Updated variant"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or variant not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/coordination/drafts/{id} [put]
func (s *Server) handleUpdateDraft(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	id, ok := s.getDraftID(w, r)
	if !ok {
		return
	}
	var req model.CoordinationDraftRequest
	if !s.parseJSON(w, r, &req) {
		return
	}
	if result := validation.ValidateCoordinationDraft(&req); result.HasErrors() {
		s.respondValidationError(w, result)
		return
	}
	draft, err := s.db.UpdateDraft(plan.ID, id, req.Name, req.Ops, req.Applied)
	if err != nil {
		s.draftError(w, err, "Failed to update variant")
		return
	}
	s.respondJSON(w, draft)
}

// handleDeleteDraft deletes a lab variant
// @Summary Delete coordination variant
// @Description Deletes a saved variant. Requires admin authentication.
// @Tags Coordination
// @Param token path string true "Plan token"
// @Param id path int true "Variant ID"
// @Param X-Admin-Token header string true "Admin token"
// @Success 204 "Deleted"
// @Failure 400 {string} string "Invalid input"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Plan or variant not found"
// @Failure 500 {string} string "Internal server error"
// @Security AdminToken
// @Router /plans/{token}/coordination/drafts/{id} [delete]
func (s *Server) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	plan, ok := s.getPlanByToken(w, r)
	if !ok {
		return
	}
	id, ok := s.getDraftID(w, r)
	if !ok {
		return
	}
	if err := s.db.DeleteDraft(plan.ID, id); err != nil {
		s.draftError(w, err, "Failed to delete variant")
		return
	}
	s.respondNoContent(w)
}
