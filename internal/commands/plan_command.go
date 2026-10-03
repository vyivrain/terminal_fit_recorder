package commands

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/youtube"
)

const planTimeout = 5 * time.Minute

// PlanCommand generates one workout through the AI model selected in the TUI.
// It is deliberately non-interactive so automations such as Hermes/Telegram can
// invoke it without allocating a terminal. Saving requires an explicit --save.
type PlanCommand struct {
	args        []string
	save        bool
	videoFinder planVideoFinder
}

func NewPlanCommand(args ...string) *PlanCommand {
	return &PlanCommand{args: args, videoFinder: youtube.NewFinder(3)}
}

type planVideoFinder interface {
	Find(ctx context.Context, exerciseName string) (string, error)
}

func (cmd *PlanCommand) Name() string { return "plan" }

func (cmd *PlanCommand) Validate() error {
	for _, arg := range cmd.args {
		switch strings.TrimSpace(arg) {
		case "--save":
			cmd.save = true
		case "":
			// Ignore empty arguments from programmatic callers.
		default:
			return fmt.Errorf("usage: terminal_fit_recorder exercise plan [--save]")
		}
	}
	return nil
}

func (cmd *PlanCommand) HelpManual() string {
	return "terminal_fit_recorder exercise plan [--save]\n    Generate a workout non-interactively with the selected TUI AI model. Preview only by default; --save stores it as planned."
}

func (cmd *PlanCommand) Execute(database *db.DB, _ api.OllamaClient) error {
	model, err := database.GetSelectedAIModel()
	if err != nil {
		return fmt.Errorf("could not load selected AI model: %w", err)
	}
	if model == nil {
		return fmt.Errorf("no AI model is selected; choose one in the TUI first")
	}

	history, err := database.GetAllWorkouts()
	if err != nil {
		return fmt.Errorf("could not load workout history: %w", err)
	}

	profile, err := database.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("could not load active profile: %w", err)
	}
	var profileDescription string
	if profile != nil {
		profileDescription = profile.Description
	}

	ctx, cancel := context.WithTimeout(context.Background(), planTimeout)
	defer cancel()

	fmt.Printf("Generating workout with %s...\n\n", model.DisplayName)
	result, err := ai.NewCLIGenerator().Generate(ctx, *model, history, profileDescription)
	if err != nil {
		return fmt.Errorf("could not generate workout: %w", err)
	}

	videoErrors := populatePlanVideos(ctx, result.Workout, history, cmd.videoFinder)
	if len(videoErrors) > 0 {
		fmt.Printf("Note: used YouTube search links for %d exercise(s) because direct video lookup failed.\n\n", len(videoErrors))
	}

	fmt.Print(FormatWorkout(result.Workout))
	if result.ProfileUpdate != "" {
		fmt.Printf("\nSuggested profile update: %s\n", result.ProfileUpdate)
	}

	if !cmd.save {
		fmt.Println("\nPreview only. Run again with --save to store it as planned.")
		return nil
	}
	if err := database.SaveGeneratedWorkout(result.Workout); err != nil {
		return fmt.Errorf("could not save planned workout: %w", err)
	}
	if result.ProfileUpdate != "" && profile != nil {
		if _, err := database.SetProfileDescription(profile.ID, result.ProfileUpdate); err != nil {
			return fmt.Errorf("could not save profile update: %w", err)
		}
		fmt.Println("✓ Profile description updated.")
	}
	fmt.Println("\n✓ Workout saved as planned.")
	return nil
}

// populatePlanVideos makes non-interactive plan output self-contained. Each
// generated exercise reuses a stored video for the same normalized name, or
// synchronously looks one up before the workout is printed. The Finder bounds
// subprocess concurrency, while these goroutines avoid serial lookup latency.
func populatePlanVideos(ctx context.Context, workout *db.WorkoutWithExercises, history []db.WorkoutWithExercises, finder planVideoFinder) []error {
	if workout == nil || finder == nil {
		return nil
	}

	known := make(map[string]string)
	for _, previous := range history {
		for _, exercise := range previous.Exercises {
			key := normalizeExerciseName(exercise.Name)
			if key != "" && exercise.YoutubeURL != "" {
				if _, exists := known[key]; !exists {
					known[key] = exercise.YoutubeURL
				}
			}
		}
	}

	type lookupResult struct {
		index int
		url   string
		err   error
	}
	results := make(chan lookupResult, len(workout.Exercises))
	var wait sync.WaitGroup
	for index := range workout.Exercises {
		exercise := &workout.Exercises[index]
		if exercise.YoutubeURL != "" {
			continue
		}
		if url := known[normalizeExerciseName(exercise.Name)]; url != "" {
			exercise.YoutubeURL = url
			continue
		}
		name := strings.TrimSpace(exercise.Name)
		if name == "" {
			continue
		}
		wait.Add(1)
		go func(index int, name string) {
			defer wait.Done()
			url, err := finder.Find(ctx, name)
			results <- lookupResult{index: index, url: url, err: err}
		}(index, name)
	}

	wait.Wait()
	close(results)
	errors := make([]error, 0)
	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
			workout.Exercises[result.index].YoutubeURL = youtubeSearchURL(workout.Exercises[result.index].Name)
			continue
		}
		if result.url != "" {
			workout.Exercises[result.index].YoutubeURL = result.url
			continue
		}
		// A direct short clip is preferred, but every generated exercise should
		// still have a useful YouTube link when no <=60s result qualifies.
		workout.Exercises[result.index].YoutubeURL = youtubeSearchURL(workout.Exercises[result.index].Name)
	}
	return errors
}

func normalizeExerciseName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func youtubeSearchURL(exerciseName string) string {
	query := strings.TrimSpace(exerciseName) + " exercise technique"
	return "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)
}
