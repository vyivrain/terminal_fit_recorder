DROP INDEX workouts_profile_date;
DROP INDEX profiles_one_active;
DROP TRIGGER workouts_require_profile_on_update;
DROP TRIGGER workouts_require_profile_on_insert;

ALTER TABLE workouts DROP COLUMN profile_id;

DROP TABLE profiles;
