package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/72nd/shifts/internal/model"
)

// MaxDraftsPerPlan is the number of saved lab variants a plan can hold.
const MaxDraftsPerPlan = 50

// ListDrafts returns the variants of a plan without their operations, newest change first.
func (db *DB) ListDrafts(planID int64) ([]model.CoordinationDraftSummary, error) {
	out := []model.CoordinationDraftSummary{}
	err := db.transaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(`SELECT id, name, json_array_length(ops), created_at, updated_at, applied_at
			FROM coordination_drafts WHERE plan_id = ? ORDER BY updated_at DESC, id DESC`, planID)
		if err != nil {
			return fmt.Errorf("failed to query drafts: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var d model.CoordinationDraftSummary
			var applied sql.NullString
			if err := rows.Scan(&d.ID, &d.Name, &d.OpCount, &d.CreatedAt, &d.UpdatedAt, &applied); err != nil {
				return fmt.Errorf("failed to scan draft: %w", err)
			}
			if applied.Valid {
				d.AppliedAt = &applied.String
			}
			out = append(out, d)
		}
		return rows.Err()
	})
	if err != nil {
		db.logError("ListDrafts", err, planID)
		return nil, err
	}
	return out, nil
}

// GetDraft returns one variant with its operations. A missing variant or one of another plan gives ErrNotFound.
func (db *DB) GetDraft(planID, id int64) (*model.CoordinationDraft, error) {
	var d *model.CoordinationDraft
	err := db.transaction(func(tx *sql.Tx) error {
		var err error
		d, err = getDraft(tx, planID, id)
		return err
	})
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			db.logError("GetDraft", err, planID, id)
		}
		return nil, err
	}
	return d, nil
}

func getDraft(tx *sql.Tx, planID, id int64) (*model.CoordinationDraft, error) {
	d := &model.CoordinationDraft{}
	var ops string
	var applied sql.NullString
	err := tx.QueryRow(`SELECT id, name, ops, created_at, updated_at, applied_at
		FROM coordination_drafts WHERE id = ? AND plan_id = ?`, id, planID).
		Scan(&d.ID, &d.Name, &ops, &d.CreatedAt, &d.UpdatedAt, &applied)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query draft: %w", err)
	}
	if err := json.Unmarshal([]byte(ops), &d.Ops); err != nil {
		return nil, fmt.Errorf("failed to decode ops of draft %d: %w", id, err)
	}
	if d.Ops == nil {
		d.Ops = []model.CoordinationOp{}
	}
	d.OpCount = len(d.Ops)
	if applied.Valid {
		d.AppliedAt = &applied.String
	}
	return d, nil
}

// CreateDraft stores a new variant. ErrLimitReached is returned when the plan already holds MaxDraftsPerPlan.
func (db *DB) CreateDraft(planID int64, name string, ops []model.CoordinationOp) (*model.CoordinationDraft, error) {
	data, err := json.Marshal(ops)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ops: %w", err)
	}
	var d *model.CoordinationDraft
	err = db.transactionImmediate(func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM coordination_drafts WHERE plan_id = ?`, planID).Scan(&count); err != nil {
			return fmt.Errorf("failed to count drafts: %w", err)
		}
		if count >= MaxDraftsPerPlan {
			return ErrLimitReached
		}
		now := time.Now().UTC().Format(time.RFC3339)
		res, err := tx.Exec(`INSERT INTO coordination_drafts (plan_id, name, ops, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)`, planID, name, string(data), now, now)
		if err != nil {
			return fmt.Errorf("failed to create draft: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}
		d, err = getDraft(tx, planID, id)
		return err
	})
	if err != nil {
		if !errors.Is(err, ErrLimitReached) {
			db.logError("CreateDraft", err, planID)
		}
		return nil, err
	}
	return d, nil
}

// UpdateDraft replaces name and operations of a variant (last write wins). With applied set,
// applied_at is set to now. A missing variant or one of another plan gives ErrNotFound.
func (db *DB) UpdateDraft(planID, id int64, name string, ops []model.CoordinationOp, applied bool) (*model.CoordinationDraft, error) {
	data, err := json.Marshal(ops)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ops: %w", err)
	}
	var d *model.CoordinationDraft
	err = db.transaction(func(tx *sql.Tx) error {
		now := time.Now().UTC().Format(time.RFC3339)
		query := `UPDATE coordination_drafts SET name = ?, ops = ?, updated_at = ? WHERE id = ? AND plan_id = ?`
		args := []any{name, string(data), now, id, planID}
		if applied {
			query = `UPDATE coordination_drafts SET name = ?, ops = ?, updated_at = ?, applied_at = ? WHERE id = ? AND plan_id = ?`
			args = []any{name, string(data), now, now, id, planID}
		}
		res, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update draft: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		} else if n == 0 {
			return ErrNotFound
		}
		d, err = getDraft(tx, planID, id)
		return err
	})
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			db.logError("UpdateDraft", err, planID, id)
		}
		return nil, err
	}
	return d, nil
}

// DeleteDraft removes a variant. A missing variant or one of another plan gives ErrNotFound.
func (db *DB) DeleteDraft(planID, id int64) error {
	err := db.transaction(func(tx *sql.Tx) error {
		res, err := tx.Exec(`DELETE FROM coordination_drafts WHERE id = ? AND plan_id = ?`, id, planID)
		if err != nil {
			return fmt.Errorf("failed to delete draft: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("failed to get affected rows: %w", err)
		} else if n == 0 {
			return ErrNotFound
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrNotFound) {
		db.logError("DeleteDraft", err, planID, id)
	}
	return err
}
