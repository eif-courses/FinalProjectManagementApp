// handlers/source_code.go
package handlers

import (
	"FinalProjectManagementApp/database"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Add these structs at the top after your existing imports
type UploadRequest struct {
	ID              string
	ResponseChan    chan *database.SubmissionResult
	StudentInfo     *database.StudentInfo
	File            multipart.File
	Header          *multipart.FileHeader
	StudentRecordID int
}

// Update your existing SourceCodeHandler struct
type SourceCodeHandler struct {
	db           *sqlx.DB
	githubConfig *database.GitHubConfig
	client       *http.Client

	// ADD these new fields:
	uploadQueue   chan *UploadRequest
	activeUploads sync.Map
	maxConcurrent int
}

// UPDATE your existing NewSourceCodeHandler function
func NewSourceCodeHandler(db *sqlx.DB, githubConfig *database.GitHubConfig) *SourceCodeHandler {
	handler := &SourceCodeHandler{
		db:           db,
		githubConfig: githubConfig,
		client:       &http.Client{Timeout: 60 * time.Second},

		// ADD these:
		uploadQueue:   make(chan *UploadRequest, 50),
		maxConcurrent: 5,
	}

	// ADD: Start upload workers
	for i := 0; i < handler.maxConcurrent; i++ {
		go handler.uploadWorker(i)
	}

	return handler
}

// ADD this new method
func (h *SourceCodeHandler) uploadWorker(workerID int) {
	log.Printf("Upload worker %d started", workerID)

	for req := range h.uploadQueue {
		log.Printf("Worker %d processing upload for student %s", workerID, req.StudentInfo.StudentID)

		// Simple delay to avoid overwhelming GitHub
		time.Sleep(10 * time.Second)

		result := h.processSourceCodeUpload(req.StudentRecordID, req.ID, req.File, req.Header, req.StudentInfo)

		req.ResponseChan <- result
		close(req.ResponseChan)
		h.activeUploads.Delete(req.ID)

		log.Printf("Worker %d completed upload for student %s", workerID, req.StudentInfo.StudentID)
	}
}

// REPLACE your existing UploadSourceCode function with this:
func (h *SourceCodeHandler) UploadSourceCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check system load
	queueLen := len(h.uploadQueue)
	if queueLen >= 45 { // Near capacity
		h.renderJSONError(w, fmt.Sprintf("System busy, please try again in 10 minutes. Queue length: %d", queueLen))
		return
	}

	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		h.renderError(w, "Failed to parse form data", err)
		return
	}

	studentInfo := &database.StudentInfo{
		Name:        strings.TrimSpace(r.FormValue("name")),
		StudentID:   strings.TrimSpace(r.FormValue("student_id")),
		Email:       strings.TrimSpace(r.FormValue("email")),
		ThesisTitle: strings.TrimSpace(r.FormValue("thesis_title")),
	}

	if errors := h.validateStudentInfo(studentInfo); len(errors) > 0 {
		h.renderJSONError(w, strings.Join(errors, "; "))
		return
	}

	// Check for duplicate upload
	uploadKey := fmt.Sprintf("%s-%s", studentInfo.StudentID, studentInfo.Email)
	if _, exists := h.activeUploads.LoadOrStore(uploadKey, true); exists {
		h.renderJSONError(w, "Upload already in progress for this student. Please wait.")
		return
	}
	defer h.activeUploads.Delete(uploadKey) // Clean up on any early return

	file, header, err := r.FormFile("source_code")
	if err != nil {
		h.renderError(w, "No file uploaded", err)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		h.renderJSONError(w, "Only ZIP files are accepted")
		return
	}

	studentRecordID, err := h.findStudentRecordID(studentInfo)
	if err != nil {
		h.renderError(w, "Student not found", err)
		return
	}

	uploadID := uuid.New().String()

	// For small queues, process immediately
	if queueLen < 3 {
		result := h.processSourceCodeUpload(studentRecordID, uploadID, file, header, studentInfo)
		h.renderJSONResult(w, result)
		return
	}

	// For larger queues, queue the request
	responseChan := make(chan *database.SubmissionResult, 1)

	uploadReq := &UploadRequest{
		ID:              uploadID,
		ResponseChan:    responseChan,
		StudentInfo:     studentInfo,
		File:            file,
		Header:          header,
		StudentRecordID: studentRecordID,
	}

	select {
	case h.uploadQueue <- uploadReq:
		queuePos := len(h.uploadQueue)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":        true,
			"message":        fmt.Sprintf("Upload queued successfully. Position: %d", queuePos),
			"submission_id":  uploadID,
			"queue_position": queuePos,
			"estimated_wait": fmt.Sprintf("%d minutes", queuePos*2),
			"status":         "queued",
		})

		// Process in background
		go func() {
			select {
			case result := <-responseChan:
				log.Printf("Background upload completed for %s: success=%v", studentInfo.StudentID, result.Success)
			case <-time.After(30 * time.Minute):
				log.Printf("Upload timeout for student %s", studentInfo.StudentID)
			}
		}()

	case <-time.After(5 * time.Second):
		h.renderJSONError(w, "System overloaded. Please try again later.")
	}
}

