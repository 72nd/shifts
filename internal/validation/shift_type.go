package validation

import (
	"github.com/72nd/shifts/internal/db"
	"github.com/72nd/shifts/internal/model"
)

// ValidateNewShiftType validates a NewShiftType struct for creation.
// It sanitizes text fields in place and returns validation errors.
// isMultilingual determines whether German fields are required.
func ValidateNewShiftType(st *db.NewShiftType, isMultilingual bool) *Result {
	r := NewResult()

	// English title is always required
	st.TitleEN = ValidateRequiredMaxLength(st.TitleEN, "title_en", "English title", MaxTitleLength, r)

	// German title is required only for multilingual plans
	if isMultilingual {
		st.TitleDE = ValidateRequiredMaxLength(st.TitleDE, "title_de", "German title", MaxTitleLength, r)
	} else {
		st.TitleDE = ValidateOptional(st.TitleDE)
	}

	// Descriptions are optional
	st.DescriptionEN = ValidateOptional(st.DescriptionEN)
	st.DescriptionDE = ValidateOptional(st.DescriptionDE)

	// Locations are optional
	st.LocationEN = ValidateOptional(st.LocationEN)
	st.LocationDE = ValidateOptional(st.LocationDE)

	return r
}

// ValidateShiftTypeUpdate validates a shift type update request.
// isMultilingual determines whether German fields are required.
func ValidateShiftTypeUpdate(st *model.ShiftTypeUpdateRequest, isMultilingual bool) *Result {
	r := NewResult()

	// English title is always required
	st.TitleEN = ValidateRequiredMaxLength(st.TitleEN, "title_en", "English title", MaxTitleLength, r)

	// German title is required only for multilingual plans
	if isMultilingual {
		st.TitleDE = ValidateRequiredMaxLength(st.TitleDE, "title_de", "German title", MaxTitleLength, r)
	} else {
		st.TitleDE = ValidateOptional(st.TitleDE)
	}

	// Descriptions are optional
	st.DescriptionEN = ValidateOptional(st.DescriptionEN)
	st.DescriptionDE = ValidateOptional(st.DescriptionDE)

	// Locations are optional
	st.LocationEN = ValidateOptional(st.LocationEN)
	st.LocationDE = ValidateOptional(st.LocationDE)

	return r
}
