// Package model defines the data transfer objects (DTOs) for the Shifts API.
// This package serves as the single source of truth for all API contracts,
// including request and response types, batch operation results, and validation constants.
//
// All HTTP handlers, validators, and other layers should reference types from this package
// to ensure consistency across the application.
//
// Package structure organized by entity:
//   - admin.go: Admin authentication DTOs (requests and responses)
//   - batch.go: Generic batch operation types
//   - constants.go: Validation and size limit constants
//   - participant.go: Participant DTOs (requests and responses)
//   - shift.go: Shift DTOs (requests and responses)
//   - shift_plan.go: Shift plan DTOs (requests and responses)
//   - shift_type.go: Shift type DTOs (requests and responses)
package model
