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

var participantSchema = `
    CREATE TABLE IF NOT EXISTS participants (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        shift_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        created_at TEXT NOT NULL,
        FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS idx_participants_shift_id ON participants(shift_id);
`

// AddParticipant adds a participant to a shift
func (db *DB) AddParticipant(shiftID int64, name string) (*Participant, error) {
	var result Participant

	err := db.transaction(func(tx *sql.Tx) error {
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

		// Get shift info and plan's collaborative mode
		var required, current int
		var collaborativeMode bool
		err = tx.QueryRow(`
            SELECT s.required, COUNT(p.id), sp.collaborative_mode
            FROM shifts s
            LEFT JOIN participants p ON s.id = p.shift_id
            JOIN shift_types st ON s.type_id = st.id
            JOIN shift_plans sp ON st.plan_id = sp.id
            WHERE s.id = ?
            GROUP BY s.id, s.required, sp.collaborative_mode
        `, shiftID).Scan(&required, &current, &collaborativeMode)
		if err != nil {
			return fmt.Errorf("failed to check shift capacity: %w", err)
		}

		// In collaborative mode, skip capacity check (people can express interest)
		// In normal mode, enforce capacity limits
		if !collaborativeMode && required != -1 && current >= required {
			return ErrShiftFull
		}

		// Add participant
		query := `
            INSERT INTO participants (shift_id, name, created_at)
            VALUES (?, ?, ?)
        `
		res, err := tx.Exec(query,
			shiftID,
			name,
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to insert participant: %w", err)
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
		return nil, err
	}

	return &result, nil
}

// RemoveParticipant removes a participant from a shift
func (db *DB) RemoveParticipant(shiftID int64, name string) error {
	query := `DELETE FROM participants WHERE shift_id = ? AND name = ?`
	result, err := db.Exec(query, shiftID, name)
	if err != nil {
		db.logError("RemoveParticipant", err, shiftID, name)
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
	query := `
        SELECT id, shift_id, name, created_at
        FROM participants
        WHERE shift_id = ?
        ORDER BY created_at
    `
	rows, err := db.Query(query, shiftID)
	if err != nil {
		db.logError("GetParticipantsByShift", err, shiftID)
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
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		participants = append(participants, p)
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

	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return p, nil
}
