// main.go - Updated with fixed middlewares
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-sql-driver/mysql" // Add this MySQL driver import
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/handlers"
	"FinalProjectManagementApp/i18n"
	custommiddleware "FinalProjectManagementApp/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Connect to MySQL database
	dbConfig := database.LoadConfig()
	db, err := dbConfig.Connect()
	if err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}
	defer db.Close()

	// Run database migrations
	if err := runMigrations(dbConfig); err != nil {
		log.Printf("Migration warning: %v", err)
	}

	// Initialize i18n
	localizer := i18n.NewLocalizer()
	if err := localizer.LoadTranslations("i18n/translations"); err != nil {
		log.Printf("Failed to load translations: %v", err)
	}

	// Initialize services
	authService, err := auth.NewAuthService(db.DB)
	if err != nil {
		log.Fatal("Failed to create auth service:", err)
	}

	authMiddleware, err := auth.NewAuthMiddleware(authService)
	if err != nil {
		log.Fatal("Failed to create auth middleware:", err)
	}

	commissionService := auth.NewCommissionAccessService(db.DB)
	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	commissionHandler := handlers.NewCommissionHandler(commissionService, authMiddleware, baseURL)

	// Set up Chi router
	r := setupRouter(authMiddleware, commissionHandler, commissionService, localizer)

	// Configure server
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("🚀 Server starting on port %s", port)
		log.Printf("🌍 Environment: %s", getEnv("ENV", "development"))
		log.Printf("🗄️  Database: MySQL (%s)", dbConfig.Host)
		log.Printf("🔗 Base URL: %s", baseURL)
		log.Printf("🌐 Supported languages: %v", localizer.GetSupportedLanguages())

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("✅ Server exited gracefully")
}

func runMigrations(config *database.Config) error {
	// Build the database URL for migrations
	databaseURL := config.GetMigrationURL()

	// Create migrations instance using the simpler New function
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Migrations completed successfully")
	return nil
}

// Remove the mustLoadMigrationSource function - it's not needed