// ===== MAIN PROCESSING LOGIC =====
func (h *SourceCodeHandler) processSourceCodeUpload(studentRecordID int, submissionID string, file multipart.File, header *multipart.FileHeader, studentInfo *database.StudentInfo) *database.SubmissionResult {
	// Create temp directories
	// ADD: Create unique directories to avoid conflicts
	timestamp := time.Now().UnixNano()
	uniqueID := fmt.Sprintf("%s_%d", submissionID[:8], timestamp) // Shorter unique ID

	uploadDir := filepath.Join("uploads", "upload_"+uniqueID)
	extractDir := filepath.Join("uploads", "extract_"+uniqueID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return &database.SubmissionResult{Success: false, Error: "Failed to create upload directory"}
	}
	defer os.RemoveAll(uploadDir)
	defer os.RemoveAll(extractDir)

	// Save ZIP file
	zipPath := filepath.Join(uploadDir, header.Filename)
	if err := h.saveFile(file, zipPath); err != nil {
		return &database.SubmissionResult{Success: false, Error: "Failed to save file"}
	}

	// Extract and validate with filtering
	filterInfo, err := h.extractZip(zipPath, extractDir)
	if err != nil {
		return &database.SubmissionResult{Success: false, Error: "Failed to extract ZIP: " + err.Error()}
	}

	validation := h.validateSubmission(extractDir)
	if !validation.Valid {
		return &database.SubmissionResult{
			Success:    false,
			Error:      "Validation failed",
			Validation: validation,
		}
	}

	// Create or update GitHub repository (one per student)
	repoInfo, err := h.createOrUpdateRepository(studentInfo)
	if err != nil {
		return &database.SubmissionResult{Success: false, Error: "Failed to create/update repository: " + err.Error()}
	}

	// Upload code using Git
	commitInfo, err := h.uploadToGit(repoInfo, extractDir, studentInfo)
	if err != nil {
		return &database.SubmissionResult{Success: false, Error: "Failed to upload code: " + err.Error()}
	}

	// Save to database
	documentID, err := h.saveToDatabase(studentRecordID, submissionID, header.Filename, header.Size, repoInfo, commitInfo)
	if err != nil {
		log.Printf("Warning: Failed to save to database: %v", err)
	}

	return &database.SubmissionResult{
		Success:        true,
		Message:        "Thesis source code uploaded successfully",
		SubmissionID:   submissionID,
		RepositoryInfo: repoInfo,
		Validation:     validation,
		CommitInfo:     commitInfo,
		DocumentID:     documentID,
		FilterInfo:     filterInfo,
	}
}

