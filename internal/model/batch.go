package model

// BatchItemError represents an error for a specific item in a batch operation
type BatchItemError struct {
	// Index of the failed item in the original request array
	Index int `json:"index"`
	// Human-readable error message
	Error string `json:"error"`
	// Machine-readable error code
	Code string `json:"code"`
}

// BatchResult represents the result of a batch operation using generics.
// T can be any type depending on the operation:
// - Shift for create operations
// - int64 for update/delete operations (IDs)
type BatchResult[T any] struct {
	// Success contains the successfully processed items
	Success []T `json:"success"`
	// Failed contains errors for items that failed processing
	Failed []BatchItemError `json:"failed,omitempty"`
}
