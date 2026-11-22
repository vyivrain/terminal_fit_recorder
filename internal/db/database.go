package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"

	migrationfiles "terminal_fit_recorder/migrations"
)

type DB struct {
	conn *sql.DB
}

// The driver is pure Go (no cgo), so release binaries cross-compile for every
// platform from one machine.
//
// _time_format=sqlite stores time.Time as "2006-01-02 15:04:05.999999999-07:00",
// the format earlier releases wrote and that SQLite's DATE() understands.
// busy_timeout lets the TUI, CLI commands, and the MCP server hold the same
// file open at once, waiting for a competing write instead of failing.
const connectionParams = "_time_format=sqlite&_pragma=busy_timeout(5000)"

func New(dbPath string) (*DB, error) {
	if dbPath == ":memory:" {
		// Shared cache so all connections see the same in-memory database
		dbPath = "file::memory:?cache=shared&" + connectionParams
	} else {
		dbPath = "file:" + dbPath + "?" + connectionParams
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) init() error {
	return db.runMigrations()
}

func (db *DB) runMigrations() error {
	source, err := iofs.New(migrationfiles.Files, ".")
	if err != nil {
		return fmt.Errorf("could not load embedded migrations: %v", err)
	}

	driver, err := sqlite.WithInstance(db.conn, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("could not create sqlite driver: %v", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		source,
		"sqlite",
		driver,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not run migrations: %v", err)
	}

	return nil
}

func (db *DB) GetConn() *sql.DB {
	return db.conn
}
