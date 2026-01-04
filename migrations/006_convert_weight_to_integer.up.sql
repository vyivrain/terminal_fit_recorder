-- SQLite doesn't support ALTER COLUMN directly, so we need to recreate the table

-- Create new exercises table with INTEGER weight, repetitions, and sets
CREATE TABLE exercises_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    weight INTEGER DEFAULT 0,
    repetitions INTEGER DEFAULT 0,
    sets INTEGER DEFAULT 0,
    duration REAL DEFAULT 0,
    workout_id INTEGER REFERENCES workouts(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Copy data from old table, converting weight, repetitions, and sets to integers
-- CAST returns 0 for non-numeric values in SQLite
INSERT INTO exercises_new (id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at)
SELECT
    id,
    name,
    CAST(COALESCE(NULLIF(TRIM(weight), ''), '0') AS INTEGER),
    CAST(COALESCE(NULLIF(TRIM(repetitions), ''), '0') AS INTEGER),
    CAST(COALESCE(NULLIF(TRIM(sets), ''), '0') AS INTEGER),
    duration,
    workout_id,
    created_at,
    updated_at
FROM exercises;

-- Drop old table
DROP TABLE exercises;

-- Rename new table to exercises
ALTER TABLE exercises_new RENAME TO exercises;
