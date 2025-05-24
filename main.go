// main.go - Complete fixed version
package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
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

	// Log available templates
	log.Printf("✅ Templates initialized. Available templates:")
	for _, tmpl := range globalTemplates.Templates() {
		log.Printf("   - %s", tmpl.Name())
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

	// Initialize handlers
	commissionHandler := handlers.NewCommissionHandler(commissionService, authMiddleware, baseURL)
	supervisorHandler := handlers.NewSupervisorHandler(db, localizer)
	reviewerHandler := handlers.NewReviewerHandler(db, localizer)
	adminHandler := handlers.NewAdminHandler(db, localizer)

	// Set up Chi router
	r := setupRouter(
		authService,
		authMiddleware,
		commissionHandler,
		commissionService,
		localizer,
		supervisorHandler,
		reviewerHandler,
		adminHandler,
		db,
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
		"float64": func(i int) float64 {
			return float64(i)
		},
		"int": func(f float64) int {
			return int(f)
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

	// Create new template with functions
	tmpl := template.New("").Funcs(funcMap)

	// Parse templates by reading files directly
	templateDirs := []string{
		"templates/layouts",
		"templates/auth",
		"templates/components",
		"templates/shared",
		"templates/supervisor",
		"templates/reviewer",
		"templates/admin",
	}

	for _, dir := range templateDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.html"))
		if err != nil {
			log.Printf("Warning: failed to glob %s: %v", dir, err)
			continue
		}

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				log.Printf("Warning: failed to read %s: %v", file, err)
				continue
			}

			name := filepath.Base(file)
			_, err = tmpl.New(name).Parse(string(content))
			if err != nil {
				log.Printf("Error parsing template %s: %v", file, err)
				return nil, fmt.Errorf("failed to parse %s: %w", file, err)
			}
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
	authService *auth.AuthService,
	authMiddleware *auth.AuthMiddleware,
	commissionHandler *handlers.CommissionHandler,
	commissionService *auth.CommissionAccessService,
	localizer *i18n.Localizer,
	supervisorHandler *handlers.SupervisorHandler,
	reviewerHandler *handlers.ReviewerHandler,
	adminHandler *handlers.AdminHandler,
	db *sqlx.DB,
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

	// Test route
	r.Get("/test", testHandler)

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
		r.Get("/microsoft", authMicrosoftHandler(authService))
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

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto p-8">
        <h1 class="text-3xl font-bold mb-4">Template System Test</h1>
        <p class="mb-4">If you see this page, the server is working!</p>
        <div class="space-y-2">
            <p><a href="/auth/login" class="text-blue-600 hover:underline">Go to Login</a></p>
            <p><a href="/health" class="text-blue-600 hover:underline">Health Check</a></p>
        </div>
    </div>
</body>
</html>
	`))
}

func authLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is already logged in
	if user := auth.GetUserFromContext(r.Context()); user != nil {
		// Already logged in, redirect to dashboard
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	// Create a simple login page with proper OAuth initiation
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - VIKO</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css">
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center">
    <div class="max-w-md w-full space-y-8">
        <div class="text-center">
            <h2 class="text-3xl font-bold text-gray-900">%s</h2>
            <p class="mt-2 text-sm text-gray-600">%s</p>
        </div>
        
        <div class="bg-white py-8 px-4 shadow-xl rounded-lg sm:px-10">
            <div class="space-y-6">
                <div>
                    <h3 class="text-lg font-medium text-gray-900">%s</h3>
                    <p class="mt-1 text-sm text-gray-600">%s</p>
                </div>
                
                %s
                
                <div>
                    <form action="/auth/microsoft" method="get">
                        <button type="submit" class="w-full flex justify-center items-center px-4 py-3 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
                            <i class="fab fa-microsoft mr-2"></i>
                            %s
                        </button>
                    </form>
                </div>
                
                <div class="text-center text-sm text-gray-600">
                    <p>%s</p>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
	`,
		lang,
		globalLocalizer.T(lang, "auth.login_title"),
		globalLocalizer.T(lang, "common.thesis_management_system"),
		globalLocalizer.T(lang, "institution.name"),
		globalLocalizer.T(lang, "auth.login_subtitle"),
		globalLocalizer.T(lang, "auth.login_description"),
		// Add error message if present
		func() string {
			if errMsg := r.URL.Query().Get("error"); errMsg != "" {
				return fmt.Sprintf(`
				<div class="rounded-md bg-red-50 p-4">
					<div class="flex">
						<div class="flex-shrink-0">
							<i class="fas fa-exclamation-circle text-red-400"></i>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-medium text-red-800">Authentication Error</h3>
							<p class="text-sm text-red-700 mt-1">%s</p>
						</div>
					</div>
				</div>`, errMsg)
			}
			return ""
		}(),
		globalLocalizer.T(lang, "auth.login_with_microsoft"),
		"Use your organizational Microsoft account to sign in",
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// Add this new handler for Microsoft OAuth initiation
func authMicrosoftHandler(authService *auth.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loginURL, err := authService.GenerateLoginURL()
		if err != nil {
			log.Printf("Failed to generate login URL: %v", err)
			http.Redirect(w, r, "/auth/login?error="+url.QueryEscape("Failed to initiate login"), http.StatusFound)
			return
		}

		log.Printf("Redirecting to Microsoft login: %s", loginURL)
		http.Redirect(w, r, loginURL, http.StatusFound)
	}
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		log.Println("No user in context, redirecting to login")
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	log.Printf("User %s with role %s accessing dashboard", user.Email, user.Role)

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
		//lang := i18n.GetLangFromContext(r.Context())
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto p-8">
        <h1 class="text-3xl font-bold mb-4">Dashboard</h1>
        <p class="mb-4">Welcome, %s (%s)</p>
        <p>Role: %s</p>
        <div class="mt-4">
            <a href="/auth/logout" class="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600">Logout</a>
        </div>
    </div>
</body>
</html>
		`, user.Name, user.Email, user.Role)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}
}

// Student handlers (placeholders - implement as needed)
func studentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Student Dashboard</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto p-8">
        <h1 class="text-3xl font-bold mb-4">Student Dashboard</h1>
        <p>Welcome, %s</p>
        <div class="mt-4">
            <a href="/auth/logout" class="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600">Logout</a>
        </div>
    </div>
</body>
</html>
	`, user.Name)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func studentProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Profile"))
}

func studentTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Topic"))
}

func studentDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Documents"))
}

// System handlers (placeholders)
func systemUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("System Users"))
}

func systemUpdateUserRoleHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Update User Role"))
}

func systemDepartmentHeadsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Department Heads"))
}

func systemAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Audit Logs"))
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

// Helper function
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
