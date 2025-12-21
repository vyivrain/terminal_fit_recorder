-- Add status column to workouts table
ALTER TABLE workouts ADD COLUMN status TEXT NOT NULL DEFAULT 'completed';
