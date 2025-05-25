package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

type Migrator struct {
	db      *sqlx.DB
	migrate *migrate.Migrate
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sqlx.DB, migrationsPath string) (*Migrator, error) {
	// Create MySQL driver
	driver, err := mysql.WithInstance(db.DB, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql driver: %w", err)
	}

	// Convert to absolute path and verify it exists
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("migrations directory does not exist: %s", absPath)
	}

	log.Printf("Using migrations path: %s", absPath)

	// List migration files for debugging
	files, err := filepath.Glob(filepath.Join(absPath, "*.sql"))
	if err != nil {
		log.Printf("Warning: Could not list migration files: %v", err)
	} else {
		log.Printf("Found migration files: %v", files)
	}

	// Create file source instance directly
	fileSource := &file.File{}
	sourceDriver, err := fileSource.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open migrations directory %s: %w", absPath, err)
	}

	// Create migrate instance with source and database drivers
	m, err := migrate.NewWithInstance("file", sourceDriver, "mysql", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		db:      db,
		migrate: m,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if err := m.migrate.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations up: %w", err)
	}
	log.Println("Migrations completed successfully")
	return nil
}

// Down rolls back one migration
func (m *Migrator) Down() error {
	if err := m.migrate.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migration down: %w", err)
	}
	log.Println("Migration rolled back successfully")
	return nil
}

// Steps runs n migrations (positive for up, negative for down)
func (m *Migrator) Steps(n int) error {
	if err := m.migrate.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run %d migration steps: %w", n, err)
	}
	log.Printf("Successfully ran %d migration steps", n)
	return nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}
	log.Printf("Forced migration to version %d", version)
	return nil
}

// Version returns current migration version
func (m *Migrator) Version() (uint, bool, error) {
	return m.migrate.Version()
}

// Close closes the migrator
func (m *Migrator) Close() error {
	sourceErr, dbErr := m.migrate.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return dbErr
}
