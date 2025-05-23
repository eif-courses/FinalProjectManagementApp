// main.go - Complete updated version with template integration
package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
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
	_ "github.com/go-sql-driver/mysql" // MySQL database driver
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql" // MySQL migrate driver
	_ "github.com/golang-migrate/migrate/v4/source/file"    // File source driver
	"github.com/joho/godotenv"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/handlers"
	"FinalProjectManagementApp/i18n"
	custommiddleware "FinalProjectManagementApp/middleware"
)

var (
	globalTemplates *template.Template
	globalLocalizer *i18n.Localizer
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
	globalLocalizer = localizer

	// Initialize templates
	templates, err := initializeTemplates()
	if err != nil {
		log.Fatal("Failed to initialize templates:", err)
	}
	globalTemplates = templates

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

	// Initialize handlers
	commissionHandler := handlers.NewCommissionHandler(commissionService, authMiddleware, baseURL)
	supervisorHandler := handlers.NewSupervisorHandler(db.DB, localizer)
	reviewerHandler := handlers.NewReviewerHandler(db.DB, localizer)
	adminHandler := handlers.NewAdminHandler(db.DB, localizer)

	// Set up Chi router
	r := setupRouter(
		authMiddleware,
		commissionHandler,
		commissionService,
		localizer,
		supervisorHandler,
		reviewerHandler,
		adminHandler,
		db.DB,
	)

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

func initializeTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int { return a / b },
		"seq": func(start, end int) []int {
			seq := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				seq = append(seq, i)
			}
			return seq
		},
		"printf": fmt.Sprintf,
		"now":    time.Now,
		"formatDate": func(t time.Time, format string) string {
			return t.Format(format)
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires even number of arguments")
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse all templates
	patterns := []string{
		"templates/layouts/*.html",
		"templates/auth/*.html",
		"templates/supervisor/*.html",
		"templates/reviewer/*.html",
		"templates/admin/*.html",
		"templates/components/*.html",
		"templates/shared/*.html",
	}

	for _, pattern := range patterns {
		_, err := tmpl.ParseGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", pattern, err)
		}
	}

	return tmpl, nil
}

func runMigrations(config *database.Config) error {
	databaseURL := config.GetMigrationURL()
	log.Printf("Attempting to run migrations with URL: %s", maskPassword(databaseURL))

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

func maskPassword(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			userPart := parts[0]
			if strings.Contains(userPart, ":") {
				userParts := strings.Split(userPart, ":")
				if len(userParts) >= 2 {
					return userParts[0] + ":***@" + parts[1]
				}
			}
		}
	}
	return url
}

func setupRouter(
	authMiddleware *auth.AuthMiddleware,
	commissionHandler *handlers.CommissionHandler,
	commissionService *auth.CommissionAccessService,
	localizer *i18n.Localizer,
	supervisorHandler *handlers.SupervisorHandler,
	reviewerHandler *handlers.ReviewerHandler,
	adminHandler *handlers.AdminHandler,
	db *sql.DB,
) chi.Router {
	r := chi.NewRouter()

	// Core Chi middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommiddleware.RecoveryMiddleware)
	r.Use(custommiddleware.LoggingMiddleware)
	r.Use(middleware.Compress(5))

	// Security middleware
	r.Use(custommiddleware.SecurityHeadersMiddleware)

	// Rate limiting
	if getEnv("ENV", "development") == "production" {
		r.Use(custommiddleware.RateLimitMiddleware(100))
	}

	// CORS middleware
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

	// Timeout middleware
	r.Use(custommiddleware.TimeoutMiddleware(30 * time.Second))

	// Health check
	r.Get("/health", healthCheckHandler)

	// Language switching
	r.Post("/switch-language", localizer.LanguageSwitchHandler)
	r.Get("/switch-language", localizer.LanguageSwitchHandler)

	// Static files
	fileServer := http.FileServer(http.Dir("./static/"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public commission access
	r.Route("/commission", func(r chi.Router) {
		r.Use(commissionService.CommissionAccessMiddleware)
		r.Get("/{accessCode}", commissionHandler.CommissionViewHandler)
		r.Get("/{accessCode}/*", commissionHandler.CommissionViewHandler)
	})

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", authLoginHandler)
		r.Get("/callback", authMiddleware.CallbackHandler)
		r.Get("/logout", authMiddleware.LogoutHandler)
		r.Post("/logout", authMiddleware.LogoutHandler)
		r.Get("/user", authMiddleware.UserInfoHandler)
	})

	// Protected routes
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
			r.Get("/", supervisorHandler.DashboardHandler)
			r.Get("/students", supervisorHandler.DashboardHandler)

			// Topic registration
			r.Get("/topic-registration/{studentID}", supervisorHandler.TopicModalHandler)
			r.Post("/topic-registration/{studentID}/review", supervisorHandler.ReviewTopicHandler)

			// Documents
			r.Get("/documents/{studentID}", supervisorHandler.DocumentsModalHandler)

			// Reports
			r.Get("/reports/create/{studentID}", supervisorHandler.CreateReportHandler)
			r.Post("/reports/{studentID}", supervisorHandler.SaveReportHandler)
			r.Get("/reports/edit/{studentID}", supervisorHandler.EditReportHandler)
			r.Get("/reports/view/{studentID}", supervisorHandler.ViewReportHandler)
		})

		// Reviewer routes
		r.Route("/reviewer", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleReviewer))
			r.Get("/", reviewerHandler.DashboardHandler)

			// Documents
			r.Get("/documents/{studentID}", reviewerHandler.DocumentsModalHandler)

			// View supervisor report
			r.Get("/supervisor-report/{studentID}", reviewerHandler.ViewSupervisorReportHandler)

			// Reviewer reports
			r.Get("/reports/create/{studentID}", reviewerHandler.CreateReportHandler)
			r.Post("/reports/{studentID}", reviewerHandler.SaveReportHandler)
			r.Get("/reports/edit/{studentID}", reviewerHandler.EditReportHandler)
			r.Get("/reports/view/{studentID}", reviewerHandler.ViewReportHandler)
		})

		// Admin routes (department heads and admins)
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleDepartmentHead, auth.RoleAdmin))

			r.Get("/", adminHandler.DashboardHandler)
			r.Get("/dashboard", adminHandler.DashboardHandler)

			// HTMX endpoints for tabs
			r.Get("/students-table", adminHandler.StudentsTableHandler)
			r.Get("/topics-table", adminHandler.TopicsTableHandler)
			r.Get("/commission-table", adminHandler.CommissionTableHandler)

			// Student management
			r.Get("/students", adminHandler.StudentsHandler)
			r.Get("/students/search", adminHandler.StudentsSearchHandler)

			// Topics
			r.Get("/topics", adminHandler.TopicsHandler)
			r.Post("/topics/{topicID}/approve", adminHandler.ApproveTopicHandler)

			// Reports
			r.Get("/reports", adminHandler.ReportsHandler)

			// Commission access management
			r.Get("/commission", commissionHandler.ListCommissionAccessesHandler)
			r.Get("/commission/create", commissionHandler.CreateCommissionAccessHandler)
			r.Post("/commission/create", commissionHandler.CreateCommissionAccessHandler)
			r.Get("/commission/create-modal", adminHandler.CreateCommissionModalHandler)
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
func needsCORS() bool {
	if getEnv("ENABLE_CORS", "false") == "true" {
		return true
	}
	if getEnv("ENV", "development") == "development" {
		return true
	}
	if getEnv("API_ONLY", "false") == "true" {
		return true
	}
	return false
}

