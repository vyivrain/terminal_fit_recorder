package testutil

import (
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"terminal_fit_recorder/internal/db"
)

// DBTestSuite provides a shared database setup for tests
type DBTestSuite struct {
	suite.Suite
	DB          *db.DB
	originalWd  string
	projectRoot string
}

// SetupSuite runs once before all tests in the suite
func (s *DBTestSuite) SetupSuite() {
	var err error

	// Get the current working directory
	s.originalWd, err = os.Getwd()
	require.NoError(s.T(), err, "Failed to get working directory")

	// Navigate to project root (migrations are at project root)
	s.projectRoot = s.findProjectRoot(s.originalWd)
	require.NotEmpty(s.T(), s.projectRoot, "Failed to find project root")
}

// TearDownSuite runs once after all tests in the suite
func (s *DBTestSuite) TearDownSuite() {
	os.Chdir(s.originalWd)
}

// SetupTest runs before each test to ensure clean state
func (s *DBTestSuite) SetupTest() {
	// Change to project root for migrations
	err := os.Chdir(s.projectRoot)
	require.NoError(s.T(), err, "Failed to change to project root")

	// Close existing DB if any
	if s.DB != nil {
		s.DB.Close()
	}

	// Create fresh in-memory database for each test
	// Using :memory: creates a private in-memory database for this connection
	s.DB, err = db.New(":memory:")
	require.NoError(s.T(), err, "Failed to create in-memory test database")
}

// TearDownTest runs after each test to clean up
func (s *DBTestSuite) TearDownTest() {
	if s.DB != nil {
		s.DB.Close()
		s.DB = nil
	}
}

// findProjectRoot walks up the directory tree to find the project root
// (directory containing go.mod)
func (s *DBTestSuite) findProjectRoot(startDir string) string {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
}

// CleanupSubtest closes the current DB and creates a fresh one
// Useful for subtests that need isolated database state
func (s *DBTestSuite) CleanupSubtest() {
	if s.DB != nil {
		s.DB.Close()
	}

	var err error
	s.DB, err = db.New(":memory:")
	require.NoError(s.T(), err, "Failed to create fresh in-memory database")
}
