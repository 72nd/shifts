package model

import "time"

// =============================================================================
// Shift Plan Request DTOs
// =============================================================================

// CreateShiftPlanRequest represents the data needed to create a new shift plan
// @Description Request body for creating a new shift plan
type CreateShiftPlanRequest struct {
	// Title in German (required if multilingual)
	TitleDE *string `json:"title_de" example:"Sommerfest 2024"`
	// Title in English
	TitleEN string `json:"title_en" example:"Summer Festival 2024" validate:"required,max=150"`
	// Description in German (supports Markdown)
	DescriptionDE *string `json:"description_de" example:"Beschreibung des Events"`
	// Description in English (supports Markdown)
	DescriptionEN string `json:"description_en" example:"Event description"`
	// Start date of the plan
	StartDate time.Time `json:"start_date" example:"2024-12-20T00:00:00Z" validate:"required"`
	// End date of the plan
	EndDate time.Time `json:"end_date" example:"2024-12-22T00:00:00Z" validate:"required"`
	// Whether the plan supports multiple languages
	IsMultilingual bool `json:"is_multilingual" example:"true"`
	// Time when a new day starts (HH:MM format, e.g., "06:00" for 6 AM)
	DayBoundaryTime string `json:"day_boundary_time" example:"00:00"`
	// Whether participants can see each other's names
	CollaborativeMode bool `json:"collaborative_mode" example:"false"`
} // @name CreateShiftPlanRequest

// ShiftPlanUpdateRequest represents the fields that can be updated on a shift plan
// @Description Request body for updating a shift plan
type ShiftPlanUpdateRequest struct {
	// Title in German (required if multilingual)
	TitleDE *string `json:"title_de" example:"Sommerfest 2024"`
	// Title in English
	TitleEN string `json:"title_en" example:"Summer Festival 2024" validate:"required,max=150"`
	// Description in German (supports Markdown)
	DescriptionDE *string `json:"description_de" example:"Beschreibung des Events"`
	// Description in English (supports Markdown)
	DescriptionEN string `json:"description_en" example:"Event description"`
	// Start date of the plan
	StartDate time.Time `json:"start_date" example:"2024-12-20T00:00:00Z" validate:"required"`
	// End date of the plan
	EndDate time.Time `json:"end_date" example:"2024-12-22T00:00:00Z" validate:"required"`
	// Whether the plan supports multiple languages
	IsMultilingual bool `json:"is_multilingual" example:"true"`
	// Time when a new day starts (HH:MM format, e.g., "06:00" for 6 AM)
	DayBoundaryTime string `json:"day_boundary_time" example:"00:00"`
	// Whether participants can see each other's names
	CollaborativeMode bool `json:"collaborative_mode" example:"false"`
} // @name ShiftPlanUpdateRequest

// ShiftTypeOrderRequest represents a request to update shift type order
// @Description Request body for updating the display order of shift types
type ShiftTypeOrderRequest struct {
	// Array of type IDs in the desired display order
	TypeIDs []int64 `json:"type_ids" validate:"required"`
} // @name ShiftTypeOrderRequest

// =============================================================================
// Shift Plan Response DTOs
// =============================================================================

// ShiftPlanResponse represents a shift plan in API responses
// @Description Shift plan data returned in API responses
type ShiftPlanResponse struct {
	// Unique identifier of the shift plan
	ID int64 `json:"id" example:"42"`
	// Public token for accessing the plan
	Token string `json:"token" example:"abc123def456"`
	// Start date of the plan
	StartDate time.Time `json:"start_date" example:"2024-12-20T00:00:00Z"`
	// End date of the plan
	EndDate time.Time `json:"end_date" example:"2024-12-22T00:00:00Z"`
	// Timestamp when the plan was created
	CreatedAt time.Time `json:"created_at" example:"2024-12-01T10:00:00Z"`
	// Whether the plan supports multiple languages
	IsMultilingual bool `json:"is_multilingual" example:"true"`
	// Title in German
	TitleDE *string `json:"title_de" example:"Sommerfest 2024"`
	// Title in English
	TitleEN string `json:"title_en" example:"Summer Festival 2024"`
	// Description in German (raw Markdown)
	DescriptionDE *string `json:"description_de" example:"Beschreibung des Events"`
	// Description in English (raw Markdown)
	DescriptionEN string `json:"description_en" example:"Event description"`
	// Rendered HTML of German description (only present if description_de is set)
	DescriptionDEHTML *string `json:"description_de_html,omitempty"`
	// Rendered HTML of English description (only present if description_en is set)
	DescriptionENHTML *string `json:"description_en_html,omitempty"`
	// Time when a new day starts (HH:MM format)
	DayBoundaryTime string `json:"day_boundary_time" example:"00:00"`
	// Whether participants can see each other's names
	CollaborativeMode bool `json:"collaborative_mode" example:"false"`
} // @name ShiftPlanResponse

// CreateShiftPlanResponse represents the response after creating a shift plan
// @Description Response body for successful shift plan creation
type CreateShiftPlanResponse struct {
	// Public token for accessing the plan
	Token string `json:"token" example:"abc123def456"`
	// URL path to access the plan
	URL string `json:"url" example:"/plans/abc123def456"`
	// Admin password for managing the plan (only returned on creation)
	Password string `json:"password" example:"secretpassword123"`
} // @name CreateShiftPlanResponse
