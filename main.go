// main.go
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/handlers"
	"FinalProjectManagementApp/i18n"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv" // Added for .env loading
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Debug environment variables
	debugEnvironmentVariables()

	// Verify critical environment variables are loaded
	requiredEnvVars := []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("❌ Missing required environment variable: %s", envVar)
		}
	}

	log.Println("✅ Environment variables loaded successfully")

	// Initialize database with migrations
	db, err := database.InitDatabase("thesis.db")
	if err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	defer db.DB.Close() // Fixed: Use db.DB.Close() since db is *ThesisDB

	// Initialize i18n
	localizer := i18n.NewLocalizer()
	if err := localizer.LoadTranslations("translations"); err != nil {
		log.Fatal("Failed to load translations:", err)
	}

	// Initialize authentication (this should now work)
	authService := auth.NewAuthService()
	authMiddleware := auth.NewAuthMiddleware(authService)
	log.Println("✅ Authentication service initialized successfully")

	// Initialize templates with i18n functions
	tmpl := handlers.NewTemplateHandlerWithI18n(localizer)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(localizer.LanguageMiddleware)

	// Static files
	fileServer := http.FileServer(http.Dir("./static/"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Language switching
	r.Get("/switch-language", localizer.LanguageSwitchHandler)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Test database connection
		if err := db.DB.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"disconnected","error":"` + err.Error() + `"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","database":"connected"}`))
	})

	// Public routes (no authentication)
	r.Group(func(r chi.Router) {
		r.Get("/", handlers.HomeHandlerWithI18n(tmpl, localizer, authMiddleware))
		r.Get("/auth/login", authMiddleware.LoginHandler)
		r.Get("/auth/callback", authMiddleware.CallbackHandler)
	})

	// Protected routes (authentication required)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// Dashboard
		r.Get("/dashboard", handlers.DashboardHandlerWithI18n(tmpl, localizer))

		// Logout
		r.Get("/auth/logout", authMiddleware.LogoutHandler)
		r.Post("/auth/logout", authMiddleware.LogoutHandler)

		// API
		r.Get("/api/user", handlers.UserInfoAPIHandler)

		// Student routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireRole("student"))
			r.Get("/student/profile", handlers.StudentProfileHandler(tmpl, localizer))
			r.Get("/student/topic/submit", handlers.SubmitTopicHandler(tmpl, localizer))
			r.Post("/student/topic/submit", handlers.SubmitTopicHandler(tmpl, localizer))
		})

		// Supervisor routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireRole("supervisor", "department_head"))
			r.Get("/supervisor/students", handlers.SupervisorStudentsHandler(tmpl, localizer))
			r.Post("/supervisor/report", handlers.CreateSupervisorReportHandler(tmpl, localizer))
		})

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.RequireRole("department_head", "admin"))
			r.Get("/students", handlers.StudentsTableHandlerWithI18n(tmpl, db.DB, localizer))
			r.Post("/admin/approve-topic", handlers.ApproveTopicHandler(tmpl, localizer))
		})

		// API routes
		r.Route("/api", func(r chi.Router) {
			r.Get("/students/search", handlers.SearchStudentsAPIWithI18n(db, localizer))            // Fixed: pass db directly
			r.Get("/students", handlers.GetStudentsAPIWithI18n(db, localizer))                      // Fixed: pass db directly
			r.Post("/supervisor-report", handlers.CreateSupervisorReportAPIWithI18n(db, localizer)) // Fixed: pass db directly
		})
	})

	log.Println("🌍 Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// debugEnvironmentVariables prints environment variables for debugging
func debugEnvironmentVariables() {
	log.Println("=== Environment Variables Debug ===")

	envVars := []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
		"AZURE_REDIRECT_URI",
		"SESSION_SECRET",
	}

	for _, envVar := range envVars {
		value := os.Getenv(envVar)
		if value == "" {
			log.Printf("❌ %s: NOT SET", envVar)
		} else {
			// Don't log sensitive values in full
			if strings.Contains(envVar, "SECRET") || strings.Contains(envVar, "CLIENT_SECRET") {
				log.Printf("✅ %s: SET (length: %d)", envVar, len(value))
			} else {
				log.Printf("✅ %s: %s", envVar, value)
			}
		}
	}
	log.Println("=== End Debug ===")
}
