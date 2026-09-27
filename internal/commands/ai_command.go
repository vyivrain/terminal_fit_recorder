package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

// aiTestTimeout bounds a diagnostic call so a hung provider cannot block forever.
// Output streams live, so a stall is visible immediately and Ctrl-C aborts sooner.
const aiTestTimeout = 2 * time.Minute

// AITestCommand runs one AI request with full observability: the exact command,
// live output, timing, and a classified outcome. It is the tool for answering
// "why did that AI call hang / fail?" without guessing.
type AITestCommand struct {
	args    []string
	modelID string
}

func NewAITestCommand(args ...string) *AITestCommand {
	return &AITestCommand{args: args}
}

func (cmd *AITestCommand) Name() string { return "ai test" }

func (cmd *AITestCommand) Validate() error {
	if len(cmd.args) > 1 {
		return fmt.Errorf("usage: terminal_fit_recorder ai test [model-id]")
	}
	if len(cmd.args) == 1 {
		cmd.modelID = strings.TrimSpace(cmd.args[0])
	}
	return nil
}

func (cmd *AITestCommand) HelpManual() string {
	return "terminal_fit_recorder ai test [model-id]\n    Run one diagnostic AI call with full observability: the exact command, live streamed output, timing, and outcome. Defaults to the selected model; pass a model id to test another without changing your selection."
}

func (cmd *AITestCommand) Execute(database *db.DB, _ api.OllamaClient) error {
	model, err := cmd.resolveModel(database)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), aiTestTimeout)
	defer cancel()

	fmt.Printf("Testing %s  (%s / %s)\n", modelLabel(model), model.Provider, model.ModelID)
	fmt.Printf("Timeout %s — press Ctrl-C to abort early.\n", aiTestTimeout)
	fmt.Println("\n─── live output ────────────────────────────────────────")

	diag := ai.NewCLIGenerator().TestCall(ctx, model, "", os.Stdout)

	fmt.Println("\n─── report ─────────────────────────────────────────────")
	fmt.Print(diag.Render())
	return nil
}

// resolveModel picks the model to test: the one named on the command line, or the
// currently selected model when none is given.
func (cmd *AITestCommand) resolveModel(database *db.DB) (db.AIModel, error) {
	models, err := database.GetAIModels()
	if err != nil {
		return db.AIModel{}, fmt.Errorf("could not load AI models: %w", err)
	}
	if len(models) == 0 {
		return db.AIModel{}, fmt.Errorf("no AI models are configured")
	}

	if cmd.modelID == "" {
		for _, model := range models {
			if model.IsSelected {
				return model, nil
			}
		}
		return models[0], nil
	}

	for _, model := range models {
		if model.ModelID == cmd.modelID {
			return model, nil
		}
	}
	return db.AIModel{}, fmt.Errorf("no AI model has id %q (run without an id to use the selected model)", cmd.modelID)
}

func modelLabel(model db.AIModel) string {
	if strings.TrimSpace(model.DisplayName) != "" {
		return model.DisplayName
	}
	return model.ModelID
}
