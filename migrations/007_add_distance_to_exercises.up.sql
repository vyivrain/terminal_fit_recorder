-- Add distance column to exercises table for run/walk exercises
ALTER TABLE exercises ADD COLUMN distance INTEGER DEFAULT 0;
