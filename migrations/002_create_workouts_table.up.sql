CREATE TABLE workouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workout_type TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create a default strength workout for existing exercises
INSERT INTO workouts (workout_type, created_at, updated_at)
SELECT 'strength', MIN(created_at), MAX(updated_at)
FROM exercises
WHERE EXISTS (SELECT 1 FROM exercises);

-- Add workout_id column to exercises table
ALTER TABLE exercises ADD COLUMN workout_id INTEGER REFERENCES workouts(id);

-- Link existing exercises to the created workout
UPDATE exercises
SET workout_id = (SELECT id FROM workouts WHERE workout_type = 'strength' LIMIT 1)
WHERE workout_id IS NULL;
