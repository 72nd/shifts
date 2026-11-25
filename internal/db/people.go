package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/72nd/shifts/internal/model"
	"github.com/72nd/shifts/internal/util"
)

// PlanPerson is the stored data of a person in a plan (table plan_people).
// A person without a row still exists as long as they have participations.
type PlanPerson struct {
	ID              int64
	PlanID          int64
	NameKey         string
	DisplayName     string
	AvailableFrom   *model.DateTime
	AvailableUntil  *model.DateTime
	TargetMinutes   *int
	PrefersLongDays bool
	Values          map[string]any // field ID -> value; choice = option ID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PlanPersonWrite is the complete data of an admin upsert (replaces everything but the key).
type PlanPersonWrite struct {
	Name            string
	AvailableFrom   *model.DateTime
	AvailableUntil  *model.DateTime
	TargetMinutes   *int
	PrefersLongDays bool
	Values          map[string]any
}

// PersonInput is the partial update a participant can send when signing up.
// Nil attendance fields are left unchanged; a nil value in Values removes that field's value.
type PersonInput struct {
	AvailableFrom  *model.DateTime
	AvailableUntil *model.DateTime
	Values         map[string]any
}

const personColumns = `id, plan_id, name_key, display_name, available_from, available_until,
	target_minutes, prefers_long_days, field_values, created_at, updated_at`

func scanPerson(row rowScanner) (PlanPerson, error) {
	var p PlanPerson
	var from, until sql.NullString
	var target sql.NullInt64
	var values, createdAt, updatedAt string
	if err := row.Scan(&p.ID, &p.PlanID, &p.NameKey, &p.DisplayName, &from, &until,
		&target, &p.PrefersLongDays, &values, &createdAt, &updatedAt); err != nil {
		return p, err
	}
	var err error
	if from.Valid {
		t, err := model.ParseDateTime(from.String)
		if err != nil {
			return p, fmt.Errorf("plan_people id %d: invalid available_from %q: %w", p.ID, from.String, err)
		}
		p.AvailableFrom = &t
	}
	if until.Valid {
		t, err := model.ParseDateTime(until.String)
		if err != nil {
			return p, fmt.Errorf("plan_people id %d: invalid available_until %q: %w", p.ID, until.String, err)
		}
		p.AvailableUntil = &t
	}
	p.TargetMinutes = nullIntPtr(target)
	p.Values = map[string]any{}
	if err := json.Unmarshal([]byte(values), &p.Values); err != nil {
		return p, fmt.Errorf("plan_people id %d: invalid field_values %q: %w", p.ID, values, err)
	}
	if p.CreatedAt, err = parseTimestamp("plan_people", p.ID, "created_at", createdAt); err != nil {
		return p, err
	}
	if p.UpdatedAt, err = parseTimestamp("plan_people", p.ID, "updated_at", updatedAt); err != nil {
		return p, err
	}
	return p, nil
}

func nullableDateTime(d *model.DateTime) any {
	if d == nil {
		return nil
	}
	return d.String()
}

// pruneValues drops values of fields that no longer exist in the plan.
func pruneValues(tx *sql.Tx, planID int64, values map[string]any) (map[string]any, error) {
	fields, err := getPlanFields(tx, planID)
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for _, f := range fields {
		ids[fmt.Sprint(f.ID)] = true
	}
	out := map[string]any{}
	for k, v := range values {
		if ids[k] && v != nil {
			out[k] = v
		}
	}
	return out, nil
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

// getPersonByKey loads the person row, or nil if there is none.
func getPersonByKey(q queryer, planID int64, key string) (*PlanPerson, error) {
	p, err := scanPerson(q.QueryRow(`SELECT `+personColumns+` FROM plan_people WHERE plan_id = ? AND name_key = ?`, planID, key))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get person: %w", err)
	}
	return &p, nil
}

// GetPersonByName loads the person row for a name, or nil if there is none.
func (db *DB) GetPersonByName(planID int64, name string) (*PlanPerson, error) {
	return getPersonByKey(db.rdb, planID, util.PersonKey(name))
}

// ListPeople returns all person rows of a plan, ordered by name.
func (db *DB) ListPeople(planID int64) ([]PlanPerson, error) {
	rows, err := db.rdb.Query(`SELECT `+personColumns+` FROM plan_people WHERE plan_id = ? ORDER BY name_key`, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to query people: %w", err)
	}
	defer rows.Close()
	people := []PlanPerson{}
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan person: %w", err)
		}
		people = append(people, p)
	}
	return people, rows.Err()
}

