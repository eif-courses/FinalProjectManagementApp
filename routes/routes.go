package routes

import (
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/handlers"
)

func SetupRoutes(authService *auth.AuthService, authMiddleware *auth.AuthMiddleware) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5))

	// Initialize handlers
	authHandlers := handlers.NewAuthHandlers(authMiddleware)

	// Static files - More comprehensive setup
	workDir, _ := filepath.Abs("./")
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(filesDir)))

	// Also serve favicon if you have one
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})

	// Public routes
	r.Group(func(r chi.Router) {
		// Login page
		r.Get("/", authHandlers.ShowLoginPage)
		r.Get("/login", authHandlers.ShowLoginPage)
		r.Get("/access-denied", authHandlers.ShowAccessDeniedPage)

		// OAuth flow
		r.Get("/auth/login", authMiddleware.LoginHandler)
		r.Get("/auth/callback", authMiddleware.CallbackHandler)
		r.Get("/auth/logout", authHandlers.ShowLogoutPage)
		r.Post("/auth/logout", authMiddleware.LogoutHandler)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// API endpoint for user info
		r.Get("/api/auth/user", authMiddleware.UserInfoHandler)

		// Dashboard
		r.Get("/dashboard", handlers.DashboardHandler)

		// Student List
		r.Get("/students-list", handlers.StudentListHandler)

		// API endpoints for student data
		r.Get("/api/students/{id}/documents", handlers.DocumentsAPIHandler)

		// Student routes
		r.Route("/students", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleStudent))
			r.Get("/profile", handlers.StudentProfileHandler)
			r.Get("/topic", handlers.StudentTopicHandler)
			r.Post("/topic/submit", handlers.SubmitTopicHandler)
		})

		// Supervisor routes
		r.Route("/supervisor", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleSupervisor, auth.RoleReviewer))
			r.Get("/", handlers.SupervisorDashboardHandler)
			r.Get("/students", handlers.SupervisorStudentsHandler)
			r.Get("/reports", handlers.SupervisorReportsHandler)
		})

		// Department head routes
		r.Route("/department", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleDepartmentHead))
			r.Get("/", handlers.DepartmentDashboardHandler)
			r.Get("/students", handlers.DepartmentStudentsHandler)
			r.Get("/topics/pending", handlers.PendingTopicsHandler)
			r.Post("/topics/{id}/approve", handlers.ApproveTopicHandler)
			r.Post("/topics/{id}/reject", handlers.RejectTopicHandler)
		})

		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleAdmin))
			r.Get("/", handlers.AdminDashboardHandler)
			r.Get("/users", handlers.AdminUsersHandler)
			r.Get("/audit-logs", handlers.AuditLogsHandler)
		})
	})

	// Add this temporary debug route in your SetupRoutes function
	r.Get("/debug/css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>CSS Debug</title>
        <link rel="stylesheet" href="/static/css/student-list.css">
        <style>
        .test { background: red; color: white; padding: 10px; }
        </style>
    </head>
    <body>
        <div class="test">This should have red background (inline CSS)</div>
        <div class="container">This should be styled if CSS loads</div>
        <button class="btn-view">This should be a styled button</button>
    </body>
    </html>
    `))
	})

	return r

}
