-- Add youtube_url column to exercises: a short (<=60s) technique video found
-- in the background via yt-dlp search after the exercise is created.
ALTER TABLE exercises ADD COLUMN youtube_url TEXT NOT NULL DEFAULT '';
