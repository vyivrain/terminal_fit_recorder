package trainer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func workoutUpdatedAt(status string, updatedAt time.Time) db.WorkoutWithExercises {
	return db.WorkoutWithExercises{Workout: db.Workout{Status: status, UpdatedAt: updatedAt}}
}

func TestDue(t *testing.T) {
	now := time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)
	lastReview := &db.TrainerReview{Guidance: "keep going", CreatedAt: now.Add(-8 * 24 * time.Hour)}
	recentReview := &db.TrainerReview{Guidance: "keep going", CreatedAt: now.Add(-3 * 24 * time.Hour)}
	afterLastReview := lastReview.CreatedAt.Add(24 * time.Hour)
	beforeLastReview := lastReview.CreatedAt.Add(-24 * time.Hour)

	cases := []struct {
		name    string
		history []db.WorkoutWithExercises
		latest  *db.TrainerReview
		want    bool
	}{
		{"no training yet", nil, nil, false},
		{"only planned workouts", []db.WorkoutWithExercises{workoutUpdatedAt("planned", afterLastReview)}, nil, false},
		{"first review once training exists", []db.WorkoutWithExercises{workoutUpdatedAt("completed", beforeLastReview)}, nil, true},
		{"week old review with new training", []db.WorkoutWithExercises{workoutUpdatedAt("completed", afterLastReview)}, lastReview, true},
		{"week old review without new training", []db.WorkoutWithExercises{workoutUpdatedAt("completed", beforeLastReview)}, lastReview, false},
		{"week old review with only new planned workouts", []db.WorkoutWithExercises{workoutUpdatedAt("planned", afterLastReview)}, lastReview, false},
		{"recent review with new training", []db.WorkoutWithExercises{workoutUpdatedAt("completed", now)}, recentReview, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Due(tc.history, tc.latest, now))
		})
	}
}

func TestClaimAttemptAllowsOneAttemptPerRetryInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), attemptFileName)
	now := time.Now()

	claimed, err := claimAttempt(path, now)
	require.NoError(t, err)
	require.True(t, claimed, "first attempt should be claimed")

	claimed, err = claimAttempt(path, now.Add(time.Hour))
	require.NoError(t, err)
	require.False(t, claimed, "a second launch within the retry interval must not start another review")

	stale := now.Add(-retryInterval - time.Minute)
	require.NoError(t, os.Chtimes(path, stale, stale))
	claimed, err = claimAttempt(path, now)
	require.NoError(t, err)
	require.True(t, claimed, "an attempt older than the retry interval should be retried")
}
