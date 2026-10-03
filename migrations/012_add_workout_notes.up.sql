-- Add notes column to workouts: persisted commentary (what changed, improved,
-- or reduced) usually written by the AI generator and surfaced via Hermes.
ALTER TABLE workouts ADD COLUMN notes TEXT NOT NULL DEFAULT '';
