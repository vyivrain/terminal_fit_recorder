package utils

import (
	"fmt"
	"os"
	"text/tabwriter"

	"terminal_fit_recorder/internal/db"
)

// PrintExercises prints a list of exercises in a formatted table
func PrintExercises(exercises []db.Exercise) {
	if len(exercises) == 0 {
		fmt.Println("No exercises to display")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Exercise\tWeight\tReps\tSets\tDuration")
	fmt.Fprintln(w, "--------\t------\t----\t----\t--------")

	for _, exercise := range exercises {
		if exercise.Duration > 0 {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.2f min\n",
				exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t-\n",
				exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets)
		}
	}

	w.Flush()
}
