package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Participant represents a person assigned to a shift
type Participant struct {
	ID        int64     `json:"id"`
	ShiftID   int64     `json:"shift_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// AddParticipant adds a participant to a shift.
// This uses an atomic INSERT with capacity check to prevent race conditions
// where two users could simultaneously take the last available spot.
func (db *DB) AddParticipant(shiftID int64, name string) (*Participant, error) {
	start := time.Now()
	defer func() {
		db.recordQuery("insert", "participants", start)
	}()

	var result Participant

	// Use IMMEDIATE transaction to acquire write lock at the start,
	// preventing race conditions in the check-then-insert pattern
	err := db.transactionImmediate(func(tx *sql.Tx) error {
		createdAt := time.Now().UTC().Format(time.RFC3339)

		// Atomic INSERT with all checks in a single query.
		// This prevents TOCTOU race conditions by making the check and insert atomic.
		// The INSERT only succeeds if:
		// 1. No participant with same name exists for this shift
		// 2. Either collaborative_mode is enabled, OR required is -1 (unlimited), OR current count < required
		query := `
            INSERT INTO participants (shift_id, name, created_at)
            SELECT ?, ?, ?
            WHERE NOT EXISTS (
                SELECT 1 FROM participants WHERE shift_id = ? AND name = ?
            )
            AND (
                SELECT CASE 
                    WHEN sp.collaborative_mode = 1 THEN 1
                    WHEN s.required = -1 THEN 1
                    WHEN (SELECT COUNT(*) FROM participants WHERE shift_id = ?) < s.required THEN 1
                    ELSE 0
                END
                FROM shifts s
                JOIN shift_types st ON s.type_id = st.id
                JOIN shift_plans sp ON st.plan_id = sp.id
                WHERE s.id = ?
            ) = 1
        `
		res, err := tx.Exec(query, shiftID, name, createdAt, shiftID, name, shiftID, shiftID)
		if err != nil {
			return fmt.Errorf("failed to insert participant: %w", err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		// If no rows were inserted, determine why
		if rowsAffected == 0 {
			// Check if participant already exists
			var exists bool
			err := tx.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM participants WHERE shift_id = ? AND name = ?)",
				shiftID, name,
			).Scan(&exists)
			if err != nil {
				return fmt.Errorf("failed to check participant existence: %w", err)
			}
			if exists {
				return ErrParticipantExists
			}

			// Otherwise the shift must be full
			return ErrShiftFull
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		result, err = db.getParticipantByID(tx, id)
		if err != nil {
			return fmt.Errorf("failed to load created participant: %w", err)
		}

		return nil
	})

	if err != nil {
		db.logError("AddParticipant", err, shiftID, name)
		db.recordError("insert", err)
		return nil, err
	}

	return &result, nil
}

// RemoveParticipant removes a participant from a shift
func (db *DB) RemoveParticipant(shiftID int64, name string) error {
	start := time.Now()
	defer func() {
		db.recordQuery("delete", "participants", start)
	}()

	query := `DELETE FROM participants WHERE shift_id = ? AND name = ?`
	result, err := db.Exec(query, shiftID, name)
	if err != nil {
		db.logError("RemoveParticipant", err, shiftID, name)
		db.recordError("delete", err)
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetParticipantsByShift retrieves all participants for a shift
func (db *DB) GetParticipantsByShift(shiftID int64) ([]Participant, error) {
	return db.getParticipantsByShift(nil, shiftID)
}

// getParticipantsByShift retrieves all participants for a shift, optionally within a transaction
func (db *DB) getParticipantsByShift(tx *sql.Tx, shiftID int64) ([]Participant, error) {
	query := `
        SELECT id, shift_id, name, created_at
        FROM participants
        WHERE shift_id = ?
        ORDER BY created_at
    `

	var rows *sql.Rows
	var err error

	if tx != nil {
		rows, err = tx.Query(query, shiftID)
	} else {
		rows, err = db.rdb.Query(query, shiftID)
	}

	if err != nil {
		db.logError("getParticipantsByShift", err, shiftID)
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []Participant
	for rows.Next() {
		var p Participant
		var createdAt string
		if err := rows.Scan(&p.ID, &p.ShiftID, &p.Name, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		if p.CreatedAt, err = parseTimestamp("participants", p.ID, "created_at", createdAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating participants: %w", err)
	}

	return participants, nil
}

// getParticipantByID retrieves a participant by ID within a transaction
func (db *DB) getParticipantByID(tx *sql.Tx, id int64) (Participant, error) {
	var p Participant
	var createdAt string

	query := `
        SELECT id, shift_id, name, created_at
        FROM participants
        WHERE id = ?
    `
	err := tx.QueryRow(query, id).Scan(
		&p.ID,
		&p.ShiftID,
		&p.Name,
		&createdAt,
	)
	if err != nil {
		return p, fmt.Errorf("failed to scan participant: %w", err)
	}

	if p.CreatedAt, err = parseTimestamp("participants", p.ID, "created_at", createdAt); err != nil {
		return p, err
	}
	return p, nil
}
