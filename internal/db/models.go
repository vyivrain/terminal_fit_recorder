package db

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	ModelProviderOpenCode = "opencode"
	ModelProviderCodex    = "codex"
	ModelProviderClaude   = "claude"
)

type AIModel struct {
	ID          int
	Provider    string
	ModelID     string
	DisplayName string
	IsSelected  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (db *DB) GetAIModels() ([]AIModel, error) {
	rows, err := db.conn.Query(`
		SELECT id, provider, model_id, display_name, is_selected, created_at, updated_at
		FROM ai_models
		ORDER BY is_selected DESC, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []AIModel
	for rows.Next() {
		var model AIModel
		if err := rows.Scan(&model.ID, &model.Provider, &model.ModelID, &model.DisplayName, &model.IsSelected, &model.CreatedAt, &model.UpdatedAt); err != nil {
			return nil, err
		}
		models = append(models, model)
	}

	return models, rows.Err()
}

func (db *DB) GetSelectedAIModel() (*AIModel, error) {
	var model AIModel
	err := db.conn.QueryRow(`
		SELECT id, provider, model_id, display_name, is_selected, created_at, updated_at
		FROM ai_models
		WHERE is_selected = 1
	`).Scan(&model.ID, &model.Provider, &model.ModelID, &model.DisplayName, &model.IsSelected, &model.CreatedAt, &model.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (db *DB) SelectAIModel(modelID int) (*AIModel, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var model AIModel
	err = tx.QueryRow(`
		SELECT id, provider, model_id, display_name, is_selected, created_at, updated_at
		FROM ai_models
		WHERE id = ?
	`, modelID).Scan(&model.ID, &model.Provider, &model.ModelID, &model.DisplayName, &model.IsSelected, &model.CreatedAt, &model.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("AI model %d does not exist", modelID)
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if _, err := tx.Exec(`UPDATE ai_models SET is_selected = 0, updated_at = ? WHERE is_selected = 1`, now); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE ai_models SET is_selected = 1, updated_at = ? WHERE id = ?`, now, model.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	model.IsSelected = true
	model.UpdatedAt = now
	return &model, nil
}
