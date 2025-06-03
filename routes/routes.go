// routes/routes.go
package routes

import (
	"FinalProjectManagementApp/database"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	// Initialize handlers
	dashboardHandlers := handlers.NewDashboardHandlers(db)
	authHandlers := handlers.NewAuthHandlers(authMiddleware)
	topicHandlers := handlers.NewTopicHandlers(db.DB)
	supervisorReportHandler := handlers.NewSupervisorReportHandler(db)
	studentListHandler := handlers.NewStudentListHandler(db)
	uploadHandlers := handlers.NewUploadHandlers(db)
	commissionHandler := handlers.NewCommissionHandler(db)

	reviewerAccessHandler := handlers.NewReviewerAccessHandler(db)

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

	// Health check endpoint
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

	// Public API routes (accessible without auth for commission members)
	r.Get("/api/documents/{id}", handlers.DocumentsAPIHandler)
	r.Get("/api/students/{id}/documents", handlers.DocumentsAPIHandler)

	// Reviewer-specific route with token
	r.Get("/reviewer/students", studentListHandler.ReviewerStudentsList)

	// Reviewer-specific routes with token (no auth required)
	r.Route("/reviewer/{accessToken}", func(r chi.Router) {
		r.Get("/", studentListHandler.ShowReviewerStudentsWithToken)
		r.Get("/student/{studentId}/review", studentListHandler.ReviewerReportModalHandlerWithToken)
		r.Post("/student/{studentId}/review/submit", studentListHandler.ReviewerReportSubmitHandlerWithToken)
	})

	// PUBLIC DOCUMENTS ROUTES
	r.Get("/api/public/documents/{id}", createPublicDocumentsHandler(db))
	r.Get("/api/public/students/{id}/documents", createPublicDocumentsHandler(db))
	r.Get("/api/public/documents/{id}/preview", createPublicDocumentPreviewHandler(db))
	r.Get("/api/public/documents/{id}/download", createPublicDocumentDownloadHandler(db))

	// Public commission access (no auth required)

	r.Route("/commission/{accessCode}", func(r chi.Router) {
		r.Get("/", commissionHandler.ShowStudentList)
		r.Get("/topic-registration/{studentId}", createCommissionTopicRegistrationHandler(db))

		// Add repository routes here
		if repositoryHandler != nil {
			r.Route("/repository", func(r chi.Router) {
				r.Get("/student/{studentId}", createCommissionRepositoryHandler(repositoryHandler, db))
				r.Get("/student/{studentId}/download", createCommissionDownloadHandler(repositoryHandler, db))
				r.Get("/student/{studentId}/browse/*", createCommissionBrowseHandler(repositoryHandler, db))
				r.Get("/student/{studentId}/file/*", createCommissionFileHandler(repositoryHandler, db))
				r.Get("/student/{studentId}/tree", createCommissionTreeHandler(repositoryHandler, db))
			})
		}
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)

		// @TODO REVIEW IF NEED TOPIC REGISTRATION
		r.Get("/topic-registration/{studentId}", topicHandlers.ShowTopicRegistrationModal)

		// TEST FOR GITHUB
		r.Get("/test-upload", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "static/upload_test.html")
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
		r.Post("/supervisor-report/{id}/submit", supervisorReportHandler.SubmitSupervisorReport)
		r.Post("/supervisor-report/{id}/save-draft", supervisorReportHandler.SaveSupervisorDraft)

		// Upload routes
		r.Get("/upload", handlers.ShowUploadPage)
		r.Post("/api/upload", handlers.UploadFileHandler)

		// API endpoint for user info
		r.Get("/api/auth/user", authMiddleware.UserInfoHandler)

		// Dashboard
		r.Get("/dashboard", dashboardHandlers.DashboardHandler)

		// Student List
		r.Get("/students-list", studentListHandler.StudentTableDisplayHandler)

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

		// REVIEWER FORM HANDLER
		r.Route("/reviewer-report", func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.Get("/{id}/compact-modal", studentListHandler.ReviewerReportModalHandler)
			r.Post("/{id}/submit", studentListHandler.ReviewerReportSubmitHandler)
			r.Post("/{id}/save-draft", studentListHandler.ReviewerReportSaveDraftHandler)
		})

		// Admin routes - MERGED WITH IMPORT/EXPORT FUNCTIONALITY
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware.RequireRole(auth.RoleAdmin, auth.RoleDepartmentHead))

			r.Get("/reviewer-access", reviewerAccessHandler.ShowManagementPage)
			r.Post("/reviewer-access/create", reviewerAccessHandler.CreateReviewerAccess)
			r.Delete("/reviewer-access/{accessToken}", reviewerAccessHandler.DeactivateReviewerAccess)

			r.Get("/commission", commissionHandler.ShowManagementPage)
			r.Post("/commission/create", commissionHandler.CreateAccess)
			r.Delete("/commission/{accessCode}", commissionHandler.DeactivateAccess)
			r.Get("/commission/list", commissionHandler.ListActiveAccess)

			r.Get("/dashboard", dashboardHandlers.DashboardHandler)

			// Import/Export routes
			r.Get("/import/modal", studentListHandler.ImportModalHandler)
			r.Post("/import/preview", studentListHandler.PreviewHandler)
			r.Post("/import/process", studentListHandler.ProcessImportHandler)
			r.Get("/import/sample-excel", studentListHandler.SampleExcelHandler)
			r.Get("/export/students", studentListHandler.ExportHandler)

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

	return r
}

