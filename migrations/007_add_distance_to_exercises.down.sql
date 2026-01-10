-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table

-- Create new exercises table without distance
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

-- Copy data from old table
INSERT INTO exercises_new (id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at)
SELECT id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at
FROM exercises;

-- Drop old table
DROP TABLE exercises;

-- Rename new table to exercises
ALTER TABLE exercises_new RENAME TO exercises;
