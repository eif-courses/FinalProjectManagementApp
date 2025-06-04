// handlers/repository.go - Complete Repository Handler
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/repository"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/types"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type RepositoryHandler struct {
	db           *sqlx.DB
	githubConfig *database.GitHubConfig
	client       *http.Client
}

func NewRepositoryHandler(db *sqlx.DB, githubConfig *database.GitHubConfig) *RepositoryHandler {
	return &RepositoryHandler{
		db:           db,
		githubConfig: githubConfig,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// ================================
// MAIN HANDLERS
// ================================

// ViewStudentRepository displays the repository page using templ
func (h *RepositoryHandler) ViewStudentRepository(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Extract access code
	accessInfo := h.extractAccessInfo(r)
	//accessCode := h.extractAccessCode(r)

	// Skip permission check for commission members
	// UPDATE PERMISSION CHECK
	if user.Role == auth.RoleCommissionMember {
		if !accessInfo.IsValid() {
			http.Error(w, "Access code required", http.StatusForbidden)
			return
		}
		// Add validation for the access code here if needed
	} else {
		if !h.canViewRepository(user, studentID) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
	}

	student, err := h.getStudentRecord(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	currentLocale := h.getCurrentLocale(r)

	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		component := repository.NoRepositoryPage(user, student, currentLocale, accessInfo)
		templ.Handler(component).ServeHTTP(w, r)
		return
	}

	repoContents, err := h.getRepositoryContents(repoInfo)
	if err != nil {
		repoContents = &types.RepositoryContents{
			Files:   []types.RepositoryFile{},
			Commits: []types.CommitInfo{},
			Stats:   types.RepositoryStats{},
			Error:   err.Error(),
		}
	}

	// Pass accessCode to the component
	component := repository.RepositoryPage(user, student, repoInfo, repoContents, currentLocale, accessInfo)
	templ.Handler(component).ServeHTTP(w, r)
}

// GetRepositoryAPI returns repository data as JSON for AJAX requests
func (h *RepositoryHandler) GetRepositoryAPI(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	if !h.canViewRepository(user, studentID) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "No repository available",
		})
		return
	}

	repoContents, err := h.getRepositoryContents(repoInfo)
	if err != nil {
		repoContents = &types.RepositoryContents{Error: err.Error()}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"repository": repoInfo,
		"contents":   repoContents,
	})
}

// DownloadRepository creates a ZIP download of the repository
func (h *RepositoryHandler) DownloadRepository(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Special handling for commission members
	if user.Role == auth.RoleCommissionMember {
		accessInfo := h.extractAccessInfo(r)
		if !accessInfo.IsValid() {
			http.Error(w, "Valid access code required for commission members", http.StatusForbidden)
			return
		}
		// Commission members can download during commission period
		if !h.isCommissionPeriod() {
			http.Error(w, "Downloads not available outside commission period", http.StatusForbidden)
			return
		}
	} else {
		// Regular permission check for other roles
		if !h.canDownloadRepository(user, studentID) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
	}

	// Rest of your download logic...
	student, err := h.getStudentRecord(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	// Check for nil repository URL
	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		http.Error(w, "No repository available for download", http.StatusNotFound)
		return
	}

	// Create download URL for GitHub repository
	repoName := h.extractRepoName(*repoInfo.RepositoryURL)
	downloadURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/main",
		h.githubConfig.Organization, repoName)

	// Proxy the download with authentication
	h.proxyRepositoryDownload(w, r, downloadURL, student)
}

// ================================
// PERMISSION CHECKS
// ================================

func (h *RepositoryHandler) canViewRepository(user *auth.AuthenticatedUser, studentID int) bool {
	switch user.Role {
	case auth.RoleAdmin, auth.RoleDepartmentHead:
		return true
	case auth.RoleSupervisor:
		return h.isSupervisorForStudent(user.Email, studentID)
	case auth.RoleReviewer:
		return h.isReviewerForStudent(user.Email, studentID)
	case auth.RoleCommissionMember:
		return h.isCommissionPeriod()
	case auth.RoleStudent:
		return h.isStudentOwnRepository(user.Email, studentID)
	default:
		return false
	}
}

func (h *RepositoryHandler) isStudentOwnRepository(email string, studentID int) bool {
	var count int
	query := "SELECT COUNT(*) FROM student_records WHERE id = ? AND student_email = ?"
	h.db.Get(&count, query, studentID, email)
	return count > 0
}

func (h *RepositoryHandler) canDownloadRepository(user *auth.AuthenticatedUser, studentID int) bool {
	// More restrictive than viewing
	switch user.Role {
	case auth.RoleAdmin, auth.RoleDepartmentHead:
		return true
	case auth.RoleSupervisor:
		return h.isSupervisorForStudent(user.Email, studentID)
	case auth.RoleReviewer:
		return h.isReviewerForStudent(user.Email, studentID)
	case auth.RoleCommissionMember:
		// Commission members can download if they're in the commission period
		return h.isCommissionPeriod()
	default:
		return false
	}
}

func (h *RepositoryHandler) isSupervisorForStudent(email string, studentID int) bool {
	var count int
	query := "SELECT COUNT(*) FROM student_records WHERE id = ? AND supervisor_email = ?"
	h.db.Get(&count, query, studentID, email)
	return count > 0
}

func (h *RepositoryHandler) isReviewerForStudent(email string, studentID int) bool {
	var count int
	query := "SELECT COUNT(*) FROM student_records WHERE id = ? AND reviewer_email = ?"
	h.db.Get(&count, query, studentID, email)
	return count > 0
}

func (h *RepositoryHandler) isCommissionPeriod() bool {
	// Check if we're in defense period (implement your logic)
	// For now, return true - you can add date range checks here
	return true
}

// ================================
// DATABASE OPERATIONS
// ================================

func (h *RepositoryHandler) getStudentRecord(studentID int) (*database.StudentRecord, error) {
	var student database.StudentRecord
	query := `
		SELECT id, student_name, student_lastname, student_email, student_number,
			   final_project_title, final_project_title_en, study_program, department, 
			   supervisor_email, reviewer_email, reviewer_name, created_at, updated_at
		FROM student_records WHERE id = ?
	`
	err := h.db.Get(&student, query, studentID)
	return &student, err
}

func (h *RepositoryHandler) getStudentRepository(studentID int) (*database.Document, error) {
	var doc database.Document
	query := `
		SELECT id, student_record_id, document_type, file_path, repository_url,
			   repository_id, commit_id, submission_id, validation_status, 
			   upload_status, uploaded_date, file_size, original_filename
		FROM documents 
		WHERE student_record_id = ? AND document_type = 'thesis_source_code'
		ORDER BY uploaded_date DESC LIMIT 1
	`
	err := h.db.Get(&doc, query, studentID)
	return &doc, err
}

// ================================
// GITHUB API OPERATIONS
// ================================

func (h *RepositoryHandler) getRepositoryContents(repoInfo *database.Document) (*types.RepositoryContents, error) {

	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		return nil, fmt.Errorf("no repository URL available")
	}

	repoName := h.extractRepoName(*repoInfo.RepositoryURL)
	if repoName == "" {
		return nil, fmt.Errorf("could not extract repository name from URL")
	}

	// Get repository files
	files, err := h.getRepositoryFiles(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository files: %w", err)
	}

	// Get recent commits
	commits, err := h.getRepositoryCommits(repoName)
	if err != nil {
		commits = []types.CommitInfo{} // Don't fail on commits error
	}

	// Calculate stats
	stats := h.calculateRepositoryStats(files, commits)

	return &types.RepositoryContents{
		Files:   files,
		Commits: commits,
		Stats:   stats,
	}, nil
}

func (h *RepositoryHandler) getRepositoryFiles(repoName string) ([]types.RepositoryFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents",
		h.githubConfig.Organization, repoName)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("repository not found or access denied")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var githubFiles []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&githubFiles); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	var files []types.RepositoryFile
	for _, file := range githubFiles {
		fileType := "file"
		if t, ok := file["type"].(string); ok {
			fileType = t
		}

		size := int64(0)
		if s, ok := file["size"].(float64); ok {
			size = int64(s)
		}

		files = append(files, types.RepositoryFile{
			Name: h.getStringValue(file, "name"),
			Path: h.getStringValue(file, "path"),
			Type: fileType,
			Size: size,
			URL:  h.getStringValue(file, "html_url"),
		})
	}

	return files, nil
}

