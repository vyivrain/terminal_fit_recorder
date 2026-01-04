package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type InitCommand struct{}

func NewInitCommand() *InitCommand {
	return &InitCommand{}
}

func (cmd *InitCommand) Name() string {
	return "init"
}

func (cmd *InitCommand) Validate() error {
	return nil
}

func (cmd *InitCommand) HelpManual() string {
	return "terminal_fit_recorder exercise init\n    Initialize the database in ~/.terminal_fit_recorder/exercises.db"
}

func (cmd *InitCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %v", err)
	}

	// Create config directory path
	configDir := filepath.Join(homeDir, ".terminal_fit_recorder")

	// Check if config directory exists, if not create it
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}
		fmt.Printf("Created config directory: %s\n", configDir)
	} else {
		fmt.Printf("Config directory already exists: %s\n", configDir)
	}

	// Database path
	dbPath := filepath.Join(configDir, "exercises.db")

	// Check if database already exists
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Printf("Database already exists at: %s\n", dbPath)
		fmt.Println("Running migrations to ensure schema is up to date...")
	} else {
		fmt.Printf("Creating new database at: %s\n", dbPath)
	}

	// Initialize/migrate the database
	exerciseDB, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer exerciseDB.Close()

	fmt.Println("Database initialized successfully!")
	return nil
}
