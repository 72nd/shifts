package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/util"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// ShiftPlan represents a complete shift planning schedule
type ShiftPlan struct {
	ID                int64      `json:"id"`
	Token             string     `json:"token"`
	StartDate         model.Date `json:"start_date"`
	EndDate           model.Date `json:"end_date"`
	CreatedAt         time.Time  `json:"created_at"`
	IsMultilingual    bool       `json:"is_multilingual"`
	TitleDE           *string    `json:"title_de"`
	TitleEN           string     `json:"title_en"`
	DescriptionDE     *string    `json:"description_de"`
	DescriptionEN     string     `json:"description_en"`
	DescriptionDEHTML *string    `json:"description_de_html,omitempty"`
	DescriptionENHTML *string    `json:"description_en_html,omitempty"`
	AdminPassword     string     `json:"-"` // Never sent to client
	DayBoundaryTime   string     `json:"day_boundary_time"`
	CollaborativeMode bool       `json:"collaborative_mode"`
	TimeZone          string     `json:"time_zone"` // IANA name, fixed at creation
}

// Location returns the plan's time zone, falling back to the default zone if
// the stored name cannot be loaded.
func (p *ShiftPlan) Location() *time.Location {
	if loc, err := time.LoadLocation(p.TimeZone); err == nil && p.TimeZone != "" {
		return loc
	}
	return model.DefaultLocation()
}

// NewShiftPlan contains the data needed to create a new shift plan
type NewShiftPlan struct {
	TitleDE           *string    `json:"title_de"`
	TitleEN           string     `json:"title_en" validate:"required" maxLength:"150"`
	DescriptionDE     *string    `json:"description_de"`
	DescriptionEN     string     `json:"description_en"`
	StartDate         model.Date `json:"start_date" swaggertype:"string" example:"2024-12-20"`
	EndDate           model.Date `json:"end_date" swaggertype:"string" example:"2024-12-22"`
	IsMultilingual    bool       `json:"is_multilingual"`
	Password          string     `json:"password" swaggerignore:"true"`
	DayBoundaryTime   string     `json:"day_boundary_time"`
	CollaborativeMode bool       `json:"collaborative_mode"`
	TimeZone          string     `json:"time_zone"` // only used on create; empty = default zone
}

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

