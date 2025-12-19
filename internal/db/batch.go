package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/72nd/shifts/internal/model"
)

// CreateShiftsBatch creates multiple shifts in a single transaction.
// It validates all shifts first, then creates them. If any validation fails,
// the entire operation is aborted. If a DB error occurs during creation,
// the transaction is rolled back.
func (db *DB) CreateShiftsBatch(shifts []NewShift, planID int64) (*model.BatchResult[Shift], error) {
	result := &model.BatchResult[Shift]{
		Success: []Shift{},
		Failed:  []model.BatchItemError{},
	}

	if len(shifts) == 0 {
		return result, nil
	}

	// Get valid type IDs for this plan
	types, err := db.GetShiftTypesByPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shift types: %w", err)
	}
	validTypeIDs := make(map[int64]bool)
	for _, t := range types {
		validTypeIDs[t.ID] = true
	}

	// Pre-validate all shifts
	for i, shift := range shifts {
		// Validate required field
		if shift.Required != -1 && shift.Required < 1 {
			result.Failed = append(result.Failed, model.BatchItemError{
				Index: i,
				Error: ErrInvalidRequired.Error(),
				Code:  "invalid_required",
			})
			continue
		}

		// Validate type belongs to plan
		if !validTypeIDs[shift.TypeID] {
			result.Failed = append(result.Failed, model.BatchItemError{
				Index: i,
				Error: "shift type not found in this plan",
				Code:  "invalid_type_id",
			})
		}
	}

	// If any validation failed, return early without creating anything
	if len(result.Failed) > 0 {
		return result, nil
	}

	// All validations passed, create shifts in a transaction
	err = db.transaction(func(tx *sql.Tx) error {
		for i, shift := range shifts {
			query := `
				INSERT INTO shifts (
					type_id, start_time, duration, required,
					comment_de, comment_en
				) VALUES (?, ?, ?, ?, ?, ?)
			`
			res, err := tx.Exec(query,
				shift.TypeID,
				shift.StartTime.Format(time.RFC3339),
				shift.Duration,
				shift.Required,
				shift.CommentDE,
				shift.CommentEN,
			)
			if err != nil {
				// On DB error, add to failed and continue (partial success)
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to insert shift: %v", err),
					Code:  "db_error",
				})
				continue
			}

			id, err := res.LastInsertId()
			if err != nil {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to get insert ID: %v", err),
					Code:  "db_error",
				})
				continue
			}

			// Load the created shift
			created, err := db.getShiftByIDInTx(tx, id)
			if err != nil {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to load created shift: %v", err),
					Code:  "db_error",
				})
				continue
			}

			result.Success = append(result.Success, created)
		}

		return nil
	})

	if err != nil {
		db.logError("CreateShiftsBatch", err, len(shifts))
		return nil, err
	}

	return result, nil
}

// ShiftBatchUpdate represents an update operation for a single shift in a batch.
// Fields are optional (use nil to skip updating that field).
type ShiftBatchUpdate struct {
	ID        int64   `json:"id"`
	TypeID    *int64  `json:"type_id,omitempty"`
	StartTime *string `json:"start_time,omitempty"` // RFC3339 format
	Duration  *int    `json:"duration,omitempty"`
	Required  *int    `json:"required,omitempty"`
	CommentDE *string `json:"comment_de,omitempty"`
	CommentEN *string `json:"comment_en,omitempty"`
}

