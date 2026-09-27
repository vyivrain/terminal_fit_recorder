CREATE TABLE profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    is_active INTEGER NOT NULL DEFAULT 0 CHECK (is_active IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO profiles (name, is_active) VALUES ('mine', 1);

-- Existing workout history belongs to the default profile. The column is
-- nullable during this migration because SQLite cannot add a referenced
-- NOT NULL column safely to a populated table. The triggers below enforce
-- the invariant for all future writes after the backfill.
ALTER TABLE workouts ADD COLUMN profile_id INTEGER REFERENCES profiles(id);

UPDATE workouts
SET profile_id = (SELECT id FROM profiles WHERE name = 'mine');

CREATE TRIGGER workouts_require_profile_on_insert
BEFORE INSERT ON workouts
WHEN NEW.profile_id IS NULL
BEGIN
    SELECT RAISE(ABORT, 'workouts.profile_id is required');
END;

CREATE TRIGGER workouts_require_profile_on_update
BEFORE UPDATE OF profile_id ON workouts
WHEN NEW.profile_id IS NULL
BEGIN
    SELECT RAISE(ABORT, 'workouts.profile_id is required');
END;

CREATE UNIQUE INDEX profiles_one_active
ON profiles(is_active)
WHERE is_active = 1;

CREATE INDEX workouts_profile_date
ON workouts(profile_id, workout_date DESC);
