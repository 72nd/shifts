-- Coordination tools: people per plan, custom person fields, participation
-- status, load factor / popularity / experience need on types and shifts,
-- and working time rules on the plan. See prj/coordination-plan.md (section 5).
-- Only CREATE TABLE and ALTER TABLE ... ADD COLUMN with defaults.

CREATE TABLE plan_fields (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL,
    label_de TEXT,
    label_en TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('choice','number','boolean','text')),
    options TEXT NOT NULL DEFAULT '[]',
    editable_by TEXT NOT NULL DEFAULT 'admin' CHECK (editable_by IN ('participant','admin')),
    required BOOLEAN NOT NULL DEFAULT false,
    FOREIGN KEY (plan_id) REFERENCES shift_plans(id) ON DELETE CASCADE
);
CREATE INDEX idx_plan_fields_plan_id ON plan_fields(plan_id);

CREATE TABLE plan_people (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    name_key TEXT NOT NULL,
    display_name TEXT NOT NULL,
    available_from TEXT,
    available_until TEXT,
    target_minutes INTEGER,
    prefers_long_days BOOLEAN NOT NULL DEFAULT false,
    field_values TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (plan_id, name_key),
    FOREIGN KEY (plan_id) REFERENCES shift_plans(id) ON DELETE CASCADE
);

ALTER TABLE shift_plans ADD COLUMN experience_field_id INTEGER;
ALTER TABLE shift_plans ADD COLUMN attendance_mode TEXT NOT NULL DEFAULT 'off';
ALTER TABLE shift_plans ADD COLUMN default_target_minutes INTEGER;
ALTER TABLE shift_plans ADD COLUMN min_rest_minutes INTEGER;
ALTER TABLE shift_plans ADD COLUMN max_day_minutes INTEGER;
ALTER TABLE shift_plans ADD COLUMN show_declined BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE shift_types ADD COLUMN load_factor REAL NOT NULL DEFAULT 1.0;
ALTER TABLE shift_types ADD COLUMN popularity TEXT NOT NULL DEFAULT 'auto';
ALTER TABLE shift_types ADD COLUMN exp_min_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE shift_types ADD COLUMN exp_min_level TEXT;

ALTER TABLE shifts ADD COLUMN load_factor REAL;
ALTER TABLE shifts ADD COLUMN popularity TEXT;
ALTER TABLE shifts ADD COLUMN exp_min_count INTEGER;
ALTER TABLE shifts ADD COLUMN exp_min_level TEXT;

ALTER TABLE participants ADD COLUMN status TEXT NOT NULL DEFAULT 'signed_up';
ALTER TABLE participants ADD COLUMN status_changed_at TEXT;
ALTER TABLE participants ADD COLUMN added_by_admin BOOLEAN NOT NULL DEFAULT false;
