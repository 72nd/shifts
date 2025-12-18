package model

// =============================================================================
// Participant Request DTOs
// =============================================================================

// AddParticipantRequest represents a request to add a participant to a shift
// @Description Request body for adding a participant to a shift
type AddParticipantRequest struct {
	// Name of the participant to add
	Name string `json:"name" example:"John Doe" validate:"required,min=1,max=100"`
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
} // @name ParticipantResponse
