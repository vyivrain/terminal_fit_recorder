-- Revert weight column back to TEXT

-- Create new exercises table with TEXT weight
CREATE TABLE exercises_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    weight TEXT NOT NULL,
    repetitions TEXT NOT NULL,
    sets TEXT NOT NULL,
    duration REAL DEFAULT 0,
    workout_id INTEGER REFERENCES workouts(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Copy data from old table, converting weight back to text
INSERT INTO exercises_new (id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at)
SELECT
    id,
    name,
    CAST(weight AS TEXT),
    repetitions,
    sets,
    duration,
    workout_id,
    created_at,
    updated_at
FROM exercises;

-- Drop old table
DROP TABLE exercises;

-- Rename new table to exercises
ALTER TABLE exercises_new RENAME TO exercises;