func getAllowedOrigins() []string {
	origins := []string{"http://localhost:8080"}

	if getEnv("ENV", "development") == "development" {
		origins = append(origins,
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:8080",
		)
	}

	if prodOrigins := getEnv("ALLOWED_ORIGINS", ""); prodOrigins != "" {
		origins = append(origins, strings.Split(prodOrigins, ",")...)
	}

	return origins
}

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

func authLoginHandler(w http.ResponseWriter, r *http.Request) {
	lang := i18n.GetLangFromContext(r.Context())

	data := map[string]interface{}{
		"Title": globalLocalizer.T(lang, "auth.login_title"),
		"Lang":  lang,
		"T": func(key string, args ...interface{}) string {
			return globalLocalizer.T(lang, key, args...)
		},
		"Error": r.URL.Query().Get("error"),
	}

	renderTemplate(w, "login.html", data)
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
	case auth.RoleReviewer:
		http.Redirect(w, r, "/reviewer/", http.StatusFound)
	case auth.RoleDepartmentHead, auth.RoleAdmin:
		http.Redirect(w, r, "/admin/", http.StatusFound)
	default:
		// Show generic dashboard
		templateData := getLocalizedTemplateData(r, "dashboard.title", user, nil)
		renderTemplate(w, "dashboard.html", templateData)
	}
}

// Student handlers (placeholders - implement as needed)
func studentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	templateData := getLocalizedTemplateData(r, "student_management.title", user, nil)
	renderTemplate(w, "student_dashboard.html", templateData)
}

func studentProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
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
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
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
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/students/documents?success=1", http.StatusFound)
		return
	}

	templateData := getLocalizedTemplateData(r, "documents.title", user, map[string]interface{}{
		"Success": r.URL.Query().Get("success") == "1",
	})
	renderTemplate(w, "student_documents.html", templateData)
}

// System handlers (placeholders)
func systemUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	templateData := getLocalizedTemplateData(r, "system.user_management", user, map[string]interface{}{
		"Users": []interface{}{},
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
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		log.Printf("Adding department head %s by %s", email, user.Email)

		http.Redirect(w, r, "/system/department-heads?added="+email, http.StatusFound)
		return
	}

	templateData := getLocalizedTemplateData(r, "system.department_heads", user, map[string]interface{}{
		"DepartmentHeads": []interface{}{},
		"Added":           r.URL.Query().Get("added"),
	})
	renderTemplate(w, "system_department_heads.html", templateData)
}

func systemAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	templateData := getLocalizedTemplateData(r, "system.audit_logs", user, map[string]interface{}{
		"AuditLogs": []interface{}{},
	})
	renderTemplate(w, "system_audit_logs.html", templateData)
}

// API handlers (placeholders)
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

// Template helper functions
func getLocalizedTemplateData(r *http.Request, titleKey string, user *auth.AuthenticatedUser, data interface{}) map[string]interface{} {
	lang := i18n.GetLangFromContext(r.Context())
	if lang == "" {
		lang = i18n.DefaultLang
	}

	currentYear := time.Now().Year()

	return map[string]interface{}{
		"Title":       globalLocalizer.T(lang, titleKey),
		"User":        user,
		"Data":        data,
		"Lang":        lang,
		"CurrentYear": currentYear,
		"BaseURL":     getEnv("BASE_URL", "http://localhost:8080"),
		"T": func(key string, args ...interface{}) string {
			return globalLocalizer.T(lang, key, args...)
		},
		"Breadcrumbs": nil, // Add breadcrumbs if needed
	}
}

func renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	if globalTemplates == nil {
		http.Error(w, "Templates not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := globalTemplates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template execution error for %s: %v", templateName, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
