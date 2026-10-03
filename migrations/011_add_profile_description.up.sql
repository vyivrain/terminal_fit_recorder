-- Add description column to profiles: free-form context (habits, injuries,
-- preferences) the AI coach should consider when generating workouts.
ALTER TABLE profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';
