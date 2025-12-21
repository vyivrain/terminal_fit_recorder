package config

import (
	"os"
)

type Config struct {
	OllamaHost   string
	OllamaModel  string
	OllamaPrompt string
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	cfg := &Config{
		OllamaHost:  getEnv("TERMINAL_FIT_RECORDER_OLLAMA_HOST", "http://192.168.1.39:11434"),
		OllamaModel: getEnv("TERMINAL_FIT_RECORDER_OLLAMA_MODEL", "qwen3-coder:480b-cloud"),
		OllamaPrompt: getEnv("TERMINAL_FIT_RECORDER_OLLAMA_PROMPT", `If there's no previously provided workout data, suggest a beginner workout.
		If previously there were a lot of strength workouts provide a cardio workout and vice versa.
		But the first thing to lookout what type of workout to suggest is provided looking into workout data and check for a routine.
		Also keep the exercises in workouts to same body parts as in previous workouts, it could be different exercises but same body parts.
		F.e. leg/shoulders, chest/back, arms, upper body, lower body, workouts. Keep this in mind when looking through exercises of the workout.
		When generating exercises for cardio workouts, suggest exercises that affect all major muscle groups.`),
	}

	return cfg
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
