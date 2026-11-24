-- Initial schema (state of v0.5.x). Existing databases that predate the
-- migration system are brought to this state by the baseline step in
-- migrate.go and then marked as version 1. Never edit this file; add a new
-- NNNN_*.sql file instead.

CREATE TABLE shift_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT UNIQUE NOT NULL,
    admin_password TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    is_multilingual BOOLEAN NOT NULL DEFAULT true,
    title_de TEXT,
    title_en TEXT NOT NULL,
    description_de TEXT,
    description_en TEXT,
    day_boundary_time TEXT NOT NULL DEFAULT '00:00',
    collaborative_mode BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_shift_plans_token ON shift_plans(token);

CREATE TABLE shift_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id INTEGER NOT NULL,
    password TEXT,
    sort_order INTEGER NOT NULL,
    title_de TEXT NOT NULL,
    title_en TEXT NOT NULL,
    description_de TEXT,
    description_en TEXT,
    location_de TEXT,
    location_en TEXT,
    FOREIGN KEY (plan_id) REFERENCES shift_plans(id) ON DELETE CASCADE
);
CREATE INDEX idx_shift_types_plan_id ON shift_types(plan_id);

CREATE TABLE shifts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type_id INTEGER NOT NULL,
    start_time TEXT NOT NULL,
    duration INTEGER NOT NULL,
    required INTEGER NOT NULL,
    comment_de TEXT,
    comment_en TEXT,
    is_locked BOOLEAN NOT NULL DEFAULT false,
    FOREIGN KEY (type_id) REFERENCES shift_types(id) ON DELETE CASCADE
);
CREATE INDEX idx_shifts_type_id ON shifts(type_id);

CREATE TABLE participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    shift_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (shift_id) REFERENCES shifts(id) ON DELETE CASCADE
);
CREATE INDEX idx_participants_shift_id ON participants(shift_id);