func (h *RepositoryHandler) getRepositoryCommits(repoName string) ([]types.CommitInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=10",
		h.githubConfig.Organization, repoName)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var githubCommits []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&githubCommits); err != nil {
		return nil, fmt.Errorf("failed to decode commits: %w", err)
	}

	var commits []types.CommitInfo
	for _, commit := range githubCommits {
		commitData, ok := commit["commit"].(map[string]interface{})
		if !ok {
			continue
		}

		author, ok := commitData["author"].(map[string]interface{})
		if !ok {
			continue
		}

		dateStr := h.getStringValue(author, "date")
		date, _ := time.Parse(time.RFC3339, dateStr)

		sha := h.getStringValue(commit, "sha")
		if len(sha) > 7 {
			sha = sha[:7] // Short SHA
		}

		commits = append(commits, types.CommitInfo{
			SHA:     sha,
			Message: h.getStringValue(commitData, "message"),
			Author:  h.getStringValue(author, "name"),
			Date:    date,
			URL:     h.getStringValue(commit, "html_url"),
		})
	}

	return commits, nil
}

func (h *RepositoryHandler) calculateRepositoryStats(files []types.RepositoryFile, commits []types.CommitInfo) types.RepositoryStats {
	stats := types.RepositoryStats{
		TotalFiles:  len(files),
		Languages:   make(map[string]int),
		CommitCount: len(commits),
	}

	for _, file := range files {
		if file.Type == "file" {
			stats.TotalSize += file.Size

			// Determine language by extension
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name), "."))
			if lang := h.getLanguageFromExtension(ext); lang != "" {
				stats.Languages[lang]++
			}
		}
	}

	if len(commits) > 0 {
		stats.LastUpdated = commits[0].Date
	}

	return stats
}

// ================================
// UTILITY METHODS
// ================================

func (h *RepositoryHandler) getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func (h *RepositoryHandler) getLanguageFromExtension(ext string) string {
	languages := map[string]string{
		"go":   "Go",
		"py":   "Python",
		"js":   "JavaScript",
		"ts":   "TypeScript",
		"java": "Java",
		"cpp":  "C++",
		"c":    "C",
		"cs":   "C#",
		"php":  "PHP",
		"rb":   "Ruby",
		"rs":   "Rust",
		"html": "HTML",
		"css":  "CSS",
		"scss": "SCSS",
		"sass": "Sass",
		"sql":  "SQL",
		"md":   "Markdown",
		"json": "JSON",
		"xml":  "XML",
		"yaml": "YAML",
		"yml":  "YAML",
		"sh":   "Shell",
		"bat":  "Batch",
		"ps1":  "PowerShell",
		"vue":  "Vue",
		"jsx":  "React",
		"tsx":  "React",
	}
	return languages[ext]
}

func (h *RepositoryHandler) extractRepoName(repoURL string) string {
	// Remove trailing slash if present
	repoURL = strings.TrimSuffix(repoURL, "/")

	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (h *RepositoryHandler) proxyRepositoryDownload(w http.ResponseWriter, r *http.Request, downloadURL string, student *database.StudentRecord) {
	req, _ := http.NewRequest("GET", downloadURL, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, "Download failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		http.Error(w, "Download not available", http.StatusNotFound)
		return
	}

	// Set download headers
	filename := fmt.Sprintf("%s_%s_source_code.zip",
		student.StudentNumber,
		strings.ReplaceAll(student.StudentName, " ", "_"))

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Copy response
	io.Copy(w, resp.Body)
}

// ================================
// HTML RENDERING
// ================================

func (h *RepositoryHandler) renderRepositoryPage(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, student *database.StudentRecord, repoInfo *database.Document, contents *types.RepositoryContents, accessInfo database.AccessInfo) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := h.generateRepositoryHTML(user, student, repoInfo, contents, accessInfo)
	w.Write([]byte(html))
}

func (h *RepositoryHandler) renderNoRepositoryPage(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, student *database.StudentRecord) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := h.generateNoRepositoryHTML(user, student)
	w.Write([]byte(html))
}

