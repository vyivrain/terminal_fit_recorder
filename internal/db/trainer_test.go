package db_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func TestTrainerReviewsArePerProfileAndLatestWins(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "trainer.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	none, err := database.GetLatestTrainerReview()
	require.NoError(t, err)
	require.Nil(t, none)

	mine, err := database.GetActiveProfile()
	require.NoError(t, err)
	_, err = database.SaveTrainerReview(mine.ID, "First week guidance")
	require.NoError(t, err)
	_, err = database.SaveTrainerReview(mine.ID, "  Second week guidance  ")
	require.NoError(t, err)
	_, err = database.SaveTrainerReview(mine.ID, "   ")
	require.Error(t, err, "empty guidance must be rejected")

	latest, err := database.GetLatestTrainerReview()
	require.NoError(t, err)
	require.Equal(t, "Second week guidance", latest.Guidance)
	require.Equal(t, mine.ID, latest.ProfileID)

	partner, err := database.CreateProfile("partner")
	require.NoError(t, err)
	_, err = database.UseProfile("partner")
	require.NoError(t, err)
	partnerReview, err := database.GetLatestTrainerReview()
	require.NoError(t, err)
	require.Nil(t, partnerReview, "another profile's guidance must not leak")

	_, err = database.SaveTrainerReview(partner.ID, "Partner guidance")
	require.NoError(t, err)
	_, err = database.UseProfile(db.DefaultProfileName)
	require.NoError(t, err)
	_, err = database.DeleteProfile(partner.ID)
	require.NoError(t, err)

	var orphaned int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM trainer_reviews WHERE profile_id = ?`, partner.ID).Scan(&orphaned))
	require.Zero(t, orphaned, "deleting a profile deletes its trainer reviews")
}
