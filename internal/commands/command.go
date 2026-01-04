package commands

import (
	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type Command interface {
	Execute(database *db.DB, ollamaClient api.OllamaClient) error
	Name() string
	Validate() error
	HelpManual() string
}
