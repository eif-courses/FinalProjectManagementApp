package main

import (
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

	// Register MIME types to fix the MIME type issues
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".js", "application/javascript")

	// Database setup (your existing code)
	dbConfig := database.LoadConfig()
	db, err := dbConfig.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Migration code (your existing code)
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

	// Auth setup (your existing code)
	authService, err := auth.NewAuthService(db)
	if err != nil {
		log.Fatal("Failed to initialize auth service:", err)
	}

	authMiddleware, err := auth.NewAuthMiddleware(authService)
	if err != nil {
		log.Fatal("Failed to initialize auth middleware:", err)
	}

	// NEW: Initialize notification service
	var notificationService *notifications.NotificationService
	if authService.GetAppGraphClient() != nil {
		notificationService = notifications.NewNotificationService(authService.GetAppGraphClient())
		log.Println("Notification service initialized successfully")
	} else {
		log.Println("Warning: App Graph client not available")
		log.Println("Email notifications will be disabled")
		notificationService = nil
	}

	// Setup routes (pass notification service to routes)
	r := routes.SetupRoutes(db, authService, authMiddleware, notificationService)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
