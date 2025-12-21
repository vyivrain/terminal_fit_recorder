package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"terminal_fit_recorder/internal/db"
	"time"
)

// Regular expression to match <think>...</think> tags (including multiline)
var thinkTagRegex = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)

// Client represents an Ollama API client with configuration
type Client struct {
	host         string
	model        string
	CustomPrompt string
}

// NewClient creates a new Ollama client with the given host and model
func NewClient(host, model, CustomPrompt string) *Client {
	return &Client{
		host:         host,
		model:        model,
		CustomPrompt: CustomPrompt,
	}
}

// stripThinkingTags removes <think>...</think> tags from DeepSeek-R1 responses
func stripThinkingTags(text string) string {
	return thinkTagRegex.ReplaceAllString(text, "")
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

// SendPrompt sends a prompt to Ollama and returns the response
func (c *Client) SendPrompt(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", c.host)

	reqBody := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var response OllamaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Response, nil
}

// SendPromptStream sends a prompt to Ollama and streams the response
// Prints response in real-time and returns the complete response
func (c *Client) SendPromptStream(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", c.host)

	reqBody := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: true, // Enable streaming
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read streaming response line by line
	scanner := bufio.NewScanner(resp.Body)
	var fullResponse string
	var buffer string
	inThinkTag := false

	for scanner.Scan() {
		line := scanner.Bytes()

		var streamResp OllamaResponse
		if err := json.Unmarshal(line, &streamResp); err != nil {
			continue // Skip malformed lines
		}

		// Add to buffer for think tag detection
		buffer += streamResp.Response

		// Track if we're inside a <think> tag
		if !inThinkTag && len(buffer) >= 7 && buffer[len(buffer)-7:] == "<think>" {
			inThinkTag = true
			buffer = ""
			continue
		}

		// If we're in a think tag, look for closing tag
		if inThinkTag {
			if len(buffer) >= 8 && buffer[len(buffer)-8:] == "</think>" {
				inThinkTag = false
				buffer = ""
			}
			continue
		}

		// Only print if we're not in a think tag
		if !inThinkTag {
			fullResponse += streamResp.Response
		}

		// Check if generation is complete
		if streamResp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fullResponse, fmt.Errorf("error reading stream: %w", err)
	}

	return fullResponse, nil
}

// AIWorkoutResponse is the structure for parsing AI JSON response
type AIWorkoutResponse struct {
	Date      string       `json:"date"`
	Type      string       `json:"type"`
	Exercises []AIExercise `json:"exercises"`
}

// AIExercise represents an exercise in AI JSON response
type AIExercise struct {
	Name     string      `json:"name"`
	Weight   interface{} `json:"weight"`   // Can be string, number, or "-"
	Reps     interface{} `json:"reps"`     // Can be string, number, or "-"
	Sets     interface{} `json:"sets"`     // Can be string, number, or "-"
	Duration interface{} `json:"duration"` // Can be string, number, or "-"
}

// ParseWorkoutResponse parses AI JSON response into db.WorkoutWithExercises
func ParseWorkoutResponse(jsonResponse string) (*db.WorkoutWithExercises, error) {
	// Extract JSON from response (in case there's extra text)
	jsonStart := strings.Index(jsonResponse, "{")
	jsonEnd := strings.LastIndex(jsonResponse, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := jsonResponse[jsonStart : jsonEnd+1]

	// Unmarshal AI response
	var aiWorkout AIWorkoutResponse
	if err := json.Unmarshal([]byte(jsonStr), &aiWorkout); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Parse date
	workoutDate, err := time.Parse("2006-01-02", aiWorkout.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Create Workout
	workout := db.Workout{
		WorkoutType: aiWorkout.Type,
		WorkoutDate: workoutDate,
	}

	// Parse exercises
	var exercises []db.Exercise
	for _, aiEx := range aiWorkout.Exercises {
		ex := db.Exercise{
			Name:        aiEx.Name,
			Weight:      convertToString(aiEx.Weight),
			Repetitions: convertToString(aiEx.Reps),
			Sets:        convertToString(aiEx.Sets),
			Duration:    convertToFloat(aiEx.Duration),
		}
		exercises = append(exercises, ex)
	}

	return &db.WorkoutWithExercises{
		Workout:   workout,
		Exercises: exercises,
	}, nil
}

// convertToString converts interface{} to string (handles "-", numbers, strings)
func convertToString(val interface{}) string {
	if val == nil {
		return "0"
	}

	switch v := val.(type) {
	case string:
		if v == "-" || v == "" {
			return "0"
		}
		return v
	case float64:
		if v == 0 {
			return "0"
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return "0"
	}
}

// convertToFloat converts interface{} to float64 (handles "-", numbers, strings)
func convertToFloat(val interface{}) float64 {
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case string:
		if v == "-" || v == "" {
			return 0
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}