// Helper function to validate commission access and return member
func validateCommissionAccess(db *sqlx.DB, accessCode string) (*database.CommissionMember, error) {
	var member database.CommissionMember
	query := `SELECT * FROM commission_members WHERE access_code = ? AND is_active = true`
	err := db.Get(&member, query, accessCode)
	if err != nil {
		return nil, fmt.Errorf("invalid access token")
	}

	// Check if expired
	if time.Now().Unix() > member.ExpiresAt {
		return nil, fmt.Errorf("access token has expired")
	}

	// Check max access limit
	if member.MaxAccess > 0 && member.AccessCount >= member.MaxAccess {
		return nil, fmt.Errorf("access limit reached")
	}

	return &member, nil
}

// Commission handler functions
func createCommissionRepositoryHandler(repositoryHandler *handlers.RepositoryHandler, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessCode := chi.URLParam(r, "accessCode")

		member, err := validateCommissionAccess(db, accessCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Update access count
		updateQuery := `UPDATE commission_members SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?`
		db.Exec(updateQuery, time.Now().Unix(), member.ID)

		// Create fake authenticated user for commission member
		fakeUser := &auth.AuthenticatedUser{
			Email: "commission_" + accessCode,
			Role:  auth.RoleCommissionMember,
			Name:  "Commission Member",
		}

		// Add to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, fakeUser)

		// Call the handler with authenticated context
		repositoryHandler.ViewStudentRepository(w, r.WithContext(ctx))
	}
}

func createCommissionBrowseHandler(repositoryHandler *handlers.RepositoryHandler, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessCode := chi.URLParam(r, "accessCode")

		member, err := validateCommissionAccess(db, accessCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Create fake authenticated user for commission member
		fakeUser := &auth.AuthenticatedUser{
			Email:      "commission_" + accessCode,
			Role:       auth.RoleCommissionMember,
			Name:       "Commission Member",
			Department: member.Department,
		}

		// Add to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, fakeUser)

		// Call the handler with authenticated context
		repositoryHandler.ViewStudentRepositoryPath(w, r.WithContext(ctx))
	}
}

