// handlers/supervisor_report.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"database/sql"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SupervisorReportHandler handles supervisor report operations
type SupervisorReportHandler struct {
	db *sqlx.DB
}

// NewSupervisorReportHandler creates a new handler instance
func NewSupervisorReportHandler(db *sqlx.DB) *SupervisorReportHandler {
	return &SupervisorReportHandler{
		db: db,
	}
}

// Add this method to SupervisorReportHandler
func (h *SupervisorReportHandler) GetCompactSupervisorModal(w http.ResponseWriter, r *http.Request) {
	studentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Check the mode parameter
	mode := r.URL.Query().Get("mode")
	isReadOnly := mode == "view"

	student, err := h.getStudentRecord(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Get the existing report
	existingReport, err := h.getSupervisorReport(studentID)
	hasReport := err == nil && existingReport != nil

	// SIMPLE RULE: If report exists, it's always read-only
	if hasReport {
		isReadOnly = true
		log.Printf("DEBUG: Report exists for student %d - forcing read-only mode", studentID)
	}

	formData := database.NewSupervisorReportFormData(existingReport)
	language := h.getLanguageFromRequest(r)
	user := auth.GetUserFromContext(r.Context())

	supervisorName := ""
	supervisorEmail := ""
	if user != nil {
		supervisorName = user.Name
		supervisorEmail = user.Email
	}

	props := database.SupervisorReportFormProps{
		StudentRecord:          *student,
		InitialReport:          existingReport,
		FormVariant:            language,
		CurrentSupervisorName:  supervisorName,
		CurrentSupervisorEmail: supervisorEmail,
		IsReadOnly:             isReadOnly,
	}

	w.Header().Set("Content-Type", "text/html")
	component := templates.CompactSupervisorForm(props, formData)
	component.Render(r.Context(), w)
}

//func (h *SupervisorReportHandler) getTopicStatus(studentID int) (string, error) {
//	var status string
//	query := `
//		SELECT COALESCE(status, '')
//		FROM project_topic_registrations
//		WHERE student_record_id = ?
//	`
//	err := h.db.Get(&status, query, studentID)
//	return status, err
//}

// SubmitSupervisorReport handles form submission
func (h *SupervisorReportHandler) SubmitSupervisorReport(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Starting supervisor report submission")

	studentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Printf("ERROR: Invalid student ID: %v", err)
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("ERROR: Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Check if this is a draft save
	isDraft := r.FormValue("is_draft") == "true"
	if isDraft {
		// Redirect to draft handler
		h.SaveSupervisorDraft(w, r)
		return
	}

	// Get supervisor info from context
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		log.Printf("ERROR: No user in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	supervisorName := user.Name
	if supervisorName == "" {
		log.Printf("ERROR: Empty supervisor name")
		http.Error(w, "Supervisor information not found", http.StatusBadRequest)
		return
	}

	formData, err := h.parseFormData(r)
	if err != nil {
		log.Printf("ERROR: Failed to parse form data: %v", err)
		http.Error(w, "Invalid form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to database model
	reportData := formData.ToSupervisorReportData(studentID, supervisorName)

	// Validate
	if err := reportData.Validate(); err != nil {
		log.Printf("ERROR: Validation failed: %v", err)
		http.Error(w, "Validation error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Save to database
	if err := h.saveSupervisorReport(reportData); err != nil {
		log.Printf("ERROR: Failed to save report: %v", err)
		http.Error(w, "Failed to save report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("DEBUG: Report saved successfully")

	// Create audit log
	h.createAuditLog(database.AuditLog{
		UserEmail:    user.Email,
		UserRole:     "supervisor",
		Action:       "create_supervisor_report",
		ResourceType: "supervisor_report",
		ResourceID:   database.NullableString(fmt.Sprintf("%d", studentID)),
		Details: func() *string {
			detailsStr := fmt.Sprintf(`{"student_id":%d,"supervisor":"%s"}`, studentID, supervisorName)
			return &detailsStr
		}(),
		IPAddress: database.NullableString(h.getClientIP(r)),
		UserAgent: database.NullableString(r.UserAgent()),
		Success:   true,
	})

	// Return success response with proper modal closing and refresh
	//w.Header().Set("Content-Type", "text/html")
	//w.Header().Set("HX-Trigger", "closeModal,refreshStudentList") // Multiple triggers
	//w.WriteHeader(http.StatusOK)
	//

	// Return success response with HTMX trigger
	//w.Header().Set("Content-Type", "text/html")
	//w.Header().Set("HX-Trigger", "closeModal,refreshStudentTable") // Custom trigger
	//w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Trigger", "supervisorReportSaved")
	w.WriteHeader(http.StatusNoContent) // 204 No Content

	successHTML := `
    <div id="success-notification" class="fixed top-4 right-4 z-50 bg-green-50 border border-green-200 rounded-lg p-4 shadow-lg max-w-sm">
        <div class="flex items-center">
            <svg class="h-5 w-5 text-green-400 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            <div class="text-green-600 text-sm font-medium">Supervisor report saved successfully!</div>
        </div>
    </div>
    <script>
        // Close modal immediately
        const modal = document.getElementById('supervisor-modal');
        if (modal && window.modalState) {
            modal.style.display = 'none';
            window.modalState.openModalId = null;
            document.body.style.overflow = '';
        }

        // Remove success notification after 3 seconds
        setTimeout(function() {
            const notification = document.getElementById('success-notification');
            if (notification) {
                notification.remove();
            }
        }, 3000);
    </script>`
	w.Write([]byte(successHTML))
}

// Database helper methods using sqlx
func (h *SupervisorReportHandler) getStudentRecord(studentID int) (*database.StudentRecord, error) {
	var record database.StudentRecord

	query := `
		SELECT id, student_group, final_project_title, final_project_title_en,
		       student_email, student_name, student_lastname, student_number,
		       supervisor_email, study_program, department, program_code,
		       current_year, reviewer_email, reviewer_name, is_favorite,
		       created_at, updated_at
		FROM student_records
		WHERE id = ?
	`

	err := h.db.Get(&record, query, studentID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (h *SupervisorReportHandler) getSupervisorReport(studentID int) (*database.SupervisorReport, error) {
	var report database.SupervisorReport

	query := `
		SELECT id, student_record_id, supervisor_comments, supervisor_name,
		       supervisor_position, supervisor_workplace, is_pass_or_failed,
		       is_signed, other_match, one_match, own_match, join_match,
		       created_date, updated_date, grade, final_comments
		FROM supervisor_reports
		WHERE student_record_id = ?
	`

	err := h.db.Get(&report, query, studentID)
	if err != nil {
		return nil, err
	}

	return &report, nil
}

func (h *SupervisorReportHandler) saveSupervisorReport(data *database.SupervisorReportData) error {
	// Check if report exists
	//existingReport, err := h.getSupervisorReport(data.StudentRecordID)
	_, err := h.getSupervisorReport(data.StudentRecordID)
	if err == sql.ErrNoRows {
		// Create new report
		query := `
            INSERT INTO supervisor_reports (
                student_record_id, supervisor_comments, supervisor_name,
                supervisor_position, supervisor_workplace, is_pass_or_failed,
                other_match, one_match, own_match, join_match, grade, 
                final_comments, is_signed
            ) VALUES (
                :student_record_id, :supervisor_comments, :supervisor_name,
                :supervisor_position, :supervisor_workplace, :is_pass_or_failed,
                :other_match, :one_match, :own_match, :join_match, :grade, 
                :final_comments, :is_signed
            )
		`

		// Convert to map for Named exec
		params := map[string]interface{}{
			"student_record_id":    data.StudentRecordID,
			"supervisor_comments":  data.SupervisorComments,
			"supervisor_name":      data.SupervisorName,
			"supervisor_position":  data.SupervisorPosition,
			"supervisor_workplace": data.SupervisorWorkplace,
			"is_pass_or_failed":    data.IsPassOrFailed,
			"other_match":          data.OtherMatch,
			"one_match":            data.OneMatch,
			"own_match":            data.OwnMatch,
			"join_match":           data.JoinMatch,
			"grade":                data.Grade,
			"final_comments":       data.FinalComments,
			"is_signed":            true, // Set to true for final submission
		}

		_, err = h.db.NamedExec(query, params)
		return err

	} else if err != nil {
		return err
	} else {
		// Update existing report using sqlx Named query
		query := `
            UPDATE supervisor_reports SET
                supervisor_comments = :supervisor_comments,
                supervisor_position = :supervisor_position,
                supervisor_workplace = :supervisor_workplace,
                is_pass_or_failed = :is_pass_or_failed,
                other_match = :other_match,
                one_match = :one_match,
                own_match = :own_match,
                join_match = :join_match,
                grade = :grade,
                final_comments = :final_comments,
                updated_date = :updated_date,
                is_signed = :is_signed
            WHERE student_record_id = :student_record_id
		`

		params := map[string]interface{}{
			"student_record_id":    data.StudentRecordID,
			"supervisor_comments":  data.SupervisorComments,
			"supervisor_position":  data.SupervisorPosition,
			"supervisor_workplace": data.SupervisorWorkplace,
			"is_pass_or_failed":    data.IsPassOrFailed,
			"other_match":          data.OtherMatch,
			"one_match":            data.OneMatch,
			"own_match":            data.OwnMatch,
			"join_match":           data.JoinMatch,
			"grade":                data.Grade,
			"final_comments":       data.FinalComments,
			"updated_date":         time.Now(),
			"is_signed":            true, // Set to true for final submission
		}

		_, err = h.db.NamedExec(query, params)
		return err
	}
}

func (h *SupervisorReportHandler) createAuditLog(log database.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			user_email, user_role, action, resource_type, resource_id,
			details, ip_address, user_agent, success, created_at
		) VALUES (
			:user_email, :user_role, :action, :resource_type, :resource_id,
			:details, :ip_address, :user_agent, :success, :created_at
		)
	`

	// Set created_at if not set
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	_, err := h.db.NamedExec(query, log)
	return err
}

// Bulk operations using sqlx (bonus methods)
func (h *SupervisorReportHandler) getMultipleStudentRecords(studentIDs []int) ([]database.StudentRecord, error) {
	if len(studentIDs) == 0 {
		return []database.StudentRecord{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, student_group, final_project_title, final_project_title_en,
		       student_email, student_name, student_lastname, student_number,
		       supervisor_email, study_program, department, program_code,
		       current_year, reviewer_email, reviewer_name, is_favorite,
		       created_at, updated_at
		FROM student_records
		WHERE id IN (?)
	`, studentIDs)

	if err != nil {
		return nil, err
	}

	// Rebind for MySQL
	query = h.db.Rebind(query)

	var records []database.StudentRecord
	err = h.db.Select(&records, query, args...)
	return records, err
}

func (h *SupervisorReportHandler) getSupervisorReportsBySupervisor(supervisorEmail string) ([]database.SupervisorReport, error) {
	var reports []database.SupervisorReport

	query := `
		SELECT sr.id, sr.student_record_id, sr.supervisor_comments, sr.supervisor_name,
		       sr.supervisor_position, sr.supervisor_workplace, sr.is_pass_or_failed,
		       sr.is_signed, sr.other_match, sr.one_match, sr.own_match, sr.join_match,
		       sr.created_date, sr.updated_date, sr.grade, sr.final_comments
		FROM supervisor_reports sr
		JOIN student_records st ON sr.student_record_id = st.id
		WHERE st.supervisor_email = ?
		ORDER BY sr.created_date DESC
	`

	err := h.db.Select(&reports, query, supervisorEmail)
	return reports, err
}

// Form parsing and validation methods
func (h *SupervisorReportHandler) parseFormData(r *http.Request) (*database.SupervisorReportFormData, error) {
	// Parse numeric fields with error handling
	otherMatch, err := strconv.ParseFloat(r.FormValue("other_match"), 64)
	if err != nil {
		otherMatch = 0
	}

	oneMatch, err := strconv.ParseFloat(r.FormValue("one_match"), 64)
	if err != nil {
		oneMatch = 0
	}

	ownMatch, err := strconv.ParseFloat(r.FormValue("own_match"), 64)
	if err != nil {
		ownMatch = 0
	}

	joinMatch, err := strconv.ParseFloat(r.FormValue("join_match"), 64)
	if err != nil {
		joinMatch = 0
	}

	// Parse pass/fail
	isPassOrFailed := r.FormValue("is_pass_or_failed") == "true"

	// Parse optional grade
	var grade *int
	if gradeStr := r.FormValue("grade"); gradeStr != "" {
		if g, err := strconv.Atoi(gradeStr); err == nil && g >= 1 && g <= 10 {
			grade = &g
		}
	}

	return &database.SupervisorReportFormData{
		SupervisorComments:  r.FormValue("supervisor_comments"),
		SupervisorWorkplace: r.FormValue("supervisor_workplace"),
		SupervisorPosition:  r.FormValue("supervisor_position"),
		OtherMatch:          otherMatch,
		OneMatch:            oneMatch,
		OwnMatch:            ownMatch,
		JoinMatch:           joinMatch,
		IsPassOrFailed:      isPassOrFailed,
		Grade:               grade,
		FinalComments:       r.FormValue("final_comments"),
		SubmissionDate:      time.Now(),
	}, nil
}

//func (h *SupervisorReportHandler) renderFormWithErrors(w http.ResponseWriter, r *http.Request, studentID int, formData *database.SupervisorReportFormData, errors map[string]string) {
//	student, _ := h.getStudentRecord(studentID)
//	existingReport, _ := h.getSupervisorReport(studentID)
//
//	if formData == nil {
//		formData = database.NewSupervisorReportFormData(existingReport)
//	}
//
//	language := h.getLanguageFromRequest(r)
//	user := auth.GetUserFromContext(r.Context())
//
//	supervisorName := ""
//	supervisorEmail := ""
//	if user != nil {
//		supervisorName = user.Name
//		supervisorEmail = user.Email
//	}
//
//	props := database.SupervisorReportFormProps{
//		StudentRecord:          *student,
//		InitialReport:          existingReport,
//		FormVariant:            language,
//		IsModalOpen:            true,
//		IsSaving:               false,
//		ValidationErrors:       errors,
//		CurrentSupervisorName:  supervisorName,
//		CurrentSupervisorEmail: supervisorEmail,
//	}
//
//	w.Header().Set("Content-Type", "text/html")
//	component := templates.SupervisorReportModal(props, formData)
//	component.Render(r.Context(), w)
//}

func (h *SupervisorReportHandler) parseValidationErrors(err error) map[string]string {
	// Parse validation errors based on your validation approach
	errorMessage := err.Error()

	// Map common validation errors to field names
	if strings.Contains(errorMessage, "supervisor comments") {
		return map[string]string{"supervisor_comments": errorMessage}
	}
	if strings.Contains(errorMessage, "supervisor name") {
		return map[string]string{"supervisor_name": errorMessage}
	}
	if strings.Contains(errorMessage, "supervisor position") {
		return map[string]string{"supervisor_position": errorMessage}
	}
	if strings.Contains(errorMessage, "supervisor workplace") {
		return map[string]string{"supervisor_workplace": errorMessage}
	}
	if strings.Contains(errorMessage, "match percentage") {
		return map[string]string{"other_match": errorMessage}
	}
	if strings.Contains(errorMessage, "grade") {
		return map[string]string{"grade": errorMessage}
	}

	// Default to general error
	return map[string]string{"general": errorMessage}
}

// Context/session helper methods
func (h *SupervisorReportHandler) getLanguageFromRequest(r *http.Request) string {
	// Try to get from session first
	if lang := h.getFromSession(r, "language"); lang != "" {
		return lang
	}

	// Try URL parameter
	if lang := r.URL.Query().Get("lang"); lang == "en" || lang == "lt" {
		return lang
	}

	// Try Accept-Language header
	if r.Header.Get("Accept-Language") != "" &&
		len(r.Header.Get("Accept-Language")) >= 2 &&
		r.Header.Get("Accept-Language")[:2] == "en" {
		return "en"
	}

	// Default to Lithuanian
	return "lt"
}

func (h *SupervisorReportHandler) getSupervisorNameFromRequest(r *http.Request) string {
	// Get from session or JWT token
	if name := h.getFromSession(r, "user_name"); name != "" {
		return name
	}

	// Or construct from first_name + last_name
	firstName := h.getFromSession(r, "first_name")
	lastName := h.getFromSession(r, "last_name")
	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}

	return ""
}

func (h *SupervisorReportHandler) getUserEmailFromRequest(r *http.Request) string {
	return h.getFromSession(r, "user_email")
}

func (h *SupervisorReportHandler) getFromSession(r *http.Request, key string) string {
	// Get user from context (your auth system)
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		return ""
	}

	switch key {
	case "user_name":
		return user.Name
	case "user_email":
		return user.Email
	case "first_name":
		// Extract first name from full name
		parts := strings.Fields(user.Name)
		if len(parts) > 0 {
			return parts[0]
		}
		return user.Name
	case "last_name":
		// Extract last name from full name
		parts := strings.Fields(user.Name)
		if len(parts) > 1 {
			return parts[len(parts)-1]
		}
		return ""
	case "language":
		// Default language or get from user preferences
		return "lt"
	default:
		return ""
	}
}

func (h *SupervisorReportHandler) getClientIP(r *http.Request) string {
	// Check for forwarded IP first
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	// Extract IP from RemoteAddr (remove port)
	parts := strings.Split(r.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return r.RemoteAddr
}

func (h *SupervisorReportHandler) getButtonLabel(language string, isEdit bool) string {
	if language == "en" {
		if isEdit {
			return "Edit Report"
		}
		return "Create Report"
	} else {
		if isEdit {
			return "Redaguoti atsiliepimą"
		}
		return "Pildyti atsiliepimą"
	}
}

func (h *SupervisorReportHandler) getModalTitle(language string) string {
	if language == "en" {
		return "Supervisor Report"
	}
	return "Vadovo atsiliepimas"
}

// SaveSupervisorDraft handles saving supervisor report as draft

// SaveSupervisorDraft handles saving supervisor report as draft
func (h *SupervisorReportHandler) SaveSupervisorDraft(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Starting supervisor report draft save")

	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		log.Printf("ERROR: User not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "id")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		log.Printf("ERROR: Invalid student ID: %v", err)
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	log.Printf("DEBUG: Processing draft for student ID: %d", studentID)

	// Get student record
	student, err := h.getStudentRecord(studentID)
	if err != nil {
		log.Printf("ERROR: Failed to get student record: %v", err)
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if user can save draft
	if user.Role != auth.RoleSupervisor || student.SupervisorEmail != user.Email {
		log.Printf("ERROR: Access denied - user role: %s, student supervisor: %s, user email: %s",
			user.Role, student.SupervisorEmail, user.Email)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("ERROR: Failed to parse form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Parse form data without strict validation for drafts
	formData, err := h.parseFormDataForDraft(r)
	if err != nil {
		log.Printf("ERROR: Failed to parse form data: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Check if report already exists
	existingReport, err := h.getSupervisorReport(studentID)
	reportExists := err == nil && existingReport != nil

	tx, err := h.db.Beginx()
	if err != nil {
		log.Printf("ERROR: Failed to begin transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if !reportExists {
		log.Printf("DEBUG: Creating new draft report")
		// Create new draft report
		_, err = tx.Exec(`
            INSERT INTO supervisor_reports (
                student_record_id,
                supervisor_comments,
                supervisor_name,
                supervisor_position,
                supervisor_workplace,
                is_pass_or_failed,
                other_match,
                one_match,
                own_match,
                join_match,
                is_signed,
                created_date
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
			studentID,
			formData.SupervisorComments,
			user.Name,
			formData.SupervisorPosition,
			formData.SupervisorWorkplace,
			formData.IsPassOrFailed,
			formData.OtherMatch,
			formData.OneMatch,
			formData.OwnMatch,
			formData.JoinMatch,
			false, // Not signed for draft
		)

		if err != nil {
			log.Printf("ERROR: Failed to insert draft: %v", err)
			http.Error(w, "Failed to save draft", http.StatusInternalServerError)
			return
		}
	} else {
		// Report exists
		if existingReport.IsSigned {
			log.Printf("ERROR: Report already signed")
			http.Error(w, "Report already signed", http.StatusBadRequest)
			return
		}

		log.Printf("DEBUG: Updating existing draft report ID: %d", existingReport.ID)
		// Update existing unsigned report
		_, err = tx.Exec(`
            UPDATE supervisor_reports SET
                supervisor_comments = ?,
                supervisor_position = ?,
                supervisor_workplace = ?,
                is_pass_or_failed = ?,
                other_match = ?,
                one_match = ?,
                own_match = ?,
                join_match = ?,
                updated_date = NOW()
            WHERE id = ?`,
			formData.SupervisorComments,
			formData.SupervisorPosition,
			formData.SupervisorWorkplace,
			formData.IsPassOrFailed,
			formData.OtherMatch,
			formData.OneMatch,
			formData.OwnMatch,
			formData.JoinMatch,
			existingReport.ID,
		)

		if err != nil {
			log.Printf("ERROR: Failed to update draft: %v", err)
			http.Error(w, "Failed to save draft", http.StatusInternalServerError)
			return
		}
	}

	// Create audit log
	auditLog := database.AuditLog{
		UserEmail:    user.Email,
		UserRole:     user.Role,
		Action:       "save_supervisor_report_draft",
		ResourceType: "supervisor_report",
		ResourceID:   database.NullableString(studentIDStr),
		Details: func() *string {
			detailsStr := fmt.Sprintf(`{"student_id":%d,"action":"draft_saved"}`, studentID)
			return &detailsStr
		}(),
		IPAddress: database.NullableString(h.getClientIP(r)),
		UserAgent: database.NullableString(r.UserAgent()),
		Success:   true,
		CreatedAt: time.Now(),
	}

	// Don't fail if audit log fails
	if err := h.createAuditLog(auditLog); err != nil {
		log.Printf("WARNING: Failed to create audit log: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("ERROR: Failed to commit transaction: %v", err)
		http.Error(w, "Failed to save draft", http.StatusInternalServerError)
		return
	}

	log.Printf("DEBUG: Draft saved successfully")

	// Return success response
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="text-xs text-green-600">Draft saved</div>`))
}

// parseFormDataForDraft is more lenient than parseFormData for draft saves
func (h *SupervisorReportHandler) parseFormDataForDraft(r *http.Request) (*database.SupervisorReportFormData, error) {
	// Parse numeric fields with defaults for empty values
	otherMatch, _ := strconv.ParseFloat(r.FormValue("other_match"), 64)
	oneMatch, _ := strconv.ParseFloat(r.FormValue("one_match"), 64)
	ownMatch, _ := strconv.ParseFloat(r.FormValue("own_match"), 64)
	joinMatch, _ := strconv.ParseFloat(r.FormValue("join_match"), 64)

	// Parse pass/fail - default to false if not set
	isPassOrFailed := r.FormValue("is_pass_or_failed") == "true"

	// Parse optional grade
	var grade *int
	if gradeStr := r.FormValue("grade"); gradeStr != "" {
		if g, err := strconv.Atoi(gradeStr); err == nil && g >= 1 && g <= 10 {
			grade = &g
		}
	}

	return &database.SupervisorReportFormData{
		SupervisorComments:  r.FormValue("supervisor_comments"),
		SupervisorWorkplace: r.FormValue("supervisor_workplace"),
		SupervisorPosition:  r.FormValue("supervisor_position"),
		OtherMatch:          otherMatch,
		OneMatch:            oneMatch,
		OwnMatch:            ownMatch,
		JoinMatch:           joinMatch,
		IsPassOrFailed:      isPassOrFailed,
		Grade:               grade,
		FinalComments:       r.FormValue("final_comments"),
		SubmissionDate:      time.Now(),
	}, nil
}
