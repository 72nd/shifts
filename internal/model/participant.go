package model

import "time"

// =============================================================================
// Participant Request DTOs
// =============================================================================

// AddParticipantRequest represents a request to add a participant to a shift
// @Description Request body for adding a participant to a shift
type AddParticipantRequest struct {
	// Name of the participant to add
	Name string `json:"name" example:"John Doe" validate:"required,min=1,max=100"`
	// Optional details about the person (attendance, person fields)
	Person *PersonInput `json:"person,omitempty"`
} // @name AddParticipantRequest

// =============================================================================
// Participant Response DTOs
// =============================================================================

// ParticipantResponse represents a participant in API responses
// @Description Participant data returned in API responses
type ParticipantResponse struct {
	// Unique identifier of the participant
	ID int64 `json:"id" example:"123"`
	// Name of the participant
	Name string `json:"name" example:"John Doe"`
	// Time of sign-up (RFC3339, UTC)
	CreatedAt time.Time `json:"created_at" example:"2024-12-01T10:00:00Z"`
	// signed_up, assigned or declined
	Status string `json:"status" example:"signed_up" enums:"signed_up,assigned,declined"`
	// Time of the last status change by the coordination (RFC3339, UTC)
	StatusChangedAt *time.Time `json:"status_changed_at,omitempty" example:"2024-12-01T10:00:00Z"`
	// Whether the coordination added this person without a request
	AddedByAdmin bool `json:"added_by_admin"`
} // @name ParticipantResponse
