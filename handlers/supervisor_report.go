// handlers/supervisor_report.go
package handlers

import (
	"FinalProjectManagementApp/database"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// SupervisorReportHandler handles supervisor report operations
type SupervisorReportHandler struct {
	// Add any dependencies like database connection, logger, etc.
	// db *database.DB
	// logger *log.Logger
}

// NewSupervisorReportHandler creates a new handler instance
func NewSupervisorReportHandler() *SupervisorReportHandler {
	return &SupervisorReportHandler{}
}

// GetSupervisorReportModal returns the modal for editing supervisor report
func (h *SupervisorReportHandler) GetSupervisorReportModal(w http.ResponseWriter, r *http.Request) {
	studentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Get student record
	student, err := database.GetStudentRecord(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Get existing supervisor report if any
	existingReport, _ := database.GetSupervisorReport(studentID)

	// Create form data
	formData := NewSupervisorReportFormData(existingReport)

	// Get language from context/session/header
	language := h.getLanguageFromRequest(r)

	// Create props
	props := SupervisorReportFormProps{
		StudentRecord: *student,
		InitialReport: existingReport,
		ButtonLabel:   h.getButtonLabel(language, existingReport != nil),
		ModalTitle:    h.getModalTitle(language),
		FormVariant:   language,
		IsModalOpen:   true,
		IsSaving:      false,
	}

	// Set content type for HTMX
	w.Header().Set("Content-Type", "text/html")

	// Render modal
	component := SupervisorReportModal(props, formData)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// SubmitSupervisorReport handles form submission
func (h *SupervisorReportHandler) SubmitSupervisorReport(w http.ResponseWriter, r *http.Request) {
	studentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.renderFormWithErrors(w, r, studentID, nil, map[string]string{
			"general": "Failed to parse form data",
		})
		return
	}

	formData, err := h.parseFormData(r)
	if err != nil {
		h.renderFormWithErrors(w, r, studentID, formData, map[string]string{
			"general": err.Error(),
		})
		return
	}

	// Get supervisor info from session/context
	supervisorName := h.getSupervisorNameFromRequest(r)
	if supervisorName == "" {
		h.renderFormWithErrors(w, r, studentID, formData, map[string]string{
			"general": "Supervisor information not found",
		})
		return
	}

	// Convert to database model
	reportData := formData.ToSupervisorReportData(studentID, supervisorName)

	// Validate
	if err := reportData.Validate(); err != nil {
		validationErrors := h.parseValidationErrors(err)
		h.renderFormWithErrors(w, r, studentID, formData, validationErrors)
		return
	}

	// Save to database
	if err := database.SaveSupervisorReport(reportData); err != nil {
		h.renderFormWithErrors(w, r, studentID, formData, map[string]string{
			"general": "Failed to save report: " + err.Error(),
		})
		return
	}

	// Create audit log
	userEmail := h.getUserEmailFromRequest(r)
	database.CreateAuditLog(database.AuditLog{
		UserEmail:    userEmail,
		UserRole:     "supervisor",
		Action:       "create_supervisor_report",
		ResourceType: "supervisor_report",
		ResourceID:   database.NullableString(fmt.Sprintf("%d", studentID)),
		Details: database.JSONMap{
			"student_id": studentID,
			"pass":       formData.IsPassOrFailed,
			"similarity": formData.OtherMatch + formData.OneMatch + formData.OwnMatch + formData.JoinMatch,
		},
		IPAddress: database.NullableString(h.getClientIP(r)),
		UserAgent: database.NullableString(r.UserAgent()),
		Success:   true,
	})

	// Return success response (close modal and trigger refresh)
	w.Header().Set("HX-Trigger", "reportSaved")
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// GetSupervisorReportButton returns just the button (for initial page load)
func (h *SupervisorReportHandler) GetSupervisorReportButton(w http.ResponseWriter, r *http.Request) {
	studentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Get student record
	student, err := database.GetStudentRecord(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if report exists
	existingReport, _ := database.GetSupervisorReport(studentID)

	language := h.getLanguageFromRequest(r)

	props := SupervisorReportFormProps{
		StudentRecord: *student,
		InitialReport: existingReport,
		ButtonLabel:   h.getButtonLabel(language, existingReport != nil),
		FormVariant:   language,
		IsModalOpen:   false,
		IsSaving:      false,
	}

	// Set content type
	w.Header().Set("Content-Type", "text/html")

	// Render just the button
	component := SupervisorReportForm(props, nil)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Helper methods

func (h *SupervisorReportHandler) parseFormData(r *http.Request) (*SupervisorReportFormData, error) {
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

	return &SupervisorReportFormData{
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

func (h *SupervisorReportHandler) renderFormWithErrors(w http.ResponseWriter, r *http.Request, studentID int, formData *SupervisorReportFormData, errors map[string]string) {
	student, _ := database.GetStudentRecord(studentID)
	existingReport, _ := database.GetSupervisorReport(studentID)

	if formData == nil {
		formData = NewSupervisorReportFormData(existingReport)
	}

	language := h.getLanguageFromRequest(r)

	props := SupervisorReportFormProps{
		StudentRecord:    *student,
		InitialReport:    existingReport,
		FormVariant:      language,
		IsModalOpen:      true,
		IsSaving:         false,
		ValidationErrors: errors,
	}

	w.Header().Set("Content-Type", "text/html")
	component := SupervisorReportModal(props, formData)
	component.Render(r.Context(), w)
}

func (h *SupervisorReportHandler) parseValidationErrors(err error) map[string]string {
	// Parse validation errors based on your validation approach
	// This is a simple example - you might want more sophisticated error parsing
	return map[string]string{
		"general": err.Error(),
	}
}

// Context/session helper methods - implement based on your auth system

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
		(r.Header.Get("Accept-Language")[:2] == "en") {
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
	// Implement based on your session management
	// This is a placeholder - replace with your actual session handling

	// Example for context-based session:
	if val := r.Context().Value(key); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}

	return ""
}

func (h *SupervisorReportHandler) getClientIP(r *http.Request) string {
	// Check for forwarded IP first
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
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
