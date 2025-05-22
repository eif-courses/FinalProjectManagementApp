// handlers/api.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/i18n"
	"encoding/json"
	"net/http"
	"strings"
)

// SearchStudentsAPIWithI18n handles student search API
func SearchStudentsAPIWithI18n(db *database.ThesisDB, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.GetLangFromContext(r.Context())
		search := r.URL.Query().Get("q")

		if search == "" {
			http.Error(w, localizer.T(lang, "invalid_search"), http.StatusBadRequest)
			return
		}

		// For now, use dummy data
		allStudents := getDummyStudents()
		var filteredStudents []Student

		searchLower := strings.ToLower(search)
		for _, student := range allStudents {
			if strings.Contains(strings.ToLower(student.StudentName), searchLower) ||
				strings.Contains(strings.ToLower(student.ProjectTitle), searchLower) ||
				strings.Contains(strings.ToLower(student.Group), searchLower) {
				filteredStudents = append(filteredStudents, student)
			}
		}

		// Return JSON for HTMX or AJAX requests
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"students": filteredStudents,
			"total":    len(filteredStudents),
			"message":  localizer.T(lang, "showing_results", 1, len(filteredStudents), len(filteredStudents)),
		})
	}
}

// GetStudentsAPIWithI18n returns all students as JSON
func GetStudentsAPIWithI18n(db *database.ThesisDB, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.GetLangFromContext(r.Context())

		// For now, use dummy data
		students := getDummyStudents()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"students": students,
			"total":    len(students),
			"message":  localizer.T(lang, "showing_results", 1, len(students), len(students)),
		})
	}
}

// CreateSupervisorReportAPIWithI18n handles supervisor report creation via API
func CreateSupervisorReportAPIWithI18n(db *database.ThesisDB, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.GetLangFromContext(r.Context())

		var report database.SupervisorReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			http.Error(w, localizer.T(lang, "invalid_data"), http.StatusBadRequest)
			return
		}

		// For now, just return success
		// In real implementation: err := db.CreateSupervisorReport(&report)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": localizer.T(lang, "success_saved"),
		})
	}
}

// SubmitTopicAPIHandler handles topic submission via API
func SubmitTopicAPIHandler(localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FIXED: Remove unused user variable since we're not using it yet
		lang := i18n.GetLangFromContext(r.Context())

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse form data
		title := r.FormValue("title")
		// FIXED: Use underscore for unused variables
		_ = r.FormValue("title_en")
		problem := r.FormValue("problem")
		objective := r.FormValue("objective")
		_ = r.FormValue("tasks")
		_ = r.FormValue("supervisor")

		// Validate required fields
		if title == "" || problem == "" || objective == "" {
			http.Error(w, localizer.T(lang, "missing_required_fields"), http.StatusBadRequest)
			return
		}

		// In real implementation, save to database
		// topicRegistration := &database.ProjectTopicRegistration{
		//     StudentRecordID: getStudentRecordID(user.Email),
		//     Title:          title,
		//     TitleEn:        titleEn,
		//     Problem:        problem,
		//     Objective:      objective,
		//     Tasks:          tasks,
		//     Supervisor:     supervisor,
		//     Status:         "submitted",
		// }
		// err := db.CreateTopicRegistration(topicRegistration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": localizer.T(lang, "topic_submitted"),
		})
	}
}

// UserInfoAPIHandler returns current user information
func UserInfoAPIHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUserRoleAPIHandler updates user role (admin only)
func UpdateUserRoleAPIHandler(localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		lang := i18n.GetLangFromContext(r.Context())

		if !user.HasPermission("manage_users") {
			http.Error(w, localizer.T(lang, "access_denied"), http.StatusForbidden)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// FIXED: Use underscore for unused variable
		_ = r.FormValue("user_id")
		newRole := r.FormValue("role")

		// Validate role
		validRoles := []string{"student", "supervisor", "department_head", "admin"}
		isValidRole := false
		for _, role := range validRoles {
			if newRole == role {
				isValidRole = true
				break
			}
		}

		if !isValidRole {
			http.Error(w, localizer.T(lang, "invalid_role"), http.StatusBadRequest)
			return
		}

		// In real implementation, update database
		// err := db.UpdateUserRole(userID, newRole)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": localizer.T(lang, "user_role_updated"),
		})
	}
}

// UploadDocumentAPIHandler handles document uploads
func UploadDocumentAPIHandler(localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FIXED: Remove unused user variable
		lang := i18n.GetLangFromContext(r.Context())

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			http.Error(w, localizer.T(lang, "file_too_large"), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("document")
		if err != nil {
			http.Error(w, localizer.T(lang, "no_file_uploaded"), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// FIXED: Use underscore for unused variable
		_ = r.FormValue("type")

		// Validate file type
		allowedTypes := []string{"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
		isValidType := false
		for _, allowedType := range allowedTypes {
			if header.Header.Get("Content-Type") == allowedType {
				isValidType = true
				break
			}
		}

		if !isValidType {
			http.Error(w, localizer.T(lang, "invalid_file_type"), http.StatusBadRequest)
			return
		}

		// In real implementation, save file and create database record
		// filePath := saveUploadedFile(file, header)
		// document := &database.Document{
		//     DocumentType:    documentType,
		//     FilePath:        filePath,
		//     StudentRecordID: getStudentRecordID(user.Email),
		// }
		// err = db.CreateDocument(document)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"message":  localizer.T(lang, "file_uploaded_successfully"),
			"filename": header.Filename,
			"size":     header.Size,
		})
	}
}

// DeleteDocumentAPIHandler handles document deletion
func DeleteDocumentAPIHandler(localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FIXED: Remove unused user variable
		lang := i18n.GetLangFromContext(r.Context())

		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		documentID := r.URL.Query().Get("id")
		if documentID == "" {
			http.Error(w, localizer.T(lang, "missing_document_id"), http.StatusBadRequest)
			return
		}

		// In real implementation, check ownership and delete
		// document, err := db.GetDocumentByID(documentID)
		// if err != nil || !userOwnsDocument(user, document) {
		//     http.Error(w, localizer.T(lang, "access_denied"), http.StatusForbidden)
		//     return
		// }
		// err = db.DeleteDocument(documentID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": localizer.T(lang, "document_deleted"),
		})
	}
}

// GetStatisticsAPIHandler returns dashboard statistics
func GetStatisticsAPIHandler(localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		stats := getDashboardStats(user)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}