// upsertPersonInput applies a participant's partial input to the person's row (creating it if needed).
func upsertPersonInput(tx *sql.Tx, planID int64, name string, in *PersonInput) (*PlanPerson, error) {
	key := util.PersonKey(name)
	p, err := getPersonByKey(tx, planID, key)
	if err != nil {
		return nil, err
	}
	now := nowUTC()
	if p == nil {
		res, err := tx.Exec(`INSERT INTO plan_people (plan_id, name_key, display_name, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?)`, planID, key, name, now, now)
		if err != nil {
			return nil, fmt.Errorf("failed to insert person: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
		np, err := scanPerson(tx.QueryRow(`SELECT `+personColumns+` FROM plan_people WHERE id = ?`, id))
		if err != nil {
			return nil, fmt.Errorf("failed to load person: %w", err)
		}
		p = &np
	}
	if in.AvailableFrom != nil {
		p.AvailableFrom = in.AvailableFrom
	}
	if in.AvailableUntil != nil {
		p.AvailableUntil = in.AvailableUntil
	}
	for k, v := range in.Values {
		if v == nil {
			delete(p.Values, k)
		} else {
			p.Values[k] = v
		}
	}
	values, err := pruneValues(tx, planID, p.Values)
	if err != nil {
		return nil, err
	}
	p.Values = values
	valuesJSON, err := json.Marshal(p.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to encode field values: %w", err)
	}
	if _, err := tx.Exec(`UPDATE plan_people SET available_from = ?, available_until = ?, field_values = ?, updated_at = ?
        WHERE id = ?`, nullableDateTime(p.AvailableFrom), nullableDateTime(p.AvailableUntil), string(valuesJSON), now, p.ID); err != nil {
		return nil, fmt.Errorf("failed to update person: %w", err)
	}
	return p, nil
}

// UpsertPerson creates or completely replaces the data of a person (admin).
func (db *DB) UpsertPerson(planID int64, in PlanPersonWrite) (*PlanPerson, error) {
	var result PlanPerson
	err := db.transaction(func(tx *sql.Tx) error {
		key := util.PersonKey(in.Name)
		values, err := pruneValues(tx, planID, in.Values)
		if err != nil {
			return err
		}
		valuesJSON, err := json.Marshal(values)
		if err != nil {
			return fmt.Errorf("failed to encode field values: %w", err)
		}
		now := nowUTC()
		if _, err := tx.Exec(`
            INSERT INTO plan_people (plan_id, name_key, display_name, available_from, available_until,
                target_minutes, prefers_long_days, field_values, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT (plan_id, name_key) DO UPDATE SET
                display_name = excluded.display_name,
                available_from = excluded.available_from,
                available_until = excluded.available_until,
                target_minutes = excluded.target_minutes,
                prefers_long_days = excluded.prefers_long_days,
                field_values = excluded.field_values,
                updated_at = excluded.updated_at
        `, planID, key, in.Name, nullableDateTime(in.AvailableFrom), nullableDateTime(in.AvailableUntil),
			in.TargetMinutes, in.PrefersLongDays, string(valuesJSON), now, now); err != nil {
			return fmt.Errorf("failed to upsert person: %w", err)
		}
		p, err := getPersonByKey(tx, planID, key)
		if err != nil {
			return err
		}
		result = *p
		return nil
	})
	if err != nil {
		db.logError("UpsertPerson", err, planID, in.Name)
		return nil, err
	}
	return &result, nil
}

// DeletePerson deletes only the person row, never participations.
func (db *DB) DeletePerson(planID int64, name string) error {
	res, err := db.Exec(`DELETE FROM plan_people WHERE plan_id = ? AND name_key = ?`, planID, util.PersonKey(name))
	if err != nil {
		db.logError("DeletePerson", err, planID, name)
		return fmt.Errorf("failed to delete person: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if n == 0 {
		return ErrPersonNotFound
	}
	return nil
}

// planParticipantKeys returns the set of person keys that have participations in the plan.
func planParticipantKeys(tx *sql.Tx, planID int64) (map[string]bool, error) {
	rows, err := tx.Query(`
        SELECT DISTINCT p.name FROM participants p
        JOIN shifts s ON p.shift_id = s.id
        JOIN shift_types st ON s.type_id = st.id
        WHERE st.plan_id = ?`, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participant names: %w", err)
	}
	defer rows.Close()
	keys := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan participant name: %w", err)
		}
		keys[util.PersonKey(name)] = true
	}
	return keys, rows.Err()
}

// ImportPeople creates person rows for names that do not exist yet (neither as row nor
// through participations). Names must be validated. Returns the counts created and skipped.
func (db *DB) ImportPeople(planID int64, names []string) (created, skipped int, err error) {
	err = db.transaction(func(tx *sql.Tx) error {
		known, err := planParticipantKeys(tx, planID)
		if err != nil {
			return err
		}
		rows, err := tx.Query(`SELECT name_key FROM plan_people WHERE plan_id = ?`, planID)
		if err != nil {
			return fmt.Errorf("failed to query people: %w", err)
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan person key: %w", err)
			}
			known[key] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		now := nowUTC()
		for _, name := range names {
			key := util.PersonKey(name)
			if known[key] {
				skipped++
				continue
			}
			if _, err := tx.Exec(`INSERT INTO plan_people (plan_id, name_key, display_name, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?)`, planID, key, name, now, now); err != nil {
				return fmt.Errorf("failed to insert person: %w", err)
			}
			known[key] = true
			created++
		}
		return nil
	})
	if err != nil {
		db.logError("ImportPeople", err, planID)
		return 0, 0, err
	}
	return created, skipped, nil
}

// statusRank orders participation statuses for merging: the stronger one wins.
func statusRank(status string) int {
	switch status {
	case StatusAssigned:
		return 3
	case StatusSignedUp:
		return 2
	default:
		return 1
	}
}

// RenamePerson renames a person in all participations of the plan and in the person row.
// If the target name belongs to another person, ErrPersonExists is returned unless merge is set;
// then the data are merged (target wins, empty values are filled from the source) and
// duplicate participations in one shift become one (strongest status wins).
func (db *DB) RenamePerson(planID int64, from, to string, merge bool) error {
	fromKey, toKey := util.PersonKey(from), util.PersonKey(to)
	err := db.transaction(func(tx *sql.Tx) error {
		type part struct {
			id      int64
			shiftID int64
			name    string
			status  string
		}
		rows, err := tx.Query(`
            SELECT p.id, p.shift_id, p.name, p.status FROM participants p
            JOIN shifts s ON p.shift_id = s.id
            JOIN shift_types st ON s.type_id = st.id
            WHERE st.plan_id = ? ORDER BY p.created_at, p.id`, planID)
		if err != nil {
			return fmt.Errorf("failed to query participants: %w", err)
		}
		var fromParts, toParts []part
		for rows.Next() {
			var p part
			if err := rows.Scan(&p.id, &p.shiftID, &p.name, &p.status); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan participant: %w", err)
			}
			switch util.PersonKey(p.name) {
			case fromKey:
				fromParts = append(fromParts, p)
			case toKey:
				if toKey != fromKey {
					toParts = append(toParts, p)
				}
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		fromRow, err := getPersonByKey(tx, planID, fromKey)
		if err != nil {
			return err
		}
		if len(fromParts) == 0 && fromRow == nil {
			return ErrPersonNotFound
		}
		var toRow *PlanPerson
		if toKey != fromKey {
			if toRow, err = getPersonByKey(tx, planID, toKey); err != nil {
				return err
			}
			if (len(toParts) > 0 || toRow != nil) && !merge {
				return ErrPersonExists
			}
		}

		now := nowUTC()
		// Duplicates: both names have a participation in the same shift
		toByShift := map[int64]part{}
		for _, p := range toParts {
			toByShift[p.shiftID] = p
		}
		for _, p := range fromParts {
			target, dup := toByShift[p.shiftID]
			if dup && statusRank(p.status) <= statusRank(target.status) {
				// keep the target participation, drop the source one
				if _, err := tx.Exec(`DELETE FROM participants WHERE id = ?`, p.id); err != nil {
					return fmt.Errorf("failed to delete duplicate participation: %w", err)
				}
				continue
			}
			if dup {
				// the source is stronger: drop the target and rename the source
				if _, err := tx.Exec(`DELETE FROM participants WHERE id = ?`, target.id); err != nil {
					return fmt.Errorf("failed to delete duplicate participation: %w", err)
				}
			}
			if _, err := tx.Exec(`UPDATE participants SET name = ? WHERE id = ?`, to, p.id); err != nil {
				return fmt.Errorf("failed to rename participation: %w", err)
			}
		}
		// Person rows
		switch {
		case fromRow != nil && toRow == nil:
			if _, err := tx.Exec(`UPDATE plan_people SET name_key = ?, display_name = ?, updated_at = ? WHERE id = ?`,
				toKey, to, now, fromRow.ID); err != nil {
				return fmt.Errorf("failed to rename person: %w", err)
			}
		case fromRow != nil && toRow != nil:
			merged := mergePeople(*toRow, *fromRow)
			valuesJSON, err := json.Marshal(merged.Values)
			if err != nil {
				return fmt.Errorf("failed to encode field values: %w", err)
			}
			if _, err := tx.Exec(`UPDATE plan_people SET display_name = ?, available_from = ?, available_until = ?,
                target_minutes = ?, prefers_long_days = ?, field_values = ?, updated_at = ? WHERE id = ?`,
				to, nullableDateTime(merged.AvailableFrom), nullableDateTime(merged.AvailableUntil),
				merged.TargetMinutes, merged.PrefersLongDays, string(valuesJSON), now, toRow.ID); err != nil {
				return fmt.Errorf("failed to merge person: %w", err)
			}
			if _, err := tx.Exec(`DELETE FROM plan_people WHERE id = ?`, fromRow.ID); err != nil {
				return fmt.Errorf("failed to delete merged person: %w", err)
			}
		case toRow != nil:
			if _, err := tx.Exec(`UPDATE plan_people SET display_name = ?, updated_at = ? WHERE id = ?`, to, now, toRow.ID); err != nil {
				return fmt.Errorf("failed to rename person: %w", err)
			}
		}
		// Target participations keep working under the new spelling
		for _, p := range toParts {
			if _, err := tx.Exec(`UPDATE participants SET name = ? WHERE id = ?`, to, p.id); err != nil {
				return fmt.Errorf("failed to rename participation: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		db.logError("RenamePerson", err, planID, from, to, merge)
	}
	return err
}

// mergePeople merges src into dst: dst wins, empty values are filled from src.
func mergePeople(dst, src PlanPerson) PlanPerson {
	if dst.AvailableFrom == nil {
		dst.AvailableFrom = src.AvailableFrom
	}
	if dst.AvailableUntil == nil {
		dst.AvailableUntil = src.AvailableUntil
	}
	if dst.TargetMinutes == nil {
		dst.TargetMinutes = src.TargetMinutes
	}
	dst.PrefersLongDays = dst.PrefersLongDays || src.PrefersLongDays
	if dst.Values == nil {
		dst.Values = map[string]any{}
	}
	for k, v := range src.Values {
		if _, ok := dst.Values[k]; !ok {
			dst.Values[k] = v
		}
	}
	return dst
}
