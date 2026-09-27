package db_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func TestAIModelsAreSeededAndSelectable(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "models.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	models, err := database.GetAIModels()
	require.NoError(t, err)
	require.Len(t, models, 4)
	require.Equal(t, "opencode-go/glm-5.3", models[0].ModelID)
	require.True(t, models[0].IsSelected)

	var deepSeek db.AIModel
	for _, model := range models {
		if model.ModelID == "opencode-go/deepseek-v4-flash" {
			deepSeek = model
		}
	}
	require.Equal(t, db.ModelProviderOpenCode, deepSeek.Provider, "DeepSeek V4 Flash should be seeded")
	require.Equal(t, "DeepSeek V4 Flash · OpenCode", deepSeek.DisplayName)
	require.False(t, deepSeek.IsSelected, "adding a model must not change the current selection")

	var codexModel db.AIModel
	for _, model := range models {
		if model.Provider == db.ModelProviderCodex {
			codexModel = model
		}
	}
	require.Equal(t, "gpt-5.6-sol", codexModel.ModelID)

	selected, err := database.SelectAIModel(codexModel.ID)
	require.NoError(t, err)
	require.Equal(t, "gpt-5.6-sol", selected.ModelID)

	selected, err = database.GetSelectedAIModel()
	require.NoError(t, err)
	require.Equal(t, codexModel.ID, selected.ID)

	_, err = database.SelectAIModel(9999)
	require.ErrorContains(t, err, "does not exist")
	selected, err = database.GetSelectedAIModel()
	require.NoError(t, err)
	require.Equal(t, codexModel.ID, selected.ID, "failed selection must preserve the current model")
}

func TestAIModelMigrationCanRollBack(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "model-rollback.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	downMigration, err := os.ReadFile("migrations/009_add_ai_models.down.sql")
	require.NoError(t, err)
	_, err = database.GetConn().Exec(string(downMigration))
	require.NoError(t, err)

	var modelTables int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'ai_models'`).Scan(&modelTables))
	require.Zero(t, modelTables)
}