// ===== REPOSITORY MANAGEMENT =====
func (h *SourceCodeHandler) generateRepoName(studentInfo *database.StudentInfo) string {
	// ONE repository per student - consistent naming
	name := fmt.Sprintf("thesis-%s-%s",
		studentInfo.StudentID,
		strings.ToLower(strings.ReplaceAll(studentInfo.Name, " ", "-")))

	reg := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	name = reg.ReplaceAllString(name, "")

	if len(name) > 90 {
		name = name[:90]
	}

	return name
}

func (h *SourceCodeHandler) createOrUpdateRepository(studentInfo *database.StudentInfo) (*database.RepositoryInfo, error) {
	repoName := h.generateRepoName(studentInfo)

	// First, try to get existing repository
	existingRepo, err := h.getExistingRepository(repoName)
	if err == nil {
		log.Printf("Repository already exists: %s - will update with new submission", existingRepo.WebURL)
		return existingRepo, nil
	}

	// Repository doesn't exist, create new one
	log.Printf("Creating new repository: %s", repoName)
	return h.createNewRepository(repoName, studentInfo)
}

func (h *SourceCodeHandler) getExistingRepository(repoName string) (*database.RepositoryInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", h.githubConfig.Organization, repoName)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("repository does not exist")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get repository info: %s", string(body))
	}

	var repoResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&repoResponse); err != nil {
		return nil, err
	}

	return &database.RepositoryInfo{
		ID:        fmt.Sprintf("%.0f", repoResponse["id"].(float64)),
		Name:      repoResponse["name"].(string),
		WebURL:    repoResponse["html_url"].(string),
		RemoteURL: repoResponse["clone_url"].(string),
		CloneURL:  repoResponse["clone_url"].(string),
	}, nil
}

func (h *SourceCodeHandler) createNewRepository(repoName string, studentInfo *database.StudentInfo) (*database.RepositoryInfo, error) {
	payload := map[string]interface{}{
		"name":    repoName,
		"private": true,
		"description": fmt.Sprintf("Final thesis: %s by %s (%s)",
			studentInfo.ThesisTitle, studentInfo.Name, studentInfo.StudentID),
		"auto_init":      true,
		"default_branch": "main",
	}

	jsonData, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos", h.githubConfig.Organization)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+h.githubConfig.PAT)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Thesis-Management-System/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create GitHub repository (status %d): %s", resp.StatusCode, string(body))
	}

	var repoResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&repoResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	repoInfo := &database.RepositoryInfo{
		ID:        fmt.Sprintf("%.0f", repoResponse["id"].(float64)),
		Name:      repoResponse["name"].(string),
		WebURL:    repoResponse["html_url"].(string),
		RemoteURL: repoResponse["clone_url"].(string),
		CloneURL:  repoResponse["clone_url"].(string),
	}

	log.Printf("Created GitHub repository: %s", repoInfo.WebURL)
	return repoInfo, nil
}

