package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// ShiftPlan represents a complete shift planning schedule
type ShiftPlan struct {
	ID             int64     `json:"id"`
	Token          string    `json:"token"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	CreatedAt      time.Time `json:"created_at"`
	IsMultilingual bool      `json:"is_multilingual"`
	TitleDE        *string   `json:"title_de"`
	TitleEN        string    `json:"title_en"`
	DescriptionDE  *string   `json:"description_de"`
	DescriptionEN  string    `json:"description_en"`
	AdminPassword  string    `json:"-"` // Never sent to client
}

// NewShiftPlan contains the data needed to create a new shift plan
type NewShiftPlan struct {
	TitleDE        *string   `json:"title_de"`
	TitleEN        string    `json:"title_en"`
	DescriptionDE  *string   `json:"description_de"`
	DescriptionEN  string    `json:"description_en"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	IsMultilingual bool      `json:"is_multilingual"`
	Password       string    `json:"password"`
}

var shiftPlanSchema = `
    CREATE TABLE IF NOT EXISTS shift_plans (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        token TEXT UNIQUE NOT NULL,
        admin_password TEXT NOT NULL,
        start_date TEXT NOT NULL,
        end_date TEXT NOT NULL,
        created_at TEXT NOT NULL,
        is_multilingual BOOLEAN NOT NULL DEFAULT true,
        title_de TEXT,
        title_en TEXT NOT NULL,
        description_de TEXT,
        description_en TEXT NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_shift_plans_token ON shift_plans(token);
`

// generateToken generates a random token for shift plans
func generateToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 12

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// generatePassword generates a random password that is easy to remember
func generatePassword() string {
	// Define word lists for generating memorable passwords
	adjectives := []string{
		"happy", "brave", "calm", "wise", "kind", "bold", "proud", "quick",
		"smart", "bright", "strong", "warm", "cool", "sweet", "fresh", "wild",
		"soft", "fair", "pure", "free", "safe", "rich", "deep", "light",
		"sharp", "clear", "clean", "broad", "fine", "firm", "grand", "plain",
		"prime", "quiet", "small", "solid", "whole", "young", "good", "great",
	}
	nouns := []string{
		"apple", "bird", "cloud", "dream", "earth", "fire", "grass", "heart",
		"king", "lake", "moon", "night", "ocean", "pearl", "queen", "river",
		"star", "tree", "wave", "wind", "world", "bear", "book", "door",
		"fish", "game", "home", "lion", "rock", "ship", "snow", "song",
		"sun", "time", "wolf", "wood", "rose", "rain", "gold", "hill",
	}

	// Generate password components
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]

	// Combine components into a memorable password (e.g., "BraveWolf7")
	return fmt.Sprintf("%s%s%02d",
		strings.Title(adj),
		strings.Title(noun),
		rand.Intn(100),
	)
}

