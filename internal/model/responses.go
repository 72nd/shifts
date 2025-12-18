package model

import "time"

// =============================================================================
// Shift Response DTO
// =============================================================================

// ShiftResponse represents a shift in API responses
// @Description Shift data returned in API responses
type ShiftResponse struct {
	// Unique identifier of the shift
	ID int64 `json:"id" example:"42"`
	// ID of the shift type this shift belongs to
	TypeID int64 `json:"type_id" example:"1"`
	// Start time of the shift
	StartTime time.Time `json:"start_time" example:"2024-12-20T09:00:00Z"`
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
// Participant DTOs
// =============================================================================

// ParticipantResponse represents a participant in API responses
// @Description Participant data returned in API responses
type ParticipantResponse struct {
	// Unique identifier of the participant
	ID int64 `json:"id" example:"123"`
	// Name of the participant
	Name string `json:"name" example:"John Doe"`
} // @name ParticipantResponse

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

// =============================================================================
// Admin DTOs
// =============================================================================

// VerifyAdminResponse represents a response to admin verification
// @Description Response body for successful admin verification
type VerifyAdminResponse struct {
	// Token to use for admin operations
	Token string `json:"token" example:"secretpassword123"`
} // @name VerifyAdminResponse