// ===== GIT OPERATIONS =====
func (h *SourceCodeHandler) uploadToGit(repoInfo *database.RepositoryInfo, sourcePath string, studentInfo *database.StudentInfo) (*database.CommitInfo, error) {
	tempDir := filepath.Join("uploads", "git_"+uuid.New().String())
	defer os.RemoveAll(tempDir)

	// Clone existing repository
	cloneURL := strings.Replace(repoInfo.CloneURL, "https://", fmt.Sprintf("https://%s@", h.githubConfig.PAT), 1)

	repo, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:  cloneURL,
		Auth: &githttp.BasicAuth{Username: h.githubConfig.PAT, Password: ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create version-specific branch for this submission
	submissionTime := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("submission-%s", submissionTime)

	// Create new branch
	branchRef := plumbing.NewBranchReferenceName(branchName)
	headRef, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	err = repo.Storer.SetReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch reference: %w", err)
	}

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to checkout branch: %w", err)
	}

	// Clear existing content (except .git and README.md)
	h.clearRepositoryContent(tempDir)

	// Copy new source files
	err = h.copyFiles(sourcePath, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to copy files: %w", err)
	}

	// Create updated README with version info
	readmeContent := h.generateVersionedReadme(studentInfo, submissionTime)
	os.WriteFile(filepath.Join(tempDir, "README.md"), []byte(readmeContent), 0644)

	// Add submission info
	submissionInfo := h.generateVersionedSubmissionInfo(studentInfo, submissionTime)
	os.WriteFile(filepath.Join(tempDir, "SUBMISSION_INFO.md"), []byte(submissionInfo), 0644)

	// Commit changes
	err = worktree.AddGlob(".")
	if err != nil {
		return nil, fmt.Errorf("failed to add files: %w", err)
	}

	fileCount := h.countFiles(tempDir)
	commitMessage := fmt.Sprintf("Thesis submission %s - %s (%s)",
		submissionTime, studentInfo.Name, studentInfo.StudentID)

	commit, err := worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  studentInfo.Name,
			Email: studentInfo.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Push the new branch
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       &githttp.BasicAuth{Username: h.githubConfig.PAT, Password: ""},
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName))},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to push branch: %w", err)
	}

	// Checkout main and merge the submission branch
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("main"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to checkout main: %w", err)
	}

	// Merge submission branch to main
	err = h.mergeToMain(repo, worktree, branchName, commitMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to merge to main: %w", err)
	}

	// Push main branch
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       &githttp.BasicAuth{Username: h.githubConfig.PAT, Password: ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to push main: %w", err)
	}

	log.Printf("Successfully updated repository: %s", repoInfo.WebURL)

	return &database.CommitInfo{
		Message:    commitMessage,
		Timestamp:  time.Now(),
		FilesCount: fileCount,
		CommitID:   commit.String(),
	}, nil
}

func (h *SourceCodeHandler) mergeToMain(repo *git.Repository, worktree *git.Worktree, branchName, commitMessage string) error {
	// Get the commit from the submission branch
	branchRef, err := repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err != nil {
		return err
	}

	// Create merge commit
	_, err = worktree.Commit(fmt.Sprintf("Merge %s into main\n\n%s", branchName, commitMessage), &git.CommitOptions{
		Parents: []plumbing.Hash{branchRef.Hash()},
		Author: &object.Signature{
			Name:  "Thesis Management System",
			Email: "system@thesis.local",
			When:  time.Now(),
		},
	})

	return err
}

func (h *SourceCodeHandler) clearRepositoryContent(repoPath string) {
	// Remove all files except .git directory and system files
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}

		fullPath := filepath.Join(repoPath, entry.Name())
		os.RemoveAll(fullPath)
	}
}

// ===== FILE FILTERING LOGIC =====
func (h *SourceCodeHandler) shouldIgnoreFile(path string, info os.FileInfo) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")

	ignoreDirs := map[string]bool{
		"node_modules": true, ".git": true, ".svn": true, ".hg": true,
		"vendor": true, "target": true, "build": true, "dist": true, "out": true,
		"bin": true, "obj": true, "Debug": true, "Release": true,
		".vs": true, ".vscode": true, ".idea": true,
		"__pycache__": true, ".pytest_cache": true, ".mypy_cache": true,
		"coverage": true, ".coverage": true, "htmlcov": true, ".tox": true,
		"venv": true, "env": true, "virtualenv": true, ".virtualenv": true,
		"logs": true, "log": true, "temp": true, "tmp": true, ".tmp": true,
		"cache": true, ".cache": true, "packages": true, ".nuget": true,
		"TestResults": true, "bower_components": true, ".gradle": true,
		".maven": true, "Pods": true, "DerivedData": true, ".next": true,
		".nuxt": true, ".nyc_output": true,
	}

	for _, part := range parts {
		if ignoreDirs[part] {
			return true
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	ignoreExtensions := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true, ".a": true,
		".lib": true, ".obj": true, ".o": true, ".class": true, ".jar": true,
		".war": true, ".ear": true, ".pyc": true, ".pyo": true, ".pyd": true,
		".whl": true, ".egg": true, ".log": true, ".tmp": true, ".temp": true,
		".swp": true, ".swo": true, ".bak": true, ".orig": true, ".rej": true,
		".cache": true,
	}

	if ignoreExtensions[ext] {
		return true
	}

	fileName := filepath.Base(path)
	if strings.HasPrefix(fileName, ".") {
		allowedHiddenFiles := map[string]bool{
			".env.example": true, ".gitignore": true, ".editorconfig": true,
			".eslintrc": true, ".prettierrc": true, ".babelrc": true,
			".dockerignore": true,
		}

		if !allowedHiddenFiles[fileName] {
			return true
		}
	}

	if info.Size() > 1024*1024 {
		largeMediaExtensions := map[string]bool{
			".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
			".mp3": true, ".wav": true, ".flac": true, ".ape": true,
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
			".tiff": true, ".psd": true, ".ai": true, ".eps": true,
			".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
			".ppt": true, ".pptx": true, ".zip": true, ".rar": true, ".7z": true,
			".tar": true, ".gz": true, ".iso": true,
		}

		if largeMediaExtensions[ext] {
			return true
		}
	}

	return false
}

