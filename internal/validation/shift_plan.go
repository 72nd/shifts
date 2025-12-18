package validation

import (
	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// ValidateCreateShiftPlanRequest validates a CreateShiftPlanRequest for creation.
// It sanitizes text fields in place and returns validation errors.
func ValidateCreateShiftPlanRequest(p *model.CreateShiftPlanRequest) *Result {
	r := NewResult()

	// English title is always required
	p.TitleEN = ValidateRequiredMaxLength(p.TitleEN, "title_en", "English title", MaxTitleLength, r)

	// German title is required only for multilingual plans
	if p.IsMultilingual && p.TitleDE != nil {
		*p.TitleDE = ValidateRequiredMaxLength(*p.TitleDE, "title_de", "German title", MaxTitleLength, r)
	} else if p.IsMultilingual && p.TitleDE == nil {
		r.AddRequired("title_de", "German title")
	}

	// Descriptions are optional, just sanitize them
	p.DescriptionEN = ValidateOptional(p.DescriptionEN)
	if p.DescriptionDE != nil && *p.DescriptionDE != "" {
		*p.DescriptionDE = ValidateOptional(*p.DescriptionDE)
	}

	// Dates validation
	if p.StartDate.IsZero() {
		r.AddRequired("start_date", "Start date")
	}
	if p.EndDate.IsZero() {
		r.AddRequired("end_date", "End date")
	}
	if !p.StartDate.IsZero() && !p.EndDate.IsZero() && p.EndDate.Before(p.StartDate) {
		r.AddInvalid("end_date", "End date must be after start date")
	}

	return r
}

// ValidateNewShiftPlan validates a NewShiftPlan struct for creation (legacy).
// Deprecated: Use ValidateCreateShiftPlanRequest instead.
func ValidateNewShiftPlan(p *db.NewShiftPlan) *Result {
	r := NewResult()

	// English title is always required
	p.TitleEN = ValidateRequiredMaxLength(p.TitleEN, "title_en", "English title", MaxTitleLength, r)

	// German title is required only for multilingual plans
	if p.IsMultilingual && p.TitleDE != nil {
		*p.TitleDE = ValidateRequiredMaxLength(*p.TitleDE, "title_de", "German title", MaxTitleLength, r)
	} else if p.IsMultilingual && p.TitleDE == nil {
		r.AddRequired("title_de", "German title")
	}

	// Descriptions are optional, just sanitize them
	p.DescriptionEN = ValidateOptional(p.DescriptionEN)
	if p.DescriptionDE != nil && *p.DescriptionDE != "" {
		*p.DescriptionDE = ValidateOptional(*p.DescriptionDE)
	}

	// Dates validation
	if p.StartDate.IsZero() {
		r.AddRequired("start_date", "Start date")
	}
	if p.EndDate.IsZero() {
		r.AddRequired("end_date", "End date")
	}
	if !p.StartDate.IsZero() && !p.EndDate.IsZero() && p.EndDate.Before(p.StartDate) {
		r.AddInvalid("end_date", "End date must be after start date")
	}

	return r
}

// ValidateShiftPlanUpdate validates a shift plan update request.
func ValidateShiftPlanUpdate(p *model.ShiftPlanUpdateRequest) *Result {
	r := NewResult()

	// English title is always required
	p.TitleEN = ValidateRequiredMaxLength(p.TitleEN, "title_en", "English title", MaxTitleLength, r)

	// German title is required only for multilingual plans
	if p.IsMultilingual && p.TitleDE != nil {
		*p.TitleDE = ValidateRequiredMaxLength(*p.TitleDE, "title_de", "German title", MaxTitleLength, r)
	} else if p.IsMultilingual && p.TitleDE == nil {
		r.AddRequired("title_de", "German title")
	}

	// Descriptions are optional, just sanitize them
	p.DescriptionEN = ValidateOptional(p.DescriptionEN)
	if p.DescriptionDE != nil && *p.DescriptionDE != "" {
		*p.DescriptionDE = ValidateOptional(*p.DescriptionDE)
	}

	return r
}
