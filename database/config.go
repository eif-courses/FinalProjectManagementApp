// database/config.go
package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// ===== DATABASE CONFIG =====
type Config struct { // Changed back to original name for compatibility
	Host     string
	Port     string
	Database string
	Username string
	Password string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// LoadConfig loads MySQL configuration from environment variables
func LoadConfig() *Config { // Changed back to original function name
	return &Config{
		Host:     getEnv("MYSQLHOST", "localhost"),
		Port:     getEnv("MYSQLPORT", "3306"),
		Database: getEnv("MYSQLDATABASE", "railway"),
		Username: getEnv("MYSQLUSER", "root"),
		Password: getEnv("MYSQLPASSWORD", ""),

		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
		ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME", 60)) * time.Second,
	}
}

// Connect creates a MySQL database connection
func (c *Config) Connect() (*sqlx.DB, error) {
	dsn := c.buildDSN()

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)
	db.SetConnMaxIdleTime(c.ConnMaxIdleTime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	return db, nil
}

// buildDSN creates MySQL data source name
func (c *Config) buildDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}

// GetMigrationURL returns the database URL for golang-migrate
func (c *Config) GetMigrationURL() string {
	return fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}

// Database interface for easier testing
type Database interface {
	sqlx.Queryer
	sqlx.Execer
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Close() error
	Ping() error
}

// Ensure *sqlx.DB implements Database interface
var _ Database = (*sqlx.DB)(nil)

// ===== APPLICATION CONFIG =====
type AppConfig struct {
	Database *Config
	GitHub   *GitHubConfig // CHANGED: From AzureDevOps to GitHub
	Server   *ServerConfig
}

// CHANGED: Renamed from AzureDevOpsConfig to GitHubConfig
type GitHubConfig struct {
	Organization string
	Project      string // Keep for compatibility, though not used much in GitHub
	PAT          string
}

type ServerConfig struct {
	Port        string
	Environment string
}

// LoadAppConfig loads all application configuration
func LoadAppConfig() *AppConfig {
	config := &AppConfig{
		Database: LoadConfig(), // Use the database config from same package

		// CHANGED: GitHub configuration instead of Azure DevOps
		GitHub: &GitHubConfig{
			Organization: getEnv("GITHUB_ORG", ""),     // CHANGED: From AZURE_ORG
			Project:      getEnv("GITHUB_PROJECT", ""), // CHANGED: From AZURE_PROJECT
			PAT:          getEnv("GITHUB_PAT", ""),     // CHANGED: From AZURE_PAT
		},

		Server: &ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Environment: getEnv("RAILWAY_ENVIRONMENT", "development"),
		},
	}

	// Log configuration (without sensitive data)
	config.logConfig()

	return config
}

// CHANGED: Updated function name and logic for GitHub
func (c *AppConfig) HasGitHub() bool {
	return c.GitHub.Organization != "" && c.GitHub.PAT != ""
}

// CHANGED: Updated logging for GitHub
func (c *AppConfig) logConfig() {
	log.Printf("Configuration loaded:")
	log.Printf("  Environment: %s", c.Server.Environment)
	log.Printf("  Database Host: %s", c.Database.Host)
	log.Printf("  Database Name: %s", c.Database.Database)
	log.Printf("  Server Port: %s", c.Server.Port)

	if c.GitHub.Organization != "" {
		log.Printf("  GitHub Org: %s", c.GitHub.Organization)
		log.Printf("  GitHub: ENABLED")
	} else {
		log.Printf("  GitHub: DISABLED")
	}
}

func (c *AppConfig) IsProduction() bool {
	return c.Server.Environment == "production" || c.Database.Host == "mysql.railway.internal"
}

func (c *AppConfig) IsLocal() bool {
	return c.Database.Host == "localhost" || c.Database.Host == "127.0.0.1"
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
