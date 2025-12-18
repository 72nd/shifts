package model

import "time"

// =============================================================================
// Shift Request DTOs
// =============================================================================

// ShiftCreateRequest represents a single shift to create
// @Description Request body for creating a single shift
type ShiftCreateRequest struct {
	// ID of the shift type this shift belongs to
	TypeID int64 `json:"type_id" example:"1" validate:"required,min=1"`
	// Start time of the shift in RFC3339 format
	StartTime time.Time `json:"start_time" example:"2024-12-20T09:00:00Z" validate:"required"`
	// Duration of the shift in minutes
	Duration int `json:"duration" example:"120" validate:"required,min=1"`
	// Number of required participants (-1 for unlimited)
	Required int `json:"required" example:"3" validate:"required,min=-1"`
	// Optional comment in German (supports Markdown)
	CommentDE string `json:"comment_de" example:"Bitte pünktlich erscheinen"`
	// Optional comment in English (supports Markdown)
	CommentEN string `json:"comment_en" example:"Please arrive on time"`
} // @name ShiftCreateRequest

// ShiftUpdateRequest represents the fields that can be updated on a shift
// @Description Request body for updating a single shift
type ShiftUpdateRequest struct {
	// ID of the shift type this shift belongs to
	TypeID int64 `json:"type_id" example:"1" validate:"required,min=1"`
	// Start time of the shift in RFC3339 format
	StartTime time.Time `json:"start_time" example:"2024-12-20T09:00:00Z" validate:"required"`
	// Duration of the shift in minutes
	Duration int `json:"duration" example:"120" validate:"required,min=1"`
	// Number of required participants (-1 for unlimited)
	Required int `json:"required" example:"3" validate:"required,min=-1"`
	// Optional comment in German (supports Markdown)
	CommentDE string `json:"comment_de" example:"Bitte pünktlich erscheinen"`
	// Optional comment in English (supports Markdown)
	CommentEN string `json:"comment_en" example:"Please arrive on time"`
} // @name ShiftUpdateRequest

// =============================================================================
// Batch Shift Request DTOs
// =============================================================================

// BatchShiftCreateRequest represents a batch create request for shifts
// @Description Request body for creating multiple shifts in a single operation
type BatchShiftCreateRequest struct {
	// Array of shifts to create (max 100 items)
	Shifts []ShiftCreateRequest `json:"shifts" validate:"required,min=1,max=100,dive"`
} // @name BatchShiftCreateRequest

// BatchShiftUpdateItem represents a single shift update in a batch operation
// @Description Single shift update item within a batch update request
type BatchShiftUpdateItem struct {
	// ID of the shift to update
	ID int64 `json:"id" example:"42" validate:"required,min=1"`
	// ID of the shift type this shift belongs to
	TypeID int64 `json:"type_id" example:"1" validate:"required,min=1"`
	// Start time of the shift in RFC3339 format
	StartTime string `json:"start_time" example:"2024-12-20T09:00:00Z" validate:"required"`
	// Duration of the shift in minutes
	Duration int `json:"duration" example:"120" validate:"required,min=1"`
	// Number of required participants (-1 for unlimited)
	Required int `json:"required" example:"3" validate:"required,min=-1"`
	// Optional comment in German (supports Markdown)
	CommentDE string `json:"comment_de" example:"Bitte pünktlich erscheinen"`
	// Optional comment in English (supports Markdown)
	CommentEN string `json:"comment_en" example:"Please arrive on time"`
} // @name BatchShiftUpdateItem

// BatchShiftUpdateRequest represents a batch update request for shifts
// @Description Request body for updating multiple shifts in a single operation
type BatchShiftUpdateRequest struct {
	// Array of shift updates (max 100 items)
	Shifts []BatchShiftUpdateItem `json:"shifts" validate:"required,min=1,max=100,dive"`
} // @name BatchShiftUpdateRequest

// BatchShiftDeleteRequest represents a batch delete request for shifts
// @Description Request body for deleting multiple shifts in a single operation
type BatchShiftDeleteRequest struct {
	// Array of shift IDs to delete (max 100 items)
	ShiftIDs []int64 `json:"shift_ids" validate:"required,min=1,max=100"`
} // @name BatchShiftDeleteRequest

// =============================================================================
// Participant DTOs
// =============================================================================

// AddParticipantRequest represents a request to add a participant to a shift
// @Description Request body for adding a participant to a shift
type AddParticipantRequest struct {
	// Name of the participant to add
	Name string `json:"name" example:"John Doe" validate:"required,min=1,max=100"`
} // @name AddParticipantRequest

// =============================================================================
// Shift Type DTOs
// =============================================================================

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
} // @name ShiftTypeUpdateRequest

// =============================================================================
// Shift Plan DTOs
// =============================================================================

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
// Admin DTOs
// =============================================================================

// VerifyAdminRequest represents a request to verify admin password
// @Description Request body for verifying admin password
type VerifyAdminRequest struct {
	// Admin password to verify
	Password string `json:"password" example:"secretpassword123" validate:"required"`
} // @name VerifyAdminRequest
