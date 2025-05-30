package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type UploadHandlers struct {
	db *sqlx.DB
}

func NewUploadHandlers(db *sqlx.DB) *UploadHandlers {
	return &UploadHandlers{db: db}
}

// RecommendationUploadHandler handles company recommendation uploads
func (h *UploadHandlers) RecommendationUploadHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleStudent {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("recommendation")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if filepath.Ext(header.Filename) != ".pdf" {
		http.Error(w, "Only PDF files are allowed", http.StatusBadRequest)
		return
	}

	// Get student record
	var studentID int
	err = h.db.Get(&studentID, "SELECT id FROM student_records WHERE student_email = ?", user.Email)
	if err != nil {
		http.Error(w, "Student record not found", http.StatusNotFound)
		return
	}

	// Create uploads directory
	uploadDir := "uploads/recommendations"
	os.MkdirAll(uploadDir, 0755)

	// Generate unique filename
	filename := fmt.Sprintf("recommendation_%d_%d.pdf", studentID, time.Now().Unix())
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Save to database
	_, err = h.db.Exec(`
        INSERT INTO documents (student_record_id, document_type, file_path, original_filename, file_size, mime_type)
        VALUES (?, ?, ?, ?, ?, ?)
    `, studentID, "company_recommendation", filePath, header.Filename, header.Size, "application/pdf")

	if err != nil {
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Recommendation uploaded successfully",
	})
}

// VideoUploadHandler handles video presentation uploads
func (h *UploadHandlers) VideoUploadHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleStudent {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form with larger limit for videos
	err := r.ParseMultipartForm(500 << 20) // 500MB max
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{".mp4": true, ".avi": true, ".mov": true, ".wmv": true}
	if !allowedExts[ext] {
		http.Error(w, "Invalid video format. Allowed: MP4, AVI, MOV, WMV", http.StatusBadRequest)
		return
	}

	// Get student record
	var studentID int
	err = h.db.Get(&studentID, "SELECT id FROM student_records WHERE student_email = ?", user.Email)
	if err != nil {
		http.Error(w, "Student record not found", http.StatusNotFound)
		return
	}

	// Create uploads directory
	uploadDir := "uploads/videos"
	os.MkdirAll(uploadDir, 0755)

	// Generate unique filename
	filename := fmt.Sprintf("video_%d_%d%s", studentID, time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Save to database
	_, err = h.db.Exec(`
        INSERT INTO documents (student_record_id, document_type, file_path, original_filename, file_size, mime_type)
        VALUES (?, ?, ?, ?, ?, ?)
    `, studentID, "video_presentation", filePath, header.Filename, header.Size, header.Header.Get("Content-Type"))

	if err != nil {
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Video uploaded successfully",
	})
}

// DocumentPreviewHandler serves document previews
func (h *UploadHandlers) DocumentPreviewHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get document ID from URL
	docIDStr := r.URL.Path[len("/api/documents/"):]
	docIDStr = docIDStr[:len(docIDStr)-len("/preview")]
	docID, err := strconv.Atoi(docIDStr)
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	// Get document from database
	var doc database.Document
	err = h.db.Get(&doc, "SELECT * FROM documents WHERE id = ?", docID)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Check permissions
	if !h.canAccessDocument(user, &doc) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Serve file
	http.ServeFile(w, r, doc.FilePath)
}

// DocumentDownloadHandler serves document downloads
func (h *UploadHandlers) DocumentDownloadHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get document ID from URL
	docIDStr := r.URL.Path[len("/api/documents/"):]
	docIDStr = docIDStr[:len(docIDStr)-len("/download")]
	docID, err := strconv.Atoi(docIDStr)
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	// Get document from database
	var doc database.Document
	err = h.db.Get(&doc, "SELECT * FROM documents WHERE id = ?", docID)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Check permissions
	if !h.canAccessDocument(user, &doc) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Set download headers
	filename := database.StringValue(doc.OriginalFilename)
	if filename == "" {
		filename = "document.pdf"
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	// Serve file
	http.ServeFile(w, r, doc.FilePath)
}

func (h *UploadHandlers) canAccessDocument(user *auth.AuthenticatedUser, doc *database.Document) bool {
	// Admin can access everything
	if user.Role == auth.RoleAdmin {
		return true
	}

	// Get student record for the document
	var studentEmail string
	err := h.db.Get(&studentEmail,
		"SELECT student_email FROM student_records WHERE id = ?",
		doc.StudentRecordID)
	if err != nil {
		return false
	}

	// Students can access their own documents
	if user.Role == auth.RoleStudent && user.Email == studentEmail {
		return true
	}

	// Get supervisor and reviewer emails
	var supervisorEmail, reviewerEmail string
	err = h.db.Get(&supervisorEmail,
		"SELECT supervisor_email FROM student_records WHERE id = ?",
		doc.StudentRecordID)
	if err == nil && user.Role == auth.RoleSupervisor && user.Email == supervisorEmail {
		return true
	}

	err = h.db.Get(&reviewerEmail,
		"SELECT reviewer_email FROM student_records WHERE id = ?",
		doc.StudentRecordID)
	if err == nil && user.Role == auth.RoleReviewer && user.Email == reviewerEmail {
		return true
	}

	// Department heads can access documents in their department
	if user.Role == auth.RoleDepartmentHead {
		var department string
		err = h.db.Get(&department,
			"SELECT sr.department FROM student_records sr WHERE sr.id = ?",
			doc.StudentRecordID)
		if err == nil {
			var userDepartment string
			err = h.db.Get(&userDepartment,
				"SELECT department FROM department_heads WHERE email = ?",
				user.Email)
			if err == nil && department == userDepartment {
				return true
			}
		}
	}

	return false
}
