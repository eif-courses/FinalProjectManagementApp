package routes

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/handlers"
	"FinalProjectManagementApp/notifications" // Add this import
)

// Update the function signature to accept notification service
func SetupRoutes(authService *auth.AuthService, authMiddleware *auth.AuthMiddleware, notificationService *notifications.NotificationService) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5))

	// Initialize handlers
	authHandlers := handlers.NewAuthHandlers(authMiddleware)

	// Static files - serve both assets and static directories
	workDir, _ := filepath.Abs("./")

	// Serve /assets/ directory (for CSS, etc.)
	assetsDir := http.Dir(filepath.Join(workDir, "assets"))
	r.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(assetsDir)))

	// Serve /static/ directory (for images, JS, etc.)
	staticDir := http.Dir(filepath.Join(workDir, "static"))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticDir)))

	// Favicon
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

		// Upload routes
		r.Get("/upload", handlers.ShowUploadPage)
		r.Post("/api/upload", handlers.UploadFileHandler)

		// API endpoint for user info
		r.Get("/api/auth/user", authMiddleware.UserInfoHandler)

		// Dashboard
		r.Get("/dashboard", handlers.DashboardHandler)

		// Student List
		r.Get("/students-list", handlers.StudentListHandler)

		// API endpoints for student data
		r.Get("/api/students/{id}/documents", handlers.DocumentsAPIHandler)

		// NEW: Notification test route (admin only)
		if notificationService != nil {
			r.Route("/admin/notifications", func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(auth.RoleAdmin))

				// Test notification endpoint
				// In your routes/routes.go, update the test notification handler:
				r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					email := r.FormValue("email")
					if email == "" {
						http.Error(w, "Email is required", http.StatusBadRequest)
						return
					}

					log.Printf("DEBUG: Starting notification test to email: %s", email)
					log.Printf("DEBUG: Notification service enabled: %v", notificationService.IsEnabled())
					log.Printf("DEBUG: System email: %s", notificationService.GetSystemEmail())

					//err := notificationService.SendTestNotification(r.Context(), email)

					err := notificationService.SendTestNotificationWithDebug(r.Context(), email)

					if err != nil {
						log.Printf("ERROR: Failed to send notification: %v", err)
						http.Error(w, fmt.Sprintf("Failed to send notification: %v", err), http.StatusInternalServerError)
						return
					}

					log.Printf("SUCCESS: Test notification sent to %s", email)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status":"success","message":"Test notification sent successfully"}`))
				})

				// Notification status endpoint
				r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
					status := "disabled"
					if notificationService.IsEnabled() {
						status = "enabled"
					}

					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(fmt.Sprintf(`{"status":"%s"}`, status)))
				})
			})
		}

		// Student routes
		r.Route("/students", func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
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

			// Update these handlers to use notification service
			r.Post("/topics/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
				// Call your existing approve handler
				handlers.ApproveTopicHandler(w, r)

				// TODO: Add notification logic here after successful approval
				// Example:
				// topicID := chi.URLParam(r, "id")
				// if notificationService != nil {
				//     // Send approval notification to student
				//     student := getStudentByTopicID(topicID) // You'll need to implement this
				//     notificationService.SendTopicApprovalNotification(r.Context(), student.Email, student.Name, student.TopicTitle, true)
				// }
			})

			r.Post("/topics/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
				// Call your existing reject handler
				handlers.RejectTopicHandler(w, r)

				// TODO: Add notification logic here after successful rejection
				// Similar to approval but with approved=false
			})
		})

		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleAdmin))
			r.Get("/", handlers.AdminDashboardHandler)
			r.Get("/users", handlers.AdminUsersHandler)
			r.Get("/audit-logs", handlers.AuditLogsHandler)
		})
	})

	// Debug route for testing CSS
	r.Get("/debug/css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>CSS Debug</title>
    <link rel="stylesheet" href="/assets/css/output.css">
    <style>
    .test { background: red; color: white; padding: 10px; }
    </style>
</head>
<body>
    <div class="test">This should have red background (inline CSS)</div>
    <div class="bg-primary text-primary-foreground p-4">This should be styled if templUI CSS loads</div>
    <button class="bg-blue-500 text-white px-4 py-2 rounded">Tailwind button test</button>
    
    <!-- Add notification test form for admins -->
    <div style="margin-top: 20px; padding: 20px; border: 1px solid #ccc;">
        <h3>Notification Test (Admin Only)</h3>
        <form id="notificationTest">
            <input type="email" id="testEmail" placeholder="Enter email to test" required style="padding: 5px; margin-right: 10px;">
            <button type="submit" style="padding: 5px 10px;">Send Test Notification</button>
        </form>
        <div id="notificationResult" style="margin-top: 10px;"></div>
    </div>
    
    <script>
    document.getElementById('notificationTest').addEventListener('submit', async function(e) {
        e.preventDefault();
        const email = document.getElementById('testEmail').value;
        const resultDiv = document.getElementById('notificationResult');
        
        try {
            const response = await fetch('/admin/notifications/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: 'email=' + encodeURIComponent(email)
            });
            
            if (response.ok) {
                resultDiv.innerHTML = '<p style="color: green;">Test notification sent successfully!</p>';
            } else {
                const text = await response.text();
                resultDiv.innerHTML = '<p style="color: red;">Error: ' + text + '</p>';
            }
        } catch (error) {
            resultDiv.innerHTML = '<p style="color: red;">Error: ' + error.message + '</p>';
        }
    });
    </script>
</body>
</html>
`))
	})

	return r
}
