package model

// =============================================================================
// Shift Request DTOs
// =============================================================================

// ShiftCreateRequest represents a single shift to create
// @Description Request body for creating a single shift
type ShiftCreateRequest struct {
	// ID of the shift type this shift belongs to
	TypeID int64 `json:"type_id" example:"1" validate:"required,min=1"`
	// Start time of the shift as local time of the plan without offset (YYYY-MM-DDTHH:MM[:SS])
	StartTime DateTime `json:"start_time" swaggertype:"string" example:"2024-12-20T09:00:00" validate:"required"`
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
	// Start time of the shift as local time of the plan without offset (YYYY-MM-DDTHH:MM[:SS])
	StartTime DateTime `json:"start_time" swaggertype:"string" example:"2024-12-20T09:00:00" validate:"required"`
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
// @Description Single shift update item within a batch update request. Only include fields you want to update (partial updates supported).
type BatchShiftUpdateItem struct {
	// ID of the shift to update
	ID int64 `json:"id" example:"42" validate:"required,min=1"`
	// ID of the shift type this shift belongs to (optional, only set if you want to change it)
	TypeID *int64 `json:"type_id,omitempty" example:"1"`
	// Start time of the shift as local time of the plan without offset (optional, only set if you want to change it)
	StartTime *DateTime `json:"start_time,omitempty" swaggertype:"string" example:"2024-12-20T09:00:00"`
	// Duration of the shift in minutes (optional, only set if you want to change it)
	Duration *int `json:"duration,omitempty" example:"120"`
	// Number of required participants, -1 for unlimited (optional, only set if you want to change it)
	Required *int `json:"required,omitempty" example:"3"`
	// Optional comment in German (supports Markdown). Use null to clear existing comment.
	CommentDE *string `json:"comment_de,omitempty" example:"Bitte pünktlich erscheinen"`
	// Optional comment in English (supports Markdown). Use null to clear existing comment.
	CommentEN *string `json:"comment_en,omitempty" example:"Please arrive on time"`
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
// Shift Response DTOs
// =============================================================================

// ShiftResponse represents a shift in API responses
// @Description Shift data returned in API responses
type ShiftResponse struct {
	// Unique identifier of the shift
	ID int64 `json:"id" example:"42"`
	// ID of the shift type this shift belongs to
	TypeID int64 `json:"type_id" example:"1"`
	// Start time of the shift as local time of the plan (no offset)
	StartTime DateTime `json:"start_time" swaggertype:"string" example:"2024-12-20T09:00:00"`
	// Duration of the shift in minutes
	Duration int `json:"duration" example:"120"`
	// Number of required participants (-1 for unlimited)
	Required int `json:"required" example:"3"`
	// Comment in German (raw Markdown)
	CommentDE string `json:"comment_de" example:"Bitte pünktlich erscheinen"`
	// Comment in English (raw Markdown)
	CommentEN string `json:"comment_en" example:"Please arrive on time"`
	// Rendered HTML of German comment (only present if comment_de is set)
	CommentDEHTML *string `json:"comment_de_html,omitempty"`
	// Rendered HTML of English comment (only present if comment_en is set)
	CommentENHTML *string `json:"comment_en_html,omitempty"`
	// Whether the shift is locked for participant changes
	IsLocked bool `json:"is_locked" example:"false"`
	// List of participants (only present when requested)
	Participants []ParticipantResponse `json:"participants,omitempty"`
} // @name ShiftResponse

// =============================================================================
// Batch Response DTOs
// =============================================================================

// BatchCreateResponse holds the results of a batch shift create operation
// @Description Response body for batch shift creation
type BatchCreateResponse struct {
	// Array of successfully created shifts
	Created []ShiftResponse `json:"created"`
	// Array of items that failed to create (only present if there were failures)
	Failed []BatchItemError `json:"failed,omitempty"`
} // @name BatchCreateResponse

// BatchUpdateResponse holds the results of a batch shift update operation
// @Description Response body for batch shift update
type BatchUpdateResponse struct {
	// Array of IDs of successfully updated shifts
	Updated []int64 `json:"updated"`
	// Array of items that failed to update (only present if there were failures)
	Failed []BatchItemError `json:"failed,omitempty"`
} // @name BatchUpdateResponse

// BatchDeleteResponse holds the results of a batch shift delete operation
// @Description Response body for batch shift deletion
type BatchDeleteResponse struct {
	// Array of IDs of successfully deleted shifts
	Deleted []int64 `json:"deleted"`
	// Array of items that failed to delete (only present if there were failures)
	Failed []BatchItemError `json:"failed,omitempty"`
} // @name BatchDeleteResponse