func setupRouter(authMiddleware *auth.AuthMiddleware, commissionHandler *handlers.CommissionHandler, commissionService *auth.CommissionAccessService, localizer *i18n.Localizer) chi.Router {
	r := chi.NewRouter()

	// Core Chi middleware (order matters!)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommiddleware.RecoveryMiddleware) // Custom recovery with better error handling
	r.Use(custommiddleware.LoggingMiddleware)  // Custom logging
	r.Use(middleware.Compress(5))

	// Security middleware
	r.Use(custommiddleware.SecurityHeadersMiddleware)

	// Rate limiting (adjust as needed)
	if getEnv("ENV", "development") == "production" {
		r.Use(custommiddleware.RateLimitMiddleware(100)) // 100 requests per minute
	}

	// CORS middleware - only if you need API access from other domains
	// For full-stack Go apps, you usually don't need this unless you have:
	// - Separate frontend applications (React, Vue, etc.)
	// - Mobile apps making API calls
	// - Third-party integrations
	if needsCORS() {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   getAllowedOrigins(),
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
			ExposedHeaders:   []string{"Link", "X-Request-ID"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Internationalization middleware
	r.Use(localizer.RequestLocalizationMiddleware)

	// Cache control middleware
	r.Use(custommiddleware.CacheControlMiddleware)

	// Maintenance mode check
	r.Use(custommiddleware.MaintenanceMiddleware)

	// Timeout middleware for long-running requests
	r.Use(custommiddleware.TimeoutMiddleware(30 * time.Second))

	// Health check (no auth required)
	r.Get("/health", healthCheckHandler)

	// Language switching endpoint
	r.Post("/switch-language", localizer.LanguageSwitchHandler)
	r.Get("/switch-language", localizer.LanguageSwitchHandler)

	// Static files with proper caching
	fileServer := http.FileServer(http.Dir("./static/"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public commission access (no auth required, but commission middleware applies)
	r.Route("/commission", func(r chi.Router) {
		r.Use(commissionService.CommissionAccessMiddleware)
		r.Get("/{accessCode}", commissionHandler.CommissionViewHandler)
		r.Get("/{accessCode}/*", commissionHandler.CommissionViewHandler)
	})

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", authMiddleware.LoginHandler)
		r.Get("/callback", authMiddleware.CallbackHandler)
		r.Get("/logout", authMiddleware.LogoutHandler)
		r.Post("/logout", authMiddleware.LogoutHandler)
		r.Get("/user", authMiddleware.UserInfoHandler)
	})

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// Dashboard
		r.Get("/", dashboardHandler)
		r.Get("/dashboard", dashboardHandler)

		// Student routes
		r.Route("/students", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleStudent))
			r.Get("/", studentDashboardHandler)
			r.Get("/profile", studentProfileHandler)
			r.Post("/profile", studentProfileHandler)
			r.Get("/topic", studentTopicHandler)
			r.Post("/topic", studentTopicHandler)
			r.Get("/documents", studentDocumentsHandler)
			r.Post("/documents", studentDocumentsHandler)
		})

		// Supervisor routes
		r.Route("/supervisor", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleSupervisor))
			r.Get("/", supervisorDashboardHandler)
			r.Get("/students", supervisorStudentsHandler)
			r.Get("/reports", supervisorReportsHandler)
			r.Post("/reports", supervisorReportsHandler)
			r.Get("/reports/{studentID}", supervisorCreateReportHandler)
			r.Post("/reports/{studentID}", supervisorCreateReportHandler)
		})

		// Admin routes (department heads and admins)
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleDepartmentHead, auth.RoleAdmin))

			// Student management
			r.Get("/students", adminStudentsHandler)
			r.Get("/students/search", adminStudentsSearchHandler)
			r.Get("/topics", adminTopicsHandler)
			r.Post("/topics/{topicID}/approve", adminApproveTopicHandler)
			r.Get("/reports", adminReportsHandler)

			// Commission access management
			r.Get("/commission", commissionHandler.ListCommissionAccessesHandler)
			r.Get("/commission/create", commissionHandler.CreateCommissionAccessHandler)
			r.Post("/commission/create", commissionHandler.CreateCommissionAccessHandler)
			r.Post("/commission/{accessCode}/deactivate", commissionHandler.DeactivateAccessHandler)
		})

		// System admin routes
		r.Route("/system", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleAdmin))
			r.Get("/users", systemUsersHandler)
			r.Post("/users/{email}/role", systemUpdateUserRoleHandler)
			r.Get("/department-heads", systemDepartmentHeadsHandler)
			r.Post("/department-heads", systemDepartmentHeadsHandler)
			r.Get("/audit-logs", systemAuditLogsHandler)
		})

		// API routes for HTMX/AJAX
		r.Route("/api", func(r chi.Router) {
			// Student API
			r.Get("/students/search", apiStudentsSearchHandler)
			r.Get("/students/{studentID}", apiStudentDetailsHandler)
			r.Get("/students/{studentID}/reports", apiStudentReportsHandler)

			// Topic API
			r.Get("/topics/{topicID}", apiTopicDetailsHandler)
			r.Post("/topics/{topicID}/approve", func(w http.ResponseWriter, r *http.Request) {
				user := auth.GetUserFromContext(r.Context())
				if !user.HasPermission(auth.PermissionApproveTopics) {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				apiApproveTopicHandler(w, r)
			})

			// Reports API
			r.Get("/reports", apiReportsHandler)
			r.Post("/reports", apiCreateReportHandler)
			r.Get("/reports/{reportID}", apiReportDetailsHandler)
		})
	})

	return r
}

// Helper functions

// needsCORS determines if CORS middleware is needed
func needsCORS() bool {
	// Enable CORS if:
	// 1. Explicitly enabled via environment variable
	if getEnv("ENABLE_CORS", "false") == "true" {
		return true
	}

	// 2. Development environment (for testing with different ports)
	if getEnv("ENV", "development") == "development" {
		return true
	}

	// 3. API-only mode
	if getEnv("API_ONLY", "false") == "true" {
		return true
	}

	// For traditional server-side rendered apps, CORS is usually not needed
	return false
}

func getAllowedOrigins() []string {
	origins := []string{"http://localhost:8080"} // Always allow self

	// Add additional origins based on environment
	if getEnv("ENV", "development") == "development" {
		origins = append(origins,
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:8080",
		)
	}

	// Add production origins from environment
	if prodOrigins := getEnv("ALLOWED_ORIGINS", ""); prodOrigins != "" {
		origins = append(origins, strings.Split(prodOrigins, ",")...)
	}

	return origins
}

