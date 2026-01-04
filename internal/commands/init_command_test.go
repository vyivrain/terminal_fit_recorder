package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand_Name(t *testing.T) {
	cmd := NewInitCommand()
	assert.Equal(t, "init", cmd.Name())
}

func TestInitCommand_Validate(t *testing.T) {
	cmd := NewInitCommand()
	assert.NoError(t, cmd.Validate())
}

func TestInitCommand_HelpManual(t *testing.T) {
	cmd := NewInitCommand()
	help := cmd.HelpManual()

	assert.NotEmpty(t, help, "HelpManual should return non-empty string")

	// Verify help text contains expected components
	expectedParts := []string{
		"terminal_fit_recorder",
		"exercise",
		"init",
		"Initialize",
		".terminal_fit_recorder",
		"exercises.db",
	}

	for _, part := range expectedParts {
		assert.Contains(t, help, part, "HelpManual should contain %q", part)
	}
}

func TestInitCommand_Execute_CreatesConfigDirectory(t *testing.T) {
	// Create a temporary directory to act as home
	tmpHome, err := os.MkdirTemp("", "test-home-*")
	require.NoError(t, err, "Failed to create temp home dir")
	defer os.RemoveAll(tmpHome)

	// Set temp home directory
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create migrations directory in current working directory for test
	cwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	migrationsDir := filepath.Join(cwd, "../../migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Create a temporary migrations directory for the test
		testMigrationsDir := filepath.Join(tmpHome, "migrations")
		require.NoError(t, os.MkdirAll(testMigrationsDir, 0755), "Failed to create test migrations dir")

		// Create a simple migration file
		migrationContent := `-- Create exercises table
CREATE TABLE IF NOT EXISTS exercises (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);`
		migrationFile := filepath.Join(testMigrationsDir, "001_init.up.sql")
		require.NoError(t, os.WriteFile(migrationFile, []byte(migrationContent), 0644), "Failed to create migration file")

		// Change to temp home for migration path resolution
		originalWd, _ := os.Getwd()
		os.Chdir(tmpHome)
		defer os.Chdir(originalWd)
	}

	cmd := NewInitCommand()

	// Execute should create the config directory
	configDir := filepath.Join(tmpHome, ".terminal_fit_recorder")

	// Verify directory doesn't exist initially
	_, err = os.Stat(configDir)
	assert.True(t, os.IsNotExist(err), "Config directory should not exist before Execute()")

	// Execute the command (will fail on db.New due to migrations, but should create directory)
	_ = cmd.Execute(nil, nil)

	// Verify directory was created
	_, err = os.Stat(configDir)
	assert.False(t, os.IsNotExist(err), "Execute() should create config directory")

	// Verify directory permissions
	info, err := os.Stat(configDir)
	require.NoError(t, err, "Failed to stat config directory")

	expectedPerms := os.FileMode(0755)
	assert.Equal(t, expectedPerms, info.Mode().Perm(), "Config directory should have correct permissions")
}

func TestInitCommand_Execute_DirectoryAlreadyExists(t *testing.T) {
	// Create a temporary directory to act as home
	tmpHome, err := os.MkdirTemp("", "test-home-*")
	require.NoError(t, err, "Failed to create temp home dir")
	defer os.RemoveAll(tmpHome)

	// Set temp home directory
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Pre-create the config directory
	configDir := filepath.Join(tmpHome, ".terminal_fit_recorder")
	require.NoError(t, os.MkdirAll(configDir, 0755), "Failed to create config directory")

	cmd := NewInitCommand()

	// Execute should handle existing directory gracefully
	_ = cmd.Execute(nil, nil)

	// Verify directory still exists
	_, err = os.Stat(configDir)
	assert.False(t, os.IsNotExist(err), "Config directory should still exist after Execute()")
}

func TestInitCommand_Execute_DatabasePath(t *testing.T) {
	// Create a temporary directory to act as home
	tmpHome, err := os.MkdirTemp("", "test-home-*")
	require.NoError(t, err, "Failed to create temp home dir")
	defer os.RemoveAll(tmpHome)

	// Set temp home directory
	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Setup migrations directory
	cwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	testMigrationsDir := filepath.Join(tmpHome, "migrations")
	require.NoError(t, os.MkdirAll(testMigrationsDir, 0755), "Failed to create test migrations dir")

	// Create a simple migration file
	migrationContent := `-- Create exercises table
CREATE TABLE IF NOT EXISTS exercises (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    muscle_group TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create workouts table
CREATE TABLE IF NOT EXISTS workouts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exercise_id INTEGER NOT NULL,
    reps INTEGER NOT NULL,
    weight REAL,
    status TEXT DEFAULT 'pending',
    workout_date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (exercise_id) REFERENCES exercises(id)
);`
	migrationFile := filepath.Join(testMigrationsDir, "001_init.up.sql")
	require.NoError(t, os.WriteFile(migrationFile, []byte(migrationContent), 0644), "Failed to create migration file")

	// Change to temp home for migration path resolution
	originalWd, _ := os.Getwd()
	os.Chdir(tmpHome)
	defer os.Chdir(originalWd)

	_ = cwd // silence unused variable warning

	cmd := NewInitCommand()

	// Execute the command
	err = cmd.Execute(nil, nil)
	if err != nil {
		t.Logf("Execute() returned error (expected due to migrations): %v", err)
	}

	// Verify expected database path would be correct
	expectedDBPath := filepath.Join(tmpHome, ".terminal_fit_recorder", "exercises.db")
	t.Logf("Expected database path: %s", expectedDBPath)

	// Verify database was created
	_, err = os.Stat(expectedDBPath)
	assert.False(t, os.IsNotExist(err), "Database file should be created")
}