func createCommissionFileHandler(repositoryHandler *handlers.RepositoryHandler, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessCode := chi.URLParam(r, "accessCode")

		member, err := validateCommissionAccess(db, accessCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Create fake authenticated user for commission member
		fakeUser := &auth.AuthenticatedUser{
			Email:      "commission_" + accessCode,
			Role:       auth.RoleCommissionMember,
			Name:       "Commission Member",
			Department: member.Department,
		}

		// Add to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, fakeUser)

		// Call the handler with authenticated context
		repositoryHandler.ViewFileContent(w, r.WithContext(ctx))
	}
}

func createCommissionTreeHandler(repositoryHandler *handlers.RepositoryHandler, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessCode := chi.URLParam(r, "accessCode")

		member, err := validateCommissionAccess(db, accessCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Create fake authenticated user for commission member
		fakeUser := &auth.AuthenticatedUser{
			Email:      "commission_" + accessCode,
			Role:       auth.RoleCommissionMember,
			Name:       "Commission Member",
			Department: member.Department,
		}

		// Add to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, fakeUser)

		// Call the handler with authenticated context
		repositoryHandler.GetRepositoryTree(w, r.WithContext(ctx))
	}
}

func createCommissionDownloadHandler(repositoryHandler *handlers.RepositoryHandler, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessCode := chi.URLParam(r, "accessCode")

		member, err := validateCommissionAccess(db, accessCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Log the download action
		updateQuery := `UPDATE commission_members SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?`
		db.Exec(updateQuery, time.Now().Unix(), member.ID)

		// Create fake authenticated user for commission member
		fakeUser := &auth.AuthenticatedUser{
			Email:      "commission_" + accessCode,
			Role:       auth.RoleCommissionMember,
			Name:       "Commission Member",
			Department: member.Department,
		}

		// Add to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, fakeUser)

		// Call the handler with authenticated context
		repositoryHandler.DownloadRepository(w, r.WithContext(ctx))
	}
}

// PUBLIC ROUTES FOR DOCUMENTS

// In routes.go, add these functions:

// createPublicDocumentsHandler - fetches documents without authentication
func createPublicDocumentsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentIDStr := chi.URLParam(r, "id")
		studentID, err := strconv.Atoi(studentIDStr)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":     "Invalid student ID",
				"documents": []interface{}{},
			})
			return
		}

		// Query documents from database
		query := `
            SELECT id, document_type, file_path, uploaded_date, 
                   file_size, mime_type, original_filename
            FROM documents 
            WHERE student_record_id = ?
            ORDER BY uploaded_date DESC
        `

		var documents []struct {
			ID               int       `db:"id"`
			DocumentType     string    `db:"document_type"`
			FilePath         string    `db:"file_path"`
			UploadedDate     time.Time `db:"uploaded_date"`
			FileSize         *int64    `db:"file_size"`
			MimeType         *string   `db:"mime_type"`
			OriginalFilename *string   `db:"original_filename"`
		}

		err = db.Select(&documents, query, studentID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":     "Failed to fetch documents",
				"documents": []interface{}{},
			})
			return
		}

		// Transform for response
		responseDocuments := make([]map[string]interface{}, 0)
		for _, doc := range documents {
			hasPreview := false
			if doc.MimeType != nil && strings.HasPrefix(*doc.MimeType, "application/pdf") {
				hasPreview = true
			}

			docType := doc.DocumentType
			if docType == "thesis" || docType == "thesis_pdf" {
				docType = "thesis_pdf"
			}

			responseDocuments = append(responseDocuments, map[string]interface{}{
				"id":         doc.ID,
				"type":       docType,
				"hasPreview": hasPreview,
				"filename":   doc.OriginalFilename,
				"size":       doc.FileSize,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"documents": responseDocuments,
		})
	}
}

