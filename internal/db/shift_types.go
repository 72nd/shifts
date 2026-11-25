package db

import (
	"database/sql"
	"fmt"

	"github.com/72nd/shifts/internal/model"
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
	// Coordination settings. Shifts inherit them unless they override them.
	LoadFactor  float64 `json:"load_factor"`      // 0.1 to 5.0, default 1.0
	Popularity  string  `json:"popularity"`       // auto | popular | normal
	ExpMinCount int     `json:"exp_min_count"`    // people with at least ExpMinLevel needed
	ExpMinLevel *string `json:"exp_min_level"`    // option ID of the experience field
	Shifts      []Shift `json:"shifts,omitempty"` // Added field for shifts
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
	LoadFactor    float64 `json:"load_factor"` // 0 = default 1.0
	Popularity    string  `json:"popularity"`  // "" = auto
	ExpMinCount   int     `json:"exp_min_count"`
	ExpMinLevel   *string `json:"exp_min_level"`
}

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
                location_de, location_en,
                load_factor, popularity, exp_min_count, exp_min_level
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
		if st.LoadFactor == 0 {
			st.LoadFactor = 1.0
		}
		if st.Popularity == "" {
			st.Popularity = "auto"
		}
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
			st.LoadFactor,
			st.Popularity,
			st.ExpMinCount,
			st.ExpMinLevel,
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

// GetShiftTypesByPlan retrieves all shift types for a plan, ordered by sort_order.
// This uses a single JOIN query to fetch all types, shifts, and participants efficiently,
// avoiding the N+1 query problem.
func (db *DB) GetShiftTypesByPlan(planID int64) ([]ShiftType, error) {
	// Single query with JOINs to fetch all data at once
	query := `
        SELECT 
            st.id, st.plan_id, st.password, st.sort_order,
            st.title_de, st.title_en, st.description_de, st.description_en,
            st.location_de, st.location_en,
            st.load_factor, st.popularity, st.exp_min_count, st.exp_min_level,
            s.id, s.start_time, s.duration, s.required,
            s.comment_de, s.comment_en, s.is_locked,
            s.load_factor, s.popularity, s.exp_min_count, s.exp_min_level,
            ` + participantJoinColumns + `
        FROM shift_types st
        LEFT JOIN shifts s ON s.type_id = st.id
        LEFT JOIN participants p ON p.shift_id = s.id
        WHERE st.plan_id = ?
        ORDER BY st.sort_order, s.start_time, p.created_at, p.id
    `
	rows, err := db.rdb.Query(query, planID)
	if err != nil {
		db.logError("GetShiftTypesByPlan", err, planID)
		return nil, fmt.Errorf("failed to query shift types: %w", err)
	}
	defer rows.Close()

	return db.groupShiftTypesResult(rows)
}