// ===== VALIDATION FUNCTIONS =====
func (h *SourceCodeHandler) validateStudentInfo(info *database.StudentInfo) []string {
	var errors []string
	if strings.TrimSpace(info.Name) == "" {
		errors = append(errors, "Student name is required")
	}
	if strings.TrimSpace(info.StudentID) == "" {
		errors = append(errors, "Student ID is required")
	}
	if strings.TrimSpace(info.Email) == "" {
		errors = append(errors, "Email is required")
	}
	if strings.TrimSpace(info.ThesisTitle) == "" {
		errors = append(errors, "Thesis title is required")
	}
	if info.Email != "" && !strings.Contains(info.Email, "@") {
		errors = append(errors, "Invalid email format")
	}
	return errors
}

func (h *SourceCodeHandler) validateSubmission(sourcePath string) *database.ValidationResult {
	result := &database.ValidationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}

	allowedExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true, ".java": true,
		".cpp": true, ".c": true, ".cs": true, ".php": true, ".rb": true, ".rs": true,
		".html": true, ".css": true, ".scss": true, ".less": true, ".sass": true,
		".sql": true, ".md": true, ".txt": true, ".json": true, ".xml": true,
		".yaml": true, ".yml": true, ".toml": true, ".ini": true, ".cfg": true,
		".sh": true, ".bat": true, ".ps1": true, ".dockerfile": true,
		".gitignore": true, ".env.example": true, ".editorconfig": true,
		".vue": true, ".jsx": true, ".tsx": true, ".svelte": true,
		".kt": true, ".swift": true, ".dart": true, ".r": true, ".m": true,
		".scala": true, ".clj": true, ".hs": true, ".elm": true, ".pl": true,
		".lua": true, ".coffee": true, ".pug": true, ".jade": true,
		".makefile": true, ".cmake": true, ".gradle": true,
		".properties": true, ".conf": true, ".config": true,
	}

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		result.FileCount++
		result.TotalSize += info.Size()

		ext := strings.ToLower(filepath.Ext(path))
		fileName := strings.ToLower(filepath.Base(path))

		allowedNoExtFiles := map[string]bool{
			"dockerfile": true, "makefile": true, "rakefile": true,
			"gemfile": true, "procfile": true, "vagrantfile": true,
			"readme": true, "license": true, "changelog": true,
			"authors": true, "contributors": true, "copying": true,
		}

		if ext == "" && !allowedNoExtFiles[fileName] {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("File without extension: %s", filepath.Base(path)))
		} else if ext != "" && !allowedExts[ext] {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Unusual file type: %s", filepath.Base(path)))
		}

		return nil
	})

	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "Failed to scan files")
		return result
	}

	readmePath := filepath.Join(sourcePath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, "Missing recommended file: README.md")
	}

	return result
}

// ===== FILE OPERATIONS =====
func (h *SourceCodeHandler) saveFile(src io.Reader, dst string) error {
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, src)
	return err
}