func (h *RepositoryHandler) generateRepositoryHTML(user *auth.AuthenticatedUser, student *database.StudentRecord, repoInfo *database.Document, contents *types.RepositoryContents, accessInfo database.AccessInfo) string {
	// Safe repository URL handling
	repoURL := ""
	if repoInfo.RepositoryURL != nil {
		repoURL = *repoInfo.RepositoryURL
	}

	// Safe handling of optional fields
	uploadStatus := "unknown"
	validationStatus := "unknown"

	if repoInfo.UploadStatus != "" {
		uploadStatus = repoInfo.UploadStatus
	}
	if repoInfo.ValidationStatus != "" {
		validationStatus = repoInfo.ValidationStatus
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Repository: %s %s</title>
	<link rel="stylesheet" href="/assets/css/output.css">
	<style>
		.file-tree { font-family: monospace; }
		.file-item { padding: 8px 12px; border-bottom: 1px solid #e5e7eb; }
		.file-item:hover { background-color: #f9fafb; }
		.file-item:last-child { border-bottom: none; }
		.language-badge { 
			padding: 4px 8px; 
			border-radius: 12px; 
			font-size: 12px; 
			margin: 2px; 
			display: inline-block; 
			background-color: #dbeafe; 
			color: #1e40af;
		}
		.commit-item { 
			border: 1px solid #d1d5db; 
			border-radius: 6px; 
			padding: 12px; 
			margin-bottom: 8px; 
			background-color: #ffffff;
		}
		.status-badge { 
			padding: 4px 8px; 
			border-radius: 12px; 
			font-size: 11px; 
			font-weight: 600; 
		}
		.status-valid { background-color: #d1fae5; color: #065f46; }
		.status-completed { background-color: #dbeafe; color: #1e40af; }
		.status-pending { background-color: #fef3c7; color: #92400e; }
		.status-unknown { background-color: #f3f4f6; color: #374151; }
		.repo-header { 
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
			color: white; 
			padding: 2rem; 
			border-radius: 12px 12px 0 0; 
		}
		.stat-card { 
			background: white; 
			border-radius: 8px; 
			padding: 1rem; 
			box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1); 
		}
		.error-message {
			background-color: #fef2f2;
			color: #dc2626;
			padding: 1rem;
			border-radius: 8px;
			border: 1px solid #fecaca;
		}
	</style>
</head>
<body class="bg-gray-100 min-h-screen">
	<div class="container mx-auto py-8 px-4 max-w-7xl">
		<div class="bg-white rounded-lg shadow-lg overflow-hidden">
			<!-- Header -->
			<div class="repo-header">
				<div class="flex justify-between items-start">
					<div>
						<h1 class="text-3xl font-bold mb-2">📂 Source Code Repository</h1>
						<p class="text-xl opacity-90">%s %s</p>
						<p class="opacity-75">%s • %s</p>
					</div>
					<div class="flex gap-3">
						%s
						%s
					</div>
				</div>
			</div>

			<div class="p-6">
				<!-- Repository Status -->
				<div class="mb-8 p-4 bg-gray-50 rounded-lg">
					<h3 class="font-semibold mb-3 text-gray-900">📊 Repository Status</h3>
					<div class="flex flex-wrap gap-4">
						<span class="status-badge %s">Upload: %s</span>
						<span class="status-badge %s">Validation: %s</span>
						<span class="text-sm text-gray-600">📅 Uploaded: %s</span>
					</div>
				</div>

				<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
					<!-- Main Content -->
					<div class="lg:col-span-2">
						<!-- Student Information -->
						<div class="stat-card mb-6">
							<h2 class="text-xl font-semibold mb-4 text-gray-900">👨‍🎓 Student Information</h2>
							<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
								<div>
									<p class="text-sm text-gray-600">Name</p>
									<p class="font-medium">%s %s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Student ID</p>
									<p class="font-medium">%s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Email</p>
									<p class="font-medium">%s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Department</p>
									<p class="font-medium">%s</p>
								</div>
								<div class="md:col-span-2">
									<p class="text-sm text-gray-600">Thesis Title</p>
									<p class="font-medium">%s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Study Program</p>
									<p class="font-medium">%s</p>
								</div>
							</div>
						</div>

						<!-- Repository Files -->
						%s
					</div>

					<!-- Sidebar -->
					<div class="space-y-6">
						<!-- Statistics -->
						<div class="stat-card">
							<h2 class="text-lg font-semibold mb-4 text-gray-900">📈 Statistics</h2>
							<div class="space-y-3">
								<div class="flex justify-between">
									<span class="text-gray-600">Total Files:</span>
									<span class="font-medium">%d</span>
								</div>
								<div class="flex justify-between">
									<span class="text-gray-600">Total Size:</span>
									<span class="font-medium">%s</span>
								</div>
								<div class="flex justify-between">
									<span class="text-gray-600">Commits:</span>
									<span class="font-medium">%d</span>
								</div>
								<div class="flex justify-between">
									<span class="text-gray-600">Last Updated:</span>
									<span class="font-medium text-sm">%s</span>
								</div>
							</div>
						</div>

						<!-- Languages -->
						%s

						<!-- Recent Commits -->
						%s
					</div>
				</div>
			</div>
		</div>

		<!-- Back Button -->
		<div class="mt-6 text-center">
			<a href="/students-list" class="inline-flex items-center px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors">
				<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
				</svg>
				Back to Student List
			</a>
		</div>
	</div>
</body>
</html>`,
		student.StudentName, student.StudentLastname,
		student.StudentName, student.StudentLastname,
		student.StudyProgram, student.Department,
		h.generateGitHubButton(repoURL),
		h.generateDownloadButton(student.ID, repoURL, accessInfo),
		h.getStatusClass(uploadStatus), uploadStatus,
		h.getStatusClass(validationStatus), validationStatus,
		repoInfo.UploadedDate.Format("Jan 2, 2006 15:04"),
		student.StudentName, student.StudentLastname,
		student.StudentNumber,
		student.StudentEmail,
		student.Department,
		student.FinalProjectTitle,
		student.StudyProgram,
		h.generateRepositorySection(contents, student.ID, accessInfo),
		contents.Stats.TotalFiles,
		h.formatFileSize(contents.Stats.TotalSize),
		contents.Stats.CommitCount,
		h.formatDate(contents.Stats.LastUpdated),
		h.generateLanguagesSection(contents.Stats.Languages),
		h.generateCommitsSection(contents.Commits))
}

func (h *RepositoryHandler) generateNoRepositoryHTML(user *auth.AuthenticatedUser, student *database.StudentRecord) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>No Repository - %s %s</title>
	<link rel="stylesheet" href="/assets/css/output.css">
</head>
<body class="bg-gray-100 min-h-screen">
	<div class="container mx-auto py-8 px-4 max-w-4xl">
		<div class="bg-white rounded-lg shadow-lg p-8 text-center">
			<div class="mb-6">
				<svg class="w-20 h-20 mx-auto text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
				</svg>
				<h1 class="text-2xl font-bold text-gray-900 mb-2">No Repository Available</h1>
				<p class="text-gray-600">This student has not uploaded their source code yet.</p>
			</div>

			<div class="bg-gray-50 p-6 rounded-lg mb-6">
				<h2 class="text-lg font-semibold mb-4">Student Information</h2>
				<div class="text-left max-w-md mx-auto space-y-2">
					<p><strong>Name:</strong> %s %s</p>
					<p><strong>Student ID:</strong> %s</p>
					<p><strong>Email:</strong> %s</p>
					<p><strong>Thesis:</strong> %s</p>
					<p><strong>Department:</strong> %s</p>
				</div>
			</div>

			<div class="text-sm text-gray-500 mb-6">
				<p>The student will need to use the source code upload system to submit their repository.</p>
			</div>

			<a href="/students-list" class="inline-flex items-center px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
				<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
				</svg>
				Back to Student List
			</a>
		</div>
	</div>
</body>
</html>`,
		student.StudentName, student.StudentLastname,
		student.StudentName, student.StudentLastname,
		student.StudentNumber,
		student.StudentEmail,
		student.FinalProjectTitle,
		student.Department)
}
func (h *RepositoryHandler) generateDownloadButton(studentID int, repoURL string, accessInfo database.AccessInfo) string {
	if repoURL == "" {
		return ""
	}

	downloadURL := fmt.Sprintf("/repository/student/%d/download", studentID)
	if accessInfo.IsValid() {
		// Use the access type from AccessInfo
		downloadURL = fmt.Sprintf("/%s/%s/repository/student/%d/download", accessInfo.Type, accessInfo.Code, studentID)
		log.Printf("DEBUG: Generated download URL with access info - Type: %s, Code: %s, URL: %s",
			accessInfo.Type, accessInfo.Code, downloadURL)
	}

	return fmt.Sprintf(`
        <a href="%s" 
           class="inline-flex items-center px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            Download ZIP
        </a>`, downloadURL)
}

// Helper methods for HTML generation
func (h *RepositoryHandler) generateGitHubButton(repoURL string) string {
	if repoURL == "" {
		return ""
	}
	return fmt.Sprintf(`
		<a href="%s" target="_blank" 
		   class="inline-flex items-center px-4 py-2 bg-white bg-opacity-20 text-white rounded-lg hover:bg-opacity-30 transition-colors">
			<svg class="w-4 h-4 mr-2" fill="currentColor" viewBox="0 0 24 24">
				<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
			</svg>
			View on GitHub
		</a>`)
}

// Add these helper methods:

func (h *RepositoryHandler) generateFileDownloadButton(fileContent *FileContent) string {
	if fileContent.DownloadURL == "" {
		return ""
	}

	return fmt.Sprintf(`
		<a href="%s" download="%s"
		   class="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 transition-colors">
			<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
			</svg>
			Download
		</a>`, fileContent.DownloadURL, fileContent.Name)
}

func (h *RepositoryHandler) generateCopyButton() string {
	return `
		<button onclick="copyToClipboard()" id="copy-button"
		        class="bg-gray-500 text-white px-3 py-2 rounded-lg text-sm hover:bg-gray-600 transition-colors">
			<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
			</svg>
			Copy
		</button>`
}

// Fixed generateLineNumbers method
func (h *RepositoryHandler) generateLineNumbers(lineCount int) string {
	if lineCount == 0 {
		return "1"
	}

	lines := make([]string, lineCount)
	for i := 1; i <= lineCount; i++ {
		lines[i-1] = fmt.Sprintf("%d", i)
	}
	return strings.Join(lines, "\n")
}

// Enhanced generateBreadcrumb method
func (h *RepositoryHandler) generateBreadcrumb(filePath string, studentID int, accessInfo database.AccessInfo) string {
	if filePath == "" {
		return ""
	}

	parts := strings.Split(filePath, "/")
	breadcrumb := ""
	currentPath := ""

	for i, part := range parts {
		if i > 0 {
			currentPath += "/"
		}
		currentPath += part

		if i == len(parts)-1 {
			// Last part (current file) - not a link
			breadcrumb += " / <span class='text-gray-600'>" + part + "</span>"
		} else {
			// Directory - make it a link (you can implement directory viewing later)
			breadcrumb += fmt.Sprintf(` / <a href="#" class="text-blue-600 hover:underline">%s</a>`, part)
		}
	}

	return breadcrumb
}

// Enhanced getFileIcon method
func (h *RepositoryHandler) getFileIcon(filename, fileType string) string {
	if fileType == "dir" {
		return "📁"
	}

	// Handle specific filenames first
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "readme") {
		return "📖"
	}
	if strings.Contains(lower, "submission") {
		return "📋"
	}

	// Handle by extension
	ext := strings.ToLower(filepath.Ext(filename))
	icons := map[string]string{
		".md":         "📝",
		".markdown":   "📝",
		".txt":        "📄",
		".go":         "🐹",
		".py":         "🐍",
		".js":         "💛",
		".ts":         "💙",
		".java":       "☕",
		".cpp":        "⚙️",
		".c":          "⚙️",
		".cs":         "💜",
		".php":        "🐘",
		".rb":         "💎",
		".html":       "🌐",
		".css":        "🎨",
		".scss":       "🎨",
		".json":       "📋",
		".xml":        "📄",
		".yaml":       "📄",
		".yml":        "📄",
		".sql":        "🗄️",
		".dockerfile": "🐳",
		".gitignore":  "🙈",
		".env":        "⚙️",
		".config":     "⚙️",
		".toml":       "📄",
		".ini":        "📄",
		".png":        "🖼️",
		".jpg":        "🖼️",
		".jpeg":       "🖼️",
		".gif":        "🖼️",
		".svg":        "🎨",
		".pdf":        "📕",
		".zip":        "📦",
		".tar":        "📦",
		".gz":         "📦",
	}

	if icon, exists := icons[ext]; exists {
		return icon
	}

	return "📄"
}

// Enhanced getSyntaxHighlightingClass method
func (h *RepositoryHandler) getSyntaxHighlightingClass(language string) string {
	classes := map[string]string{
		"Go":         "language-go",
		"Python":     "language-python",
		"JavaScript": "language-javascript",
		"TypeScript": "language-typescript",
		"Java":       "language-java",
		"C++":        "language-cpp",
		"C":          "language-c",
		"C#":         "language-csharp",
		"PHP":        "language-php",
		"Ruby":       "language-ruby",
		"HTML":       "language-html",
		"CSS":        "language-css",
		"SCSS":       "language-scss",
		"JSON":       "language-json",
		"XML":        "language-xml",
		"YAML":       "language-yaml",
		"SQL":        "language-sql",
		"Shell":      "language-bash",
		"Markdown":   "language-markdown",
		"Rust":       "language-rust",
		"Swift":      "language-swift",
		"Kotlin":     "language-kotlin",
	}

	if class, exists := classes[language]; exists {
		return class
	}

	return "language-text"
}

// Enhanced getFileTypeDescription method
func (h *RepositoryHandler) getFileTypeDescription(fileContent *FileContent) string {
	if fileContent.IsBinary {
		return "Binary file"
	}
	if fileContent.Language != "" {
		return fileContent.Language + " source file"
	}

	ext := strings.ToLower(filepath.Ext(fileContent.Name))
	switch ext {
	case ".md", ".markdown":
		return "Markdown document"
	case ".txt":
		return "Text file"
	case ".json":
		return "JSON data"
	case ".yaml", ".yml":
		return "YAML configuration"
	case ".xml":
		return "XML document"
	case ".log":
		return "Log file"
	case ".env":
		return "Environment file"
	case ".gitignore":
		return "Git ignore file"
	case ".dockerfile":
		return "Docker file"
	default:
		return "Text file"
	}
}

// Enhanced generateRawViewButton method
func (h *RepositoryHandler) generateRawViewButton(fileContent *FileContent) string {
	if fileContent.DownloadURL == "" {
		return ""
	}

	return fmt.Sprintf(`
		<a href="%s" target="_blank" 
		   class="bg-gray-500 text-white px-3 py-2 rounded-lg text-sm hover:bg-gray-600 transition-colors">
			<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
			</svg>
			Raw
		</a>`, fileContent.DownloadURL)
}

func (h *RepositoryHandler) getStatusClass(status string) string {
	switch strings.ToLower(status) {
	case "valid", "completed":
		return "status-valid"
	case "pending", "processing":
		return "status-pending"
	case "unknown":
		return "status-unknown"
	default:
		return "status-pending"
	}
}

func (h *RepositoryHandler) generateRepositorySection(contents *types.RepositoryContents, studentID int, accessInfo database.AccessInfo) string {
	if contents.Error != "" {
		return fmt.Sprintf(`
            <div class="stat-card">
                <h2 class="text-lg font-semibold mb-3 text-gray-900">📁 Repository Files</h2>
                <div class="error-message">
                    <div class="flex items-center mb-2">
                        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <span class="font-medium">Unable to load repository contents</span>
                    </div>
                    <p class="text-sm">%s</p>
                    <p class="text-sm mt-2 opacity-75">The repository may be private or the access token may not have sufficient permissions.</p>
                </div>
            </div>`, contents.Error)
	}

	fileListHTML := h.generateFileListHTML(contents.Files, studentID, accessInfo)

	return fmt.Sprintf(`
        <div class="stat-card">
            <div class="flex items-center justify-between mb-4">
                <h2 class="text-lg font-semibold text-gray-900">📁 Repository Files</h2>
                <span class="text-sm text-gray-500">%d files</span>
            </div>
            %s
        </div>`, len(contents.Files), fileListHTML)
}

func (h *RepositoryHandler) generateLanguagesSection(languages map[string]int) string {
	if len(languages) == 0 {
		return ""
	}

	html := `<div class="stat-card"><h3 class="font-semibold mb-3 text-gray-900">💻 Languages</h3><div class="flex flex-wrap">`
	for lang, count := range languages {
		html += fmt.Sprintf(`<span class="language-badge">%s (%d)</span>`, lang, count)
	}
	html += "</div></div>"
	return html
}

func (h *RepositoryHandler) generateCommitsSection(commits []types.CommitInfo) string {
	if len(commits) == 0 {
		return `<div class="stat-card"><h3 class="font-semibold mb-3 text-gray-900">🔄 Recent Commits</h3><p class="text-gray-500 text-sm">No commits found</p></div>`
	}

	html := `<div class="stat-card"><h3 class="font-semibold mb-3 text-gray-900">🔄 Recent Commits</h3><div class="space-y-3">`
	for i, commit := range commits {
		if i >= 5 { // Limit to 5 commits
			break
		}
		html += fmt.Sprintf(`
			<div class="commit-item">
				<a href="%s" target="_blank" class="text-blue-600 hover:underline font-mono text-sm font-medium">%s</a>
				<p class="text-gray-700 text-sm mt-1 leading-relaxed">%s</p>
				<div class="flex items-center justify-between mt-2 text-xs text-gray-500">
					<span>%s</span>
					<span>%s</span>
				</div>
			</div>`,
			commit.URL, commit.SHA,
			h.truncateString(commit.Message, 80),
			commit.Author,
			commit.Date.Format("Jan 2, 15:04"))
	}
	html += "</div></div>"
	return html
}

func (h *RepositoryHandler) generateFileListHTML(files []types.RepositoryFile, studentID int, accessInfo database.AccessInfo) string {
	if len(files) == 0 {
		return `<div class="p-6 text-center text-gray-500">
            <svg class="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
            </svg>
            <p>No files found or unable to access repository</p>
        </div>`
	}

	html := `<div class="border border-gray-300 rounded-lg overflow-hidden bg-white">`

	for i, file := range files {
		icon := h.getFileIcon(file.Name, file.Type)

		sizeStr := ""
		if file.Type == "file" {
			sizeStr = fmt.Sprintf(`<span class="text-xs text-gray-500 ml-auto">%s</span>`, h.formatFileSize(file.Size))
		} else {
			sizeStr = `<span class="text-xs text-gray-500 ml-auto">Directory</span>`
		}

		// Add border between items
		borderClass := ""
		if i > 0 {
			borderClass = "border-t border-gray-200"
		}

		html += fmt.Sprintf(`
            <div class="file-item %s flex items-center p-3 hover:bg-gray-50 cursor-pointer transition-colors" 
                 data-file-path="%s" 
                 data-file-type="%s"
                 onclick="handleFileClick('%s', '%s')">
                <div class="flex items-center flex-1 min-w-0">
                    <span class="text-lg mr-3" title="%s">%s</span>
                    <div class="flex-1 min-w-0">
                        <span class="text-blue-600 hover:text-blue-800 font-medium truncate block">%s</span>
                    </div>
                    %s
                </div>
                <div class="flex items-center ml-4 space-x-2">
                    <a href="%s" target="_blank" 
                       class="text-gray-400 hover:text-blue-600 transition-colors" 
                       onclick="event.stopPropagation();"
                       title="View on GitHub">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                        </svg>
                    </a>
                </div>
            </div>`,
			borderClass,
			file.Path,
			file.Type,
			file.Path,
			file.Type,
			file.Type,
			icon,
			file.Name,
			sizeStr,
			file.URL)
	}

	html += `</div>`

	// Add improved JavaScript
	html += fmt.Sprintf(`
<script>
    function handleFileClick(filePath, fileType) {
        console.log('File clicked:', {filePath, fileType});
        
        const studentId = %d;
        const accessCode = '%s';  // This is still accessInfo.Code
        const accessType = '%s';  // This is accessInfo.Type
        
        // Build the base URL based on whether we have an access code
        let baseUrl;
        if (accessCode && accessType) {
            // Use the correct access type (reviewer or commission)
            baseUrl = '/' + accessType + '/' + accessCode + '/repository/student/' + studentId;
        } else {
            // Regular authenticated access
            baseUrl = '/repository/student/' + studentId;
        }
        
        console.log('Base URL:', baseUrl);
        
        if (fileType === 'dir') {
            // Navigate to directory
            const newUrl = baseUrl + '/browse/' + encodeURIComponent(filePath);
            console.log('Navigating to directory:', newUrl);
            window.location.href = newUrl;
        } else {
            // View file content in new tab
            const fileUrl = baseUrl + '/file/' + encodeURIComponent(filePath);
            console.log('Opening file in new tab:', fileUrl);
            window.open(fileUrl, '_blank');
        }
    }
    
    // Initialize when DOM is ready
    document.addEventListener('DOMContentLoaded', function() {
        console.log('Repository file list initialized');
        console.log('Found file items:', document.querySelectorAll('.file-item').length);
        console.log('Access code:', '%s');  // accessInfo.Code
        console.log('Access type:', '%s');  // accessInfo.Type
        console.log('Student ID:', %d);
        
        // Add keyboard navigation support
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && e.target.classList.contains('file-item')) {
                e.target.click();
            }
        });
    });
</script>`, studentID, accessInfo.Code, accessInfo.Type, accessInfo.Code, accessInfo.Type, studentID)

	return html
}
func (h *RepositoryHandler) generateFileLink(file types.RepositoryFile) string {
	if file.Type == "file" {
		return fmt.Sprintf(`
			<span class="text-blue-600 hover:underline flex-1 truncate cursor-pointer" 
				  title="Click to view file content">%s</span>`, file.Name)
	} else {
		return fmt.Sprintf(`
			<a href="%s" target="_blank" class="text-blue-600 hover:underline flex-1 truncate">%s</a>`,
			file.URL, file.Name)
	}
}

// Utility methods
func (h *RepositoryHandler) formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

func (h *RepositoryHandler) formatDate(date time.Time) string {
	if date.IsZero() {
		return "Unknown"
	}
	return date.Format("Jan 2, 2006")
}

func (h *RepositoryHandler) truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// ViewFileContent displays file content using templ
func (h *RepositoryHandler) ViewFileContent(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	filePath := chi.URLParam(r, "*")

	// Extract access code
	accessInfo := h.extractAccessInfo(r)

	// Skip permission check for commission members
	if user.Role != auth.RoleCommissionMember {
		if !h.canViewRepository(user, studentID) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil || repoInfo.RepositoryURL == nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	fileContent, err := h.getFileContent(*repoInfo.RepositoryURL, filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For HTMX requests, only return the file viewer component
	if r.Header.Get("HX-Request") == "true" {
		component := repository.FileViewer(studentID, filePath, fileContent, accessInfo)
		templ.Handler(component).ServeHTTP(w, r)
		return
	}

	// For full page loads, return the complete page
	student, _ := h.getStudentRecord(studentID)
	currentLocale := h.getCurrentLocale(r)
	component := repository.FileViewerPage(user, student, repoInfo, filePath, fileContent, currentLocale, accessInfo)
	templ.Handler(component).ServeHTTP(w, r)
}

// GetRepositoryTree returns the full repository file tree
func (h *RepositoryHandler) GetRepositoryTree(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	if !h.canViewRepository(user, studentID) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil || repoInfo.RepositoryURL == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repoName := h.extractRepoName(*repoInfo.RepositoryURL)
	tree, err := h.getRepositoryTreeRecursive(repoName, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

// ================================
// FILE CONTENT FETCHING
// ================================

type FileContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	Size        int64  `json:"size"`
	Type        string `json:"type"`
	Language    string `json:"language"`
	IsText      bool   `json:"is_text"`
	IsBinary    bool   `json:"is_binary"`
	Encoding    string `json:"encoding"`
	SHA         string `json:"sha"`
	DownloadURL string `json:"download_url"`
	Error       string `json:"error,omitempty"`
}

type FileTreeNode struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Type     string          `json:"type"` // file, dir
	Size     int64           `json:"size"`
	Children []*FileTreeNode `json:"children,omitempty"`
	Language string          `json:"language,omitempty"`
}

func (h *RepositoryHandler) getFileContent(repoURL, filePath string) (*types.FileContent, error) {
	repoName := h.extractRepoName(repoURL)
	if repoName == "" {
		return nil, fmt.Errorf("invalid repository URL")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		h.githubConfig.Organization, repoName, filePath)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("file not found")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var githubFile map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&githubFile); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	// Check if it's a file (not directory)
	fileType := h.getStringValue(githubFile, "type")
	if fileType != "file" {
		return nil, fmt.Errorf("path points to a directory, not a file")
	}

	fileContent := &types.FileContent{
		Name:        h.getStringValue(githubFile, "name"),
		Path:        h.getStringValue(githubFile, "path"),
		Type:        fileType,
		SHA:         h.getStringValue(githubFile, "sha"),
		DownloadURL: h.getStringValue(githubFile, "download_url"),
		Encoding:    h.getStringValue(githubFile, "encoding"),
	}

	// Get size
	if size, ok := githubFile["size"].(float64); ok {
		fileContent.Size = int64(size)
	}

	// Determine language
	fileContent.Language = h.getLanguageFromExtension(
		strings.ToLower(strings.TrimPrefix(filepath.Ext(fileContent.Name), ".")))

	// Get content if it's text-based and not too large
	if fileContent.Size < 1024*1024 { // Limit to 1MB
		content := h.getStringValue(githubFile, "content")
		encoding := h.getStringValue(githubFile, "encoding")

		if encoding == "base64" && content != "" {
			// Decode base64 content
			decoded, err := h.decodeBase64Content(content)
			if err == nil {
				fileContent.Content = decoded
				fileContent.IsText = h.isTextFile(fileContent.Name, decoded)
				fileContent.IsBinary = !fileContent.IsText
			}
		}
	} else {
		fileContent.IsBinary = true
		fileContent.Content = fmt.Sprintf("File too large to display (%s). Use download link instead.",
			h.formatFileSize(fileContent.Size))
	}

	return fileContent, nil
}

func (h *RepositoryHandler) getRepositoryTreeRecursive(repoName, path string) ([]*FileTreeNode, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		h.githubConfig.Organization, repoName, path)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var githubFiles []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&githubFiles); err != nil {
		return nil, err
	}

	var nodes []*FileTreeNode
	for _, file := range githubFiles {
		node := &FileTreeNode{
			Name: h.getStringValue(file, "name"),
			Path: h.getStringValue(file, "path"),
			Type: h.getStringValue(file, "type"),
		}

		if size, ok := file["size"].(float64); ok {
			node.Size = int64(size)
		}

		if node.Type == "file" {
			node.Language = h.getLanguageFromExtension(
				strings.ToLower(strings.TrimPrefix(filepath.Ext(node.Name), ".")))
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// ================================
// UTILITY METHODS
// ================================

func (h *RepositoryHandler) decodeBase64Content(content string) (string, error) {
	// Remove newlines from base64 content
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, "\r", "")

	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func (h *RepositoryHandler) isTextFile(filename, content string) bool {
	// Check by extension first
	ext := strings.ToLower(filepath.Ext(filename))
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".markdown": true, ".rst": true,
		".go": true, ".py": true, ".js": true, ".ts": true, ".java": true,
		".cpp": true, ".c": true, ".cs": true, ".php": true, ".rb": true,
		".html": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
		".json": true, ".xml": true, ".yaml": true, ".yml": true, ".toml": true,
		".sql": true, ".sh": true, ".bat": true, ".ps1": true, ".dockerfile": true,
		".gitignore": true, ".env": true, ".conf": true, ".config": true,
		".log": true, ".ini": true, ".cfg": true, ".properties": true,
	}

	if textExtensions[ext] {
		return true
	}

	// Check if content appears to be text (simple heuristic)
	if len(content) == 0 {
		return true
	}

	// Count non-printable characters
	nonPrintable := 0
	for i, r := range content {
		if i > 1000 { // Only check first 1000 characters
			break
		}
		if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, LF, CR
			nonPrintable++
		}
	}

	// If more than 10% non-printable, consider it binary
	threshold := len(content) / 10
	if threshold > 100 {
		threshold = 100
	}

	return nonPrintable < threshold
}

// ================================
// HTML RENDERING FOR FILE CONTENT
// ================================

func (h *RepositoryHandler) renderFileContentPage(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, student *database.StudentRecord, repoInfo *database.Document, filePath string, fileContent *FileContent, accessInfo database.AccessInfo) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := h.generateFileContentHTML(user, student, repoInfo, filePath, fileContent, accessInfo)
	w.Write([]byte(html))
}

func (h *RepositoryHandler) renderFileError(w http.ResponseWriter, r *http.Request, student *database.StudentRecord, filePath, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := h.generateFileErrorHTML(student, filePath, errorMsg)
	w.Write([]byte(html))
}

func (h *RepositoryHandler) generateFileContentHTML(user *auth.AuthenticatedUser, student *database.StudentRecord, repoInfo *database.Document, filePath string, fileContent *FileContent, accessInfo database.AccessInfo) string {
	// Determine syntax highlighting class
	syntaxClass := h.getSyntaxHighlightingClass(fileContent.Language)

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s - %s %s</title>
	<link rel="stylesheet" href="/assets/css/output.css">
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.24.1/themes/prism.min.css">
	<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.24.1/components/prism-core.min.js"></script>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.24.1/plugins/autoloader/prism-autoloader.min.js"></script>
	<style>
		.file-content {
			font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
			font-size: 14px;
			line-height: 1.5;
			white-space: pre-wrap;
			word-wrap: break-word;
			margin: 0;
		}
		.line-numbers {
			background-color: #f8f9fa;
			border-right: 1px solid #e9ecef;
			color: #6c757d;
			padding: 1rem 0.5rem;
			text-align: right;
			user-select: none;
			min-width: 4rem;
			font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
			font-size: 14px;
			line-height: 1.5;
		}
		.file-viewer {
			border: 1px solid #e9ecef;
			border-radius: 8px;
			overflow: hidden;
		}
		.file-header {
			background-color: #f8f9fa;
			padding: 1rem;
			border-bottom: 1px solid #e9ecef;
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.binary-file {
			text-align: center;
			padding: 2rem;
			color: #6c757d;
		}
		.breadcrumb {
			background-color: #e9ecef;
			padding: 0.5rem 1rem;
			font-size: 0.9rem;
		}
		.breadcrumb a {
			color: #007bff;
			text-decoration: none;
		}
		.breadcrumb a:hover {
			text-decoration: underline;
		}
		.code-container {
			display: flex;
			max-height: 80vh;
			overflow: auto;
		}
		.content-area {
			flex: 1;
			padding: 1rem;
			overflow: auto;
		}
	</style>
</head>
<body class="bg-gray-100 min-h-screen">
	<div class="container mx-auto py-8 px-4 max-w-7xl">
		<!-- Header -->
		<div class="bg-white rounded-lg shadow-lg mb-6 p-6">
			<div class="flex justify-between items-center">
				<div>
					<h1 class="text-2xl font-bold text-gray-900">📄 %s</h1>
					<p class="text-gray-600">%s %s • %s</p>
				</div>
				<div class="flex gap-3">
					<a href="/repository/student/%d" 
					   class="bg-gray-600 text-white px-4 py-2 rounded-lg hover:bg-gray-700 transition-colors">
						← Back to Repository
					</a>
					%s
				</div>
			</div>
		</div>

		<!-- Breadcrumb -->
		<div class="bg-white rounded-lg shadow mb-6">
			<div class="breadcrumb">
				<a href="/repository/student/%d">📁 Repository</a>
				%s
			</div>
		</div>

		<!-- File Content -->
		<div class="bg-white rounded-lg shadow-lg overflow-hidden">
			<div class="file-header">
				<div class="flex items-center gap-3">
					<span class="text-2xl">%s</span>
					<div>
						<h3 class="font-semibold text-lg">%s</h3>
						<p class="text-sm text-gray-600">
							%s • %s • %s
						</p>
					</div>
				</div>
				<div class="flex gap-2">
					%s
					%s
				</div>
			</div>

			%s
		</div>
	</div>
</body>
</html>`,
		fileContent.Name, student.StudentName, student.StudentLastname,
		fileContent.Name,
		student.StudentName, student.StudentLastname, student.StudyProgram,
		student.ID,
		h.generateFileDownloadButton(fileContent),
		student.ID,
		h.generateBreadcrumb(filePath, student.ID, accessInfo),
		h.getFileIcon(fileContent.Name, fileContent.Type),
		fileContent.Name,
		h.formatFileSize(fileContent.Size),
		fileContent.Language,
		h.getFileTypeDescription(fileContent),
		h.generateRawViewButton(fileContent),
		h.generateCopyButton(),
		h.generateFileContentSection(fileContent, syntaxClass))
}

// Fixed generateFileContentSection method
func (h *RepositoryHandler) generateFileContentSection(fileContent *FileContent, syntaxClass string) string {
	if fileContent.IsBinary {
		return fmt.Sprintf(`
			<div class="binary-file">
				<svg class="w-16 h-16 mx-auto mb-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
				</svg>
				<h3 class="text-lg font-semibold mb-2 text-gray-700">Binary File</h3>
				<p class="mb-4 text-gray-600">This file contains binary data and cannot be displayed.</p>
				<p class="text-sm text-gray-500">%s</p>
				<div class="mt-4">
					<a href="%s" target="_blank" 
					   class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
						Download File
					</a>
				</div>
			</div>`, fileContent.Content, fileContent.DownloadURL)
	}

	if fileContent.Content == "" {
		return `
			<div class="binary-file">
				<svg class="w-12 h-12 mx-auto mb-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
				</svg>
				<p class="text-gray-500">This file is empty</p>
			</div>`
	}

	// Count lines for line numbers
	lines := strings.Split(fileContent.Content, "\n")
	lineCount := len(lines)

	// For text files, show with syntax highlighting
	return fmt.Sprintf(`
		<div class="code-container">
			<div class="line-numbers">%s</div>
			<div class="content-area">
				<pre class="file-content"><code class="%s" id="file-content">%s</code></pre>
			</div>
		</div>
		<script>
			// Initialize syntax highlighting
			if (typeof Prism !== 'undefined') {
				Prism.highlightAll();
			}
			
			// Add copy functionality
			function copyToClipboard() {
				const content = document.getElementById('file-content').textContent;
				navigator.clipboard.writeText(content).then(function() {
					const button = document.getElementById('copy-button');
					const originalText = button.innerHTML;
					button.innerHTML = '✅ Copied!';
					button.classList.remove('bg-gray-500', 'hover:bg-gray-600');
					button.classList.add('bg-green-500');
					
					setTimeout(function() {
						button.innerHTML = originalText;
						button.classList.remove('bg-green-500');
						button.classList.add('bg-gray-500', 'hover:bg-gray-600');
					}, 2000);
				}).catch(function(err) {
					console.error('Could not copy text: ', err);
					alert('Failed to copy to clipboard');
				});
			}
		</script>`,
		h.generateLineNumbers(lineCount),
		syntaxClass,
		h.escapeHTML(fileContent.Content))
}

// Helper methods for file content display

func (h *RepositoryHandler) generateFileErrorHTML(student *database.StudentRecord, filePath, errorMsg string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>File Error - %s %s</title>
	<link rel="stylesheet" href="/assets/css/output.css">
</head>
<body class="bg-gray-100 min-h-screen">
	<div class="container mx-auto py-8 px-4 max-w-4xl">
		<div class="bg-white rounded-lg shadow-lg p-8 text-center">
			<svg class="w-20 h-20 mx-auto text-red-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
			</svg>
			<h1 class="text-2xl font-bold text-gray-900 mb-2">File Not Found</h1>
			<p class="text-gray-600 mb-4">Could not load file: <code class="bg-gray-100 px-2 py-1 rounded">%s</code></p>
			<p class="text-red-600 mb-6">%s</p>
			
			<div class="space-x-4">
				<a href="/repository/student/%d" 
				   class="bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700">
					← Back to Repository
				</a>
				<a href="/students-list" 
				   class="bg-gray-600 text-white px-6 py-3 rounded-lg hover:bg-gray-700">
					← Back to Student List
				</a>
			</div>
		</div>
	</div>
</body>
</html>`,
		student.StudentName, student.StudentLastname,
		filePath, errorMsg, student.ID)
}

func (h *RepositoryHandler) escapeHTML(content string) string {
	content = strings.ReplaceAll(content, "&", "&amp;")
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	content = strings.ReplaceAll(content, "\"", "&quot;")
	content = strings.ReplaceAll(content, "'", "&#39;")
	return content
}

func (h *RepositoryHandler) ViewStudentRepositoryPath(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	dirPath := chi.URLParam(r, "*")

	// Extract access code
	accessInfo := h.extractAccessInfo(r)
	log.Printf("DEBUG: Extracted access code from URL %s: %s", r.URL.Path, accessInfo)

	// Skip permission check for commission members
	if user.Role != auth.RoleCommissionMember {
		if !h.canViewRepository(user, studentID) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
	}

	student, err := h.getStudentRecord(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	currentLocale := h.getCurrentLocale(r)

	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		component := repository.NoRepositoryPage(user, student, currentLocale, accessInfo)
		templ.Handler(component).ServeHTTP(w, r)
		return
	}

	repoContents, err := h.getRepositoryContentsForPath(repoInfo, dirPath)
	if err != nil {
		repoContents = &types.RepositoryContents{
			Files:   []types.RepositoryFile{},
			Commits: []types.CommitInfo{},
			Stats:   types.RepositoryStats{},
			Error:   err.Error(),
		}
	}

	// Pass accessCode to the component
	component := repository.DirectoryPage(user, student, repoInfo, repoContents, dirPath, currentLocale, accessInfo)
	templ.Handler(component).ServeHTTP(w, r)
}

// GetRepositoryPathAPI returns repository data for a specific path as JSON

// GetFileContentAPI returns file content as JSON for API requests
func (h *RepositoryHandler) GetFileContentAPI(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Unauthorized",
		})
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid student ID",
		})
		return
	}

	// Get file path from query parameter or URL param
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		filePath = chi.URLParam(r, "*")
	}

	if filePath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "File path required",
		})
		return
	}

	// Skip permission check for commission members
	if user.Role != auth.RoleCommissionMember {
		if !h.canViewRepository(user, studentID) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Access denied",
			})
			return
		}
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil || repoInfo.RepositoryURL == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Repository not found",
		})
		return
	}

	fileContent, err := h.getFileContent(*repoInfo.RepositoryURL, filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileContent)
}