// Handler functions
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"status": "healthy",
		"timestamp": "%s",
		"service": "thesis-management",
		"version": "%s",
		"environment": "%s"
	}`,
		time.Now().Format(time.RFC3339),
		getEnv("APP_VERSION", "1.0.0"),
		getEnv("ENV", "development"),
	)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Get localized template data
	templateData := getLocalizedTemplateData(r, "dashboard.title", user, nil)

	// Redirect based on user role
	switch user.Role {
	case auth.RoleStudent:
		http.Redirect(w, r, "/students/", http.StatusFound)
	case auth.RoleSupervisor:
		http.Redirect(w, r, "/supervisor/", http.StatusFound)
	case auth.RoleDepartmentHead, auth.RoleAdmin:
		http.Redirect(w, r, "/admin/students", http.StatusFound)
	default:
		renderTemplate(w, "dashboard.html", templateData)
	}
}

// Student handlers
func studentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	templateData := getLocalizedTemplateData(r, "student_management.title", user, nil)
	renderTemplate(w, "student_dashboard.html", templateData)
}

func studentProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle profile update
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// TODO: Implement profile update logic
		// For now, just redirect with success
		http.Redirect(w, r, "/students/profile?success=1", http.StatusFound)
		return
	}

	templateData := getLocalizedTemplateData(r, "navigation.profile", user, map[string]interface{}{
		"Success": r.URL.Query().Get("success") == "1",
	})
	renderTemplate(w, "student_profile.html", templateData)
}

func studentTopicHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle topic submission
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// TODO: Implement topic submission logic
		// Validate form data, save to database, etc.

		http.Redirect(w, r, "/students/topic?success=1", http.StatusFound)
		return
	}

	templateData := getLocalizedTemplateData(r, "topic_management.title", user, map[string]interface{}{
		"Success": r.URL.Query().Get("success") == "1",
	})
	renderTemplate(w, "student_topic.html", templateData)
}

func studentDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle document upload
		// Parse multipart form
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB limit
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// TODO: Implement document upload logic
		// Handle file upload, virus scanning, storage, etc.

		http.Redirect(w, r, "/students/documents?success=1", http.StatusFound)
		return
	}

	templateData := getLocalizedTemplateData(r, "documents.title", user, map[string]interface{}{
		"Success": r.URL.Query().Get("success") == "1",
	})
	renderTemplate(w, "student_documents.html", templateData)
}

// Supervisor handlers
func supervisorDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	templateData := getLocalizedTemplateData(r, "dashboard.supervisor_dashboard", user, nil)
	renderTemplate(w, "supervisor_dashboard.html", templateData)
}

func supervisorStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch students assigned to this supervisor
	students := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "student_management.title", user, map[string]interface{}{
		"Students": students,
	})
	renderTemplate(w, "supervisor_students.html", templateData)
}

func supervisorReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch reports for this supervisor's students
	reports := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "reports.title", user, map[string]interface{}{
		"Reports": reports,
	})
	renderTemplate(w, "supervisor_reports.html", templateData)
}

func supervisorCreateReportHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	studentID := chi.URLParam(r, "studentID")

	if r.Method == "POST" {
		// Handle report creation
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// TODO: Implement report creation logic
		// Validate form data, save to database

		http.Redirect(w, r, "/supervisor/reports?created="+studentID, http.StatusFound)
		return
	}

	// TODO: Fetch student details
	templateData := getLocalizedTemplateData(r, "reports.create_report", user, map[string]interface{}{
		"StudentID": studentID,
		"Student":   nil, // TODO: Load student data
	})
	renderTemplate(w, "supervisor_create_report.html", templateData)
}

// Admin handlers
func adminStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch students with filtering/pagination
	students := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "student_management.title", user, map[string]interface{}{
		"Students": students,
	})
	renderTemplate(w, "admin_students.html", templateData)
}

func adminStudentsSearchHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement search logic with proper filtering
	query := r.URL.Query().Get("q")
	department := r.URL.Query().Get("department")
	program := r.URL.Query().Get("program")

	// Placeholder response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"students": [],
		"total": 0,
		"query": "%s",
		"filters": {"department": "%s", "program": "%s"}
	}`, query, department, program)
}

func adminTopicsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch topics pending approval
	topics := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "topic_management.title", user, map[string]interface{}{
		"Topics": topics,
	})
	renderTemplate(w, "admin_topics.html", templateData)
}

func adminApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	topicID := chi.URLParam(r, "topicID")

	if !user.CanApproveTopics() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if r.Method == "POST" {
		// Parse approval data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		approved := r.FormValue("approved") == "true"
		comments := r.FormValue("comments")

		// TODO: Implement topic approval logic
		log.Printf("Topic %s %s by %s: %s", topicID,
			map[bool]string{true: "approved", false: "rejected"}[approved],
			user.Email, comments)

		// Return appropriate response
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Trigger", "topicApproved")
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"success": true, "approved": %t}`, approved)
		} else {
			http.Redirect(w, r, "/admin/topics?approved="+topicID, http.StatusFound)
		}
		return
	}

	// Show approval form
	templateData := getLocalizedTemplateData(r, "topic_management.approve_topic", user, map[string]interface{}{
		"TopicID": topicID,
		"Topic":   nil, // TODO: Load topic data
	})
	renderTemplate(w, "admin_approve_topic.html", templateData)
}

func adminReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch all reports with filtering
	reports := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "reports.title", user, map[string]interface{}{
		"Reports": reports,
	})
	renderTemplate(w, "admin_reports.html", templateData)
}

// System handlers
func systemUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch all users
	users := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "system.user_management", user, map[string]interface{}{
		"Users": users,
	})
	renderTemplate(w, "system_users.html", templateData)
}

func systemUpdateUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	email := chi.URLParam(r, "email")

	if !user.IsAdmin() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// TODO: Implement user role update logic
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	newRole := r.FormValue("role")
	log.Printf("Updating user %s role to %s by %s", email, newRole, user.Email)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "email": "%s", "role": "%s"}`, email, newRole)
}

func systemDepartmentHeadsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle adding new department head
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// TODO: Implement department head creation
		email := r.FormValue("email")
		log.Printf("Adding department head %s by %s", email, user.Email)

		http.Redirect(w, r, "/system/department-heads?added="+email, http.StatusFound)
		return
	}

	// TODO: Fetch department heads
	departmentHeads := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "system.department_heads", user, map[string]interface{}{
		"DepartmentHeads": departmentHeads,
		"Added":           r.URL.Query().Get("added"),
	})
	renderTemplate(w, "system_department_heads.html", templateData)
}

func systemAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// TODO: Fetch audit logs with pagination
	auditLogs := []interface{}{} // Placeholder

	templateData := getLocalizedTemplateData(r, "system.audit_logs", user, map[string]interface{}{
		"AuditLogs": auditLogs,
	})
	renderTemplate(w, "system_audit_logs.html", templateData)
}

// API handlers (placeholder implementations)
func apiStudentsSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Implement actual search
	w.Write([]byte(`{"students": [], "total": 0}`))
}

func apiStudentDetailsHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	w.Header().Set("Content-Type", "application/json")
	// TODO: Fetch actual student data
	fmt.Fprintf(w, `{"id": "%s", "name": "Student Name"}`, studentID)
}

func apiStudentReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Fetch student reports
	w.Write([]byte(`{"reports": []}`))
}

func apiTopicDetailsHandler(w http.ResponseWriter, r *http.Request) {
	topicID := chi.URLParam(r, "topicID")
	w.Header().Set("Content-Type", "application/json")
	// TODO: Fetch actual topic data
	fmt.Fprintf(w, `{"id": "%s", "title": "Topic Title"}`, topicID)
}

func apiApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	topicID := chi.URLParam(r, "topicID")
	w.Header().Set("Content-Type", "application/json")
	// TODO: Implement approval logic
	fmt.Fprintf(w, `{"success": true, "topic_id": "%s"}`, topicID)
}

func apiReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Fetch reports
	w.Write([]byte(`{"reports": []}`))
}

func apiCreateReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: Create report
	w.Write([]byte(`{"success": true}`))
}

func apiReportDetailsHandler(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportID")
	w.Header().Set("Content-Type", "application/json")
	// TODO: Fetch actual report data
	fmt.Fprintf(w, `{"id": "%s", "content": "Report content"}`, reportID)
}

// Helper functions

func getLocalizedTemplateData(r *http.Request, titleKey string, user *auth.AuthenticatedUser, data interface{}) map[string]interface{} {
	// Get localizer from context or create basic one
	lang := i18n.GetLangFromContext(r.Context())
	if lang == "" {
		lang = i18n.DefaultLang
	}

	// This is a simplified version - in production you'd get the localizer from context
	localizer := i18n.NewLocalizer()

	return map[string]interface{}{
		"Title": localizer.T(lang, titleKey),
		"User":  user,
		"Data":  data,
		"Lang":  lang,
		"T": func(key string, args ...interface{}) string {
			return localizer.T(lang, key, args...)
		},
	}
}

func renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	// This is a placeholder implementation
	// In production, you'd use your actual template engine (html/template, Gin, etc.)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	dataMap := data.(map[string]interface{})
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="%s">
	<head>
		<title>%s</title>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<script src="https://cdn.tailwindcss.com"></script>
	</head>
	<body class="bg-gray-100 min-h-screen">
		<div class="container mx-auto px-4 py-8">
			<h1 class="text-3xl font-bold mb-6">%s</h1>
			<div class="bg-white rounded-lg shadow p-6">
				<p>Template: <code>%s</code></p>
				<pre class="mt-4 bg-gray-50 p-4 rounded text-sm overflow-auto">%+v</pre>
			</div>
		</div>
	</body>
</html>`,
		dataMap["Lang"],
		dataMap["Title"],
		dataMap["Title"],
		templateName,
		data,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
