package db

import (
	"database/sql"
	"fmt"
)

// ShiftType represents a category of shifts
type ShiftType struct {
	ID                int64   `json:"id"`
	PlanID            int64   `json:"plan_id"`
	Password          *string `json:"-"` // Optional, never sent to client
	SortOrder         int     `json:"sort_order"`
	TitleDE           string  `json:"title_de"`
	TitleEN           string  `json:"title_en"`
	DescriptionDE     string  `json:"description_de"`
	DescriptionEN     string  `json:"description_en"`
	DescriptionDEHTML *string `json:"description_de_html,omitempty"`
	DescriptionENHTML *string `json:"description_en_html,omitempty"`
	LocationDE        string  `json:"location_de"`
	LocationEN        string  `json:"location_en"`
	Shifts            []Shift `json:"shifts,omitempty"` // Added field for shifts
}

// NewShiftType contains the data needed to create a new shift type
type NewShiftType struct {
	PlanID        int64   `json:"plan_id"`
	Password      *string `json:"password,omitempty"`
	TitleDE       string  `json:"title_de"`
	TitleEN       string  `json:"title_en"`
	DescriptionDE string  `json:"description_de"`
	DescriptionEN string  `json:"description_en"`
	LocationDE    string  `json:"location_de"`
	LocationEN    string  `json:"location_en"`
}

var shiftTypeSchema = `
    CREATE TABLE IF NOT EXISTS shift_types (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        plan_id INTEGER NOT NULL,
        password TEXT,
        sort_order INTEGER NOT NULL,
        title_de TEXT NOT NULL,
        title_en TEXT NOT NULL,
        description_de TEXT,
        description_en TEXT,
        location_de TEXT,
        location_en TEXT,
        FOREIGN KEY (plan_id) REFERENCES shift_plans(id) ON DELETE CASCADE
    );
    CREATE INDEX IF NOT EXISTS idx_shift_types_plan_id ON shift_types(plan_id);
`

