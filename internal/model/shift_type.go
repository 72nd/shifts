package model

// =============================================================================
// Shift Type Request DTOs
// =============================================================================

// CreateShiftTypeRequest represents the data needed to create a new shift type
// @Description Request body for creating a new shift type
type CreateShiftTypeRequest struct {
	// Title in German
	TitleDE string `json:"title_de" example:"Küchendienst" validate:"required,max=150"`
	// Title in English
	TitleEN string `json:"title_en" example:"Kitchen Duty" validate:"required,max=150"`
	// Description in German (supports Markdown)
	DescriptionDE string `json:"description_de" example:"Hilfe in der Küche"`
	// Description in English (supports Markdown)
	DescriptionEN string `json:"description_en" example:"Help in the kitchen"`
	// Location in German
	LocationDE string `json:"location_de" example:"Hauptküche"`
	// Location in English
	LocationEN string `json:"location_en" example:"Main Kitchen"`
	// Optional password to restrict participation (if set, participants must provide this password)
	Password *string `json:"password,omitempty" example:"secret123"`
	// Load factor of the shifts (0.1 to 5.0, default 1.0)
	LoadFactor *float64 `json:"load_factor,omitempty" example:"1.0"`
	// Popularity: auto, popular or normal
	Popularity *string `json:"popularity,omitempty" example:"auto" enums:"auto,popular,normal"`
	// Number of people with at least exp_min_level needed per shift
	ExpMinCount *int `json:"exp_min_count,omitempty" example:"0"`
	// Experience level (option ID of the experience field) required for exp_min_count people. Empty string clears.
	ExpMinLevel *string `json:"exp_min_level,omitempty" example:"a1b2c3d4"`
} // @name CreateShiftTypeRequest

// ShiftTypeUpdateRequest represents the fields that can be updated on a shift type
// @Description Request body for updating a shift type
type ShiftTypeUpdateRequest struct {
	// Title in German
	TitleDE string `json:"title_de" example:"Küchendienst" validate:"max=150"`
	// Title in English
	TitleEN string `json:"title_en" example:"Kitchen Duty" validate:"required,max=150"`
	// Description in German (supports Markdown)
	DescriptionDE string `json:"description_de" example:"Hilfe in der Küche"`
	// Description in English (supports Markdown)
	DescriptionEN string `json:"description_en" example:"Help in the kitchen"`
	// Location in German
	LocationDE string `json:"location_de" example:"Hauptküche"`
	// Location in English
	LocationEN string `json:"location_en" example:"Main Kitchen"`
	// Load factor of the shifts (0.1 to 5.0, default 1.0)
	LoadFactor *float64 `json:"load_factor,omitempty" example:"1.0"`
	// Popularity: auto, popular or normal
	Popularity *string `json:"popularity,omitempty" example:"auto" enums:"auto,popular,normal"`
	// Number of people with at least exp_min_level needed per shift
	ExpMinCount *int `json:"exp_min_count,omitempty" example:"0"`
	// Experience level (option ID of the experience field) required for exp_min_count people. Empty string clears.
	ExpMinLevel *string `json:"exp_min_level,omitempty" example:"a1b2c3d4"`
	// Omitted coordination fields are left unchanged.
} // @name ShiftTypeUpdateRequest

// =============================================================================
// Shift Type Response DTOs
// =============================================================================

// ShiftTypeResponse represents a shift type in API responses
// @Description Shift type data returned in API responses
type ShiftTypeResponse struct {
	// Unique identifier of the shift type
	ID int64 `json:"id" example:"1"`
	// ID of the shift plan this type belongs to
	PlanID int64 `json:"plan_id" example:"42"`
	// Display order of this type
	SortOrder int `json:"sort_order" example:"1"`
	// Title in German
	TitleDE string `json:"title_de" example:"Küchendienst"`
	// Title in English
	TitleEN string `json:"title_en" example:"Kitchen Duty"`
	// Description in German (raw Markdown)
	DescriptionDE string `json:"description_de" example:"Hilfe in der Küche"`
	// Description in English (raw Markdown)
	DescriptionEN string `json:"description_en" example:"Help in the kitchen"`
	// Rendered HTML of German description (only present if description_de is set)
	DescriptionDEHTML *string `json:"description_de_html,omitempty"`
	// Rendered HTML of English description (only present if description_en is set)
	DescriptionENHTML *string `json:"description_en_html,omitempty"`
	// Location in German
	LocationDE string `json:"location_de" example:"Hauptküche"`
	// Location in English
	LocationEN string `json:"location_en" example:"Main Kitchen"`
	// Load factor of the shifts (0.1 to 5.0, default 1.0)
	LoadFactor float64 `json:"load_factor" example:"1.0"`
	// Popularity: auto, popular or normal
	Popularity string `json:"popularity" example:"auto" enums:"auto,popular,normal"`
	// Number of people with at least exp_min_level needed per shift
	ExpMinCount int `json:"exp_min_count" example:"0"`
	// Experience level (option ID of the experience field)
	ExpMinLevel *string `json:"exp_min_level" example:"a1b2c3d4"`
	// Array of shifts belonging to this type (only present when requested)
	Shifts []ShiftResponse `json:"shifts,omitempty"`
} // @name ShiftTypeResponse
