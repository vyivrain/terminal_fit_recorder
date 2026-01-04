package ui

// InputProvider interface allows for mocking user input in tests
type InputProvider interface {
	GetInputWithType(prompt string, suggestions []string, inputType InputType) (string, bool)
}

// DefaultInputProvider uses the real UI functions
type DefaultInputProvider struct{}

func (d *DefaultInputProvider) GetInputWithType(prompt string, suggestions []string, inputType InputType) (string, bool) {
	return GetInputWithType(prompt, suggestions, inputType)
}

// NewDefaultInputProvider creates a new default input provider
func NewDefaultInputProvider() InputProvider {
	return &DefaultInputProvider{}
}
