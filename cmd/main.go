package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/commands"
	"terminal_fit_recorder/internal/config"
	"terminal_fit_recorder/internal/db"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize Ollama client
	ollamaClient := api.NewClient(cfg.OllamaHost, cfg.OllamaModel, cfg.OllamaPrompt)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get user home directory:", err)
	}

	// Create config directory path
	configDir := filepath.Join(homeDir, ".terminal_fit_recorder")

	// Check if config directory exists, if not create it
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Fatal("Failed to create config directory:", err)
		}
	}

	// Database path
	dbPath := filepath.Join(configDir, "exercises.db")

	// Parse command first to check if it's init
	cmd, err := commands.ParseArgs(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Check if database exists
	dbExists := false
	if _, err := os.Stat(dbPath); err == nil {
		dbExists = true
	}

	// If database doesn't exist and command is not init, show error
	if !dbExists && cmd.Name() != "init" {
		fmt.Println("Database not found. Please run 'terminal_fit_recorder exercise init' first to initialize the database.")
		os.Exit(1)
	}

	// Initialize database (for init command or if it already exists)
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nExiting...")
		database.Close()
		os.Exit(0)
	}()

	err = cmd.Execute(database, ollamaClient)
	if err != nil {
		log.Fatal("Error executing command:", err)
	}
}