// FIX 3: Update GetRepositoryPathAPI method
func (h *RepositoryHandler) GetRepositoryPathAPI(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Get path from query parameter OR wildcard
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath = chi.URLParam(r, "*")
	}

	if !h.canViewRepository(user, studentID) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	repoInfo, err := h.getStudentRepository(studentID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "No repository available",
		})
		return
	}

	repoContents, err := h.getRepositoryContentsForPath(repoInfo, dirPath)
	if err != nil {
		repoContents = &types.RepositoryContents{Error: err.Error()}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"repository": repoInfo,
		"contents":   repoContents,
		"path":       dirPath,
	})
}

func (h *RepositoryHandler) getRepositoryContentsForPath(repoInfo *database.Document, dirPath string) (*types.RepositoryContents, error) {
	if repoInfo.RepositoryURL == nil || *repoInfo.RepositoryURL == "" {
		return nil, fmt.Errorf("no repository URL available")
	}

	repoName := h.extractRepoName(*repoInfo.RepositoryURL)
	if repoName == "" {
		return nil, fmt.Errorf("could not extract repository name from URL: %s", *repoInfo.RepositoryURL)
	}

	log.Printf("DEBUG: Fetching repository contents for repo '%s', path '%s'", repoName, dirPath)

	// Get repository files for specific path
	files, err := h.getRepositoryFilesForPath(repoName, dirPath)
	if err != nil {
		log.Printf("ERROR: Failed to get repository files: %v", err)
		return nil, fmt.Errorf("failed to get repository files: %w", err)
	}

	log.Printf("DEBUG: Successfully fetched %d files from GitHub", len(files))

	// Get recent commits (same as before)
	commits, err := h.getRepositoryCommits(repoName)
	if err != nil {
		log.Printf("WARNING: Failed to get commits: %v", err)
		commits = []types.CommitInfo{} // Don't fail on commits error
	}

	// Calculate stats
	stats := h.calculateRepositoryStats(files, commits)

	return &types.RepositoryContents{
		Files:   files,
		Commits: commits,
		Stats:   stats,
	}, nil
}

