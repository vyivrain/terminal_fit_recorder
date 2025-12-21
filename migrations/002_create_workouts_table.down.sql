-- Remove workout_id column from exercises table
ALTER TABLE exercises DROP COLUMN workout_id;

-- Drop workouts table
DROP TABLE workouts;
