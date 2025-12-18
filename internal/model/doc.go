// Package model defines the data transfer objects (models) for the Shifts API.
// This package serves as the single source of truth for all API contracts,
// including request and response types, batch operation results, and validation constants.
//
// All HTTP handlers, validators, and other layers should reference types from this package
// to ensure consistency across the application.
//
// Package structure:
//   - batch.go: Generic batch operation types
//   - constants.go: Validation and size limit constants
//   - requests.go: All API request types
//   - responses.go: All API response types
package model
