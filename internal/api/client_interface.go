package api

import "context"

// OllamaClient interface defines the methods needed for Ollama API interactions
// This interface allows for mocking in tests
type OllamaClient interface {
	SendPromptStream(ctx context.Context, prompt string) (string, error)
	SendPrompt(ctx context.Context, prompt string) (string, error)
	GetCustomPrompt() string
}

// GetCustomPrompt returns the custom prompt from the client
func (c *Client) GetCustomPrompt() string {
	return c.CustomPrompt
}
