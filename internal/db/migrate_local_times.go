package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/72nd/shifts/internal/model"
)

// migrateLocalTimes is the Go step of migration 0002_local_times. It adds
// shift_plans.time_zone (value: the default zone from the config) and turns
// the stored instants into floating local times (see timezone-plan.md). It is
// idempotent: values already in the new format are left alone.
func migrateLocalTimes(tx *sql.Tx, logger *slog.Logger) error {
	defaultZone := model.DefaultLocation().String()

	hasZone, err := columnExists(tx, "shift_plans", "time_zone")
	if err != nil {
		return err
	}
	if !hasZone {
		stmt := fmt.Sprintf("ALTER TABLE shift_plans ADD COLUMN time_zone TEXT NOT NULL DEFAULT '%s'", defaultZone)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}

	type planRow struct {
		id                                          int64
		zone, startDate, endDate, createdAt, bounds string
	}
	rows, err := tx.Query("SELECT id, time_zone, start_date, end_date, created_at, day_boundary_time FROM shift_plans ORDER BY id")
	if err != nil {
		return fmt.Errorf("failed to read shift_plans: %w", err)
	}
	var plans []planRow
	for rows.Next() {
		var p planRow
		if err := rows.Scan(&p.id, &p.zone, &p.startDate, &p.endDate, &p.createdAt, &p.bounds); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan shift_plans: %w", err)
		}
		plans = append(plans, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var nPlans, nShifts, nParticipants int
	for _, p := range plans {
		loc, err := time.LoadLocation(p.zone)
		if err != nil {
			return fmt.Errorf("shift_plans id %d: invalid time_zone %q: %w", p.id, p.zone, err)
		}

		start, c1, err := parseLegacyDate(p.startDate)
		if err != nil {
			return fmt.Errorf("shift_plans id %d: invalid start_date %q: %w", p.id, p.startDate, err)
		}
		end, c2, err := parseLegacyDate(p.endDate)
		if err != nil {
			return fmt.Errorf("shift_plans id %d: invalid end_date %q: %w", p.id, p.endDate, err)
		}
		created, err := time.Parse(time.RFC3339, p.createdAt)
		if err != nil {
			return fmt.Errorf("shift_plans id %d: invalid created_at %q: %w", p.id, p.createdAt, err)
		}
		createdUTC := created.UTC().Format(time.RFC3339)
		bounds := p.bounds
		if bounds == "" {
			bounds = "00:00"
		}
		if c1 || c2 || createdUTC != p.createdAt || bounds != p.bounds {
			if _, err := tx.Exec(
				"UPDATE shift_plans SET start_date = ?, end_date = ?, created_at = ?, day_boundary_time = ? WHERE id = ?",
				start.String(), end.String(), createdUTC, bounds, p.id,
			); err != nil {
				return fmt.Errorf("shift_plans id %d: update failed: %w", p.id, err)
			}
			nPlans++
		}

		n, err := migratePlanShifts(tx, p.id, loc)
		if err != nil {
			return err
		}
		nShifts += n
	}

	nParticipants, err = migrateParticipantTimes(tx)
	if err != nil {
		return err
	}

	logger.Info("migrated times to local event time",
		"default_zone", defaultZone,
		"shift_plans_rewritten", nPlans,
		"shifts_rewritten", nShifts,
		"participants_rewritten", nParticipants)
	return nil
}

// migratePlanShifts converts the start times of all shifts of one plan from
// instants to wall clock in loc and returns the number of rewritten rows.
func migratePlanShifts(tx *sql.Tx, planID int64, loc *time.Location) (int, error) {
	rows, err := tx.Query(`
		SELECT s.id, s.start_time FROM shifts s
		JOIN shift_types st ON s.type_id = st.id
		WHERE st.plan_id = ?`, planID)
	if err != nil {
		return 0, fmt.Errorf("failed to read shifts of plan %d: %w", planID, err)
	}
	type update struct {
		id    int64
		value string
	}
	var updates []update
	for rows.Next() {
		var id int64
		var v string
		if err := rows.Scan(&id, &v); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan shifts: %w", err)
		}
		dt, changed, err := parseLegacyDateTime(v, loc)
		if err != nil {
			rows.Close()
			return 0, fmt.Errorf("shifts id %d: invalid start_time %q: %w", id, v, err)
		}
		if changed {
			updates = append(updates, update{id, dt.String()})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, u := range updates {
		if _, err := tx.Exec("UPDATE shifts SET start_time = ? WHERE id = ?", u.value, u.id); err != nil {
			return 0, fmt.Errorf("shifts id %d: update failed: %w", u.id, err)
		}
	}
	return len(updates), nil
}

// migrateParticipantTimes rewrites participants.created_at as RFC3339 UTC.
func migrateParticipantTimes(tx *sql.Tx) (int, error) {
	rows, err := tx.Query("SELECT id, created_at FROM participants")
	if err != nil {
		return 0, fmt.Errorf("failed to read participants: %w", err)
	}
	type update struct {
		id    int64
		value string
	}
	var updates []update
	for rows.Next() {
		var id int64
		var v string
		if err := rows.Scan(&id, &v); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan participants: %w", err)
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			rows.Close()
			return 0, fmt.Errorf("participants id %d: invalid created_at %q: %w", id, v, err)
		}
		if utc := t.UTC().Format(time.RFC3339); utc != v {
			updates = append(updates, update{id, utc})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, u := range updates {
		if _, err := tx.Exec("UPDATE participants SET created_at = ? WHERE id = ?", u.value, u.id); err != nil {
			return 0, fmt.Errorf("participants id %d: update failed: %w", u.id, err)
		}
	}
	return len(updates), nil
}