// hashPassword creates a simple hash of the password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// CreateShiftPlan creates a new shift plan and returns it with the generated ID and token
func (db *DB) CreateShiftPlan(plan NewShiftPlan) (*ShiftPlan, error) {
	start := time.Now()
	defer func() {
		db.recordQuery("insert", "shift_plans", start)
	}()

	var result ShiftPlan

	// Validate required fields
	if plan.TitleEN == "" {
		return nil, fmt.Errorf("title_en is required")
	}
	// Beschreibung ist jetzt optional
	// if plan.DescriptionEN == "" {
	// 	return nil, fmt.Errorf("description_en is required")
	// }

	if plan.DayBoundaryTime == "" {
		plan.DayBoundaryTime = "00:00"
	}
	if plan.TimeZone == "" {
		plan.TimeZone = model.DefaultLocation().String()
	}

	// Generate token and password
	token := generateToken()
	password, err := util.PwdGenerator.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}
	hashedPassword := hashPassword(password)

	// For single language plans, use English fields only
	if !plan.IsMultilingual {
		plan.TitleDE = nil
		plan.DescriptionDE = nil
	}

	err = db.transaction(func(tx *sql.Tx) error {
		query := `
            INSERT INTO shift_plans (
                token, admin_password, start_date, end_date, created_at,
                is_multilingual, title_de, title_en, description_de, description_en,
                day_boundary_time, collaborative_mode, time_zone
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        `
		res, err := tx.Exec(query,
			token,
			hashedPassword,
			plan.StartDate,
			plan.EndDate,
			time.Now().UTC().Format(time.RFC3339),
			plan.IsMultilingual,
			plan.TitleDE,
			plan.TitleEN,
			plan.DescriptionDE,
			plan.DescriptionEN,
			plan.DayBoundaryTime,
			plan.CollaborativeMode,
			plan.TimeZone,
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
		db.recordError("insert", err)
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
	for day := 0; day < 2; day++ {
		currentDate := plan.StartDate.AddDays(day)

		// Define shifts for the day
		shifts := []NewShift{
			{
				TypeID:    typeID,
				StartTime: currentDate.At(8, 0),
				Duration:  120, // 2 hours
				Required:  2,
				CommentDE: "Frühstücksvorbereitung und Abwasch",
				CommentEN: "Breakfast preparation and cleanup",
			},
			{
				TypeID:    typeID,
				StartTime: currentDate.At(11, 30),
				Duration:  180, // 3 hours
				Required:  3,
				CommentDE: "Mittagessen vorbereiten und aufräumen",
				CommentEN: "Lunch preparation and cleanup",
			},
			{
				TypeID:    typeID,
				StartTime: currentDate.At(17, 0),
				Duration:  180, // 3 hours
				Required:  3,
				CommentDE: "Abendessen vorbereiten und aufräumen",
				CommentEN: "Dinner preparation and cleanup",
			},
			{
				TypeID:    typeID,
				StartTime: currentDate.At(20, 0),
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
				shift.StartTime,
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

// renderMarkdown converts markdown text to HTML
func renderMarkdown(text string) string {
	// Create markdown parser with common extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	// Create HTML renderer with safe options
	opts := html.RendererOptions{
		Flags: html.CommonFlags | html.HrefTargetBlank | html.SkipHTML,
		// Add rel="nofollow" to links
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			if link, ok := node.(*ast.Link); ok && entering {
				io.WriteString(w, "<a href=\"")
				html.EscapeHTML(w, link.Destination)
				io.WriteString(w, "\" rel=\"nofollow\" target=\"_blank\">")
				return ast.GoToNext, true
			}
			return ast.GoToNext, false
		},
	}
	renderer := html.NewRenderer(opts)

	// Parse and render
	md := p.Parse([]byte(text))
	return string(markdown.Render(md, renderer))
}

// GetShiftPlanByToken retrieves a shift plan by its token
func (db *DB) GetShiftPlanByToken(token string) (*ShiftPlan, error) {
	start := time.Now()
	defer func() {
		db.recordQuery("select", "shift_plans", start)
	}()

	var plan ShiftPlan
	var createdAt string

	query := `
        SELECT id, token, admin_password, start_date, end_date, created_at,
               is_multilingual, title_de, title_en, description_de, description_en,
               day_boundary_time, collaborative_mode, time_zone
        FROM shift_plans
        WHERE token = ?
    `
	err := db.QueryRow(query, token).Scan(
		&plan.ID,
		&plan.Token,
		&plan.AdminPassword,
		&plan.StartDate,
		&plan.EndDate,
		&createdAt,
		&plan.IsMultilingual,
		&plan.TitleDE,
		&plan.TitleEN,
		&plan.DescriptionDE,
		&plan.DescriptionEN,
		&plan.DayBoundaryTime,
		&plan.CollaborativeMode,
		&plan.TimeZone,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.logError("GetShiftPlanByToken", err, token)
		db.recordError("select", err)
		return nil, fmt.Errorf("failed to get shift plan: %w", err)
	}

	if plan.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("shift_plans id %d: invalid created_at %q: %w", plan.ID, createdAt, err)
	}

	// Render markdown to HTML
	if plan.DescriptionEN != "" {
		html := renderMarkdown(plan.DescriptionEN)
		plan.DescriptionENHTML = &html
	}
	if plan.DescriptionDE != nil && *plan.DescriptionDE != "" {
		html := renderMarkdown(*plan.DescriptionDE)
		plan.DescriptionDEHTML = &html
	}

	return &plan, nil
}

// getShiftPlanByID retrieves a shift plan by ID within a transaction
func (db *DB) getShiftPlanByID(tx *sql.Tx, id int64) (ShiftPlan, error) {
	var plan ShiftPlan
	var createdAt string

	query := `
        SELECT id, token, admin_password, start_date, end_date, created_at,
               is_multilingual, title_de, title_en, description_de, description_en,
               day_boundary_time, collaborative_mode, time_zone
        FROM shift_plans
        WHERE id = ?
    `
	err := tx.QueryRow(query, id).Scan(
		&plan.ID,
		&plan.Token,
		&plan.AdminPassword,
		&plan.StartDate,
		&plan.EndDate,
		&createdAt,
		&plan.IsMultilingual,
		&plan.TitleDE,
		&plan.TitleEN,
		&plan.DescriptionDE,
		&plan.DescriptionEN,
		&plan.DayBoundaryTime,
		&plan.CollaborativeMode,
		&plan.TimeZone,
	)
	if err != nil {
		return plan, fmt.Errorf("failed to scan shift plan: %w", err)
	}

	if plan.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return plan, fmt.Errorf("shift_plans id %d: invalid created_at %q: %w", plan.ID, createdAt, err)
	}

	return plan, nil
}

// getShiftPlanByToken retrieves a shift plan by token within a transaction
func (db *DB) getShiftPlanByToken(tx *sql.Tx, token string) (ShiftPlan, error) {
	var plan ShiftPlan
	var createdAt string

	query := `
        SELECT id, token, admin_password, start_date, end_date, created_at,
               is_multilingual, title_de, title_en, description_de, description_en,
               day_boundary_time, collaborative_mode, time_zone
        FROM shift_plans
        WHERE token = ?
    `
	err := tx.QueryRow(query, token).Scan(
		&plan.ID,
		&plan.Token,
		&plan.AdminPassword,
		&plan.StartDate,
		&plan.EndDate,
		&createdAt,
		&plan.IsMultilingual,
		&plan.TitleDE,
		&plan.TitleEN,
		&plan.DescriptionDE,
		&plan.DescriptionEN,
		&plan.DayBoundaryTime,
		&plan.CollaborativeMode,
		&plan.TimeZone,
	)
	if err != nil {
		return plan, fmt.Errorf("failed to scan shift plan: %w", err)
	}

	if plan.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return plan, fmt.Errorf("shift_plans id %d: invalid created_at %q: %w", plan.ID, createdAt, err)
	}

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
	var result ShiftPlan

	// Validate required fields
	if plan.TitleEN == "" {
		return nil, fmt.Errorf("title_en is required")
	}
	// Beschreibung ist jetzt optional
	// if plan.DescriptionEN == "" {
	// 	return nil, fmt.Errorf("description_en is required")
	// }

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
                description_en = ?,
                day_boundary_time = ?,
                collaborative_mode = ?
            WHERE token = ?
        `
		res, err := tx.Exec(query,
			plan.StartDate,
			plan.EndDate,
			plan.IsMultilingual,
			plan.TitleDE,
			plan.TitleEN,
			plan.DescriptionDE,
			plan.DescriptionEN,
			plan.DayBoundaryTime,
			plan.CollaborativeMode,
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

		// Get the updated plan within the transaction to avoid deadlock
		result, err = db.getShiftPlanByToken(tx, token)
		if err != nil {
			return fmt.Errorf("failed to load updated shift plan: %w", err)
		}

		return nil
	})

	if err != nil {
		db.logError("UpdateShiftPlan", err, plan)
		return nil, err
	}

	return &result, nil
}

// DeleteShiftPlan deletes a shift plan and all its related data (shifts, types, participants)
func (db *DB) DeleteShiftPlan(token string, hashedPassword string) error {
	start := time.Now()
	defer func() {
		db.recordQuery("delete", "shift_plans", start)
	}()

	err := db.transaction(func(tx *sql.Tx) error {
		// First verify the plan exists and password matches
		var planID int64
		var storedHash string
		err := tx.QueryRow("SELECT id, admin_password FROM shift_plans WHERE token = ?", token).Scan(&planID, &storedHash)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("failed to get shift plan: %w", err)
		}

		// Verify password
		if hashedPassword != storedHash {
			return ErrInvalidPassword
		}

		// Delete all participants first (no foreign key constraints)
		if _, err := tx.Exec(`DELETE FROM participants WHERE shift_id IN (
			SELECT s.id FROM shifts s
			JOIN shift_types st ON s.type_id = st.id
			WHERE st.plan_id = ?
		)`, planID); err != nil {
			return fmt.Errorf("failed to delete participants: %w", err)
		}

		// Delete all shifts
		if _, err := tx.Exec(`DELETE FROM shifts WHERE type_id IN (
			SELECT id FROM shift_types WHERE plan_id = ?
		)`, planID); err != nil {
			return fmt.Errorf("failed to delete shifts: %w", err)
		}

		// Delete all shift types
		if _, err := tx.Exec("DELETE FROM shift_types WHERE plan_id = ?", planID); err != nil {
			return fmt.Errorf("failed to delete shift types: %w", err)
		}

		// Finally delete the plan itself
		if _, err := tx.Exec("DELETE FROM shift_plans WHERE id = ?", planID); err != nil {
			return fmt.Errorf("failed to delete shift plan: %w", err)
		}

		return nil
	})

	if err != nil {
		db.recordError("delete", err)
	}

	return err
}
