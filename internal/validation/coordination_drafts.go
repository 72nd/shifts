package validation

import (
	"fmt"

	"github.com/72nd/shifts/internal/model"
)

// Limits of the coordination lab variants.
const (
	MaxDraftNameLength = 80
	MaxDraftsPerPlan   = 50
)

// ValidateCoordinationDraft validates and sanitizes a lab variant request in place.
// An empty operation list is allowed; a nil list becomes empty.
func ValidateCoordinationDraft(req *model.CoordinationDraftRequest) *Result {
	r := NewResult()
	req.Name = ValidateRequiredMaxLength(req.Name, "name", "Name", MaxDraftNameLength, r)
	if req.Ops == nil {
		req.Ops = []model.CoordinationOp{}
	}
	if len(req.Ops) > MaxApplyOps {
		r.AddInvalid("ops", fmt.Sprintf("At most %d operations can be saved", MaxApplyOps))
		return r
	}
	validateCoordinationOps(req.Ops, r)
	return r
}