func (h *SourceCodeHandler) extractZip(src, dst string) (*database.FilterInfo, error) {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return nil, err
	}

	filterInfo := &database.FilterInfo{}

	for _, file := range reader.File {
		path := filepath.Join(dst, file.Name)

		if !strings.HasPrefix(path, filepath.Clean(dst)+string(os.PathSeparator)) {
			return nil, fmt.Errorf("invalid file path: %s", file.Name)
		}

		filterInfo.TotalFilesInZip++
		filterInfo.OriginalSize += file.FileInfo().Size()

		if h.shouldIgnoreFile(file.Name, file.FileInfo()) {
			filterInfo.FilesSkipped++
			continue
		}

		filterInfo.FilesAfterFilter++
		filterInfo.SizeAfterFilter += file.FileInfo().Size()

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}

		fileReader, err := file.Open()
		if err != nil {
			return nil, err
		}

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			fileReader.Close()
			return nil, err
		}

		_, err = io.Copy(targetFile, fileReader)
		fileReader.Close()
		targetFile.Close()
		if err != nil {
			return nil, err
		}
	}

	log.Printf("Extraction completed: %d files extracted, %d files skipped (%.1fMB → %.1fMB)",
		filterInfo.FilesAfterFilter, filterInfo.FilesSkipped,
		float64(filterInfo.OriginalSize)/1024/1024, float64(filterInfo.SizeAfterFilter)/1024/1024)

	return filterInfo, nil
}

func (h *SourceCodeHandler) copyFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || (info.IsDir() && info.Name() == ".git") {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return h.copyFile(path, destPath)
	})
}

func (h *SourceCodeHandler) copyFile(src, dst string) error {
	os.MkdirAll(filepath.Dir(dst), 0755)

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (h *SourceCodeHandler) countFiles(dirPath string) int {
	count := 0
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && !strings.Contains(path, ".git") {
			count++
		}
		return nil
	})
	return count
}