func (h *RepositoryHandler) getRepositoryFilesForPath(repoName, dirPath string) ([]types.RepositoryFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s",
		h.githubConfig.Organization, repoName, dirPath)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("directory not found or access denied")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var githubFiles []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&githubFiles); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	var files []types.RepositoryFile
	for _, file := range githubFiles {
		fileType := "file"
		if t, ok := file["type"].(string); ok {
			fileType = t
		}

		size := int64(0)
		if s, ok := file["size"].(float64); ok {
			size = int64(s)
		}

		files = append(files, types.RepositoryFile{
			Name: h.getStringValue(file, "name"),
			Path: h.getStringValue(file, "path"),
			Type: fileType,
			Size: size,
			URL:  h.getStringValue(file, "html_url"),
		})
	}

	return files, nil
}
func (h *RepositoryHandler) renderRepositoryPageWithPath(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, student *database.StudentRecord, repoInfo *database.Document, contents *types.RepositoryContents, dirPath string, accessInfo database.AccessInfo) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := h.generateRepositoryHTMLWithPath(user, student, repoInfo, contents, dirPath, accessInfo)
	w.Write([]byte(html))
}
func (h *RepositoryHandler) generateRepositoryHTMLWithPath(user *auth.AuthenticatedUser, student *database.StudentRecord, repoInfo *database.Document, contents *types.RepositoryContents, dirPath string, accessInfo database.AccessInfo) string {
	// Safe repository URL handling
	repoURL := ""
	if repoInfo.RepositoryURL != nil {
		repoURL = *repoInfo.RepositoryURL
	}

	// Safe handling of optional fields
	uploadStatus := "unknown"
	validationStatus := "unknown"

	if repoInfo.UploadStatus != "" {
		uploadStatus = repoInfo.UploadStatus
	}
	if repoInfo.ValidationStatus != "" {
		validationStatus = repoInfo.ValidationStatus
	}

	// Generate breadcrumb for current path
	breadcrumbHTML := h.generatePathBreadcrumb(dirPath, student.ID, accessInfo)

	// Generate back button if not in root
	backButtonHTML := ""
	if dirPath != "" {
		parentPath := filepath.Dir(dirPath)
		if parentPath == "." {
			parentPath = ""
		}

		backURL := fmt.Sprintf("/repository/student/%d", student.ID)
		if parentPath != "" {
			backURL = fmt.Sprintf("/repository/student/%d/browse/%s", student.ID, parentPath)
		}

		backButtonHTML = fmt.Sprintf(`
			<a href="%s" 
			   class="inline-flex items-center px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors mr-3">
				<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
				</svg>
				Back
			</a>`, backURL)
	}

	// Current directory display
	currentDirDisplay := "Repository Root"
	if dirPath != "" {
		currentDirDisplay = "📁 " + dirPath
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Repository: %s %s</title>
	<link rel="stylesheet" href="/assets/css/output.css">
	<style>
		.file-tree { font-family: monospace; }
		.file-item { padding: 8px 12px; border-bottom: 1px solid #e5e7eb; }
		.file-item:hover { background-color: #f9fafb; }
		.file-item:last-child { border-bottom: none; }
		.language-badge { 
			padding: 4px 8px; 
			border-radius: 12px; 
			font-size: 12px; 
			margin: 2px; 
			display: inline-block; 
			background-color: #dbeafe; 
			color: #1e40af;
		}
		.commit-item { 
			border: 1px solid #d1d5db; 
			border-radius: 6px; 
			padding: 12px; 
			margin-bottom: 8px; 
			background-color: #ffffff;
		}
		.status-badge { 
			padding: 4px 8px; 
			border-radius: 12px; 
			font-size: 11px; 
			font-weight: 600; 
		}
		.status-valid { background-color: #d1fae5; color: #065f46; }
		.status-completed { background-color: #dbeafe; color: #1e40af; }
		.status-pending { background-color: #fef3c7; color: #92400e; }
		.status-unknown { background-color: #f3f4f6; color: #374151; }
		.repo-header { 
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
			color: white; 
			padding: 2rem; 
			border-radius: 12px 12px 0 0; 
		}
		.stat-card { 
			background: white; 
			border-radius: 8px; 
			padding: 1rem; 
			box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1); 
		}
		.error-message {
			background-color: #fef2f2;
			color: #dc2626;
			padding: 1rem;
			border-radius: 8px;
			border: 1px solid #fecaca;
		}
		.breadcrumb {
			background-color: #f8f9fa;
			padding: 1rem;
			border-radius: 8px;
			margin-bottom: 1rem;
			font-family: monospace;
		}
		.breadcrumb a {
			color: #007bff;
			text-decoration: none;
		}
		.breadcrumb a:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body class="bg-gray-100 min-h-screen">
	<div class="container mx-auto py-8 px-4 max-w-7xl">
		<div class="bg-white rounded-lg shadow-lg overflow-hidden">
			<!-- Header -->
			<div class="repo-header">
				<div class="flex justify-between items-start">
					<div>
						<h1 class="text-3xl font-bold mb-2">📂 Source Code Repository</h1>
						<p class="text-xl opacity-90">%s %s</p>
						<p class="opacity-75">%s • %s</p>
						<p class="opacity-75 mt-2">%s</p>
					</div>
					<div class="flex gap-3">
						%s
						%s
						%s
					</div>
				</div>
			</div>

			<div class="p-6">
				<!-- Breadcrumb Navigation -->
				<div class="breadcrumb">
					%s
				</div>

				<!-- Repository Status -->
				<div class="mb-8 p-4 bg-gray-50 rounded-lg">
					<h3 class="font-semibold mb-3 text-gray-900">📊 Repository Status</h3>
					<div class="flex flex-wrap gap-4">
						<span class="status-badge %s">Upload: %s</span>
						<span class="status-badge %s">Validation: %s</span>
						<span class="text-sm text-gray-600">📅 Uploaded: %s</span>
					</div>
				</div>

				<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
					<!-- Main Content -->
					<div class="lg:col-span-2">
						<!-- Student Information -->
						<div class="stat-card mb-6">
							<h2 class="text-xl font-semibold mb-4 text-gray-900">👨‍🎓 Student Information</h2>
							<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
								<div>
									<p class="text-sm text-gray-600">Name</p>
									<p class="font-medium">%s %s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Student ID</p>
									<p class="font-medium">%s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Email</p>
									<p class="font-medium">%s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Department</p>
									<p class="font-medium">%s</p>
								</div>
								<div class="md:col-span-2">
									<p class="text-sm text-gray-600">Thesis Title</p>
									<p class="font-medium">%s</p>
								</div>
								<div>
									<p class="text-sm text-gray-600">Study Program</p>
									<p class="font-medium">%s</p>
								</div>
							</div>
						</div>

						<!-- Repository Files -->
						%s
					</div>

					<!-- Sidebar -->
					<div class="space-y-6">
						<!-- Statistics -->
						<div class="stat-card">
							<h2 class="text-lg font-semibold mb-4 text-gray-900">📈 Statistics</h2>
							<div class="space-y-3">
								<div class="flex justify-between">
									<span class="text-gray-600">Total Files:</span>
									<span class="font-medium">%d</span>
								</div>
								<div class="flex justify-between">
									<span class="text-gray-600">Total Size:</span>
									<span class="font-medium">%s</span>
								</div>
								<div class="flex justify-between">
									<span class="text-gray-600">Commits:</span>
									<span class="font-medium">%d</span>
								</div>
								<div class="flex justify-between">
									<span class="text-gray-600">Last Updated:</span>
									<span class="font-medium text-sm">%s</span>
								</div>
							</div>
						</div>

						<!-- Languages -->
						%s

						<!-- Recent Commits -->
						%s
					</div>
				</div>
			</div>
		</div>

		<!-- Back Button -->
		<div class="mt-6 text-center">
			<a href="/students-list" class="inline-flex items-center px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors">
				<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
				</svg>
				Back to Student List
			</a>
		</div>
	</div>
</body>
</html>`,
		student.StudentName, student.StudentLastname,
		student.StudentName, student.StudentLastname,
		student.StudyProgram, student.Department,
		currentDirDisplay,
		backButtonHTML,
		h.generateGitHubButton(repoURL),
		h.generateDownloadButton(student.ID, repoURL, accessInfo),
		breadcrumbHTML,
		h.getStatusClass(uploadStatus), uploadStatus,
		h.getStatusClass(validationStatus), validationStatus,
		repoInfo.UploadedDate.Format("Jan 2, 2006 15:04"),
		student.StudentName, student.StudentLastname,
		student.StudentNumber,
		student.StudentEmail,
		student.Department,
		student.FinalProjectTitle,
		student.StudyProgram,
		h.generateRepositorySection(contents, student.ID, accessInfo),
		contents.Stats.TotalFiles,
		h.formatFileSize(contents.Stats.TotalSize),
		contents.Stats.CommitCount,
		h.formatDate(contents.Stats.LastUpdated),
		h.generateLanguagesSection(contents.Stats.Languages),
		h.generateCommitsSection(contents.Commits))
}
func (h *RepositoryHandler) generatePathBreadcrumb(dirPath string, studentID int, accessInfo database.AccessInfo) string {
	if dirPath == "" {
		return `<span class="text-gray-900 font-medium">📁 Repository Root</span>`
	}

	// Determine base URL based on access code
	baseURL := fmt.Sprintf("/repository/student/%d", studentID)
	if accessInfo.IsValid() {
		// Use accessInfo.Type instead of hardcoded "reviewer"
		baseURL = fmt.Sprintf("/%s/%s/repository/student/%d", accessInfo.Type, accessInfo.Code, studentID)
	}

	// Split path into parts
	parts := strings.Split(dirPath, "/")
	breadcrumb := fmt.Sprintf(`<a href="%s" class="text-blue-600 hover:underline">📁 Repository</a>`, baseURL)

	currentPath := ""
	for i, part := range parts {
		if part == "" {
			continue
		}

		if i > 0 {
			currentPath += "/"
		}
		currentPath += part

		if i == len(parts)-1 {
			// Last part (current directory) - not a link
			breadcrumb += fmt.Sprintf(` <span class="text-gray-400 mx-2">/</span> <span class="text-gray-900 font-medium">📁 %s</span>`, part)
		} else {
			// Directory - make it a link
			breadcrumb += fmt.Sprintf(` <span class="text-gray-400 mx-2">/</span> <a href="%s/browse/%s" class="text-blue-600 hover:underline">📁 %s</a>`, baseURL, currentPath, part)
		}
	}

	return breadcrumb
}

// Helper method to get current locale
func (h *RepositoryHandler) getCurrentLocale(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		// Check cookie or session
		cookie, err := r.Cookie("locale")
		if err == nil {
			locale = cookie.Value
		}
	}
	if locale != "en" && locale != "lt" {
		locale = "lt" // Default to Lithuanian
	}
	return locale
}

// Add this helper method to your RepositoryHandler
func (h *RepositoryHandler) extractAccessInfo(r *http.Request) database.AccessInfo {
	pathParts := strings.Split(r.URL.Path, "/")
	for i, part := range pathParts {
		if (part == "commission" || part == "reviewer") && i+1 < len(pathParts) {
			return database.AccessInfo{
				Code: pathParts[i+1],
				Type: part,
			}
		}
	}
	return database.AccessInfo{}
}