// hashPassword creates a simple hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// CreateShiftPlan creates a new shift plan and returns it with the generated ID and token
func (db *DB) CreateShiftPlan(plan NewShiftPlan) (*ShiftPlan, error) {
	var result ShiftPlan

	// Validate required fields
	if plan.TitleEN == "" {
		return nil, fmt.Errorf("title_en is required")
	}
	if plan.DescriptionEN == "" {
		return nil, fmt.Errorf("description_en is required")
	}

	// Generate token and password
	token := generateToken()
	password := generatePassword()
	hashedPassword := hashPassword(password)

	// For single language plans, use English fields only
	if !plan.IsMultilingual {
		plan.TitleDE = nil
		plan.DescriptionDE = nil
	}

	err := db.transaction(func(tx *sql.Tx) error {
		query := `
            INSERT INTO shift_plans (
                token, admin_password, start_date, end_date, created_at,
                is_multilingual, title_de, title_en, description_de, description_en
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
		res, err := tx.Exec(query,
			token,
			hashedPassword,
			plan.StartDate.Format(time.RFC3339),
			plan.EndDate.Format(time.RFC3339),
			time.Now().Format(time.RFC3339),
			plan.IsMultilingual,
			plan.TitleDE,
			plan.TitleEN,
			plan.DescriptionDE,
			plan.DescriptionEN,
		)
		if err != nil {
			return fmt.Errorf("failed to insert shift plan: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

		result, err = db.getShiftPlanByID(tx, id)
		if err != nil {
			return fmt.Errorf("failed to load created shift plan: %w", err)
		}

		// Create example data
		if err := db.createExampleShifts(tx, &result); err != nil {
			return fmt.Errorf("failed to create example shifts: %w", err)
		}

		// Set the generated password for the result
		result.AdminPassword = password

		return nil
	})

	if err != nil {
		db.logError("CreateShiftPlan", err, plan)
		return nil, err
	}

	return &result, nil
}

// createExampleShifts creates example shift types and shifts for a new plan
func (db *DB) createExampleShifts(tx *sql.Tx, plan *ShiftPlan) error {
	// Create example shift type
	shiftType := NewShiftType{
		PlanID:        plan.ID,
		TitleDE:       "Küchendienst",
		TitleEN:       "Kitchen Duty",
		DescriptionDE: "Hilfe in der Küche beim Kochen und Abwaschen",
		DescriptionEN: "Help in the kitchen with cooking and dishes",
		LocationDE:    "Hauptküche",
		LocationEN:    "Main Kitchen",
	}

	// Use the transaction to create the shift type
	query := `
		INSERT INTO shift_types (
			plan_id, sort_order,
			title_de, title_en,
			description_de, description_en,
			location_de, location_en
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := tx.Exec(query,
		shiftType.PlanID,
		0, // First shift type
		shiftType.TitleDE,
		shiftType.TitleEN,
		shiftType.DescriptionDE,
		shiftType.DescriptionEN,
		shiftType.LocationDE,
		shiftType.LocationEN,
	)
	if err != nil {
		return fmt.Errorf("failed to create shift type: %w", err)
	}

	typeID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get shift type ID: %w", err)
	}

	// Create shifts for two days
	startDate := plan.StartDate
	for day := 0; day < 2; day++ {
		currentDate := startDate.AddDate(0, 0, day)

		// Define shifts for the day
		shifts := []NewShift{
			{
				TypeID:    typeID,
				StartTime: time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 8, 0, 0, 0, currentDate.Location()),
				Duration:  120, // 2 hours
				Required:  2,
				CommentDE: "Frühstücksvorbereitung und Abwasch",
				CommentEN: "Breakfast preparation and cleanup",
			},
			{
				TypeID:    typeID,
				StartTime: time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 11, 30, 0, 0, currentDate.Location()),
				Duration:  180, // 3 hours
				Required:  3,
				CommentDE: "Mittagessen vorbereiten und aufräumen",
				CommentEN: "Lunch preparation and cleanup",
			},
			{
				TypeID:    typeID,
				StartTime: time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 17, 0, 0, 0, currentDate.Location()),
				Duration:  180, // 3 hours
				Required:  3,
				CommentDE: "Abendessen vorbereiten und aufräumen",
				CommentEN: "Dinner preparation and cleanup",
			},
			{
				TypeID:    typeID,
				StartTime: time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 20, 0, 0, 0, currentDate.Location()),
				Duration:  90, // 1.5 hours
				Required:  2,
				CommentDE: "Küche aufräumen und vorbereiten für den nächsten Tag",
				CommentEN: "Kitchen cleanup and preparation for next day",
			},
		}

		// Create each shift
		for _, shift := range shifts {
			query := `
				INSERT INTO shifts (
					type_id, start_time, duration, required,
					comment_de, comment_en
				) VALUES (?, ?, ?, ?, ?, ?)
			`
			if _, err := tx.Exec(query,
				shift.TypeID,
				shift.StartTime.Format(time.RFC3339),
				shift.Duration,
				shift.Required,
				shift.CommentDE,
				shift.CommentEN,
			); err != nil {
				return fmt.Errorf("failed to create shift: %w", err)
			}
		}
	}

	return nil
}

