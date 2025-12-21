package utils

import (
	"fmt"
	"strconv"
	"time"
)

// ParseEUDate parses EU date format DD-MM-YY
func ParseEUDate(dateStr string) (time.Time, error) {
	date, err := time.Parse("02-01-06", dateStr)
	if err != nil {
		return time.Time{}, err
	}
	return date, nil
}

// ParseExerciseCount validates and parses exercise count (1-20)
func ParseExerciseCount(countStr string) (int, error) {
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return 0, fmt.Errorf("invalid exercise count. Must be a number between 1 and 20")
	}
	if count < 1 || count > 20 {
		return 0, fmt.Errorf("exercise count must be between 1 and 20, got %d", count)
	}
	return count, nil
}
