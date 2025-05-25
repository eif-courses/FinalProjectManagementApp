package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/routes"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load database config
	dbConfig := database.LoadConfig()

	// Connect to database
	db, err := dbConfig.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run database migrations with dirty state handling
	migrator, err := database.NewMigrator(db, "migrations")
	if err != nil {
		log.Fatal("Failed to create migrator:", err)
	}
	defer migrator.Close()

	log.Println("Checking database migration state...")

	// Check for dirty state and clean if needed
	version, dirty, err := migrator.Version()
	if err != nil && err.Error() != "no migration" {
		log.Printf("Migration version check: %v", err)
	}

	if dirty {
		log.Printf("Database is in dirty state at version %d. Forcing to clean state...", version)
		if err := migrator.Force(0); err != nil {
			log.Fatal("Failed to force clean migration state:", err)
		}
		log.Println("Migration state cleaned successfully")
	}

	log.Println("Running database migrations...")
	if err := migrator.Up(); err != nil {
		log.Fatal("Migration failed:", err)
	}
	log.Println("Database migrations completed successfully")

	// Initialize auth service
	authService, err := auth.NewAuthService(db.DB)
	if err != nil {
		log.Fatal("Failed to initialize auth service:", err)
	}

	// Initialize auth middleware
	authMiddleware, err := auth.NewAuthMiddleware(authService)
	if err != nil {
		log.Fatal("Failed to initialize auth middleware:", err)
	}

	// Setup routes
	r := routes.SetupRoutes(authService, authMiddleware)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