// ADD these new endpoint methods
func (h *SourceCodeHandler) GetUploadStatus(w http.ResponseWriter, r *http.Request) {
	submissionID := r.URL.Query().Get("id")
	if submissionID == "" {
		h.renderJSONError(w, "Missing submission ID")
		return
	}

	queueLength := len(h.uploadQueue)
	activeCount := 0
	h.activeUploads.Range(func(key, value interface{}) bool {
		activeCount++
		return true
	})

	status := map[string]interface{}{
		"submission_id":  submissionID,
		"queue_length":   queueLength,
		"active_uploads": activeCount,
		"system_status":  "operational",
	}

	if queueLength > 40 {
		status["system_status"] = "busy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *SourceCodeHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	activeCount := 0
	h.activeUploads.Range(func(key, value interface{}) bool {
		activeCount++
		return true
	})

	queueLen := len(h.uploadQueue)

	health := map[string]interface{}{
		"status":          "healthy",
		"active_uploads":  activeCount,
		"queue_length":    queueLen,
		"max_concurrent":  h.maxConcurrent,
		"queue_capacity":  cap(h.uploadQueue),
		"load_percentage": fmt.Sprintf("%.1f%%", float64(activeCount)/float64(h.maxConcurrent)*100),
		"timestamp":       time.Now().Format("2006-01-02 15:04:05"),
	}

	if activeCount >= h.maxConcurrent {
		health["status"] = "busy"
	}

	if queueLen >= 40 {
		health["status"] = "overloaded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// ===== CONTENT GENERATION =====
func (h *SourceCodeHandler) generateVersionedReadme(studentInfo *database.StudentInfo, submissionTime string) string {
	return fmt.Sprintf(`# %s

> **Final Thesis Source Code Repository**

## 👨‍🎓 Student Information
- **Name:** %s
- **Student ID:** %s
- **Email:** %s
- **Latest Submission:** %s

## 📚 Thesis Details
**Title:** %s

## 📂 Repository Structure
This repository contains the final source code for the thesis project. The code is automatically organized and filtered to include only relevant files.

### 🗂️ What's Included:
- ✅ Source code files
- ✅ Configuration files  
- ✅ Documentation
- ✅ Build scripts
- ❌ Dependencies (filtered out)
- ❌ Build artifacts (filtered out)
- ❌ Cache files (filtered out)

## 🔄 Version History
Each submission creates a new branch and is merged to main. Check the branch history to see all submissions:
- **Main branch:** Latest/final submission
- **Submission branches:** Individual submission history

## 📞 Contact
For questions about this thesis implementation:
- **Student:** %s
- **Email:** %s

---
**Repository:** %s/%s  
**Organization:** %s  
*Last updated: %s*
`,
		studentInfo.ThesisTitle,
		studentInfo.Name,
		studentInfo.StudentID,
		studentInfo.Email,
		time.Now().Format("January 2, 2006 at 15:04"),
		studentInfo.ThesisTitle,
		studentInfo.Name,
		studentInfo.Email,
		h.githubConfig.Organization,
		h.generateRepoName(studentInfo),
		h.githubConfig.Organization,
		time.Now().Format("January 2, 2006 at 15:04"))
}

func (h *SourceCodeHandler) generateVersionedSubmissionInfo(studentInfo *database.StudentInfo, submissionTime string) string {
	return fmt.Sprintf(`# Submission Information

## Student Details
- **Name:** %s
- **Student ID:** %s
- **Email:** %s
- **Thesis Title:** %s

## Submission Details
- **Submission ID:** %s
- **Submission Date:** %s
- **Repository:** %s/%s
- **Branch:** submission-%s

## Processing Information
- **Filtered:** Build artifacts, dependencies, and cache files automatically removed
- **Included:** Source code, documentation, and configuration files only
- **Organization:** %s
- **System:** Thesis Management System v1.0

## Academic Integrity
This submission represents the original work of the student listed above. The repository maintains a complete history of all submissions for academic review.

---
*This file is automatically generated by the Thesis Management System*
`,
		studentInfo.Name,
		studentInfo.StudentID,
		studentInfo.Email,
		studentInfo.ThesisTitle,
		uuid.New().String(),
		time.Now().Format("January 2, 2006 at 15:04:05"),
		h.githubConfig.Organization,
		h.generateRepoName(studentInfo),
		submissionTime,
		h.githubConfig.Organization)
}

// ===== DATABASE OPERATIONS =====
func (h *SourceCodeHandler) findStudentRecordID(studentInfo *database.StudentInfo) (int, error) {
	var studentRecordID int
	query := "SELECT id FROM student_records WHERE student_email = ? OR student_number = ?"
	err := h.db.Get(&studentRecordID, query, studentInfo.Email, studentInfo.StudentID)
	return studentRecordID, err
}

func (h *SourceCodeHandler) saveToDatabase(studentRecordID int, submissionID, filename string, fileSize int64, repoInfo *database.RepositoryInfo, commitInfo *database.CommitInfo) (int, error) {
	query := `
        INSERT INTO documents (
            student_record_id, document_type, file_path, original_filename,
            file_size, mime_type, repository_url, repository_id, commit_id,
            submission_id, validation_status, upload_status, is_confidential
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	result, err := h.db.Exec(query,
		studentRecordID, "thesis_source_code", repoInfo.WebURL, filename,
		fileSize, "application/zip", repoInfo.WebURL, repoInfo.ID,
		commitInfo.CommitID, submissionID, "valid", "completed", true)

	if err != nil {
		return 0, err
	}

	id, _ := result.LastInsertId()
	return int(id), nil
}

// ===== RESPONSE HELPERS =====
func (h *SourceCodeHandler) renderError(w http.ResponseWriter, message string, err error) {
	log.Printf("Source code upload error: %s - %v", message, err)
	h.renderJSONError(w, fmt.Sprintf("%s: %v", message, err))
}

func (h *SourceCodeHandler) renderJSONError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func (h *SourceCodeHandler) renderJSONResult(w http.ResponseWriter, result *database.SubmissionResult) {
	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(result)
}
