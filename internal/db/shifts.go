package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Shift represents a single shift instance
type Shift struct {
	ID            int64         `json:"id"`
	TypeID        int64         `json:"type_id"`
	StartTime     time.Time     `json:"start_time"`
	Duration      int           `json:"duration"` // in minutes
	Required      int           `json:"required"`
	CommentDE     string        `json:"comment_de"`
	CommentEN     string        `json:"comment_en"`
	CommentDEHTML *string       `json:"comment_de_html,omitempty"`
	CommentENHTML *string       `json:"comment_en_html,omitempty"`
	IsLocked      bool          `json:"is_locked"`
	Participants  []Participant `json:"participants,omitempty"`
}

// NewShift contains the data needed to create a new shift
type NewShift struct {
	TypeID    int64     `json:"type_id"`
	StartTime time.Time `json:"start_time"`
	Duration  int       `json:"duration"`
	Required  int       `json:"required"`
	CommentDE string    `json:"comment_de"`
	CommentEN string    `json:"comment_en"`
}

var shiftSchema = `
    CREATE TABLE IF NOT EXISTS shifts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        type_id INTEGER NOT NULL,
        start_time TEXT NOT NULL,
        duration INTEGER NOT NULL,
        required INTEGER NOT NULL,
        comment_de TEXT,
        comment_en TEXT,
        is_locked BOOLEAN NOT NULL DEFAULT false,
        FOREIGN KEY (type_id) REFERENCES shift_types(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS idx_shifts_type_id ON shifts(type_id);
`