// groupShiftTypesResult groups flat JOIN query results into nested ShiftType -> Shift -> Participant structure
func (db *DB) groupShiftTypesResult(rows *sql.Rows) ([]ShiftType, error) {
	typeMap := make(map[int64]*ShiftType)
	shiftMap := make(map[int64]*Shift)
	typeOrder := []int64{}

	for rows.Next() {
		// ShiftType fields
		var typeID int64
		var typePlanID int64
		var typePassword *string
		var typeSortOrder int
		var typeTitleDE, typeTitleEN string
		var typeDescDE, typeDescEN string
		var typeLocDE, typeLocEN string
		var typeLoadFactor float64
		var typePopularity string
		var typeExpMinCount int
		var typeExpMinLevel *string

		// Shift fields (nullable due to LEFT JOIN)
		var shiftID sql.NullInt64
		var shiftStartTime sql.NullString
		var shiftDuration sql.NullInt64
		var shiftRequired sql.NullInt64
		var shiftCommentDE sql.NullString
		var shiftCommentEN sql.NullString
		var shiftIsLocked sql.NullBool
		var overrides shiftOverrides

		// Participant fields (nullable due to LEFT JOIN)
		var np nullableParticipant

		dest := []any{
			&typeID, &typePlanID, &typePassword, &typeSortOrder,
			&typeTitleDE, &typeTitleEN, &typeDescDE, &typeDescEN,
			&typeLocDE, &typeLocEN,
			&typeLoadFactor, &typePopularity, &typeExpMinCount, &typeExpMinLevel,
			&shiftID, &shiftStartTime, &shiftDuration, &shiftRequired,
			&shiftCommentDE, &shiftCommentEN, &shiftIsLocked,
			&overrides.loadFactor, &overrides.popularity, &overrides.expMinCount, &overrides.expMinLevel,
		}
		if err := rows.Scan(append(dest, np.scanDest()...)...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Get or create ShiftType
		st, exists := typeMap[typeID]
		if !exists {
			st = &ShiftType{
				ID:            typeID,
				PlanID:        typePlanID,
				Password:      typePassword,
				SortOrder:     typeSortOrder,
				TitleDE:       typeTitleDE,
				TitleEN:       typeTitleEN,
				DescriptionDE: typeDescDE,
				DescriptionEN: typeDescEN,
				LocationDE:    typeLocDE,
				LocationEN:    typeLocEN,
				LoadFactor:    typeLoadFactor,
				Popularity:    typePopularity,
				ExpMinCount:   typeExpMinCount,
				ExpMinLevel:   typeExpMinLevel,
				Shifts:        []Shift{},
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

			typeMap[typeID] = st
			typeOrder = append(typeOrder, typeID)
		}

		// Handle shift if present
		if shiftID.Valid {
			shift, shiftExists := shiftMap[shiftID.Int64]
			if !shiftExists {
				var startTime model.DateTime
				if err := startTime.Scan(shiftStartTime.String); err != nil {
					return nil, fmt.Errorf("shifts id %d: invalid start_time %q: %w", shiftID.Int64, shiftStartTime.String, err)
				}
				shift = &Shift{
					ID:           shiftID.Int64,
					TypeID:       typeID,
					StartTime:    startTime,
					Duration:     int(shiftDuration.Int64),
					Required:     int(shiftRequired.Int64),
					CommentDE:    shiftCommentDE.String,
					CommentEN:    shiftCommentEN.String,
					IsLocked:     shiftIsLocked.Bool,
					Participants: []Participant{},
				}
				overrides.apply(shift)
				renderShiftComments(shift)

				shiftMap[shiftID.Int64] = shift
				st.Shifts = append(st.Shifts, *shift)
			}

			// Handle participant if present
			if np.ID.Valid {
				// Find the shift in st.Shifts and add participant
				for i := range st.Shifts {
					if st.Shifts[i].ID == shiftID.Int64 {
						// Check if participant already added (shouldn't happen with proper ORDER BY)
						participantExists := false
						for _, p := range st.Shifts[i].Participants {
							if p.ID == np.ID.Int64 {
								participantExists = true
								break
							}
						}
						if !participantExists {
							p, err := np.toParticipant(shiftID.Int64)
							if err != nil {
								return nil, err
							}
							st.Shifts[i].Participants = append(st.Shifts[i].Participants, p)
						}
						break
					}
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Convert map to ordered slice
	types := make([]ShiftType, 0, len(typeOrder))
	for _, id := range typeOrder {
		types = append(types, *typeMap[id])
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
            location_de = ?, location_en = ?,
            load_factor = ?, popularity = ?, exp_min_count = ?, exp_min_level = ?
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
		st.LoadFactor,
		st.Popularity,
		st.ExpMinCount,
		st.ExpMinLevel,
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
		return ErrNotFound
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
				return fmt.Errorf("%w: shift type %d", ErrNotFound, id)
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
		return ErrNotFound
	}

	return nil
}

// ValidateShiftTypePassword checks if the provided password matches
func (db *DB) ValidateShiftTypePassword(typeID int64, hashedPassword string) (bool, error) {
	var storedHash sql.NullString
	query := `SELECT password FROM shift_types WHERE id = ?`

	err := db.rdb.QueryRow(query, typeID).Scan(&storedHash)
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
               location_de, location_en,
               load_factor, popularity, exp_min_count, exp_min_level
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
		&st.LoadFactor,
		&st.Popularity,
		&st.ExpMinCount,
		&st.ExpMinLevel,
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

// GetShiftTypeByID retrieves a single shift type of a plan (without shifts).
// It returns nil if the type does not exist in the plan.
func (db *DB) GetShiftTypeByID(planID, id int64) (*ShiftType, error) {
	var st ShiftType
	err := db.rdb.QueryRow(`
        SELECT id, plan_id, password, sort_order, title_de, title_en,
               description_de, description_en, location_de, location_en,
               load_factor, popularity, exp_min_count, exp_min_level
        FROM shift_types
        WHERE id = ? AND plan_id = ?
    `, id, planID).Scan(
		&st.ID, &st.PlanID, &st.Password, &st.SortOrder, &st.TitleDE, &st.TitleEN,
		&st.DescriptionDE, &st.DescriptionEN, &st.LocationDE, &st.LocationEN,
		&st.LoadFactor, &st.Popularity, &st.ExpMinCount, &st.ExpMinLevel,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.logError("GetShiftTypeByID", err, planID, id)
		return nil, fmt.Errorf("failed to get shift type: %w", err)
	}
	return &st, nil
}