// UpdateShiftsBatch updates multiple shifts in a single transaction.
// It validates all updates first, then applies them.
func (db *DB) UpdateShiftsBatch(updates []ShiftBatchUpdate, planID int64) (*model.BatchResult[int64], error) {
	result := &model.BatchResult[int64]{
		Success: []int64{},
		Failed:  []model.BatchItemError{},
	}

	if len(updates) == 0 {
		return result, nil
	}

	// Get valid type IDs for this plan
	types, err := db.GetShiftTypesByPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shift types: %w", err)
	}
	validTypeIDs := make(map[int64]bool)
	for _, t := range types {
		validTypeIDs[t.ID] = true
	}

	// Pre-validate all updates
	for i, update := range updates {
		// Validate required field if provided
		if update.Required != nil {
			if *update.Required != -1 && *update.Required < 1 {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: ErrInvalidRequired.Error(),
					Code:  "invalid_required",
				})
				continue
			}
		}

		// Validate type belongs to plan if provided
		if update.TypeID != nil {
			if !validTypeIDs[*update.TypeID] {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: "shift type not found in this plan",
					Code:  "invalid_type_id",
				})
				continue
			}
		}

		// Validate shift belongs to plan
		belongs, err := db.VerifyShiftBelongsToPlan(update.ID, planID)
		if err != nil {
			result.Failed = append(result.Failed, model.BatchItemError{
				Index: i,
				Error: fmt.Sprintf("failed to verify shift ownership: %v", err),
				Code:  "db_error",
			})
			continue
		}
		if !belongs {
			result.Failed = append(result.Failed, model.BatchItemError{
				Index: i,
				Error: "shift not found in this plan",
				Code:  "not_found",
			})
		}
	}

	// If any validation failed, return early without updating anything
	if len(result.Failed) > 0 {
		return result, nil
	}

	// All validations passed, update shifts in a transaction
	err = db.transaction(func(tx *sql.Tx) error {
		for i, update := range updates {
			// Build dynamic UPDATE query based on provided fields
			setClauses := []string{}
			args := []interface{}{}

			if update.TypeID != nil {
				setClauses = append(setClauses, "type_id = ?")
				args = append(args, *update.TypeID)
			}
			if update.StartTime != nil {
				setClauses = append(setClauses, "start_time = ?")
				args = append(args, *update.StartTime)
			}
			if update.Duration != nil {
				setClauses = append(setClauses, "duration = ?")
				args = append(args, *update.Duration)
			}
			if update.Required != nil {
				setClauses = append(setClauses, "required = ?")
				args = append(args, *update.Required)
			}
			if update.CommentDE != nil {
				setClauses = append(setClauses, "comment_de = ?")
				args = append(args, *update.CommentDE)
			}
			if update.CommentEN != nil {
				setClauses = append(setClauses, "comment_en = ?")
				args = append(args, *update.CommentEN)
			}

			// Skip if no fields to update
			if len(setClauses) == 0 {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: "no fields to update",
					Code:  "invalid_input",
				})
				continue
			}

			// Construct query with dynamic SET clause
			query := "UPDATE shifts SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"

			// Add ID to args
			args = append(args, update.ID)

			res, err := tx.Exec(query, args...)
			if err != nil {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to update shift: %v", err),
					Code:  "db_error",
				})
				continue
			}

			rows, err := res.RowsAffected()
			if err != nil {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to get rows affected: %v", err),
					Code:  "db_error",
				})
				continue
			}

			if rows == 0 {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: "shift not found",
					Code:  "not_found",
				})
				continue
			}

			result.Success = append(result.Success, update.ID)
		}

		return nil
	})

	if err != nil {
		db.logError("UpdateShiftsBatch", err, len(updates))
		return nil, err
	}

	return result, nil
}

// DeleteShiftsBatch deletes multiple shifts in a single transaction.
// It validates all IDs first, then deletes them.
func (db *DB) DeleteShiftsBatch(shiftIDs []int64, planID int64) (*model.BatchResult[int64], error) {
	result := &model.BatchResult[int64]{
		Success: []int64{},
		Failed:  []model.BatchItemError{},
	}

	if len(shiftIDs) == 0 {
		return result, nil
	}

	// Pre-validate all shift IDs belong to the plan
	for i, shiftID := range shiftIDs {
		belongs, err := db.VerifyShiftBelongsToPlan(shiftID, planID)
		if err != nil {
			result.Failed = append(result.Failed, model.BatchItemError{
				Index: i,
				Error: fmt.Sprintf("failed to verify shift ownership: %v", err),
				Code:  "db_error",
			})
			continue
		}
		if !belongs {
			result.Failed = append(result.Failed, model.BatchItemError{
				Index: i,
				Error: "shift not found in this plan",
				Code:  "not_found",
			})
		}
	}

	// If any validation failed, return early without deleting anything
	if len(result.Failed) > 0 {
		return result, nil
	}

	// All validations passed, delete shifts in a transaction
	err := db.transaction(func(tx *sql.Tx) error {
		for i, shiftID := range shiftIDs {
			// Participants are deleted automatically via CASCADE
			res, err := tx.Exec("DELETE FROM shifts WHERE id = ?", shiftID)
			if err != nil {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to delete shift: %v", err),
					Code:  "db_error",
				})
				continue
			}

			rows, err := res.RowsAffected()
			if err != nil {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: fmt.Sprintf("failed to get rows affected: %v", err),
					Code:  "db_error",
				})
				continue
			}

			if rows == 0 {
				result.Failed = append(result.Failed, model.BatchItemError{
					Index: i,
					Error: "shift not found",
					Code:  "not_found",
				})
				continue
			}

			result.Success = append(result.Success, shiftID)
		}

		return nil
	})

	if err != nil {
		db.logError("DeleteShiftsBatch", err, len(shiftIDs))
		return nil, err
	}

	return result, nil
}

// getShiftByIDInTx retrieves a shift by ID within a transaction (without loading participants)
func (db *DB) getShiftByIDInTx(tx *sql.Tx, id int64) (Shift, error) {
	var s Shift
	var startTime string

	query := `
		SELECT id, type_id, start_time, duration, required,
			   comment_de, comment_en, is_locked
		FROM shifts
		WHERE id = ?
	`
	err := tx.QueryRow(query, id).Scan(
		&s.ID,
		&s.TypeID,
		&startTime,
		&s.Duration,
		&s.Required,
		&s.CommentDE,
		&s.CommentEN,
		&s.IsLocked,
	)
	if err != nil {
		return s, fmt.Errorf("failed to scan shift: %w", err)
	}

	s.StartTime, _ = time.Parse(time.RFC3339, startTime)

	return s, nil
}