// CreateShift creates a new shift
func (db *DB) CreateShift(shift NewShift) (*Shift, error) {
	// Validate required field
	if shift.Required != -1 && shift.Required < 1 {
		return nil, fmt.Errorf("required participants must be either -1 for unlimited or a positive number")
	}

	var result Shift

	err := db.transaction(func(tx *sql.Tx) error {
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
			return fmt.Errorf("failed to insert shift: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		result, err = db.getShiftByID(tx, id)
		if err != nil {
			return fmt.Errorf("failed to load created shift: %w", err)
		}

		return nil
	})

	if err != nil {
		db.logError("CreateShift", err, shift)
		return nil, err
	}

	return &result, nil
}

// GetShiftsByType retrieves all shifts for a shift type, ordered by start time
func (db *DB) GetShiftsByType(typeID int64) ([]Shift, error) {
	query := `
		SELECT s.id, s.type_id, s.start_time, s.duration, s.required,
			   s.comment_de, s.comment_en, s.is_locked
		FROM shifts s
		WHERE s.type_id = ?
		ORDER BY s.start_time
	`
	rows, err := db.Query(query, typeID)
	if err != nil {
		db.logError("GetShiftsByType", err, typeID)
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	var shifts []Shift
	for rows.Next() {
		var s Shift
		var startTime string
		if err := rows.Scan(
			&s.ID,
			&s.TypeID,
			&startTime,
			&s.Duration,
			&s.Required,
			&s.CommentDE,
			&s.CommentEN,
			&s.IsLocked,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shift: %w", err)
		}

		// Parse start time
		s.StartTime, err = time.Parse(time.RFC3339, startTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse start time: %w", err)
		}

		// Load participants for this shift
		participants, err := db.GetParticipantsByShift(s.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load participants for shift %d: %w", s.ID, err)
		}
		s.Participants = participants

		// Render markdown to HTML for comments
		if s.CommentEN != "" {
			html := renderMarkdown(s.CommentEN)
			s.CommentENHTML = &html
		}
		if s.CommentDE != "" {
			html := renderMarkdown(s.CommentDE)
			s.CommentDEHTML = &html
		}

		shifts = append(shifts, s)
	}

	return shifts, nil
}

// GetShiftsByPlan retrieves all shifts for a plan
func (db *DB) GetShiftsByPlan(planID int64) ([]Shift, error) {
	query := `
        SELECT s.id, s.type_id, s.start_time, s.duration, s.required,
               s.comment_de, s.comment_en, s.is_locked
        FROM shifts s
        JOIN shift_types st ON s.type_id = st.id
        WHERE st.plan_id = ?
        ORDER BY s.start_time
    `
	rows, err := db.Query(query, planID)
	if err != nil {
		db.logError("GetShiftsByPlan", err, planID)
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	var shifts []Shift
	for rows.Next() {
		var s Shift
		var startTime string
		if err := rows.Scan(
			&s.ID,
			&s.TypeID,
			&startTime,
			&s.Duration,
			&s.Required,
			&s.CommentDE,
			&s.CommentEN,
			&s.IsLocked,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shift: %w", err)
		}

		s.StartTime, _ = time.Parse(time.RFC3339, startTime)

		// Load participants
		s.Participants, err = db.GetParticipantsByShift(s.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load participants: %w", err)
		}

		// Render markdown to HTML for comments
		if s.CommentEN != "" {
			html := renderMarkdown(s.CommentEN)
			s.CommentENHTML = &html
		}
		if s.CommentDE != "" {
			html := renderMarkdown(s.CommentDE)
			s.CommentDEHTML = &html
		}

		shifts = append(shifts, s)
	}

	return shifts, nil
}

// UpdateShift updates an existing shift
func (db *DB) UpdateShift(id int64, shift Shift) error {
	// Validate required field
	if shift.Required != -1 && shift.Required < 1 {
		return fmt.Errorf("required participants must be either -1 for unlimited or a positive number")
	}

	query := `
        UPDATE shifts SET
            start_time = ?,
            duration = ?,
            required = ?,
            comment_de = ?,
            comment_en = ?
        WHERE id = ? AND type_id = ?
    `
	result, err := db.Exec(query,
		shift.StartTime.Format(time.RFC3339),
		shift.Duration,
		shift.Required,
		shift.CommentDE,
		shift.CommentEN,
		id,
		shift.TypeID,
	)
	if err != nil {
		db.logError("UpdateShift", err, id, shift)
		return fmt.Errorf("failed to update shift: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("shift not found or no permission")
	}

	return nil
}

// DeleteShift deletes a shift and all its participants
func (db *DB) DeleteShift(id int64) error {
	query := `DELETE FROM shifts WHERE id = ?`
	result, err := db.Exec(query, id)
	if err != nil {
		db.logError("DeleteShift", err, id)
		return fmt.Errorf("failed to delete shift: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("shift not found")
	}

	return nil
}

// getShiftByID retrieves a shift by ID within a transaction
func (db *DB) getShiftByID(tx *sql.Tx, id int64) (Shift, error) {
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

	// Load participants
	s.Participants, err = db.GetParticipantsByShift(s.ID)
	if err != nil {
		return s, fmt.Errorf("failed to load participants: %w", err)
	}

	// Render markdown to HTML for comments
	if s.CommentEN != "" {
		html := renderMarkdown(s.CommentEN)
		s.CommentENHTML = &html
	}
	if s.CommentDE != "" {
		html := renderMarkdown(s.CommentDE)
		s.CommentDEHTML = &html
	}

	return s, nil
}

// VerifyShiftBelongsToPlan checks if a shift belongs to a specific plan
func (db *DB) VerifyShiftBelongsToPlan(shiftID, planID int64) (bool, error) {
	var exists bool
	query := `
        SELECT EXISTS(
            SELECT 1 
            FROM shifts s
            JOIN shift_types t ON s.type_id = t.id
            WHERE s.id = ? AND t.plan_id = ?
        )
    `
	err := db.QueryRow(query, shiftID, planID).Scan(&exists)
	if err != nil {
		db.logError("VerifyShiftBelongsToPlan", err, shiftID, planID)
		return false, fmt.Errorf("failed to verify shift ownership: %w", err)
	}
	return exists, nil
}

// LockShift locks a shift, preventing non-admin users from joining or leaving
func (db *DB) LockShift(shiftID int64) error {
	query := `UPDATE shifts SET is_locked = true WHERE id = ?`
	result, err := db.Exec(query, shiftID)
	if err != nil {
		db.logError("LockShift", err, shiftID)
		return fmt.Errorf("failed to lock shift: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("shift not found")
	}

	return nil
}

// UnlockShift unlocks a shift, allowing all users to join or leave
func (db *DB) UnlockShift(shiftID int64) error {
	query := `UPDATE shifts SET is_locked = false WHERE id = ?`
	result, err := db.Exec(query, shiftID)
	if err != nil {
		db.logError("UnlockShift", err, shiftID)
		return fmt.Errorf("failed to unlock shift: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("shift not found")
	}

	return nil
}

// GetShiftByID retrieves a shift by ID (public version for handlers)
func (db *DB) GetShiftByID(shiftID int64) (*Shift, error) {
	var s Shift
	var startTime string

	query := `
        SELECT id, type_id, start_time, duration, required,
               comment_de, comment_en, is_locked
        FROM shifts
        WHERE id = ?
    `
	err := db.QueryRow(query, shiftID).Scan(
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
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.logError("GetShiftByID", err, shiftID)
		return nil, fmt.Errorf("failed to get shift: %w", err)
	}

	s.StartTime, _ = time.Parse(time.RFC3339, startTime)

	// Load participants
	s.Participants, err = db.GetParticipantsByShift(s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load participants: %w", err)
	}

	// Render markdown to HTML for comments
	if s.CommentEN != "" {
		html := renderMarkdown(s.CommentEN)
		s.CommentENHTML = &html
	}
	if s.CommentDE != "" {
		html := renderMarkdown(s.CommentDE)
		s.CommentDEHTML = &html
	}

	return &s, nil
}