// GetShiftPlanByToken retrieves a shift plan by its token
func (db *DB) GetShiftPlanByToken(token string) (*ShiftPlan, error) {
	var plan ShiftPlan
	var startDate, endDate, createdAt string

	query := `
        SELECT id, token, admin_password, start_date, end_date, created_at,
               is_multilingual, title_de, title_en, description_de, description_en
        FROM shift_plans
        WHERE token = ?
    `
	err := db.QueryRow(query, token).Scan(
		&plan.ID,
		&plan.Token,
		&plan.AdminPassword,
		&startDate,
		&endDate,
		&createdAt,
		&plan.IsMultilingual,
		&plan.TitleDE,
		&plan.TitleEN,
		&plan.DescriptionDE,
		&plan.DescriptionEN,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.logError("GetShiftPlanByToken", err, token)
		return nil, fmt.Errorf("failed to get shift plan: %w", err)
	}

	// Parse dates
	plan.StartDate, _ = time.Parse(time.RFC3339, startDate)
	plan.EndDate, _ = time.Parse(time.RFC3339, endDate)
	plan.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &plan, nil
}

// getShiftPlanByID retrieves a shift plan by ID within a transaction
func (db *DB) getShiftPlanByID(tx *sql.Tx, id int64) (ShiftPlan, error) {
	var plan ShiftPlan
	var startDate, endDate, createdAt string

	query := `
        SELECT id, token, admin_password, start_date, end_date, created_at,
               is_multilingual, title_de, title_en, description_de, description_en
        FROM shift_plans
        WHERE id = ?
    `
	err := tx.QueryRow(query, id).Scan(
		&plan.ID,
		&plan.Token,
		&plan.AdminPassword,
		&startDate,
		&endDate,
		&createdAt,
		&plan.IsMultilingual,
		&plan.TitleDE,
		&plan.TitleEN,
		&plan.DescriptionDE,
		&plan.DescriptionEN,
	)
	if err != nil {
		return plan, fmt.Errorf("failed to scan shift plan: %w", err)
	}

	// Parse dates
	plan.StartDate, _ = time.Parse(time.RFC3339, startDate)
	plan.EndDate, _ = time.Parse(time.RFC3339, endDate)
	plan.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return plan, nil
}

// ValidateShiftPlanPassword checks if the provided password matches the plan's admin password
func (db *DB) ValidateShiftPlanPassword(planID int64, hashedPassword string) (bool, error) {
	var storedHash string
	query := `SELECT admin_password FROM shift_plans WHERE id = ?`

	err := db.QueryRow(query, planID).Scan(&storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		db.logError("ValidateShiftPlanPassword", err, planID)
		return false, fmt.Errorf("failed to validate password: %w", err)
	}

	return storedHash == hashedPassword, nil
}

// UpdateShiftPlan updates an existing shift plan with new data
func (db *DB) UpdateShiftPlan(token string, plan NewShiftPlan) (*ShiftPlan, error) {
	var result *ShiftPlan

	// Validate required fields
	if plan.TitleEN == "" {
		return nil, fmt.Errorf("title_en is required")
	}
	if plan.DescriptionEN == "" {
		return nil, fmt.Errorf("description_en is required")
	}

	// For single language plans, use English fields only
	if !plan.IsMultilingual {
		plan.TitleDE = nil
		plan.DescriptionDE = nil
	}

	err := db.transaction(func(tx *sql.Tx) error {
		query := `
            UPDATE shift_plans SET
                start_date = ?,
                end_date = ?,
                is_multilingual = ?,
                title_de = ?,
                title_en = ?,
                description_de = ?,
                description_en = ?
            WHERE token = ?
        `
		res, err := tx.Exec(query,
			plan.StartDate.Format(time.RFC3339),
			plan.EndDate.Format(time.RFC3339),
			plan.IsMultilingual,
			plan.TitleDE,
			plan.TitleEN,
			plan.DescriptionDE,
			plan.DescriptionEN,
			token,
		)
		if err != nil {
			return fmt.Errorf("failed to update shift plan: %w", err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("shift plan not found")
		}

		// Get the updated plan
		result, err = db.GetShiftPlanByToken(token)
		if err != nil {
			return fmt.Errorf("failed to load updated shift plan: %w", err)
		}

		return nil
	})

	if err != nil {
		db.logError("UpdateShiftPlan", err, plan)
		return nil, err
	}

	return result, nil
}
