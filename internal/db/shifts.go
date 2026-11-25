package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/72nd/shifts/internal/model"
)

// Shift represents a single shift instance
type Shift struct {
	ID            int64          `json:"id"`
	TypeID        int64          `json:"type_id"`
	StartTime     model.DateTime `json:"start_time"`
	Duration      int            `json:"duration"` // in minutes
	Required      int            `json:"required"`
	CommentDE     string         `json:"comment_de"`
	CommentEN     string         `json:"comment_en"`
	CommentDEHTML *string        `json:"comment_de_html,omitempty"`
	CommentENHTML *string        `json:"comment_en_html,omitempty"`
	IsLocked      bool           `json:"is_locked"`
	Participants  []Participant  `json:"participants,omitempty"`
	// Overrides of the shift type's coordination settings (nil = as the type).
	LoadFactor  *float64 `json:"load_factor,omitempty"`
	Popularity  *string  `json:"popularity,omitempty"`
	ExpMinCount *int     `json:"exp_min_count,omitempty"`
	ExpMinLevel *string  `json:"exp_min_level,omitempty"`
}

// shiftColumns are the shifts columns in the order scanShift expects.
const shiftColumns = `id, type_id, start_time, duration, required, comment_de, comment_en, is_locked,
	load_factor, popularity, exp_min_count, exp_min_level`

// shiftJoinColumns are the same columns qualified with the alias s, for JOIN queries.
const shiftJoinColumns = `s.id, s.type_id, s.start_time, s.duration, s.required, s.comment_de, s.comment_en, s.is_locked,
	s.load_factor, s.popularity, s.exp_min_count, s.exp_min_level`

// nullableShiftOverrides scans the four override columns.
type shiftOverrides struct {
	loadFactor  sql.NullFloat64
	popularity  sql.NullString
	expMinCount sql.NullInt64
	expMinLevel sql.NullString
}

// apply copies the scanned overrides to the shift.
func (o shiftOverrides) apply(s *Shift) {
	if o.loadFactor.Valid {
		v := o.loadFactor.Float64
		s.LoadFactor = &v
	}
	if o.popularity.Valid {
		v := o.popularity.String
		s.Popularity = &v
	}
	if o.expMinCount.Valid {
		v := int(o.expMinCount.Int64)
		s.ExpMinCount = &v
	}
	if o.expMinLevel.Valid {
		v := o.expMinLevel.String
		s.ExpMinLevel = &v
	}
}

// scanShift reads a row selected with shiftColumns (without participants or rendered comments).
func scanShift(row rowScanner) (Shift, error) {
	var s Shift
	var o shiftOverrides
	err := row.Scan(&s.ID, &s.TypeID, &s.StartTime, &s.Duration, &s.Required, &s.CommentDE, &s.CommentEN, &s.IsLocked,
		&o.loadFactor, &o.popularity, &o.expMinCount, &o.expMinLevel)
	if err != nil {
		return s, err
	}
	o.apply(&s)
	return s, nil
}

// renderShiftComments fills the HTML fields of a shift from its Markdown comments.
func renderShiftComments(s *Shift) {
	if s.CommentEN != "" {
		html := renderMarkdown(s.CommentEN)
		s.CommentENHTML = &html
	}
	if s.CommentDE != "" {
		html := renderMarkdown(s.CommentDE)
		s.CommentDEHTML = &html
	}
}

// NewShift contains the data needed to create a new shift
type NewShift struct {
	TypeID    int64          `json:"type_id"`
	StartTime model.DateTime `json:"start_time" swaggertype:"string" example:"2024-12-20T09:00:00"`
	Duration  int            `json:"duration"`
	Required  int            `json:"required"`
	CommentDE string         `json:"comment_de"`
	CommentEN string         `json:"comment_en"`
	// Overrides of the shift type's coordination settings (nil = as the type).
	LoadFactor  *float64 `json:"load_factor,omitempty"`
	Popularity  *string  `json:"popularity,omitempty"`
	ExpMinCount *int     `json:"exp_min_count,omitempty"`
	ExpMinLevel *string  `json:"exp_min_level,omitempty"`
}

