// database/db.go - Manual migration approach (if automated migrations fail)
package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// InitDatabase initializes the database connection and runs migrations
func InitDatabase(databaseURL string) (*ThesisDB, error) {
	log.Printf("🔄 Initializing database: %s", databaseURL)

	// Connect to database first
	db, err := sqlx.Connect("sqlite3", databaseURL+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run manual migrations
	if err := runManualMigrations(db.DB); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Set connection pool settings for SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	log.Println("✅ Database initialized successfully")

	return NewThesisDB(db), nil
}

func runManualMigrations(db *sql.DB) error {
	// Check if migrations table exists
	var count int
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check schema_migrations table: %w", err)
	}

	// Create schema_migrations table if it doesn't exist
	if count == 0 {
		_, err = db.Exec(`
            CREATE TABLE schema_migrations (
                version INTEGER PRIMARY KEY,
                dirty INTEGER NOT NULL DEFAULT 0
            )
        `)
		if err != nil {
			return fmt.Errorf("failed to create schema_migrations table: %w", err)
		}
		log.Println("🔧 Created schema_migrations table")
	}

	// Check if migration has already been applied
	var migrationCount int
	err = db.QueryRow("SELECT count(*) FROM schema_migrations WHERE version = 1").Scan(&migrationCount)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if migrationCount > 0 {
		log.Println("🔧 Migration already applied - Current version: 1")
		return nil
	}

	// Read and execute migration file
	migrationPath := filepath.Join("migrations", "000001_initial_schema.up.sql")
	log.Printf("🔧 Reading migration file: %s", migrationPath)

	migrationSQL, err := ioutil.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
	}

	// Execute migration
	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Mark migration as applied
	_, err = db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (1, 0)")
	if err != nil {
		return fmt.Errorf("failed to mark migration as applied: %w", err)
	}

	log.Println("🔧 Migration completed - Current version: 1")
	return nil
}

// GetMigrationVersion returns current migration version (manual implementation)
func GetMigrationVersion(databaseURL string) (uint, bool, error) {
	db, err := sql.Open("sqlite3", databaseURL)
	if err != nil {
		return 0, false, err
	}
	defer db.Close()

	var version uint
	var dirty int
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	return version, dirty == 1, nil
}
