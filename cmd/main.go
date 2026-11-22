package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/commands"
	"terminal_fit_recorder/internal/config"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/mcpserver"
	"terminal_fit_recorder/internal/tui"
)

func main() {
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
	if len(os.Args) == 1 {
		database, err := db.New(dbPath)
		if err != nil {
			log.Fatal("Failed to initialize database:", err)
		}
		defer database.Close()

		if err := tui.Run(database, ai.NewCLIGenerator()); err != nil {
			log.Fatal("TUI failed:", err)
		}
		return
	}

	// MCP stdio server for AI agents. stdout carries the protocol, so nothing
	// else may print to it; like the TUI, it creates the database on demand.
	if len(os.Args) == 2 && os.Args[1] == "mcp" {
		database, err := db.New(dbPath)
		if err != nil {
			log.Fatal("Failed to initialize database:", err)
		}
		defer database.Close()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := mcpserver.Run(ctx, database); err != nil && ctx.Err() == nil {
			log.Fatal("MCP server failed:", err)
		}
		return
	}

	// Explicit commands remain available for scripts and existing workflows.
	cfg := config.Load()
	ollamaClient := api.NewClient(cfg.OllamaHost, cfg.OllamaModel, cfg.OllamaAPIKey, cfg.OllamaPrompt)

	// Parse command first to check if it's init
	cmd, err := commands.ParseArgs(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Help needs no database, so it works before 'exercise init'
	if cmd.Name() == "help" {
		cmd.Execute(nil, ollamaClient)
		return
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
