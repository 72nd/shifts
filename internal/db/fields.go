package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Field kinds and editors of plan_fields.
const (
	FieldChoice  = "choice"
	FieldNumber  = "number"
	FieldBoolean = "boolean"
	FieldText    = "text"

	EditableParticipant = "participant"
	EditableAdmin       = "admin"
)

// FieldOption is one option of a choice field. The position in the list is the level.
type FieldOption struct {
	ID string `json:"id"`
	DE string `json:"de"`
	EN string `json:"en"`
}

// PlanField is a custom person field of a plan.
type PlanField struct {
	ID         int64         `json:"id"`
	PlanID     int64         `json:"plan_id"`
	SortOrder  int           `json:"sort_order"`
	LabelDE    *string       `json:"label_de"`
	LabelEN    string        `json:"label_en"`
	Kind       string        `json:"kind"`
	Options    []FieldOption `json:"options"`
	EditableBy string        `json:"editable_by"`
	Required   bool          `json:"required"`
}

// NewPlanField contains the data to create or update a field. Kind is ignored on update.
type NewPlanField struct {
	LabelDE    *string
	LabelEN    string
	Kind       string
	Options    []FieldOption
	EditableBy string
	Required   bool
}

// queryer is satisfied by *sql.DB and *sql.Tx.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// newOptionID returns a short random option ID.
func newOptionID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// getPlanFields loads all fields of a plan ordered by sort_order.
func getPlanFields(q queryer, planID int64) ([]PlanField, error) {
	rows, err := q.Query(`
        SELECT id, plan_id, sort_order, label_de, label_en, kind, options, editable_by, required
        FROM plan_fields WHERE plan_id = ? ORDER BY sort_order, id
    `, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to query plan fields: %w", err)
	}
	defer rows.Close()

	fields := []PlanField{}
	for rows.Next() {
		var f PlanField
		var options string
		if err := rows.Scan(&f.ID, &f.PlanID, &f.SortOrder, &f.LabelDE, &f.LabelEN, &f.Kind, &options, &f.EditableBy, &f.Required); err != nil {
			return nil, fmt.Errorf("failed to scan plan field: %w", err)
		}
		if err := json.Unmarshal([]byte(options), &f.Options); err != nil {
			return nil, fmt.Errorf("plan_fields id %d: invalid options %q: %w", f.ID, options, err)
		}
		if f.Options == nil {
			f.Options = []FieldOption{}
		}
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

// GetPlanFields returns the person fields of a plan.
func (db *DB) GetPlanFields(planID int64) ([]PlanField, error) {
	return getPlanFields(db.rdb, planID)
}

// normalizeOptions gives new options an ID and keeps existing IDs from known.
// Options with an ID that is not in known (or duplicated) get a new one.
func normalizeOptions(options []FieldOption, known map[string]bool) []FieldOption {
	out := make([]FieldOption, len(options))
	seen := map[string]bool{}
	for i, o := range options {
		if o.ID == "" || !known[o.ID] || seen[o.ID] {
			o.ID = newOptionID()
		}
		seen[o.ID] = true
		out[i] = o
	}
	return out
}

func optionsJSON(options []FieldOption) (string, error) {
	if options == nil {
		options = []FieldOption{}
	}
	b, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("failed to encode options: %w", err)
	}
	return string(b), nil
}

// CreatePlanField appends a field to the plan.
func (db *DB) CreatePlanField(planID int64, f NewPlanField) (*PlanField, error) {
	var result PlanField
	err := db.transaction(func(tx *sql.Tx) error {
		var maxOrder int
		if err := tx.QueryRow(`SELECT COALESCE(MAX(sort_order), -1) FROM plan_fields WHERE plan_id = ?`, planID).Scan(&maxOrder); err != nil {
			return fmt.Errorf("failed to get max sort order: %w", err)
		}
		options := []FieldOption{}
		if f.Kind == FieldChoice {
			options = normalizeOptions(f.Options, nil)
		}
		optJSON, err := optionsJSON(options)
		if err != nil {
			return err
		}
		res, err := tx.Exec(`
            INSERT INTO plan_fields (plan_id, sort_order, label_de, label_en, kind, options, editable_by, required)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        `, planID, maxOrder+1, f.LabelDE, f.LabelEN, f.Kind, optJSON, f.EditableBy, f.Required)
		if err != nil {
			return fmt.Errorf("failed to insert plan field: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		fields, err := getPlanFields(tx, planID)
		if err != nil {
			return err
		}
		for _, x := range fields {
			if x.ID == id {
				result = x
			}
		}
		return nil
	})
	if err != nil {
		db.logError("CreatePlanField", err, planID, f)
		return nil, err
	}
	return &result, nil
}

// UpdatePlanField changes a field. The kind cannot be changed. Options that are
// removed from the experience field are cleared from exp_min_level of types and shifts.
func (db *DB) UpdatePlanField(planID, fieldID int64, f NewPlanField) (*PlanField, error) {
	var result PlanField
	err := db.transaction(func(tx *sql.Tx) error {
		fields, err := getPlanFields(tx, planID)
		if err != nil {
			return err
		}
		var current *PlanField
		for i := range fields {
			if fields[i].ID == fieldID {
				current = &fields[i]
			}
		}
		if current == nil {
			return ErrFieldNotFound
		}

		options := []FieldOption{}
		if current.Kind == FieldChoice {
			known := map[string]bool{}
			for _, o := range current.Options {
				known[o.ID] = true
			}
			options = normalizeOptions(f.Options, known)
		}
		optJSON, err := optionsJSON(options)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
            UPDATE plan_fields SET label_de = ?, label_en = ?, options = ?, editable_by = ?, required = ?
            WHERE id = ? AND plan_id = ?
        `, f.LabelDE, f.LabelEN, optJSON, f.EditableBy, f.Required, fieldID, planID); err != nil {
			return fmt.Errorf("failed to update plan field: %w", err)
		}

		// Clear experience levels that point to removed options
		var expField sql.NullInt64
		if err := tx.QueryRow(`SELECT experience_field_id FROM shift_plans WHERE id = ?`, planID).Scan(&expField); err != nil {
			return fmt.Errorf("failed to read experience field: %w", err)
		}
		if expField.Valid && expField.Int64 == fieldID {
			kept := map[string]bool{}
			for _, o := range options {
				kept[o.ID] = true
			}
			for _, o := range current.Options {
				if kept[o.ID] {
					continue
				}
				if err := clearExperienceLevel(tx, planID, o.ID); err != nil {
					return err
				}
			}
		}

		fields, err = getPlanFields(tx, planID)
		if err != nil {
			return err
		}
		for _, x := range fields {
			if x.ID == fieldID {
				result = x
			}
		}
		return nil
	})
	if err != nil {
		db.logError("UpdatePlanField", err, planID, fieldID, f)
		return nil, err
	}
	return &result, nil
}

// clearExperienceLevel removes the experience need (level and count) of all types
// and shifts of the plan that refer to the given option. An empty optionID clears all.
func clearExperienceLevel(tx *sql.Tx, planID int64, optionID string) error {
	cond := ""
	args := []any{planID}
	if optionID != "" {
		cond = " AND exp_min_level = ?"
		args = append(args, optionID)
	}
	if _, err := tx.Exec(`UPDATE shift_types SET exp_min_level = NULL, exp_min_count = 0
        WHERE plan_id = ?`+cond, args...); err != nil {
		return fmt.Errorf("failed to clear experience level of types: %w", err)
	}
	if _, err := tx.Exec(`UPDATE shifts SET exp_min_level = NULL, exp_min_count = NULL
        WHERE type_id IN (SELECT id FROM shift_types WHERE plan_id = ?)`+cond, args...); err != nil {
		return fmt.Errorf("failed to clear experience level of shifts: %w", err)
	}
	return nil
}

// DeletePlanField deletes a field. If it was the experience field, the experience
// setting and all experience needs of types and shifts are cleared in the same transaction.
// Values in plan_people.field_values stay orphaned and are dropped when the person is saved next.
func (db *DB) DeletePlanField(planID, fieldID int64) error {
	err := db.transaction(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM plan_fields WHERE id = ? AND plan_id = ?`, fieldID, planID)
		if err != nil {
			return fmt.Errorf("failed to delete plan field: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if n == 0 {
			return ErrFieldNotFound
		}
		res, err = tx.Exec(`UPDATE shift_plans SET experience_field_id = NULL WHERE id = ? AND experience_field_id = ?`, planID, fieldID)
		if err != nil {
			return fmt.Errorf("failed to clear experience field: %w", err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return clearExperienceLevel(tx, planID, "")
		}
		return nil
	})
	if err != nil {
		db.logError("DeletePlanField", err, planID, fieldID)
	}
	return err
}

// UpdatePlanFieldOrder sets the order of the fields. All IDs must belong to the plan.
func (db *DB) UpdatePlanFieldOrder(planID int64, fieldIDs []int64) error {
	return db.transaction(func(tx *sql.Tx) error {
		for i, id := range fieldIDs {
			res, err := tx.Exec(`UPDATE plan_fields SET sort_order = ? WHERE id = ? AND plan_id = ?`, i, id, planID)
			if err != nil {
				return fmt.Errorf("failed to update sort order: %w", err)
			}
			n, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}
			if n == 0 {
				return fmt.Errorf("%w: field %d", ErrFieldNotFound, id)
			}
		}
		return nil
	})
}