// createPublicDocumentPreviewHandler - handles document preview without authentication
func createPublicDocumentPreviewHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		documentIDStr := chi.URLParam(r, "id")
		documentID, err := strconv.Atoi(documentIDStr)
		if err != nil {
			http.Error(w, "Invalid document ID", http.StatusBadRequest)
			return
		}

		// Get document info
		var doc struct {
			ID               int     `db:"id"`
			StudentRecordID  int     `db:"student_record_id"`
			DocumentType     string  `db:"document_type"`
			FilePath         string  `db:"file_path"`
			MimeType         *string `db:"mime_type"`
			OriginalFilename *string `db:"original_filename"`
		}

		query := `SELECT id, student_record_id, document_type, file_path, mime_type, original_filename 
                  FROM documents WHERE id = ?`
		err = db.Get(&doc, query, documentID)
		if err != nil {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}

		// Check if document can be previewed
		if doc.MimeType == nil || !strings.HasPrefix(*doc.MimeType, "application/pdf") {
			http.Error(w, "Document cannot be previewed", http.StatusBadRequest)
			return
		}

		// Read file
		filePath := doc.FilePath
		if !filepath.IsAbs(filePath) {
			// Assuming uploads are in ./uploads directory
			filePath = filepath.Join("./uploads", filePath)
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Error opening file %s: %v", filePath, err)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		// Get file info
		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		// Set headers for PDF viewing
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "inline; filename=\""+*doc.OriginalFilename+"\"")
		w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

		// Copy file to response
		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("Error serving file: %v", err)
		}
	}
}

// createPublicDocumentDownloadHandler - handles document download without authentication
func createPublicDocumentDownloadHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		documentIDStr := chi.URLParam(r, "id")
		documentID, err := strconv.Atoi(documentIDStr)
		if err != nil {
			http.Error(w, "Invalid document ID", http.StatusBadRequest)
			return
		}

		// Get document info
		var doc struct {
			ID               int     `db:"id"`
			StudentRecordID  int     `db:"student_record_id"`
			DocumentType     string  `db:"document_type"`
			FilePath         string  `db:"file_path"`
			MimeType         *string `db:"mime_type"`
			OriginalFilename *string `db:"original_filename"`
		}

		query := `SELECT id, student_record_id, document_type, file_path, mime_type, original_filename 
                  FROM documents WHERE id = ?`
		err = db.Get(&doc, query, documentID)
		if err != nil {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}

		// Read file
		filePath := doc.FilePath
		if !filepath.IsAbs(filePath) {
			// Assuming uploads are in ./uploads directory
			filePath = filepath.Join("./uploads", filePath)
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Error opening file %s: %v", filePath, err)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		// Get file info
		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		// Determine content type
		contentType := "application/octet-stream"
		if doc.MimeType != nil {
			contentType = *doc.MimeType
		}

		// Set download headers
		filename := "document"
		if doc.OriginalFilename != nil {
			filename = *doc.OriginalFilename
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

		// Copy file to response
		_, err = io.Copy(w, file)
		if err != nil {
			log.Printf("Error serving file: %v", err)
		}
	}
}

// Add this function to routes.go
func createCommissionTopicRegistrationHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accessCode := chi.URLParam(r, "accessCode")
		studentIDStr := chi.URLParam(r, "studentId")

		// Validate student ID format
		_, err := strconv.Atoi(studentIDStr)
		if err != nil {
			http.Error(w, "Invalid student ID", http.StatusBadRequest)
			return
		}

		// Validate commission access
		member, err := validateCommissionAccess(db, accessCode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Update access count
		updateQuery := `UPDATE commission_members SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?`
		db.Exec(updateQuery, time.Now().Unix(), member.ID)

		// Create topic handler with database connection
		topicHandlers := handlers.NewTopicHandlers(db.DB)

		// Create fake authenticated user for commission member
		fakeUser := &auth.AuthenticatedUser{
			Email:      "commission_" + accessCode,
			Role:       auth.RoleCommissionMember,
			Name:       "Commission Member",
			Department: member.Department,
		}

		// Add to context
		ctx := context.WithValue(r.Context(), auth.UserContextKey, fakeUser)

		// Call the topic registration modal handler
		topicHandlers.ShowTopicRegistrationModal(w, r.WithContext(ctx))
	}
}