// CreateShiftType creates a new shift type
func (db *DB) CreateShiftType(st NewShiftType) (*ShiftType, error) {
	var result ShiftType

	err := db.transaction(func(tx *sql.Tx) error {
		// Get the highest sort_order for the plan
		var maxOrder int
		err := tx.QueryRow(`
            SELECT COALESCE(MAX(sort_order), -1) 
            FROM shift_types 
            WHERE plan_id = ?
        `, st.PlanID).Scan(&maxOrder)
		if err != nil {
			return fmt.Errorf("failed to get max sort order: %w", err)
		}

		// Insert with next sort order
		query := `
            INSERT INTO shift_types (
                plan_id, password, sort_order,
                title_de, title_en,
                description_de, description_en,
                location_de, location_en
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
		res, err := tx.Exec(query,
			st.PlanID,
			st.Password,
			maxOrder+1,
			st.TitleDE,
			st.TitleEN,
			st.DescriptionDE,
			st.DescriptionEN,
			st.LocationDE,
			st.LocationEN,
		)
		if err != nil {
			return fmt.Errorf("failed to insert shift type: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		result, err = db.getShiftTypeByID(tx, id)
		if err != nil {
			return fmt.Errorf("failed to load created shift type: %w", err)
		}

		return nil
	})

	if err != nil {
		db.logError("CreateShiftType", err, st)
		return nil, err
	}

	return &result, nil
}

// GetShiftTypesByPlan retrieves all shift types for a plan, ordered by sort_order
func (db *DB) GetShiftTypesByPlan(planID int64) ([]ShiftType, error) {
	query := `
        SELECT id, plan_id, password, sort_order,
               title_de, title_en,
               description_de, description_en,
               location_de, location_en
        FROM shift_types
        WHERE plan_id = ?
        ORDER BY sort_order
    `
	rows, err := db.Query(query, planID)
	if err != nil {
		db.logError("GetShiftTypesByPlan", err, planID)
		return nil, fmt.Errorf("failed to query shift types: %w", err)
	}
	defer rows.Close()

	var types []ShiftType
	for rows.Next() {
		var st ShiftType
		if err := rows.Scan(
			&st.ID,
			&st.PlanID,
			&st.Password,
			&st.SortOrder,
			&st.TitleDE,
			&st.TitleEN,
			&st.DescriptionDE,
			&st.DescriptionEN,
			&st.LocationDE,
			&st.LocationEN,
		); err != nil {
			return nil, fmt.Errorf("failed to scan shift type: %w", err)
		}

		// Render markdown to HTML for descriptions
		if st.DescriptionEN != "" {
			html := renderMarkdown(st.DescriptionEN)
			st.DescriptionENHTML = &html
		}
		if st.DescriptionDE != "" {
			html := renderMarkdown(st.DescriptionDE)
			st.DescriptionDEHTML = &html
		}

		// Load shifts for this type
		shifts, err := db.GetShiftsByType(st.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load shifts for type %d: %w", st.ID, err)
		}
		st.Shifts = shifts

		types = append(types, st)
	}

	return types, nil
}

// UpdateShiftType updates an existing shift type
func (db *DB) UpdateShiftType(id int64, st ShiftType) error {
	query := `
        UPDATE shift_types SET
            password = ?,
            title_de = ?, title_en = ?,
            description_de = ?, description_en = ?,
            location_de = ?, location_en = ?
        WHERE id = ? AND plan_id = ?
    `
	result, err := db.Exec(query,
		st.Password,
		st.TitleDE,
		st.TitleEN,
		st.DescriptionDE,
		st.DescriptionEN,
		st.LocationDE,
		st.LocationEN,
		id,
		st.PlanID,
	)
	if err != nil {
		db.logError("UpdateShiftType", err, id, st)
		return fmt.Errorf("failed to update shift type: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("shift type not found or no permission")
	}

	return nil
}

// UpdateShiftTypeOrder updates the sort order of shift types
func (db *DB) UpdateShiftTypeOrder(planID int64, typeIDs []int64) error {
	return db.transaction(func(tx *sql.Tx) error {
		for i, id := range typeIDs {
			query := `
                UPDATE shift_types 
                SET sort_order = ? 
                WHERE id = ? AND plan_id = ?
            `
			result, err := tx.Exec(query, i, id, planID)
			if err != nil {
				return fmt.Errorf("failed to update sort order: %w", err)
			}

			rows, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}
			if rows == 0 {
				return fmt.Errorf("shift type %d not found or no permission", id)
			}
		}
		return nil
	})
}

// DeleteShiftType deletes a shift type and all its shifts
func (db *DB) DeleteShiftType(planID, id int64) error {
	query := `DELETE FROM shift_types WHERE id = ? AND plan_id = ?`
	result, err := db.Exec(query, id, planID)
	if err != nil {
		db.logError("DeleteShiftType", err, id, planID)
		return fmt.Errorf("failed to delete shift type: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("shift type not found or no permission")
	}

	return nil
}

// ValidateShiftTypePassword checks if the provided password matches
func (db *DB) ValidateShiftTypePassword(typeID int64, hashedPassword string) (bool, error) {
	var storedHash sql.NullString
	query := `SELECT password FROM shift_types WHERE id = ?`

	err := db.QueryRow(query, typeID).Scan(&storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		db.logError("ValidateShiftTypePassword", err, typeID)
		return false, fmt.Errorf("failed to validate password: %w", err)
	}

	// If no password is set, access is allowed
	if !storedHash.Valid {
		return true, nil
	}

	return storedHash.String == hashedPassword, nil
}

// getShiftTypeByID retrieves a shift type by ID within a transaction
func (db *DB) getShiftTypeByID(tx *sql.Tx, id int64) (ShiftType, error) {
	var st ShiftType
	query := `
        SELECT id, plan_id, password, sort_order,
               title_de, title_en,
               description_de, description_en,
               location_de, location_en
        FROM shift_types
        WHERE id = ?
    `
	err := tx.QueryRow(query, id).Scan(
		&st.ID,
		&st.PlanID,
		&st.Password,
		&st.SortOrder,
		&st.TitleDE,
		&st.TitleEN,
		&st.DescriptionDE,
		&st.DescriptionEN,
		&st.LocationDE,
		&st.LocationEN,
	)
	if err != nil {
		return st, fmt.Errorf("failed to scan shift type: %w", err)
	}

	// Render markdown to HTML for descriptions
	if st.DescriptionEN != "" {
		html := renderMarkdown(st.DescriptionEN)
		st.DescriptionENHTML = &html
	}
	if st.DescriptionDE != "" {
		html := renderMarkdown(st.DescriptionDE)
		st.DescriptionDEHTML = &html
	}

	return st, nil
}
