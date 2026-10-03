package commands

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/utils"
)

type ShowLastWorkoutCommand struct{}

func NewShowLastWorkoutCommand() *ShowLastWorkoutCommand {
	return &ShowLastWorkoutCommand{}
}

func (cmd *ShowLastWorkoutCommand) Name() string {
	return "show last workout"
}

func (cmd *ShowLastWorkoutCommand) Validate() error {
	return nil
}

func (cmd *ShowLastWorkoutCommand) HelpManual() string {
	return "terminal_fit_recorder exercise last\n    Display the most recent workout with all exercises."
}

func (cmd *ShowLastWorkoutCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	workout, err := database.GetLastWorkout()
	if err != nil {
		return fmt.Errorf("error fetching last workout: %v", err)
	}

	if workout == nil {
		fmt.Println("No workouts found")
		return nil
	}

	fmt.Printf("\nWorkout Type: %s\n", workout.Workout.WorkoutType)
	fmt.Printf("Date: %s\n", workout.Workout.WorkoutDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Status: %s\n", workout.Workout.Status)
	if workout.Workout.Notes != "" {
		fmt.Printf("Notes: %s\n", workout.Workout.Notes)
	}
	fmt.Println()

	utils.PrintExercises(workout.Exercises)
	return nil
}

// FormatWorkout formats a workout into a readable string with tabular layout
func FormatWorkout(workout *db.WorkoutWithExercises) string {
	if workout == nil {
		return "No workout data"
	}

	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("\nWorkout Type: %s\n", workout.Workout.WorkoutType))
	buf.WriteString(fmt.Sprintf("Date: %s\n", workout.Workout.WorkoutDate.Format("2006-01-02 15:04:05")))
	buf.WriteString(fmt.Sprintf("Status: %s\n", workout.Workout.Status))
	if workout.Workout.Notes != "" {
		buf.WriteString(fmt.Sprintf("Notes: %s\n", workout.Workout.Notes))
	}
	buf.WriteString("\n")

	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Exercise\tWeight\tReps\tSets\tDuration\tDistance\tVideo")
	fmt.Fprintln(w, "--------\t------\t----\t----\t--------\t--------\t-----")

	for _, exercise := range workout.Exercises {
		weight := "-"
		if exercise.Weight > 0 {
			weight = fmt.Sprintf("%d kg", exercise.Weight)
		}

		duration := "-"
		if exercise.Duration > 0 {
			duration = fmt.Sprintf("%.2f min", exercise.Duration)
		}

		distance := "-"
		if exercise.Distance > 0 {
			distance = fmt.Sprintf("%d m", exercise.Distance)
		}

		video := "-"
		if exercise.YoutubeURL != "" {
			video = exercise.YoutubeURL
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\t%s\n",
			exercise.Name, weight, exercise.Repetitions, exercise.Sets, duration, distance, video)
	}

	w.Flush()

	return buf.String()
}

type ShowAllWorkoutsCommand struct{}

func NewShowAllWorkoutsCommand() *ShowAllWorkoutsCommand {
	return &ShowAllWorkoutsCommand{}
}

func (cmd *ShowAllWorkoutsCommand) Name() string {
	return "show all workouts"
}

func (cmd *ShowAllWorkoutsCommand) Validate() error {
	return nil
}

func (cmd *ShowAllWorkoutsCommand) HelpManual() string {
	return "terminal_fit_recorder exercise all\n    Display all workouts with their exercises."
}

func (cmd *ShowAllWorkoutsCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	workouts, err := database.GetAllWorkouts()
	if err != nil {
		return fmt.Errorf("error fetching workouts: %v", err)
	}

	if len(workouts) == 0 {
		fmt.Println("No workouts found")
		return nil
	}

	for i, workout := range workouts {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Workout Type: %s\n", workout.Workout.WorkoutType)
		fmt.Printf("Date: %s\n", workout.Workout.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Status: %s\n", workout.Workout.Status)
		if workout.Workout.Notes != "" {
			fmt.Printf("Notes: %s\n", workout.Workout.Notes)
		}
		fmt.Println()

		utils.PrintExercises(workout.Exercises)
		fmt.Println("─────────────────────────────────────")
	}

	return nil
}
