package web

import (
	"errors"
	"net/http"
	"strconv"

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
