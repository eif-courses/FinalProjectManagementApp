package main

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/routes"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
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

	// Setup static file serving for CSS and assets
	setupStaticFiles()

	// Setup routes (now includes templUI components)
	r := routes.SetupRoutes(authService, authMiddleware)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090" // Changed to 8090 to match templUI dev setup
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("templUI dev server available at http://localhost:7331")
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// setupStaticFiles configures static file serving for CSS and assets
func setupStaticFiles() {
	// Serve CSS files
	http.Handle("/assets/", http.StripPrefix("/assets/",
		http.FileServer(http.Dir("assets/"))))

	// Serve any other static files if needed
	if _, err := os.Stat("static"); err == nil {
		http.Handle("/static/", http.StripPrefix("/static/",
			http.FileServer(http.Dir("static/"))))
	}
}
