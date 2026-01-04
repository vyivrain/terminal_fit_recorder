package utils

import (
	"fmt"
	"strconv"
	"strings"
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

// ParseWeight converts weight string to integer
// Handles formats like "100", "100kg", "bodyweight", empty strings, etc.
func ParseWeight(weight string) int {
	weight = strings.TrimSpace(strings.ToLower(weight))

	// Handle empty or bodyweight
	if weight == "" || weight == "bodyweight" || weight == "-" {
		return 0
	}

	// Remove "kg" suffix if present
	weight = strings.TrimSuffix(weight, "kg")
	weight = strings.TrimSpace(weight)

	// Try to parse as integer
	w, err := strconv.Atoi(weight)
	if err != nil {
		// Try parsing as float and convert to int
		f, err := strconv.ParseFloat(weight, 64)
		if err != nil {
			return 0
		}
		return int(f)
	}

	return w
}

// ParseInt parses a string to int, returning 0 for empty or invalid strings
func ParseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}
