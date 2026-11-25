package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/72nd/shifts/internal/util"
)

// StatusRemove is the pseudo status of a coordination operation that deletes the participation.
const StatusRemove = "remove"

// CoordinationOp sets the status of one person in one shift.
// Status is signed_up, assigned, declined or remove.
type CoordinationOp struct {
	ShiftID int64
	Name    string
	Status  string
}

// CreatedParticipation reports a participation that an operation created.
type CreatedParticipation struct {
	ShiftID       int64
	ParticipantID int64
	Name          string
}

// CoordinationResult is the outcome of ApplyCoordinationOps.
type CoordinationResult struct {
	Applied int
	Created []CreatedParticipation
}

// ApplyCoordinationOps applies all operations and locks in one transaction (all or nothing).
// For each operation the participation is looked up by normalised name; an existing one gets
// the new status (or is removed), otherwise one is created with added_by_admin set. There is
// no capacity check. All shifts must belong to the plan, otherwise ErrNotFound is returned.
// Names must be validated by the caller.
func (db *DB) ApplyCoordinationOps(planID int64, ops []CoordinationOp, lock []int64) (*CoordinationResult, error) {
	result := &CoordinationResult{Created: []CreatedParticipation{}}
	err := db.transaction(func(tx *sql.Tx) error {
		// Shifts of the plan, and the existing participations per shift
		rows, err := tx.Query(`SELECT s.id FROM shifts s JOIN shift_types st ON s.type_id = st.id WHERE st.plan_id = ?`, planID)
		if err != nil {
			return fmt.Errorf("failed to query shifts: %w", err)
		}
		inPlan := map[int64]bool{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan shift: %w", err)
			}
			inPlan[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, op := range ops {
			if !inPlan[op.ShiftID] {
				return fmt.Errorf("%w: shift %d", ErrNotFound, op.ShiftID)
			}
		}
		for _, id := range lock {
			if !inPlan[id] {
				return fmt.Errorf("%w: shift %d", ErrNotFound, id)
			}
		}

		// participation id by shift and person key; loaded lazily per shift
		byShift := map[int64]map[string]int64{}
		load := func(shiftID int64) (map[string]int64, error) {
			if m, ok := byShift[shiftID]; ok {
				return m, nil
			}
			ps, err := db.getParticipantsByShift(tx, shiftID)
			if err != nil {
				return nil, err
			}
			m := map[string]int64{}
			for _, p := range ps {
				m[util.PersonKey(p.Name)] = p.ID
			}
			byShift[shiftID] = m
			return m, nil
		}

		now := time.Now().UTC().Format(time.RFC3339)
		for _, op := range ops {
			m, err := load(op.ShiftID)
			if err != nil {
				return err
			}
			key := util.PersonKey(op.Name)
			id, exists := m[key]
			switch {
			case exists && op.Status == StatusRemove:
				if _, err := tx.Exec(`DELETE FROM participants WHERE id = ?`, id); err != nil {
					return fmt.Errorf("failed to remove participation: %w", err)
				}
				delete(m, key)
			case exists:
				if _, err := tx.Exec(`UPDATE participants SET status = ?, status_changed_at = ? WHERE id = ?`, op.Status, now, id); err != nil {
					return fmt.Errorf("failed to set status: %w", err)
				}
			case op.Status == StatusRemove:
				// nothing to remove
			default:
				res, err := tx.Exec(`INSERT INTO participants (shift_id, name, created_at, status, status_changed_at, added_by_admin)
                    VALUES (?, ?, ?, ?, ?, true)`, op.ShiftID, op.Name, now, op.Status, now)
				if err != nil {
					return fmt.Errorf("failed to create participation: %w", err)
				}
				newID, err := res.LastInsertId()
				if err != nil {
					return fmt.Errorf("failed to get last insert ID: %w", err)
				}
				m[key] = newID
				result.Created = append(result.Created, CreatedParticipation{ShiftID: op.ShiftID, ParticipantID: newID, Name: op.Name})
			}
			result.Applied++
		}

		for _, id := range lock {
			if _, err := tx.Exec(`UPDATE shifts SET is_locked = true WHERE id = ?`, id); err != nil {
				return fmt.Errorf("failed to lock shift: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		db.logError("ApplyCoordinationOps", err, planID, len(ops), len(lock))
		return nil, err
	}
	return result, nil
}
