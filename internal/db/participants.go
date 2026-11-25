package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/72nd/shifts/internal/util"
)

// Participation status values (participants.status).
const (
	StatusSignedUp = "signed_up"
	StatusAssigned = "assigned"
	StatusDeclined = "declined"
)

// Participant represents a person assigned to a shift
type Participant struct {
	ID              int64      `json:"id"`
	ShiftID         int64      `json:"shift_id"`
	Name            string     `json:"name"`
	CreatedAt       time.Time  `json:"created_at"`
	Status          string     `json:"status"`
	StatusChangedAt *time.Time `json:"status_changed_at,omitempty"`
	AddedByAdmin    bool       `json:"added_by_admin"`
}

// participantColumns are the participants columns in the order scanParticipant expects.
// Prefix them with a table alias where needed (see participantColumnsAs).
const participantColumns = "id, shift_id, name, created_at, status, status_changed_at, added_by_admin"

// participantColumnsAs returns participantColumns qualified with a table alias.
func participantColumnsAs(alias string) string {
	return alias + ".id, " + alias + ".shift_id, " + alias + ".name, " + alias + ".created_at, " +
		alias + ".status, " + alias + ".status_changed_at, " + alias + ".added_by_admin"
}

// rowScanner is satisfied by *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanParticipant reads a row selected with participantColumns.
func scanParticipant(row rowScanner) (Participant, error) {
	var p Participant
	var createdAt string
	var changedAt sql.NullString
	if err := row.Scan(&p.ID, &p.ShiftID, &p.Name, &createdAt, &p.Status, &changedAt, &p.AddedByAdmin); err != nil {
		return p, err
	}
	var err error
	if p.CreatedAt, err = parseTimestamp("participants", p.ID, "created_at", createdAt); err != nil {
		return p, err
	}
	if changedAt.Valid {
		t, err := parseTimestamp("participants", p.ID, "status_changed_at", changedAt.String)
		if err != nil {
			return p, err
		}
		p.StatusChangedAt = &t
	}
	return p, nil
}

// nullableParticipant holds the participant columns of a LEFT JOIN row (all nullable).
type nullableParticipant struct {
	ID              sql.NullInt64
	Name            sql.NullString
	CreatedAt       sql.NullString
	Status          sql.NullString
	StatusChangedAt sql.NullString
	AddedByAdmin    sql.NullBool
}

// toParticipant converts a joined row into a Participant for the given shift.
func (n nullableParticipant) toParticipant(shiftID int64) (Participant, error) {
	p := Participant{
		ID:           n.ID.Int64,
		ShiftID:      shiftID,
		Name:         n.Name.String,
		Status:       n.Status.String,
		AddedByAdmin: n.AddedByAdmin.Bool,
	}
	var err error
	if p.CreatedAt, err = parseTimestamp("participants", p.ID, "created_at", n.CreatedAt.String); err != nil {
		return p, err
	}
	if n.StatusChangedAt.Valid {
		t, err := parseTimestamp("participants", p.ID, "status_changed_at", n.StatusChangedAt.String)
		if err != nil {
			return p, err
		}
		p.StatusChangedAt = &t
	}
	return p, nil
}

// participantJoinColumns are the p.* columns appended to the shift JOIN queries
// (the shift id is known from the shift columns).
const participantJoinColumns = "p.id, p.name, p.created_at, p.status, p.status_changed_at, p.added_by_admin"

// scanDest returns the Scan destinations for participantJoinColumns.
func (n *nullableParticipant) scanDest() []any {
	return []any{&n.ID, &n.Name, &n.CreatedAt, &n.Status, &n.StatusChangedAt, &n.AddedByAdmin}
}

// AddParticipant adds a participant to a shift.
// This uses an atomic INSERT with capacity check to prevent race conditions
// where two users could simultaneously take the last available spot.
// Declined participations do not count against the capacity. If person is not
// nil, the person's row in plan_people is created or updated in the same
// transaction.
func (db *DB) AddParticipant(shiftID int64, name string, person *PersonInput) (*Participant, error) {
	start := time.Now()
	defer func() {
		db.recordQuery("insert", "participants", start)
	}()

	var result Participant

	// Use IMMEDIATE transaction to acquire write lock at the start,
	// preventing race conditions in the check-then-insert pattern
	err := db.transactionImmediate(func(tx *sql.Tx) error {
		createdAt := time.Now().UTC().Format(time.RFC3339)

		// A person who was declined for this shift cannot sign up again; only the
		// coordination can change that. Compared by normalised name.
		existing, err := db.getParticipantsByShift(tx, shiftID)
		if err != nil {
			return err
		}
		key := util.PersonKey(name)
		for _, e := range existing {
			if e.Status == StatusDeclined && util.PersonKey(e.Name) == key {
				return ErrParticipantDeclined
			}
		}

		// Atomic INSERT with all checks in a single query.
		// This prevents TOCTOU race conditions by making the check and insert atomic.
		// The INSERT only succeeds if:
		// 1. No participant with same name exists for this shift
		// 2. Either collaborative_mode is enabled, OR required is -1 (unlimited), OR current count < required
		//    (declined participations do not count)
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
                    WHEN (SELECT COUNT(*) FROM participants WHERE shift_id = ? AND status <> 'declined') < s.required THEN 1
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

		if person != nil {
			planID, err := planIDOfShift(tx, shiftID)
			if err != nil {
				return err
			}
			if _, err := upsertPersonInput(tx, planID, name, person); err != nil {
				return err
			}
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
        SELECT ` + participantColumns + `
        FROM participants
        WHERE shift_id = ?
        ORDER BY created_at, id
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
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
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
	p, err := scanParticipant(tx.QueryRow(`SELECT `+participantColumns+` FROM participants WHERE id = ?`, id))
	if err != nil {
		return p, fmt.Errorf("failed to scan participant: %w", err)
	}
	return p, nil
}

// planIDOfShift returns the plan a shift belongs to.
func planIDOfShift(tx *sql.Tx, shiftID int64) (int64, error) {
	var planID int64
	err := tx.QueryRow(`SELECT st.plan_id FROM shifts s JOIN shift_types st ON s.type_id = st.id WHERE s.id = ?`, shiftID).Scan(&planID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("failed to find plan of shift: %w", err)
	}
	return planID, nil
}

// ActiveParticipants returns the participations that occupy a place: all but declined ones.
func ActiveParticipants(participants []Participant) []Participant {
	active := make([]Participant, 0, len(participants))
	for _, p := range participants {
		if p.Status != StatusDeclined {
			active = append(active, p)
		}
	}
	return active
}
