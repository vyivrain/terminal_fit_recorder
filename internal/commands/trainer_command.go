package commands

import (
	"context"
	"fmt"
	"time"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

// trainerReviewTimeout matches workout generation: a review is one AI call of
// similar size, and it usually runs detached where nobody is waiting on it.
const trainerReviewTimeout = 5 * time.Minute

// trainerReviewer is the subset of *ai.CLIGenerator the review depends on.
type trainerReviewer interface {
	ReviewTraining(ctx context.Context, model db.AIModel, history []db.WorkoutWithExercises, previousGuidance, profileDescription string) (string, error)
}

// latestTrainerGuidance returns the active profile's current trainer guidance,
// or "" before the first review. Every workout generation path includes it.
func latestTrainerGuidance(database *db.DB) (string, error) {
	review, err := database.GetLatestTrainerReview()
	if err != nil {
		return "", fmt.Errorf("could not load trainer guidance: %w", err)
	}
	if review == nil {
		return "", nil
	}
	return review.Guidance, nil
}

// TrainerReviewCommand runs the AI trainer's review now. The app also launches
// it as a detached background process about once a week.
type TrainerReviewCommand struct {
	args     []string
	reviewer trainerReviewer
}

func NewTrainerReviewCommand(args ...string) *TrainerReviewCommand {
	return &TrainerReviewCommand{args: args, reviewer: ai.NewCLIGenerator()}
}

func (cmd *TrainerReviewCommand) Name() string { return "trainer review" }

func (cmd *TrainerReviewCommand) Validate() error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: terminal_fit_recorder trainer review")
	}
	return nil
}

func (cmd *TrainerReviewCommand) HelpManual() string {
	return "terminal_fit_recorder trainer review\n    Have the AI trainer review recent workouts, notes, and its previous guidance now, and save new guidance for workout generation. Runs automatically in the background about once a week."
}

func (cmd *TrainerReviewCommand) Execute(database *db.DB, _ api.OllamaClient) error {
	model, err := database.GetSelectedAIModel()
	if err != nil {
		return fmt.Errorf("could not load selected AI model: %w", err)
	}
	if model == nil {
		return fmt.Errorf("no AI model is selected; choose one in the TUI first")
	}
	profile, err := database.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("could not load active profile: %w", err)
	}
	if profile == nil {
		return fmt.Errorf("no active profile")
	}
	history, err := database.GetAllWorkouts()
	if err != nil {
		return fmt.Errorf("could not load workout history: %w", err)
	}
	previous, err := latestTrainerGuidance(database)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), trainerReviewTimeout)
	defer cancel()

	fmt.Printf("%s  Reviewing @%s's training with %s...\n", time.Now().Format(time.DateTime), profile.Name, model.DisplayName)
	guidance, err := cmd.reviewer.ReviewTraining(ctx, *model, history, previous, profile.Description)
	if err != nil {
		return fmt.Errorf("trainer review failed: %w", err)
	}
	if _, err := database.SaveTrainerReview(profile.ID, guidance); err != nil {
		return fmt.Errorf("could not save trainer guidance: %w", err)
	}
	fmt.Printf("\nNew trainer guidance:\n%s\n", guidance)
	return nil
}

// TrainerShowCommand prints the guidance workout generation currently uses.
type TrainerShowCommand struct {
	args []string
}

func NewTrainerShowCommand(args ...string) *TrainerShowCommand {
	return &TrainerShowCommand{args: args}
}

func (cmd *TrainerShowCommand) Name() string { return "trainer show" }

func (cmd *TrainerShowCommand) Validate() error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: terminal_fit_recorder trainer show")
	}
	return nil
}

func (cmd *TrainerShowCommand) HelpManual() string {
	return "terminal_fit_recorder trainer show\n    Show the AI trainer's current guidance for the active profile."
}

func (cmd *TrainerShowCommand) Execute(database *db.DB, _ api.OllamaClient) error {
	review, err := database.GetLatestTrainerReview()
	if err != nil {
		return fmt.Errorf("could not load trainer guidance: %w", err)
	}
	if review == nil {
		fmt.Println("The trainer has not reviewed this profile yet. Run 'terminal_fit_recorder trainer review' to start.")
		return nil
	}
	fmt.Printf("Trainer guidance from %s:\n\n%s\n", review.CreatedAt.Format("2006-01-02 15:04"), review.Guidance)
	return nil
}
