-- Weekly AI trainer reviews: guidance distilled from a profile's recent
-- workouts and notes, fed into every workout generation prompt. Each review
-- is kept so the trainer can build on what it said last time.
CREATE TABLE trainer_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL REFERENCES profiles(id),
    guidance TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX trainer_reviews_profile_created
ON trainer_reviews(profile_id, created_at DESC);
