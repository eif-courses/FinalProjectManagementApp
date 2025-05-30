// routes/routes.go
package routes

import (
	"FinalProjectManagementApp/database"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/handlers"
	"FinalProjectManagementApp/notifications"
)

// Updated function signature to accept database and notification service
func SetupRoutes(db *sqlx.DB,
	authService *auth.AuthService,
	authMiddleware *auth.AuthMiddleware,
	notificationService *notifications.NotificationService,
	sourceCodeHandler *handlers.SourceCodeHandler) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "thesis-management-app",
		}
		json.NewEncoder(w).Encode(response)
	})

	dashboardHandlers := handlers.NewDashboardHandlers(db)

	// Initialize handlers with database
	authHandlers := handlers.NewAuthHandlers(authMiddleware)
	topicHandlers := handlers.NewTopicHandlers(db.DB)
	supervisorReportHandler := handlers.NewSupervisorReportHandler(db)
	studentListHandler := handlers.NewStudentListHandler(db)
	// Get app config for GitHub settings
	appConfig := database.LoadAppConfig()

	// Initialize repository handler only if GitHub is configured
	var repositoryHandler *handlers.RepositoryHandler
	if appConfig.HasGitHub() {
		repositoryHandler = handlers.NewRepositoryHandler(db, appConfig.GitHub)
		log.Println("Repository handler initialized successfully")
	} else {
		log.Println("Repository viewing disabled - GitHub not configured")
	}
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

	//r.Route("/students", func(r chi.Router) {
	//	r.Use(authMiddleware.RequireAuth)
	//	r.Get("/list", studentListHandler.GetStudentsList) // Role-filtered
	//})

	r.Get("/students-list", studentListHandler.StudentTableDisplayHandler)
	r.Get("/api/documents/{id}", handlers.DocumentsAPIHandler)

	// Reviewer-specific route with token
	r.Get("/reviewer/students", studentListHandler.ReviewerStudentsList)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// TEST FOR GITHUB
		r.Get("/test-upload", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "static/upload_test.html")
		})

		// Add this to your routes setup for testing
		r.Get("/test-repository", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			html := `
<!DOCTYPE html>
<html>
<head>
    <title>Repository Test</title>
    <link rel="stylesheet" href="/assets/css/output.css">
</head>
<body class="bg-gray-100 p-8">
    <div class="max-w-4xl mx-auto bg-white rounded-lg p-6">
        <h1 class="text-2xl font-bold mb-6">Repository System Test</h1>
        
        <div class="space-y-4">
            <h2 class="text-lg font-semibold">Test Repository Viewing:</h2>
            
            <div class="space-y-2">
                <a href="/repository/student/1" target="_blank" 
                   class="block bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
                    Test Repository View (Student ID: 1)
                </a>
                
                <a href="/repository/student/2" target="_blank" 
                   class="block bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700">
                    Test Repository View (Student ID: 2)
                </a>
                
                <a href="/api/repository/student/1" target="_blank" 
                   class="block bg-purple-600 text-white px-4 py-2 rounded hover:bg-purple-700">
                    Test API Response (Student ID: 1)
                </a>
            </div>
            
            <h2 class="text-lg font-semibold mt-6">Test JavaScript Function:</h2>
            <button onclick="viewStudentRepository(1)" 
                    class="bg-orange-600 text-white px-4 py-2 rounded hover:bg-orange-700">
                Test JavaScript Function
            </button>
            
            <h2 class="text-lg font-semibold mt-6">Current User Info:</h2>
            <div id="user-info" class="bg-gray-100 p-4 rounded">
                Loading user info...
            </div>
        </div>
    </div>
    
    <script>
        // Test the viewStudentRepository function
        function viewStudentRepository(studentId) {
            console.log('Opening repository for student ID:', studentId);
            window.open('/repository/student/' + studentId, '_blank');
        }
        
        // Fetch current user info
        fetch('/api/auth/user')
            .then(response => response.json())
            .then(data => {
                document.getElementById('user-info').innerHTML = 
                    '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
            })
            .catch(error => {
                document.getElementById('user-info').innerHTML = 
                    'Error: ' + error.message;
            });
    </script>
</body>
</html>`
			w.Write([]byte(html))
		})

		// UPDATE: Add the new endpoints
		r.Route("/api/source-code", func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.Post("/upload", sourceCodeHandler.UploadSourceCode)
			r.Get("/status", sourceCodeHandler.GetUploadStatus) // NEW
			r.Get("/health", sourceCodeHandler.GetSystemHealth) // NEW
		})

		// Repository viewing routes (only if GitHub is configured)
		if repositoryHandler != nil {
			r.Route("/repository", func(r chi.Router) {
				// Require supervisor, reviewer, department head, admin, or commission member
				r.Use(authMiddleware.RequireRole(
					auth.RoleSupervisor,
					auth.RoleReviewer,
					auth.RoleDepartmentHead,
					auth.RoleAdmin,
					auth.RoleCommissionMember))

				r.Get("/student/{studentId}", repositoryHandler.ViewStudentRepository)
				r.Get("/student/{studentId}/download", repositoryHandler.DownloadRepository)
				// NEW: File content viewing
				r.Get("/student/{studentId}/browse/*", repositoryHandler.ViewStudentRepositoryPath)
				r.Get("/student/{studentId}/file/*", repositoryHandler.ViewFileContent)
				r.Get("/student/{studentId}/tree", repositoryHandler.GetRepositoryTree)
			})

			// API routes for repository data
			r.Route("/api/repository", func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(
					auth.RoleSupervisor,
					auth.RoleReviewer,
					auth.RoleDepartmentHead,
					auth.RoleAdmin,
					auth.RoleCommissionMember))

				r.Get("/student/{studentId}", repositoryHandler.GetRepositoryAPI)
				// NEW: File content API
				r.Get("/student/{studentId}/browse", repositoryHandler.GetRepositoryPathAPI)
				r.Get("/student/{studentId}/file", repositoryHandler.GetFileContentAPI)
			})
		}

		r.Get("/supervisor-report/{id}/compact-modal", supervisorReportHandler.GetCompactSupervisorModal)

		//r.Route("/supervisor-report", func(r chi.Router) {
		//	// Only supervisors and admins can access
		//	r.Use(authMiddleware.RequireRole(auth.RoleSupervisor, auth.RoleAdmin, auth.RoleDepartmentHead))
		//	r.Get("/modal/{id}", supervisorReportHandler.GetSupervisorReportModal)
		//	r.Post("/submit/{id}", supervisorReportHandler.SubmitSupervisorReport)
		//	r.Get("/button/{id}", supervisorReportHandler.GetSupervisorReportButton)
		//
		//	// Full page view
		//	r.Get("/{id}", supervisorReportHandler.GetSupervisorReportPage)
		//
		//	// Modal for HTMX
		//	r.Get("/modal/{id}", supervisorReportHandler.GetSupervisorReportModal)
		//
		//})
		// Upload routes
		r.Get("/upload", handlers.ShowUploadPage)
		r.Post("/api/upload", handlers.UploadFileHandler)

		// API endpoint for user info
		r.Get("/api/auth/user", authMiddleware.UserInfoHandler)

		// Dashboard
		//r.Get("/dashboard", handlers.DashboardHandler)

		r.Get("/dashboard", dashboardHandlers.DashboardHandler)

		// You can also create specific role routes if needed
		r.Route("/student", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleStudent))
			//	r.Get("/profile", dashboardHandlers.StudentProfileHandler)
			//	r.Get("/topic", dashboardHandlers.StudentTopicHandler)
		})

		r.Route("/supervisor", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleSupervisor))
			//r.Get("/students", dashboardHandlers.SupervisorStudentsHandler)
			//r.Get("/reports", dashboardHandlers.SupervisorReportsHandler)
		})

		// Student List
		r.Get("/students-list", studentListHandler.StudentTableDisplayHandler)

		// API endpoints for student data
		r.Get("/api/students/{id}/documents", handlers.DocumentsAPIHandler)

		// Topic registration routes
		r.Route("/topic", func(r chi.Router) {
			r.Get("/register", topicHandlers.ShowTopicRegistrationForm)
			r.Get("/", topicHandlers.ShowTopicRegistrationForm) // Alternative route
		})

		// API routes
		r.Route("/api", func(r chi.Router) {
			// Topic API endpoints
			r.Post("/topic/submit", topicHandlers.SubmitTopic)
			r.Post("/topic/save-draft", topicHandlers.SaveDraft)
			r.Post("/topic/{id}/comment", topicHandlers.AddComment)

			// Department head/Admin only routes
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(auth.RoleDepartmentHead, auth.RoleAdmin))
				r.Post("/topic/{id}/approve", topicHandlers.ApproveTopic)
				r.Post("/topic/{id}/reject", topicHandlers.RejectTopic)
			})
		})

		// Notification test route (admin only)
		if notificationService != nil {
			r.Route("/admin/notifications", func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(auth.RoleAdmin))

				// Test notification endpoint
				r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					email := r.FormValue("email")
					if email == "" {
						http.Error(w, "Email is required", http.StatusBadRequest)
						return
					}

					log.Printf("DEBUG: Starting notification test to email: %s", email)
					log.Printf("DEBUG: Notification service enabled: %v", notificationService.IsEnabled())
					log.Printf("DEBUG: System email: %s", notificationService.GetSystemEmail())

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
			//	r.Get("/profile", handlers.StudentProfileHandler)
			r.Get("/topic", topicHandlers.ShowTopicRegistrationForm) // Redirect to topic registration
			r.Post("/topic/submit", topicHandlers.SubmitTopic)       // Legacy route
		})

		// Supervisor routes
		//r.Route("/supervisor", func(r chi.Router) {
		//r.Use(authMiddleware.RequireRole(auth.RoleSupervisor, auth.RoleReviewer))
		//r.Get("/", handlers.SupervisorDashboardHandler)
		//r.Get("/students", handlers.SupervisorStudentsHandler)
		//	r.Get("/reports", handlers.SupervisorReportsHandler)
		//})

		// Department head routes
		r.Route("/department", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleDepartmentHead))
			//r.Get("/", handlers.DepartmentDashboardHandler)
			//r.Get("/students", handlers.DepartmentStudentsHandler)
			//r.Get("/topics/pending", handlers.PendingTopicsHandler)

			// Topic approval routes with notification integration
			r.Post("/topics/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
				// Call the topic approval handler
				topicHandlers.ApproveTopic(w, r)

				// Add notification logic after successful approval
				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					// TODO: Get student info and send notification
					// student := getStudentByTopicID(topicID)
					// notificationService.SendTopicApprovalNotification(r.Context(), student.Email, student.Name, student.TopicTitle, true)
					log.Printf("Topic %s approved - notification would be sent here", topicID)
				}
			})

			r.Post("/topics/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
				// Call the topic rejection handler
				topicHandlers.RejectTopic(w, r)

				// Add notification logic after successful rejection
				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					// TODO: Get student info and send notification
					// student := getStudentByTopicID(topicID)
					// notificationService.SendTopicApprovalNotification(r.Context(), student.Email, student.Name, student.TopicTitle, false)
					log.Printf("Topic %s rejected - notification would be sent here", topicID)
				}
			})
		})

		// Admin routes
		//r.Route("/admin", func(r chi.Router) {
		//r.Use(authMiddleware.RequireRole(auth.RoleAdmin))
		//r.Get("/", handlers.AdminDashboardHandler)
		//r.Get("/users", handlers.AdminUsersHandler)
		//r.Get("/audit-logs", handlers.AuditLogsHandler)
		//})
	})

	// Debug route for testing CSS and notifications
	r.Get("/debug/css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>CSS Debug & Notification Test</title>
    <link rel="stylesheet" href="/assets/css/output.css">
    <style>
    .test { background: red; color: white; padding: 10px; }
    </style>
</head>
<body>
    <div class="test">This should have red background (inline CSS)</div>
    <div class="bg-primary text-primary-foreground p-4">This should be styled if TemplUI CSS loads</div>
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

    <!-- Topic Registration Test -->
    <div style="margin-top: 20px; padding: 20px; border: 1px solid #ccc;">
        <h3>Topic Registration Test</h3>
        <a href="/topic/register" style="padding: 10px 15px; background: #007bff; color: white; text-decoration: none; border-radius: 5px;">
            Go to Topic Registration Form
        </a>
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
