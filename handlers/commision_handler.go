package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/services"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"
)

type CommissionHandler struct {
	commissionService  *services.CommissionService
	studentListHandler *StudentListHandler
}

func NewCommissionHandler(cs *services.CommissionService, slh *StudentListHandler) *CommissionHandler {
	return &CommissionHandler{
		commissionService:  cs,
		studentListHandler: slh,
	}
}

// Helper function to get user from context
func (h *CommissionHandler) getUserFromContext(r *http.Request) *auth.AuthenticatedUser {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		return nil
	}
	return user
}

// Helper function to get locale
func (h *CommissionHandler) getLocale(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}
	return locale
}

// Admin handlers
func (h *CommissionHandler) ShowManagementPage(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Check if user has permission
	if user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	locale := h.getLocale(r)

	// Get active access codes
	accessCodes, err := h.commissionService.ListActiveAccess(r.Context(), user.Email)
	if err != nil {
		accessCodes = []database.CommissionMember{}
	}

	// Get departments and study programs for the form
	departments := []string{
		"Elektronikos ir informatikos fakultetas",
		"Informacijos technologijų katedra",
		"Verslo vadybos katedra",
	}

	studyPrograms := []string{
		"Programų sistemos",
		"Informacinės sistemos",
		"Kompiuterių tinklai",
		"Elektronikos inžinerija",
	}

	component := templates.CommissionManagement(user, locale, templates.CommissionManagementData{
		AccessCodes:   accessCodes,
		Departments:   departments,
		StudyPrograms: studyPrograms,
	})

	component.Render(r.Context(), w)
}

func (h *CommissionHandler) CreateAccess(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	durationDays, _ := strconv.Atoi(r.FormValue("duration_days"))
	if durationDays <= 0 {
		durationDays = 7
	}

	maxAccess, _ := strconv.Atoi(r.FormValue("max_access"))
	year, _ := strconv.Atoi(r.FormValue("year"))

	params := services.CreateAccessParams{
		Department:     r.FormValue("department"),
		StudyProgram:   r.FormValue("study_program"),
		Year:           year,
		Description:    r.FormValue("description"),
		DurationDays:   durationDays,
		MaxAccess:      maxAccess,
		CreatedBy:      user.Email,
		AccessLevel:    r.FormValue("access_level"),
		CommissionType: "defense",
	}

	// Handle allowed groups
	if groups := r.FormValue("allowed_groups"); groups != "" {
		params.AllowedGroups = strings.Split(groups, ",")
	}

	member, err := h.commissionService.CreateAccess(r.Context(), params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the new access code row as HTML (HTMX response)
	locale := h.getLocale(r)
	component := templates.AccessCodeRow(member, locale)
	component.Render(r.Context(), w)
}

func (h *CommissionHandler) DeactivateAccess(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accessCode := chi.URLParam(r, "accessCode")

	err := h.commissionService.DeactivateAccess(r.Context(), accessCode, user.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Add this method to get a student by ID
func (h *CommissionHandler) getStudentByID(studentID int) (*database.StudentRecord, error) {
	var student database.StudentRecord
	err := h.studentListHandler.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	return &student, err
}

// Commission member handlers
func (h *CommissionHandler) ShowAccessPage(w http.ResponseWriter, r *http.Request) {
	accessCode := chi.URLParam(r, "accessCode")

	member, err := h.commissionService.ValidateAndRecordAccess(r.Context(), accessCode)
	if err != nil {
		component := templates.CommissionAccessError(err.Error())
		component.Render(r.Context(), w)
		return
	}

	// Check if member is nil
	if member == nil {
		component := templates.CommissionAccessError("Invalid commission member")
		component.Render(r.Context(), w)
		return
	}

	// Get students based on commission member's access
	students, err := h.commissionService.GetStudentsForCommission(r.Context(), member)
	if err != nil {
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	locale := h.getLocale(r)
	component := templates.CommissionPortal(member, students, locale)
	component.Render(r.Context(), w)
}
func (h *CommissionHandler) ViewStudent(w http.ResponseWriter, r *http.Request) {
	accessCode := chi.URLParam(r, "accessCode")
	studentID := chi.URLParam(r, "studentID")

	// Validate access
	member, err := h.commissionService.ValidateAndRecordAccess(r.Context(), accessCode)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if member is nil
	if member == nil {
		http.Error(w, "Invalid commission member", http.StatusInternalServerError)
		return
	}

	// Get student details
	sid, _ := strconv.Atoi(studentID)
	student, err := h.getStudentByID(sid)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if student is nil
	if student == nil {
		http.Error(w, "Student data is invalid", http.StatusInternalServerError)
		return
	}

	// Verify student belongs to commission's department
	// Make sure both are not empty before comparing
	if student.Department != "" && member.Department != "" {
		if student.Department != member.Department {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
	}

	// Get all related data for the student
	dashboardData, err := h.getStudentDashboardData(sid)
	if err != nil {
		http.Error(w, "Failed to load student data", http.StatusInternalServerError)
		return
	}

	locale := h.getLocale(r)
	component := templates.CommissionStudentView(member, student, dashboardData, locale, accessCode)
	component.Render(r.Context(), w)
}

// Add this method to get student dashboard data

// Add this method to get student dashboard data
func (h *CommissionHandler) getStudentDashboardData(studentID int) (*database.StudentDashboardData, error) {
	data := &database.StudentDashboardData{}

	// Get documents
	err := h.studentListHandler.db.Select(&data.Documents,
		"SELECT * FROM documents WHERE student_record_id = ?", studentID)
	if err != nil {
		return nil, err
	}

	// Check for specific document types
	for _, doc := range data.Documents {
		switch doc.DocumentType {
		case "thesis", "thesis_pdf":
			data.HasThesisPDF = true
			data.ThesisDocument = &doc
		case "video":
			data.HasVideo = true
		case "recommendation":
			// HasRecommendation doesn't exist in StudentDashboardData
			// Set CompanyRecommendation instead
			data.CompanyRecommendation = &doc
		case "thesis_source_code":
			data.HasSourceCode = true
			data.SourceCodeRepository = &doc
		}
	}

	// Get supervisor report
	var supervisorReport database.SupervisorReport
	err = h.studentListHandler.db.Get(&supervisorReport,
		"SELECT * FROM supervisor_reports WHERE student_record_id = ?", studentID)
	if err == nil {
		data.SupervisorReport = &supervisorReport
		data.HasSupervisorReport = true
	}

	// Get reviewer report
	var reviewerReport database.ReviewerReport
	err = h.studentListHandler.db.Get(&reviewerReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?", studentID)
	if err == nil {
		data.ReviewerReport = &reviewerReport
		data.HasReviewerReport = true
	}

	return data, nil
}

// Add this method to list active access codes
func (h *CommissionHandler) ListActiveAccess(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accessCodes, err := h.commissionService.ListActiveAccess(r.Context(), user.Email)
	if err != nil {
		http.Error(w, "Failed to load access codes", http.StatusInternalServerError)
		return
	}

	locale := h.getLocale(r)

	// Render just the table body rows
	for _, code := range accessCodes {
		templates.AccessCodeRow(&code, locale).Render(r.Context(), w)
	}
}
