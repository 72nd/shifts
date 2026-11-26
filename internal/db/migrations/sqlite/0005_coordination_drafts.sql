-- Saved variants of the coordination lab (prj/coordination-improvements-plan.md, 6.1).
-- ops: JSON array of {shift_id, name, status} as sent to POST /coordination/apply.
CREATE TABLE coordination_drafts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    ops TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    applied_at TEXT,
    FOREIGN KEY (plan_id) REFERENCES shift_plans(id) ON DELETE CASCADE
);
CREATE INDEX idx_coordination_drafts_plan ON coordination_drafts(plan_id);
