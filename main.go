// main.go
package main

import (
	"FinalProjectManagementApp/handlers"
	"FinalProjectManagementApp/i18n"
	"FinalProjectManagementApp/notifications"
	"github.com/joho/godotenv"
	"log"
	"mime"
	"net/http"
	"os"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/routes"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Load environment variables from .env file
	translator := i18n.GetTranslator()
	if err := translator.LoadTranslations(); err != nil {
		log.Fatal("Failed to load translations:", err)
	}

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Register MIME types
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".js", "application/javascript")

	// Load application configuration
	appConfig := database.LoadAppConfig()

	// Database setup using new config structure
	db, err := appConfig.Database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Migration code
	migrator, err := database.NewMigrator(db, "migrations")
	if err != nil {
		log.Fatal("Failed to create migrator:", err)
	}
	defer migrator.Close()

	// Migration logic (your existing code)
	log.Println("Checking database migration state...")
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

	// Auth setup
	authService, err := auth.NewAuthService(db)
	if err != nil {
		log.Fatal("Failed to initialize auth service:", err)
	}

	authMiddleware, err := auth.NewAuthMiddleware(authService)
	if err != nil {
		log.Fatal("Failed to initialize auth middleware:", err)
	}

	// Notification service
	var notificationService *notifications.NotificationService
	if authService.GetAppGraphClient() != nil {
		notificationService = notifications.NewNotificationService(authService.GetAppGraphClient())
		log.Println("Notification service initialized successfully")
	} else {
		log.Println("Warning: App Graph client not available")
		log.Println("Email notifications will be disabled")
		notificationService = nil
	}

	// Initialize source code upload handler
	var sourceCodeHandler *handlers.SourceCodeHandler
	if appConfig.HasGitHub() {
		sourceCodeHandler = handlers.NewSourceCodeHandler(db, appConfig.GitHub)
	} else {
		log.Println("Github configuration not found - source code upload will be disabled")
		sourceCodeHandler = nil
	}

	// Create uploads directory
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Printf("Warning: Failed to create uploads directory: %v", err)
		log.Println("Source code uploads may not work properly")
	} else {
		log.Println("Uploads directory created/verified successfully")
	}

	// Setup routes
	r := routes.SetupRoutes(db, authService, authMiddleware, notificationService, sourceCodeHandler)

	// Get port from environment or use config
	port := appConfig.Server.Port

	// Log startup information
	log.Printf("=== Final Project Management App Starting ===")
	log.Printf("Server starting on port %s", port)
	log.Printf("Environment: %s", appConfig.Server.Environment)
	log.Printf("Database: %s@%s:%s/%s",
		appConfig.Database.Username, appConfig.Database.Host,
		appConfig.Database.Port, appConfig.Database.Database)

	if sourceCodeHandler != nil {
		log.Printf("Github API active integration: ENABLED (%s/%s)",
			appConfig.GitHub.Organization, appConfig.GitHub.Project)
	} else {
		log.Printf("Github API integration: DISABLED")
	}

	if notificationService != nil {
		log.Printf("Email notifications: ENABLED")
	} else {
		log.Printf("Email notifications: DISABLED")
	}

	log.Printf("=== Server Ready ===")

	// Start the server
	if err := http.ListenAndServe("0.0.0.0:"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
