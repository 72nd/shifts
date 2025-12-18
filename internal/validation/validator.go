// Package validation provides centralized input validation for the shifts application.
// It supports collecting multiple validation errors for batch operations.
package validation

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// Common validation limits
const (
	MaxTitleLength       = 150
	MaxNameLength        = 20
	MaxDescriptionLength = 5000
	MaxLocationLength    = 200
)

// Common validation error codes
const (
	CodeRequired     = "required"
	CodeTooLong      = "too_long"
	CodeInvalidValue = "invalid_value"
)

// ValidationError represents a single validation error for a specific field.
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Result collects validation errors and provides utility methods.
type Result struct {
	Errors []ValidationError `json:"errors,omitempty"`
}

// NewResult creates a new empty validation result.
func NewResult() *Result {
	return &Result{
		Errors: []ValidationError{},
	}
}

// HasErrors returns true if there are any validation errors.
func (r *Result) HasErrors() bool {
	return len(r.Errors) > 0
}

// Add appends a new validation error to the result.
func (r *Result) Add(field, code, message string) {
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Code:    code,
		Message: message,
	})
}

// AddRequired adds a "field is required" error.
func (r *Result) AddRequired(field, displayName string) {
	r.Add(field, CodeRequired, displayName+" is required")
}

// AddTooLong adds a "field exceeds max length" error.
func (r *Result) AddTooLong(field, displayName string, maxLen int) {
	r.Add(field, CodeTooLong, displayName+" cannot exceed "+itoa(maxLen)+" characters")
}

// AddInvalid adds an "invalid value" error with a custom message.
func (r *Result) AddInvalid(field, message string) {
	r.Add(field, CodeInvalidValue, message)
}

// Merge combines another validation result into this one.
// Useful for combining results from nested validations.
func (r *Result) Merge(other *Result) {
	if other == nil {
		return
	}
	r.Errors = append(r.Errors, other.Errors...)
}

// MergeWithPrefix combines another validation result, prefixing all field names.
// Useful for batch operations: MergeWithPrefix(itemResult, "items[0].")
func (r *Result) MergeWithPrefix(other *Result, prefix string) {
	if other == nil {
		return
	}
	for _, err := range other.Errors {
		r.Errors = append(r.Errors, ValidationError{
			Field:   prefix + err.Field,
			Code:    err.Code,
			Message: err.Message,
		})
	}
}

// FirstError returns the first error message, or empty string if no errors.
func (r *Result) FirstError() string {
	if len(r.Errors) == 0 {
		return ""
	}
	return r.Errors[0].Message
}

// sanitizer is the shared bluemonday policy for text sanitization.
var sanitizer *bluemonday.Policy

func init() {
	sanitizer = bluemonday.StrictPolicy()
}

// SanitizeText removes potentially dangerous HTML tags and trims whitespace.
func SanitizeText(text string) string {
	if text == "" {
		return ""
	}
	text = strings.TrimSpace(text)
	return sanitizer.Sanitize(text)
}

// ValidateRequired checks if a string is non-empty after trimming.
func ValidateRequired(value, field, displayName string, r *Result) string {
	value = SanitizeText(value)
	if value == "" {
		r.AddRequired(field, displayName)
	}
	return value
}

// ValidateOptional sanitizes an optional string field.
func ValidateOptional(value string) string {
	return SanitizeText(value)
}

// ValidateMaxLength checks if a string exceeds the maximum length.
// Returns the sanitized value.
func ValidateMaxLength(value, field, displayName string, maxLen int, r *Result) string {
	value = SanitizeText(value)
	if len(value) > maxLen {
		r.AddTooLong(field, displayName, maxLen)
	}
	return value
}

// ValidateRequiredMaxLength combines required and max length validation.
// Returns the sanitized value.
func ValidateRequiredMaxLength(value, field, displayName string, maxLen int, r *Result) string {
	value = SanitizeText(value)
	if value == "" {
		r.AddRequired(field, displayName)
		return value
	}
	if len(value) > maxLen {
		r.AddTooLong(field, displayName, maxLen)
	}
	return value
}

// itoa is a simple int to string converter to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
