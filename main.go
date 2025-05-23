// main.go - Chi router with MySQL
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
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/handlers"
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
	r := setupRouter(authMiddleware, commissionHandler, commissionService)

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
		log.Printf("🌍 Environment: %s", getEnv("ENVIRONMENT", "development"))
		log.Printf("🗄️  Database: MySQL (%s)", dbConfig.Host)
		log.Printf("🔗 Base URL: %s", baseURL)

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
	db, err := config.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := mysql.WithInstance(db.DB, &mysql.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("file", mustLoadMigrationSource(), "mysql", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Migrations completed successfully")
	return nil
}

func mustLoadMigrationSource() migrate.Source {
	source, err := (&migrate.FileMigrationSource{}).Open("file://migrations")
	if err != nil {
		log.Fatal("Failed to load migration source:", err)
	}
	return source
}

func setupRouter(authMiddleware *auth.AuthMiddleware, commissionHandler *handlers.CommissionHandler, commissionService *auth.CommissionAccessService) chi.Router {
	r := chi.NewRouter()

	// Chi middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Custom middleware
	r.Use(securityHeadersMiddleware)

	// Health check
	r.Get("/health", healthCheckHandler)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Public commission access (no auth required)
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
			r.Use(authMiddleware.RequirePermission(auth.PermissionViewOwnData))
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
			r.Use(authMiddleware.RequirePermission(auth.PermissionViewAssignedStudents))
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

// Middleware functions
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		if getEnv("ENVIRONMENT", "development") == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// Handler functions
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","service":"thesis-management"}`, time.Now().Format(time.RFC3339))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Redirect based on user role
	switch user.Role {
	case auth.RoleStudent:
		http.Redirect(w, r, "/students/", http.StatusFound)
	case auth.RoleSupervisor:
		http.Redirect(w, r, "/supervisor/", http.StatusFound)
	case auth.RoleDepartmentHead, auth.RoleAdmin:
		http.Redirect(w, r, "/admin/students", http.StatusFound)
	default:
		renderTemplate(w, "dashboard.html", map[string]interface{}{
			"User":  user,
			"Title": "Dashboard",
		})
	}
}

// Student handlers
func studentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "student_dashboard.html", map[string]interface{}{
		"User":  user,
		"Title": "Student Dashboard",
	})
}

func studentProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle profile update
		// TODO: Implement profile update logic
		http.Redirect(w, r, "/students/profile?success=1", http.StatusFound)
		return
	}

	renderTemplate(w, "student_profile.html", map[string]interface{}{
		"User":  user,
		"Title": "My Profile",
	})
}

func studentTopicHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle topic submission
		// TODO: Implement topic submission logic
		http.Redirect(w, r, "/students/topic?success=1", http.StatusFound)
		return
	}

	renderTemplate(w, "student_topic.html", map[string]interface{}{
		"User":  user,
		"Title": "My Topic",
	})
}

func studentDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		// Handle document upload
		// TODO: Implement document upload logic
		http.Redirect(w, r, "/students/documents?success=1", http.StatusFound)
		return
	}

	renderTemplate(w, "student_documents.html", map[string]interface{}{
		"User":  user,
		"Title": "My Documents",
	})
}

// Supervisor handlers
func supervisorDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "supervisor_dashboard.html", map[string]interface{}{
		"User":  user,
		"Title": "Supervisor Dashboard",
	})
}

func supervisorStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "supervisor_students.html", map[string]interface{}{
		"User":  user,
		"Title": "My Students",
	})
}

func supervisorReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "supervisor_reports.html", map[string]interface{}{
		"User":  user,
		"Title": "Student Reports",
	})
}

func supervisorCreateReportHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	studentID := chi.URLParam(r, "studentID")

	renderTemplate(w, "supervisor_create_report.html", map[string]interface{}{
		"User":      user,
		"StudentID": studentID,
		"Title":     "Create Report",
	})
}

// Admin handlers
func adminStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "admin_students.html", map[string]interface{}{
		"User":  user,
		"Title": "Student Management",
	})
}

func adminStudentsSearchHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement search logic
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"students": []}`))
}

func adminTopicsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "admin_topics.html", map[string]interface{}{
		"User":  user,
		"Title": "Topic Management",
	})
}

func adminApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	topicID := chi.URLParam(r, "topicID")

	if !user.CanApproveTopics() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// TODO: Implement topic approval logic
	log.Printf("Topic %s approved by %s", topicID, user.Email)

	w.Header().Set("HX-Trigger", "topicApproved")
	w.WriteHeader(http.StatusOK)
}

func adminReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "admin_reports.html", map[string]interface{}{
		"User":  user,
		"Title": "Reports",
	})
}

// System handlers
func systemUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "system_users.html", map[string]interface{}{
		"User":  user,
		"Title": "User Management",
	})
}

func systemUpdateUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement user role update logic
	w.WriteHeader(http.StatusOK)
}

func systemDepartmentHeadsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "system_department_heads.html", map[string]interface{}{
		"User":  user,
		"Title": "Department Heads",
	})
}

func systemAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	renderTemplate(w, "system_audit_logs.html", map[string]interface{}{
		"User":  user,
		"Title": "Audit Logs",
	})
}

// API handlers (placeholder implementations)
func apiStudentsSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"students": [], "total": 0}`))
}

func apiStudentDetailsHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "%s", "name": "Student Name"}`, studentID)
}

func apiStudentReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"reports": []}`))
}

func apiTopicDetailsHandler(w http.ResponseWriter, r *http.Request) {
	topicID := chi.URLParam(r, "topicID")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "%s", "title": "Topic Title"}`, topicID)
}

func apiApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	topicID := chi.URLParam(r, "topicID")
	// TODO: Implement approval logic
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "topic_id": "%s"}`, topicID)
}

func apiReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"reports": []}`))
}

func apiCreateReportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true}`))
}

func apiReportDetailsHandler(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportID")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"id": "%s", "content": "Report content"}`, reportID)
}

// Helper functions
func renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	// TODO: Implement your template rendering logic
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
		<html>
			<head><title>%s</title></head>
			<body>
				<h1>Template: %s</h1>
				<pre>%+v</pre>
			</body>
		</html>
	`, data.(map[string]interface{})["Title"], templateName, data)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
