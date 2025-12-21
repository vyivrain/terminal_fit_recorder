-- Add workout_date column without default (SQLite limitation)
ALTER TABLE workouts ADD COLUMN workout_date DATETIME;

-- Copy existing updated_at values to workout_date for existing records
UPDATE workouts SET workout_date = updated_at;

-- For any NULL values (shouldn't happen but just in case), set to current timestamp
UPDATE workouts SET workout_date = CURRENT_TIMESTAMP WHERE workout_date IS NULL;
