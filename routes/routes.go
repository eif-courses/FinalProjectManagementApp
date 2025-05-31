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

	// Initialize upload handlers
	uploadHandlers := handlers.NewUploadHandlers(db)

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

	r.Get("/students-list", studentListHandler.StudentTableDisplayHandler)
	r.Get("/api/documents/{id}", handlers.DocumentsAPIHandler)

	// Reviewer-specific route with token
	r.Get("/reviewer/students", studentListHandler.ReviewerStudentsList)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// @TODO REVIEW IF NEED TOPIC REGISTRATION
		r.Get("/topic-registration/{studentId}", topicHandlers.ShowTopicRegistrationModal)

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

		// Repository viewing routes (only if GitHub is configured)
		if repositoryHandler != nil {
			r.Route("/repository", func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(
					auth.RoleSupervisor,
					auth.RoleReviewer,
					auth.RoleDepartmentHead,
					auth.RoleAdmin,
					auth.RoleStudent,
					auth.RoleCommissionMember))

				r.Get("/student/{studentId}", repositoryHandler.ViewStudentRepository)
				r.Get("/student/{studentId}/download", repositoryHandler.DownloadRepository)

				// FIXED: Proper wildcard handling for file paths
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
					auth.RoleStudent,
					auth.RoleCommissionMember))

				r.Get("/student/{studentId}", repositoryHandler.GetRepositoryAPI)
				r.Get("/student/{studentId}/browse", repositoryHandler.GetRepositoryPathAPI)
				r.Get("/student/{studentId}/file", repositoryHandler.GetFileContentAPI)
			})
		}

		r.Get("/supervisor-report/{id}/compact-modal", supervisorReportHandler.GetCompactSupervisorModal)

		// Upload routes
		r.Get("/upload", handlers.ShowUploadPage)
		r.Post("/api/upload", handlers.UploadFileHandler)

		// API endpoint for user info
		r.Get("/api/auth/user", authMiddleware.UserInfoHandler)

		// Dashboard
		r.Get("/dashboard", dashboardHandlers.DashboardHandler)

		// Student routes
		r.Route("/student", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleStudent))
			r.Get("/dashboard", dashboardHandlers.DashboardHandler)
			r.Get("/topic", topicHandlers.ShowTopicRegistrationForm)
			r.Post("/topic/submit", topicHandlers.SubmitTopic)
		})

		// Supervisor routes - ENHANCED WITH TOPIC WORKFLOW
		r.Route("/supervisor", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleSupervisor))
			r.Get("/dashboard", dashboardHandlers.DashboardHandler)

			// Topic review routes for supervisors
			r.Get("/topics", topicHandlers.ShowSupervisorTopics)
			r.Get("/topics/pending", topicHandlers.ShowPendingSupervisorTopics)

			// Supervisor topic actions with notifications
			r.Post("/topics/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
				topicHandlers.SupervisorApproveTopic(w, r)

				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					log.Printf("Supervisor approved topic %s - notifying department head", topicID)
					// TODO: Implement notification to department head
				}
			})

			r.Post("/topics/{id}/revision", func(w http.ResponseWriter, r *http.Request) {
				topicHandlers.SupervisorRequestRevision(w, r)

				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					log.Printf("Supervisor requested revision for topic %s - notifying student", topicID)
					// TODO: Implement notification to student
				}
			})
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

		// API routes - COMPLETE TOPIC WORKFLOW IMPLEMENTATION
		r.Route("/api", func(r chi.Router) {
			// Document operations
			r.Get("/documents/{id}/preview", uploadHandlers.DocumentPreviewHandler)
			r.Get("/documents/{id}/download", uploadHandlers.DocumentDownloadHandler)

			// Student-only upload routes
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(auth.RoleStudent))
				r.Post("/recommendation/upload", uploadHandlers.RecommendationUploadHandler)
				r.Post("/video/upload", uploadHandlers.VideoUploadHandler)
			})

			// Source code routes
			r.Route("/source-code", func(r chi.Router) {
				r.Post("/upload", sourceCodeHandler.UploadSourceCode)
				r.Get("/status", sourceCodeHandler.GetUploadStatus)
				r.Get("/health", sourceCodeHandler.GetSystemHealth)
			})

			// Topic API endpoints - COMPLETE WORKFLOW
			r.Post("/topic/submit", topicHandlers.SubmitTopic) // Legacy - for compatibility
			r.Post("/topic/save-draft", topicHandlers.SaveDraft)

			// NEW: Enhanced submission workflow
			r.Post("/topic/submit-for-review", topicHandlers.SubmitTopicForReview)
			r.Post("/topic/{id}/comment", topicHandlers.AddComment)

			// Supervisor approval routes - NEW SECTION
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(auth.RoleSupervisor, auth.RoleAdmin))
				r.Post("/topic/{id}/supervisor-approve", topicHandlers.SupervisorApproveTopic)
				r.Post("/topic/{id}/supervisor-revision", topicHandlers.SupervisorRequestRevision)
			})

			// Department head/Admin final approval routes
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

		// Department head routes - ENHANCED WITH TOPIC WORKFLOW
		r.Route("/department", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleDepartmentHead))
			r.Get("/dashboard", dashboardHandlers.DashboardHandler)

			// Topic management routes
			r.Get("/topics", topicHandlers.ShowDepartmentTopics)
			r.Get("/topics/pending", topicHandlers.ShowPendingDepartmentTopics)

			// Topic approval routes with notification integration
			r.Post("/topics/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
				// Call the topic approval handler
				topicHandlers.ApproveTopic(w, r)

				// Add notification logic after successful approval
				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					log.Printf("Topic %s approved by department head - notification would be sent to student and supervisor", topicID)
					// TODO: Implement notifications
					// - Send approval notification to student
					// - Send confirmation to supervisor
				}
			})

			r.Post("/topics/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
				// Call the topic rejection handler
				topicHandlers.RejectTopic(w, r)

				// Add notification logic after successful rejection
				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					log.Printf("Topic %s rejected by department head - notification would be sent to student and supervisor", topicID)
					// TODO: Implement notifications
					// - Send rejection notification to student with reasons
					// - Send notification to supervisor
				}
			})

			// Routes to handle supervisor-approved topics
			r.Post("/topics/{id}/final-approve", func(w http.ResponseWriter, r *http.Request) {
				topicHandlers.ApproveTopic(w, r)

				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					log.Printf("Topic %s given final approval by department head", topicID)
				}
			})

			r.Post("/topics/{id}/final-reject", func(w http.ResponseWriter, r *http.Request) {
				topicHandlers.RejectTopic(w, r)

				if notificationService != nil {
					topicID := chi.URLParam(r, "id")
					log.Printf("Topic %s rejected by department head after supervisor approval", topicID)
				}
			})
		})

		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleAdmin))
			r.Get("/dashboard", dashboardHandlers.DashboardHandler)

			// Admin can access all topic management functions
			r.Get("/topics", topicHandlers.ShowAllTopics)
			r.Get("/topics/analytics", topicHandlers.ShowTopicAnalytics)

			// Admin override actions
			r.Post("/topics/{id}/force-approve", topicHandlers.ApproveTopic)
			r.Post("/topics/{id}/force-reject", topicHandlers.RejectTopic)
			r.Post("/topics/{id}/reset-workflow", topicHandlers.ResetTopicWorkflow)
		})
	})

	// Debug routes for testing
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

	// Topic Workflow Debug Route
	r.Get("/debug/topic-workflow", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Topic Workflow Debug</title>
    <link rel="stylesheet" href="/assets/css/output.css">
</head>
<body class="bg-gray-100 p-8">
    <div class="max-w-6xl mx-auto bg-white rounded-lg p-6">
        <h1 class="text-3xl font-bold mb-8">Topic Registration Workflow Test</h1>
        
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
            <!-- Student Actions -->
            <div class="border-2 border-blue-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-blue-700">📚 Student Actions</h2>
                <div class="space-y-3">
                    <a href="/topic/register" class="block bg-blue-600 text-white px-4 py-3 rounded hover:bg-blue-700 text-center">
                        Topic Registration Form
                    </a>
                    <button onclick="testAction('Student submitted topic for review', 'blue')" 
                            class="w-full bg-green-600 text-white px-4 py-3 rounded hover:bg-green-700">
                        Test Submit Topic
                    </button>
                    <button onclick="testAction('Student saved topic as draft', 'gray')" 
                            class="w-full bg-gray-600 text-white px-4 py-3 rounded hover:bg-gray-700">
                        Test Save Draft
                    </button>
                </div>
            </div>

            <!-- Supervisor Actions -->
            <div class="border-2 border-yellow-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-yellow-700">👨‍🏫 Supervisor Actions</h2>
                <div class="space-y-3">
                    <button onclick="testAction('Supervisor approved topic → Sent to Department Head', 'green')" 
                            class="w-full bg-yellow-600 text-white px-4 py-3 rounded hover:bg-yellow-700">
                        Test Supervisor Approve
                    </button>
                    <button onclick="testAction('Supervisor requested revision → Sent back to Student', 'orange')" 
                            class="w-full bg-orange-600 text-white px-4 py-3 rounded hover:bg-orange-700">
                        Test Request Revision
                    </button>
                    <a href="/supervisor/topics" class="block bg-yellow-500 text-white px-4 py-3 rounded hover:bg-yellow-600 text-center">
                        View Supervisor Dashboard
                    </a>
                </div>
            </div>

            <!-- Department Head Actions -->
            <div class="border-2 border-purple-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-purple-700">🎓 Department Head Actions</h2>
                <div class="space-y-3">
                    <button onclick="testAction('Department Head approved topic → Process Complete ✅', 'green')" 
                            class="w-full bg-purple-600 text-white px-4 py-3 rounded hover:bg-purple-700">
                        Test Final Approve
                    </button>
                    <button onclick="testAction('Department Head rejected topic → Sent back to Student ❌', 'red')" 
                            class="w-full bg-red-600 text-white px-4 py-3 rounded hover:bg-red-700">
                        Test Final Reject
                    </button>
                    <a href="/department/topics" class="block bg-purple-500 text-white px-4 py-3 rounded hover:bg-purple-600 text-center">
                        View Department Dashboard
                    </a>
                </div>
            </div>
        </div>

        <!-- Workflow Status Display -->
        <div class="mt-8 border-2 border-gray-200 p-6 rounded-lg">
            <h2 class="text-xl font-semibold mb-4">🔄 Workflow Status Log</h2>
            <div id="workflowStatus" class="bg-gray-50 p-4 rounded-lg min-h-[200px]">
                <div class="text-gray-500 text-center py-8">
                    Click buttons above to test workflow actions.<br>
                    The complete flow is: <strong>Student Submit → Supervisor Review → Department Head Approval</strong>
                </div>
            </div>
            <button onclick="clearLog()" class="mt-4 bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600">
                Clear Log
            </button>
        </div>

        <!-- Workflow Diagram -->
        <div class="mt-8 border-2 border-green-200 p-6 rounded-lg">
            <h2 class="text-xl font-semibold mb-4 text-green-700">📋 Workflow Diagram</h2>
            <div class="flex flex-wrap items-center justify-center space-x-4 text-sm">
                <div class="bg-blue-100 border border-blue-300 px-3 py-2 rounded">Draft</div>
                <span>→</span>
                <div class="bg-yellow-100 border border-yellow-300 px-3 py-2 rounded">Submitted</div>
                <span>→</span>
                <div class="bg-orange-100 border border-orange-300 px-3 py-2 rounded">Supervisor Review</div>
                <span>→</span>
                <div class="bg-purple-100 border border-purple-300 px-3 py-2 rounded">Department Review</div>
                <span>→</span>
                <div class="bg-green-100 border border-green-300 px-3 py-2 rounded">Approved</div>
            </div>
            <div class="mt-4 text-center text-gray-600">
                <small>Topics can be sent back for revision at any review stage</small>
            </div>
        </div>

        <!-- API Endpoints Reference -->
        <div class="mt-8 border-2 border-indigo-200 p-6 rounded-lg">
            <h2 class="text-xl font-semibold mb-4 text-indigo-700">🔧 API Endpoints</h2>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                <div>
                    <h3 class="font-semibold mb-2">Student Endpoints:</h3>
                    <ul class="space-y-1 text-gray-600">
                        <li><code>POST /api/topic/save-draft</code></li>
                        <li><code>POST /api/topic/submit-for-review</code></li>
                        <li><code>POST /api/topic/{id}/comment</code></li>
                    </ul>
                </div>
                <div>
                    <h3 class="font-semibold mb-2">Supervisor Endpoints:</h3>
                    <ul class="space-y-1 text-gray-600">
                        <li><code>POST /api/topic/{id}/supervisor-approve</code></li>
                        <li><code>POST /api/topic/{id}/supervisor-revision</code></li>
                    </ul>
                </div>
                <div>
                    <h3 class="font-semibold mb-2">Department Head Endpoints:</h3>
                    <ul class="space-y-1 text-gray-600">
                        <li><code>POST /api/topic/{id}/approve</code></li>
                        <li><code>POST /api/topic/{id}/reject</code></li>
                    </ul>
                </div>
                <div>
                    <h3 class="font-semibold mb-2">View Endpoints:</h3>
                    <ul class="space-y-1 text-gray-600">
                        <li><code>GET /topic/register</code></li>
                        <li><code>GET /supervisor/topics</code></li>
                        <li><code>GET /department/topics</code></li>
                    </ul>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        function testAction(message, color) {
            logAction(message, color);
        }
        
        function logAction(message, color = 'blue') {
            const statusDiv = document.getElementById('workflowStatus');
            const timestamp = new Date().toLocaleTimeString();
            const colorClasses = {
                'blue': 'bg-blue-50 border-l-4 border-blue-400 text-blue-700',
                'green': 'bg-green-50 border-l-4 border-green-400 text-green-700',
                'yellow': 'bg-yellow-50 border-l-4 border-yellow-400 text-yellow-700',
                'orange': 'bg-orange-50 border-l-4 border-orange-400 text-orange-700',
                'red': 'bg-red-50 border-l-4 border-red-400 text-red-700',
                'purple': 'bg-purple-50 border-l-4 border-purple-400 text-purple-700',
                'gray': 'bg-gray-50 border-l-4 border-gray-400 text-gray-700'
            };
            
            if (statusDiv.innerHTML.includes('Click buttons above')) {
                statusDiv.innerHTML = '';
            }
            
            statusDiv.innerHTML += '<div class="mb-3 p-3 rounded ' + (colorClasses[color] || colorClasses['blue']) + '">' + 
                                  '<div class="flex justify-between items-start">' +
                                  '<span class="font-medium">' + message + '</span>' +
                                  '<span class="text-xs opacity-75">' + timestamp + '</span>' +
                                  '</div></div>';
            statusDiv.scrollTop = statusDiv.scrollHeight;
        }
        
        function clearLog() {
            const statusDiv = document.getElementById('workflowStatus');
            statusDiv.innerHTML = '<div class="text-gray-500 text-center py-8">' +
                                'Log cleared. Click buttons above to test workflow actions.' +
                                '</div>';
        }
        
        // Auto-scroll to bottom when new entries are added
        function scrollToBottom() {
            const statusDiv = document.getElementById('workflowStatus');
            statusDiv.scrollTop = statusDiv.scrollHeight;
        }
    </script>
</body>
</html>
`))
	})

	// Add this route to your routes.go
	r.Get("/test/topic-workflow", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Topic Workflow Testing</title>
    <link rel="stylesheet" href="/assets/css/output.css">
    <script src="https://unpkg.com/htmx.org@1.9.3"></script>
</head>
<body class="bg-gray-100 p-8">
    <div class="max-w-4xl mx-auto bg-white rounded-lg p-6">
        <h1 class="text-3xl font-bold mb-8">🧪 Topic Workflow Testing Dashboard</h1>
        
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <!-- Form Access Tests -->
            <div class="border-2 border-blue-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-blue-700">📝 Form Access</h2>
                <div class="space-y-3">
                    <a href="/topic/register?locale=en" target="_blank" 
                       class="block bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 text-center">
                        English Form
                    </a>
                    <a href="/topic/register?locale=lt" target="_blank"
                       class="block bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 text-center">
                        Lithuanian Form
                    </a>
                    <a href="/topic/register" target="_blank"
                       class="block bg-blue-400 text-white px-4 py-2 rounded hover:bg-blue-500 text-center">
                        Default Form
                    </a>
                </div>
            </div>

            <!-- API Tests -->
            <div class="border-2 border-green-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-green-700">🔌 API Testing</h2>
                <div class="space-y-3">
                    <button onclick="testSaveDraft()" 
                            class="w-full bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700">
                        Test Save Draft
                    </button>
                    <button onclick="testSubmit()" 
                            class="w-full bg-green-500 text-white px-4 py-2 rounded hover:bg-green-600">
                        Test Submit
                    </button>
                    <button onclick="testAddComment()" 
                            class="w-full bg-green-400 text-white px-4 py-2 rounded hover:bg-green-500">
                        Test Add Comment
                    </button>
                </div>
            </div>

            <!-- Workflow Tests -->
            <div class="border-2 border-purple-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-purple-700">🔄 Workflow Testing</h2>
                <div class="space-y-3">
                    <button onclick="simulateWorkflow()" 
                            class="w-full bg-purple-600 text-white px-4 py-2 rounded hover:bg-purple-700">
                        Simulate Full Workflow
                    </button>
                    <button onclick="resetTopic()" 
                            class="w-full bg-purple-500 text-white px-4 py-2 rounded hover:bg-purple-600">
                        Reset Topic (Admin)
                    </button>
                    <a href="/debug/topic-workflow" target="_blank"
                       class="block bg-purple-400 text-white px-4 py-2 rounded hover:bg-purple-500 text-center">
                        Workflow Debug
                    </a>
                </div>
            </div>

            <!-- Role Tests -->
            <div class="border-2 border-orange-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-orange-700">👥 Role Testing</h2>
                <div class="space-y-3">
                    <button onclick="testAsStudent()" 
                            class="w-full bg-orange-600 text-white px-4 py-2 rounded hover:bg-orange-700">
                        Test as Student
                    </button>
                    <button onclick="testAsSupervisor()" 
                            class="w-full bg-orange-500 text-white px-4 py-2 rounded hover:bg-orange-600">
                        Test as Supervisor
                    </button>
                    <button onclick="testAsDepartmentHead()" 
                            class="w-full bg-orange-400 text-white px-4 py-2 rounded hover:bg-orange-500">
                        Test as Dept. Head
                    </button>
                </div>
            </div>

            <!-- Database Tests -->
            <div class="border-2 border-red-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-red-700">🗄️ Database Testing</h2>
                <div class="space-y-3">
                    <button onclick="checkDatabase()" 
                            class="w-full bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700">
                        Check DB Status
                    </button>
                    <button onclick="viewTopics()" 
                            class="w-full bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600">
                        View All Topics
                    </button>
                    <button onclick="createTestData()" 
                            class="w-full bg-red-400 text-white px-4 py-2 rounded hover:bg-red-500">
                        Create Test Data
                    </button>
                </div>
            </div>

            <!-- Network/Console Tests -->
            <div class="border-2 border-gray-200 p-6 rounded-lg">
                <h2 class="text-xl font-semibold mb-4 text-gray-700">🔍 Debug Tools</h2>
                <div class="space-y-3">
                    <button onclick="openDevTools()" 
                            class="w-full bg-gray-600 text-white px-4 py-2 rounded hover:bg-gray-700">
                        Open Dev Tools
                    </button>
                    <button onclick="clearStorage()" 
                            class="w-full bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600">
                        Clear Local Storage
                    </button>
                    <button onclick="showNetworkLog()" 
                            class="w-full bg-gray-400 text-white px-4 py-2 rounded hover:bg-gray-500">
                        Show Network Log
                    </button>
                </div>
            </div>
        </div>

        <!-- Test Results -->
        <div class="mt-8 border-2 border-gray-200 p-6 rounded-lg">
            <h2 class="text-xl font-semibold mb-4">📊 Test Results</h2>
            <div id="test-results" class="bg-gray-50 p-4 rounded-lg min-h-[200px]">
                <div class="text-gray-500 text-center py-8">
                    Click test buttons above to see results here
                </div>
            </div>
            <button onclick="clearResults()" class="mt-4 bg-gray-500 text-white px-4 py-2 rounded hover:bg-gray-600">
                Clear Results
            </button>
        </div>

        <!-- Quick Test Form -->
        <div class="mt-8 border-2 border-cyan-200 p-6 rounded-lg">
            <h2 class="text-xl font-semibold mb-4 text-cyan-700">⚡ Quick Test Form</h2>
            <form id="quick-test-form" class="space-y-4">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <input type="text" id="test-title" placeholder="Topic Title" class="border rounded px-3 py-2">
                    <input type="text" id="test-supervisor" placeholder="Supervisor Name" class="border rounded px-3 py-2">
                </div>
                <textarea id="test-problem" placeholder="Problem Description (50+ chars)" class="w-full border rounded px-3 py-2" rows="3"></textarea>
                <div class="flex gap-4">
                    <button type="button" onclick="quickSaveDraft()" class="bg-cyan-600 text-white px-4 py-2 rounded hover:bg-cyan-700">
                        Quick Save Draft
                    </button>
                    <button type="button" onclick="quickSubmit()" class="bg-cyan-500 text-white px-4 py-2 rounded hover:bg-cyan-600">
                        Quick Submit
                    </button>
                </div>
            </form>
        </div>
    </div>
    
    <script>
        function logResult(message, type = 'info') {
            const results = document.getElementById('test-results');
            const timestamp = new Date().toLocaleTimeString();
            const colorClass = {
                'info': 'text-blue-600',
                'success': 'text-green-600', 
                'error': 'text-red-600',
                'warning': 'text-yellow-600'
            }[type] || 'text-gray-600';
            
            if (results.innerHTML.includes('Click test buttons')) {
                results.innerHTML = '';
            }
            
            results.innerHTML += '<div class="mb-2 p-2 border-l-4 border-' + type + '-400 bg-' + type + '-50">' +
                '<span class="text-xs text-gray-500">' + timestamp + '</span> ' +
                '<span class="' + colorClass + '">' + message + '</span></div>';
            results.scrollTop = results.scrollHeight;
        }

        function testSaveDraft() {
            logResult('Testing save draft API...', 'info');
            fetch('/api/topic/save-draft', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: 'title=Test Topic&supervisor=Test Supervisor&problem=Test problem description'
            })
            .then(response => response.text())
            .then(data => logResult('Save draft response: ' + data.substring(0, 100), 'success'))
            .catch(error => logResult('Save draft error: ' + error, 'error'));
        }

        function testSubmit() {
            logResult('Testing submit API...', 'info');
            const testData = {
                title: 'Complete Test Topic',
                title_en: 'Complete Test Topic English',
                supervisor: 'Test Supervisor',
                problem: 'This is a complete problem description that is longer than 50 characters',
                objective: 'Complete test objective description',
                tasks: 'Complete test tasks description that is longer than required'
            };
            
            const formData = new URLSearchParams(testData);
            
            fetch('/api/topic/submit-for-review', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: formData
            })
            .then(response => response.text())
            .then(data => logResult('Submit response: ' + data.substring(0, 100), 'success'))
            .catch(error => logResult('Submit error: ' + error, 'error'));
        }

        function testAddComment() {
            logResult('Testing add comment...', 'info');
            // This would need a topic ID - implement based on your needs
            logResult('Add comment test requires existing topic', 'warning');
        }

        function simulateWorkflow() {
            logResult('Simulating full workflow...', 'info');
            logResult('1. Creating draft topic', 'info');
            setTimeout(() => logResult('2. Submitting for review', 'info'), 1000);
            setTimeout(() => logResult('3. Supervisor approval', 'info'), 2000);
            setTimeout(() => logResult('4. Department approval', 'success'), 3000);
        }

        function resetTopic() {
            logResult('Topic reset (admin function)', 'warning');
        }

        function testAsStudent() {
            logResult('Testing student view...', 'info');
            window.open('/topic/register', '_blank');
        }

        function testAsSupervisor() {
            logResult('Testing supervisor view...', 'info');
            window.open('/supervisor/topics', '_blank');
        }

        function testAsDepartmentHead() {
            logResult('Testing department head view...', 'info');
            window.open('/department/topics', '_blank');
        }

        function checkDatabase() {
            logResult('Checking database status...', 'info');
            fetch('/health')
            .then(response => response.json())
            .then(data => logResult('Database status: ' + data.status, 'success'))
            .catch(error => logResult('Database check error: ' + error, 'error'));
        }

        function viewTopics() {
            logResult('Opening topics view...', 'info');
            window.open('/admin/topics', '_blank');
        }

        function createTestData() {
            logResult('Creating test data...', 'info');
            logResult('Test data creation would go here', 'warning');
        }

        function openDevTools() {
            logResult('Open browser dev tools (F12) to see network requests', 'info');
        }

        function clearStorage() {
            localStorage.clear();
            sessionStorage.clear();
            logResult('Local storage cleared', 'success');
        }

        function showNetworkLog() {
            logResult('Check browser network tab for API calls', 'info');
        }

        function quickSaveDraft() {
            const title = document.getElementById('test-title').value;
            const supervisor = document.getElementById('test-supervisor').value;
            const problem = document.getElementById('test-problem').value;
            
            if (!title || !supervisor || !problem) {
                logResult('Please fill all quick test fields', 'warning');
                return;
            }
            
            logResult('Quick saving: ' + title, 'info');
            const formData = new URLSearchParams({title, supervisor, problem});
            
            fetch('/api/topic/save-draft', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: formData
            })
            .then(response => response.text())
            .then(data => logResult('Quick save successful', 'success'))
            .catch(error => logResult('Quick save error: ' + error, 'error'));
        }

        function quickSubmit() {
            const title = document.getElementById('test-title').value;
            const supervisor = document.getElementById('test-supervisor').value;
            const problem = document.getElementById('test-problem').value;
            
            if (!title || !supervisor || problem.length < 50) {
                logResult('Quick submit requires all fields (problem 50+ chars)', 'warning');
                return;
            }
            
            logResult('Quick submitting: ' + title, 'info');
            const formData = new URLSearchParams({
                title, 
                title_en: title + ' (EN)',
                supervisor, 
                problem,
                objective: 'Quick test objective',
                tasks: 'Quick test tasks description'
            });
            
            fetch('/api/topic/submit-for-review', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: formData
            })
            .then(response => response.text())
            .then(data => logResult('Quick submit successful', 'success'))
            .catch(error => logResult('Quick submit error: ' + error, 'error'));
        }

        function clearResults() {
            document.getElementById('test-results').innerHTML = 
                '<div class="text-gray-500 text-center py-8">Test results cleared</div>';
        }
    </script>
</body>
</html>`
		w.Write([]byte(html))
	})

	return r
}
