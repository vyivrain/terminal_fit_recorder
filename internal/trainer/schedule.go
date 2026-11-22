// Package trainer decides when the weekly AI trainer review is due and runs it
// as a detached background process, so the person keeps using the app exactly
// as before while the review happens.
package trainer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"terminal_fit_recorder/internal/db"
)

// ReviewInterval is how often the trainer reviews a profile's training.
const ReviewInterval = 7 * 24 * time.Hour

// retryInterval spaces out launch attempts, so a broken AI CLI costs one failed
// background call every few hours instead of one per app launch.
const retryInterval = 6 * time.Hour

const (
	attemptFileName = "trainer.attempt"
	logFileName     = "trainer.log"
)

// Due reports whether a review should run: the profile has completed training
// and either has never been reviewed, or its last review is at least a week
// old and workouts (or their notes) changed since. With no new activity a
// fresh review would only repeat the last one.
func Due(history []db.WorkoutWithExercises, latest *db.TrainerReview, now time.Time) bool {
	changedSince := func(since time.Time) bool {
		for _, workout := range history {
			if workout.Workout.Status == "completed" && workout.Workout.UpdatedAt.After(since) {
				return true
			}
		}
		return false
	}
	if latest == nil {
		return changedSince(time.Time{})
	}
	return now.Sub(latest.CreatedAt) >= ReviewInterval && changedSince(latest.CreatedAt)
}

// SpawnIfDue starts `<this binary> trainer review` detached from the current
// process when a review is due for the active profile. It returns quickly and
// never blocks the caller on the AI; the child's output is appended to
// trainer.log in configDir. A nil error with started=false means nothing was due.
func SpawnIfDue(database *db.DB, configDir string, now time.Time) (started bool, err error) {
	model, err := database.GetSelectedAIModel()
	if err != nil || model == nil {
		return false, err
	}
	latest, err := database.GetLatestTrainerReview()
	if err != nil {
		return false, err
	}
	history, err := database.GetAllWorkouts()
	if err != nil {
		return false, err
	}
	if !Due(history, latest, now) {
		return false, nil
	}

	claimed, err := claimAttempt(filepath.Join(configDir, attemptFileName), now)
	if err != nil || !claimed {
		return false, err
	}

	executable, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("could not locate the app binary: %w", err)
	}
	logFile, err := os.OpenFile(filepath.Join(configDir, logFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return false, fmt.Errorf("could not open trainer log: %w", err)
	}
	defer logFile.Close()

	command := exec.Command(executable, "trainer", "review")
	command.Stdout = logFile
	command.Stderr = logFile
	detach(command)
	if err := command.Start(); err != nil {
		return false, fmt.Errorf("could not start trainer review: %w", err)
	}
	// The child outlives this process; release it so it is never waited on.
	return true, command.Process.Release()
}

// claimAttempt records a launch attempt at path unless one was made within
// retryInterval. Creation is exclusive, so of two app launches racing to start
// a review, only one wins.
func claimAttempt(path string, now time.Time) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil && now.Sub(info.ModTime()) < retryInterval:
		return false, nil
	case err == nil:
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, file.Close()
}
