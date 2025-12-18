package model

// =============================================================================
// Admin Request DTOs
// =============================================================================

// VerifyAdminRequest represents a request to verify admin password
// @Description Request body for verifying admin password
type VerifyAdminRequest struct {
	// Admin password to verify
	Password string `json:"password" example:"secretpassword123" validate:"required"`
} // @name VerifyAdminRequest

// =============================================================================
// Admin Response DTOs
// =============================================================================

// VerifyAdminResponse represents a response to admin verification
// @Description Response body for successful admin verification
type VerifyAdminResponse struct {
	// Token to use for admin operations
	Token string `json:"token" example:"secretpassword123"`
} // @name VerifyAdminResponse
