package main

import (
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./schipla.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create tables if they don't exist
	createTables()

	// Insert example data if database is empty
	if isEmpty() {
		insertExampleData()
	}
}

func createTables() {
	// Create events table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			start_date DATETIME NOT NULL,
			end_date DATETIME NOT NULL,
			admin_password TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			day_zero TEXT,
			debug_mode INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create shift_types table with event reference
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shift_types (
			id TEXT PRIMARY KEY,
			event_id TEXT NOT NULL,
			title_de TEXT NOT NULL,
			title_en TEXT NOT NULL,
			description_de TEXT,
			description_en TEXT,
			location TEXT,
			password TEXT,
			sort_order INTEGER DEFAULT 0,
			FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create shifts table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shifts (
			id TEXT PRIMARY KEY,
			type_id TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			duration INTEGER NOT NULL,
			required INTEGER NOT NULL,
			FOREIGN KEY (type_id) REFERENCES shift_types(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Create participants table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS participants (
			shift_id TEXT NOT NULL,
			name TEXT NOT NULL,
			PRIMARY KEY (shift_id, name),
			FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func isEmpty() bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM shift_types").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	return count == 0
}

func insertExampleData() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Insert example shift type
	_, err := db.Exec(`
		INSERT INTO shift_types (id, event_id, title_de, title_en, description_de, description_en, location)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "example", defaultEventID, "Getränkestand", "Beverage Stand", "Getränke ausschenken und Kasse verwalten", "Serve drinks and manage the cash register", "Hauptbühne")
	if err != nil {
		log.Fatal(err)
	}

	// Insert example shifts
	shifts := []struct {
		id        string
		startTime time.Time
		duration  int
		required  int
	}{
		{"shift1", today.Add(18 * time.Hour), 120, 2},
		{"shift2", today.Add(20 * time.Hour), 120, 2},
		{"shift3", today.Add(22 * time.Hour), 120, 3},
		{"shift4", today.Add(24 * time.Hour).Add(14 * time.Hour), 180, 2},
		{"shift5", today.Add(24 * time.Hour).Add(17 * time.Hour), 180, 2},
	}

	for _, shift := range shifts {
		_, err := db.Exec(`
			INSERT INTO shifts (id, type_id, start_time, duration, required)
			VALUES (?, ?, ?, ?, ?)
		`, shift.id, "example", shift.startTime, shift.duration, shift.required)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Insert example participants
	participants := []struct {
		shiftID string
		name    string
	}{
		{"shift1", "Anna"},
		{"shift1", "Bob"},
		{"shift2", "Charlie"},
		{"shift4", "David"},
		{"shift4", "Eva"},
		{"shift5", "Frank"},
	}

	for _, p := range participants {
		_, err := db.Exec(`
			INSERT INTO participants (shift_id, name)
			VALUES (?, ?)
		`, p.shiftID, p.name)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Legacy function - will be removed after migration
func getAllShiftTypes() (map[string]ShiftType, error) {
	rows, err := db.Query(`
		SELECT 
			st.id,
			st.title_de,
			st.title_en,
			st.description_de,
			st.description_en,
			st.location,
			st.password,
			st.sort_order,
			s.id,
			s.start_time,
			s.duration,
			s.required,
			GROUP_CONCAT(p.name)
		FROM shift_types st
		LEFT JOIN shifts s ON s.type_id = st.id
		LEFT JOIN participants p ON p.shift_id = s.id
		GROUP BY st.id, s.id
		ORDER BY st.sort_order, st.title_de, s.start_time
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shiftTypes := make(map[string]ShiftType)
	currentType := ""
	var currentShifts []Shift

	for rows.Next() {
		var typeID, titleDE, titleEN, descriptionDE, descriptionEN, location string
		var password *string
		var sortOrder *int
		var shiftID *string
		var startTime *time.Time
		var duration, required *int
		var participants *string

		err := rows.Scan(&typeID, &titleDE, &titleEN, &descriptionDE, &descriptionEN, &location, &password, &sortOrder, &shiftID, &startTime, &duration, &required, &participants)
		if err != nil {
			return nil, err
		}

		// If we're starting a new type or this is the first row
		if typeID != currentType {
			// If this isn't the first type, save the previous one
			if currentType != "" {
				st := shiftTypes[currentType]
				st.Shifts = currentShifts
				shiftTypes[currentType] = st
			}

			// Start a new type
			currentType = typeID
			currentShifts = []Shift{}
			shiftTypes[typeID] = ShiftType{
				ID:            typeID,
				TitleDE:       titleDE,
				TitleEN:       titleEN,
				DescriptionDE: descriptionDE,
				DescriptionEN: descriptionEN,
				Location:      location,
				Password:      stringOrEmpty(password),
				SortOrder:     *sortOrder,
			}
		}

		// Add shift if it exists
		if shiftID != nil {
			participantList := []string{}
			if participants != nil && *participants != "" {
				participantList = strings.Split(*participants, ",")
			}

			currentShifts = append(currentShifts, Shift{
				ID:           *shiftID,
				StartTime:    *startTime,
				Duration:     *duration,
				Required:     *required,
				Participants: participantList,
			})
		}
	}

	// Don't forget to save the last type's shifts
	if currentType != "" {
		st := shiftTypes[currentType]
		st.Shifts = currentShifts
		shiftTypes[currentType] = st
	}

	return shiftTypes, nil
}

// New function to get shift types for a specific event
func getShiftTypes(eventID string) (map[string]ShiftType, error) {
	rows, err := db.Query(`
		SELECT 
			st.id,
			st.event_id,
			st.title_de,
			st.title_en,
			st.description_de,
			st.description_en,
			st.location,
			st.password,
			st.sort_order,
			s.id,
			s.start_time,
			s.duration,
			s.required,
			GROUP_CONCAT(p.name)
		FROM shift_types st
		LEFT JOIN shifts s ON s.type_id = st.id
		LEFT JOIN participants p ON p.shift_id = s.id
		WHERE st.event_id = ?
		GROUP BY st.id, s.id
		ORDER BY st.sort_order, st.title_de, s.start_time
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shiftTypes := make(map[string]ShiftType)
	currentType := ""
	var currentShifts []Shift

	for rows.Next() {
		var typeID, titleDE, titleEN, descriptionDE, descriptionEN, location string
		var password *string
		var sortOrder *int
		var shiftID *string
		var startTime *time.Time
		var duration, required *int
		var participants *string

		err := rows.Scan(&typeID, &shiftID, &titleDE, &titleEN, &descriptionDE, &descriptionEN, &location, &password, &sortOrder, &shiftID, &startTime, &duration, &required, &participants)
		if err != nil {
			return nil, err
		}

		// If we're starting a new type or this is the first row
		if typeID != currentType {
			// If this isn't the first type, save the previous one
			if currentType != "" {
				st := shiftTypes[currentType]
				st.Shifts = currentShifts
				shiftTypes[currentType] = st
			}

			// Start a new type
			currentType = typeID
			currentShifts = []Shift{}
			shiftTypes[typeID] = ShiftType{
				ID:            typeID,
				TitleDE:       titleDE,
				TitleEN:       titleEN,
				DescriptionDE: descriptionDE,
				DescriptionEN: descriptionEN,
				Location:      location,
				Password:      stringOrEmpty(password),
				SortOrder:     *sortOrder,
			}
		}

		// Add shift if it exists
		if shiftID != nil {
			participantList := []string{}
			if participants != nil && *participants != "" {
				participantList = strings.Split(*participants, ",")
			}

			currentShifts = append(currentShifts, Shift{
				ID:           *shiftID,
				StartTime:    *startTime,
				Duration:     *duration,
				Required:     *required,
				Participants: participantList,
			})
		}
	}

	// Don't forget to save the last type's shifts
	if currentType != "" {
		st := shiftTypes[currentType]
		st.Shifts = currentShifts
		shiftTypes[currentType] = st
	}

	return shiftTypes, nil
}

func createShiftType(shiftType ShiftType) error {
	_, err := db.Exec(`
		INSERT INTO shift_types (id, event_id, title_de, title_en, description_de, description_en, location, password)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, shiftType.ID, shiftType.EventID, shiftType.TitleDE, shiftType.TitleEN, shiftType.DescriptionDE, shiftType.DescriptionEN, shiftType.Location, shiftType.Password)
	return err
}

func updateShiftType(id string, shiftType ShiftType) error {
	_, err := db.Exec(`
		UPDATE shift_types
		SET event_id = ?, title_de = ?, title_en = ?, description_de = ?, description_en = ?, location = ?, password = ?
		WHERE id = ?
	`, shiftType.EventID, shiftType.TitleDE, shiftType.TitleEN, shiftType.DescriptionDE, shiftType.DescriptionEN, shiftType.Location, shiftType.Password, id)
	return err
}

func deleteShiftType(id string) error {
	_, err := db.Exec("DELETE FROM shift_types WHERE id = ?", id)
	return err
}

func createShift(typeID string, shift Shift) error {
	_, err := db.Exec(`
		INSERT INTO shifts (id, type_id, start_time, duration, required)
		VALUES (?, ?, ?, ?, ?)
	`, shift.ID, typeID, shift.StartTime, shift.Duration, shift.Required)
	return err
}

func updateShift(typeID string, shift Shift) error {
	_, err := db.Exec(`
		UPDATE shifts
		SET start_time = ?, duration = ?, required = ?
		WHERE id = ? AND type_id = ?
	`, shift.StartTime, shift.Duration, shift.Required, shift.ID, typeID)
	return err
}

func deleteShift(typeID string, shiftID string) error {
	_, err := db.Exec("DELETE FROM shifts WHERE id = ? AND type_id = ?", shiftID, typeID)
	return err
}

func joinShift(typeID string, shiftID string, name string) error {
	_, err := db.Exec(`
		INSERT INTO participants (shift_id, name)
		VALUES (?, ?)
	`, shiftID, name)
	return err
}

func leaveShift(typeID string, shiftID string, name string) error {
	_, err := db.Exec(`
		DELETE FROM participants
		WHERE shift_id = ? AND name = ?
	`, shiftID, name)
	return err
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func updateShiftTypeOrder(id string, sortOrder int) error {
	_, err := db.Exec(`
		UPDATE shift_types
		SET sort_order = ?
		WHERE id = ?
	`, sortOrder, id)
	return err
}

// New function to create an event
func createEvent(event Event) error {
	_, err := db.Exec(`
		INSERT INTO events (id, title, start_date, end_date, admin_password, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID, event.Title, event.StartDate, event.EndDate, event.AdminPassword, time.Now())
	return err
}

// New function to get event by ID
func getEvent(id string) (*Event, error) {
	var event Event
	err := db.QueryRow(`
		SELECT id, title, start_date, end_date, created_at
		FROM events
		WHERE id = ?
	`, id).Scan(&event.ID, &event.Title, &event.StartDate, &event.EndDate, &event.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// New function to get event configuration
func getEventConfig(id string) (*EventConfig, error) {
	var config EventConfig
	err := db.QueryRow(`
		SELECT day_zero, debug_mode
		FROM events
		WHERE id = ?
	`, id).Scan(&config.DayZero, &config.DebugMode)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// New function to get all events
func getAllEvents() ([]Event, error) {
	rows, err := db.Query(`
		SELECT id, title, start_date, end_date, created_at
		FROM events
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.ID, &event.Title, &event.StartDate, &event.EndDate, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}
