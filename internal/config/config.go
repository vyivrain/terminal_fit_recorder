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
		OllamaPrompt: getEnv("TERMINAL_FIT_RECORDER_OLLAMA_PROMPT", `Act as the person's coach and improve their training when the data supports it; do not just copy the last session. If there is no history, suggest a beginner workout.
			Aim for about 3 sessions per week: 2 strength and 1 cardio. Pick this workout's type to keep that balance from the recent sessions, and never work the same primary muscle group two sessions in a row.
			Across the week the strength sessions together must cover all major muscle groups (chest, back, shoulders, arms, quadriceps, hamstrings and glutes, core), while each single session targets only part of that using a split scaled to the number of strength sessions: 1 means full body, 2 means an upper/lower split, 3 means push/pull/legs. With 2 strength sessions make this workout the upper or lower half the recent sessions did not already cover so the pair completes full coverage. Cardio should engage the major muscle groups.
			Treat an exercise as plateaued when its last 3 or more sessions used the same weight and reps. Then progress it: add 1-2 reps first, and once at the top of its usual range add 2.5 kg (5 kg for large lower-body compounds like squat and deadlift) and reset the reps.
			Find the most recent completed session (ignore any planned/future ones) and compare its date to this workout's date. If that gap is about a week or more, do not count the muscle group it trained as already covered this week; pick this session's split by whichever groups have gone longest untrained. If that gap is about two weeks or more, also do not progress: reduce the working weight, rebuild to the previous form first, then resume progressing once it is held again.
			If the most recent session dropped in weight or reps, hold the load steady. Keep the main compound lifts stable and only swap accessories that have been repeated many times unchanged.
			Keep the same body-part routine the history shows (leg/shoulders, chest/back, arms, upper body, lower body), using different exercises for the same muscles when useful. For cardio workouts, choose exercises that affect all major muscle groups.`),
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
