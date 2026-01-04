package prompt

import (
	"fmt"
	"strings"

	"terminal_fit_recorder/internal/db"
)

// BuildPrompt creates a complete prompt for the LLM with workout data and user question
func BuildPrompt(workouts []db.WorkoutWithExercises, exerciseCount int, customPrompt string) string {
	var sb strings.Builder

	sb.WriteString(
		"You are a fitness coach analyzing workout data. Generate only next(1) workout based on provided data." + customPrompt,
	)

	// Add exercise count instruction if specified
	if exerciseCount > 0 {
		sb.WriteString(fmt.Sprintf("\t\tGenerate exactly %d exercises for this workout.\n", exerciseCount))
	}

	sb.WriteString(`
		The output of the next workout provide in json format. {
			"type": "strength",
			"date": "2006-01-02",
			"exercises": [
				{
					"name": "Exercise Name:str",
					"weight": "Weight:int",
					"reps": "Reps:int",
					"sets": "Sets:int",
					"duration": "Duration:int"
				}
			]
		}
		`,
	)
	sb.WriteString(formatWorkoutData(workouts))
	return sb.String()
}

// formatWorkoutData converts workout data into a compact, LLM-friendly format
// Format: Name|Weight|Reps×Sets|Duration
func formatWorkoutData(workouts []db.WorkoutWithExercises) string {
	var sb strings.Builder

	sb.WriteString("Workout History\n")
	sb.WriteString("Format: Exercise Name|Weight|Reps×Sets|Duration\n")
	sb.WriteString("All durations in minutes. Empty fields indicated by '-'.\n\n")

	for _, workout := range workouts {
		// Write workout header with date and type
		sb.WriteString(fmt.Sprintf("## %s (%s)\n",
			workout.Workout.WorkoutDate.Format("2006-01-02"),
			workout.Workout.WorkoutType))

		// Write exercises
		for _, ex := range workout.Exercises {
			sb.WriteString(ex.Name)
			sb.WriteString("|")

			// Weight
			if ex.Weight > 0 {
				sb.WriteString(fmt.Sprintf("%d kg", ex.Weight))
			} else {
				sb.WriteString("-")
			}
			sb.WriteString("|")

			// Reps × Sets
			if ex.Repetitions > 0 || ex.Sets > 0 {
				reps := "-"
				if ex.Repetitions > 0 {
					reps = fmt.Sprintf("%d", ex.Repetitions)
				}
				sets := "-"
				if ex.Sets > 0 {
					sets = fmt.Sprintf("%d", ex.Sets)
				}
				sb.WriteString(fmt.Sprintf("%s×%s", reps, sets))
			} else {
				sb.WriteString("-")
			}
			sb.WriteString("|")

			// Duration
			if ex.Duration > 0 {
				sb.WriteString(fmt.Sprintf("%.1fmin", ex.Duration))
			} else {
				sb.WriteString("-")
			}

			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