// CreateShift creates a new shift
func (db *DB) CreateShift(shift NewShift) (*Shift, error) {
	start := time.Now()
	defer func() {
		db.recordQuery("insert", "shifts", start)
	}()

	// Validate required field
	if shift.Required != -1 && shift.Required < 1 {
		return nil, ErrInvalidRequired
	}

	var result Shift

	err := db.transaction(func(tx *sql.Tx) error {
		query := `
            INSERT INTO shifts (
                type_id, start_time, duration, required,
                comment_de, comment_en,
                load_factor, popularity, exp_min_count, exp_min_level
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
		res, err := tx.Exec(query,
			shift.TypeID,
			shift.StartTime,
			shift.Duration,
			shift.Required,
			shift.CommentDE,
			shift.CommentEN,
			shift.LoadFactor,
			shift.Popularity,
			shift.ExpMinCount,
			shift.ExpMinLevel,
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
		db.recordError("insert", err)
		return nil, err
	}

	return &result, nil
}

// GetShiftsByType retrieves all shifts for a shift type, ordered by start time.
// This uses a single JOIN query to fetch shifts and participants efficiently.
func (db *DB) GetShiftsByType(typeID int64) ([]Shift, error) {
	query := `
		SELECT ` + shiftJoinColumns + `,
			   ` + participantJoinColumns + `
		FROM shifts s
		LEFT JOIN participants p ON p.shift_id = s.id
		WHERE s.type_id = ?
		ORDER BY s.start_time, p.created_at, p.id
	`
	rows, err := db.rdb.Query(query, typeID)
	if err != nil {
		db.logError("GetShiftsByType", err, typeID)
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	return db.groupShiftsResult(rows)
}

// GetShiftsByPlan retrieves all shifts for a plan.
// This uses a single JOIN query to fetch shifts and participants efficiently.
func (db *DB) GetShiftsByPlan(planID int64) ([]Shift, error) {
	query := `
        SELECT ` + shiftJoinColumns + `,
               ` + participantJoinColumns + `
        FROM shifts s
        JOIN shift_types st ON s.type_id = st.id
        LEFT JOIN participants p ON p.shift_id = s.id
        WHERE st.plan_id = ?
        ORDER BY s.start_time, p.created_at, p.id
    `
	rows, err := db.rdb.Query(query, planID)
	if err != nil {
		db.logError("GetShiftsByPlan", err, planID)
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	return db.groupShiftsResult(rows)
}

// groupShiftsResult groups flat JOIN query results into Shift -> Participant structure
func (db *DB) groupShiftsResult(rows *sql.Rows) ([]Shift, error) {
	shiftMap := make(map[int64]*Shift)
	shiftOrder := []int64{}

	for rows.Next() {
		// Shift fields
		var shiftID int64
		var shiftTypeID int64
		var shiftStartTime model.DateTime
		var shiftDuration int
		var shiftRequired int
		var shiftCommentDE string
		var shiftCommentEN string
		var shiftIsLocked bool
		var overrides shiftOverrides

		// Participant fields (nullable due to LEFT JOIN)
		var np nullableParticipant

		dest := []any{
			&shiftID, &shiftTypeID, &shiftStartTime, &shiftDuration, &shiftRequired,
			&shiftCommentDE, &shiftCommentEN, &shiftIsLocked,
			&overrides.loadFactor, &overrides.popularity, &overrides.expMinCount, &overrides.expMinLevel,
		}
		if err := rows.Scan(append(dest, np.scanDest()...)...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Get or create Shift
		s, exists := shiftMap[shiftID]
		if !exists {
			s = &Shift{
				ID:           shiftID,
				TypeID:       shiftTypeID,
				StartTime:    shiftStartTime,
				Duration:     shiftDuration,
				Required:     shiftRequired,
				CommentDE:    shiftCommentDE,
				CommentEN:    shiftCommentEN,
				IsLocked:     shiftIsLocked,
				Participants: []Participant{},
			}
			overrides.apply(s)
			renderShiftComments(s)

			shiftMap[shiftID] = s
			shiftOrder = append(shiftOrder, shiftID)
		}

		// Handle participant if present
		if np.ID.Valid {
			p, err := np.toParticipant(shiftID)
			if err != nil {
				return nil, err
			}
			s.Participants = append(s.Participants, p)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Convert map to ordered slice
	shifts := make([]Shift, 0, len(shiftOrder))
	for _, id := range shiftOrder {
		shifts = append(shifts, *shiftMap[id])
	}

	return shifts, nil
}

// UpdateShift updates an existing shift
func (db *DB) UpdateShift(id int64, shift Shift) error {
	// Validate required field
	if shift.Required != -1 && shift.Required < 1 {
		return ErrInvalidRequired
	}

	query := `
        UPDATE shifts SET
            start_time = ?,
            duration = ?,
            required = ?,
            comment_de = ?,
            comment_en = ?,
            load_factor = ?,
            popularity = ?,
            exp_min_count = ?,
            exp_min_level = ?
        WHERE id = ? AND type_id = ?
    `
	result, err := db.Exec(query,
		shift.StartTime,
		shift.Duration,
		shift.Required,
		shift.CommentDE,
		shift.CommentEN,
		shift.LoadFactor,
		shift.Popularity,
		shift.ExpMinCount,
		shift.ExpMinLevel,
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
		return ErrNotFound
	}

	return nil
}

// DeleteShift deletes a shift and all its participants
func (db *DB) DeleteShift(id int64) error {
	start := time.Now()
	defer func() {
		db.recordQuery("delete", "shifts", start)
	}()

	query := `DELETE FROM shifts WHERE id = ?`
	result, err := db.Exec(query, id)
	if err != nil {
		db.logError("DeleteShift", err, id)
		db.recordError("delete", err)
		return fmt.Errorf("failed to delete shift: %w", err)
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

// getShiftByID retrieves a shift by ID within a transaction
func (db *DB) getShiftByID(tx *sql.Tx, id int64) (Shift, error) {
	s, err := scanShift(tx.QueryRow(`SELECT `+shiftColumns+` FROM shifts WHERE id = ?`, id))
	if err != nil {
		return s, fmt.Errorf("failed to scan shift: %w", err)
	}

	// Load participants using the transaction
	s.Participants, err = db.getParticipantsByShift(tx, s.ID)
	if err != nil {
		return s, fmt.Errorf("failed to load participants: %w", err)
	}

	renderShiftComments(&s)
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
	err := db.rdb.QueryRow(query, shiftID, planID).Scan(&exists)
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
		return ErrNotFound
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
		return ErrNotFound
	}

	return nil
}

// GetShiftByID retrieves a shift by ID (public version for handlers)
func (db *DB) GetShiftByID(shiftID int64) (*Shift, error) {
	s, err := scanShift(db.rdb.QueryRow(`SELECT `+shiftColumns+` FROM shifts WHERE id = ?`, shiftID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.logError("GetShiftByID", err, shiftID)
		return nil, fmt.Errorf("failed to get shift: %w", err)
	}

	// Load participants
	s.Participants, err = db.GetParticipantsByShift(s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load participants: %w", err)
	}

	renderShiftComments(&s)
	return &s, nil
}
