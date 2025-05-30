// handlers/dashboard.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
)

type DashboardHandlers struct {
	db *sqlx.DB
}

func NewDashboardHandlers(db *sqlx.DB) *DashboardHandlers {
	return &DashboardHandlers{db: db}
}

func (h *DashboardHandlers) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Get current locale
	locale := h.getLocale(r, w)

	switch user.Role {
	case auth.RoleStudent:
		h.renderStudentDashboard(w, r, user, locale)
		return

	case auth.RoleSupervisor:
		h.renderSupervisorDashboard(w, r, user, locale)
		return

	case auth.RoleDepartmentHead:
		h.renderDepartmentDashboard(w, r, user, locale)
		return

	case auth.RoleReviewer:
		h.renderReviewerDashboard(w, r, user, locale)
		return

	case auth.RoleCommissionMember:
		h.renderCommissionDashboard(w, r, user, locale)
		return

	case auth.RoleAdmin:
		h.renderAdminDashboard(w, r, user, locale)
		return

	default:
		// Generic dashboard for unknown roles
		err := templates.Dashboard(user, "Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

func (h *DashboardHandlers) renderStudentDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get student-specific data
	data, err := h.getStudentDashboardData(user.Email)
	if err != nil {
		log.Printf("Error getting student dashboard data: %v", err)
		// Fall back to basic dashboard
		err = templates.Dashboard(user, "Student Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// Render student dashboard with data
	err = templates.StudentDashboard(user, data, locale).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering student dashboard: %v", err)
		// Fall back to basic dashboard
		err = templates.Dashboard(user, "Student Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

func (h *DashboardHandlers) renderSupervisorDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get supervisor-specific data
	data, err := h.getSupervisorDashboardData(user.Email)
	if err != nil {
		log.Printf("Error getting supervisor dashboard data: %v", err)
		// Fall back to basic dashboard
		err = templates.Dashboard(user, "Supervisor Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// TODO: Create SupervisorDashboard template when ready
	// For now, use basic dashboard
	_ = data // Suppress unused variable warning until template is implemented
	err = templates.Dashboard(user, "Supervisor Dashboard").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) renderDepartmentDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get department-specific data
	data, err := h.getDepartmentDashboardData(user.Email)
	if err != nil {
		log.Printf("Error getting department dashboard data: %v", err)
		// Fall back to basic dashboard
		err = templates.Dashboard(user, "Department Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// TODO: Create DepartmentDashboard template when ready
	// For now, use basic dashboard
	_ = data // Suppress unused variable warning until template is implemented
	err = templates.Dashboard(user, "Department Dashboard").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) renderAdminDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get admin-specific data
	data, err := h.getAdminDashboardData()
	if err != nil {
		log.Printf("Error getting admin dashboard data: %v", err)
		// Fall back to basic dashboard
		err = templates.Dashboard(user, "Admin Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// TODO: Create AdminDashboard template when ready
	// For now, use basic dashboard
	_ = data // Suppress unused variable warning until template is implemented
	err = templates.Dashboard(user, "Admin Dashboard").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) renderReviewerDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get reviewer-specific data
	data, err := h.getReviewerDashboardData(user.Email)
	if err != nil {
		log.Printf("Error getting reviewer dashboard data: %v", err)
		// Fall back to basic dashboard
		err = templates.Dashboard(user, "Reviewer Dashboard").Render(r.Context(), w)
		if err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
		return
	}

	// TODO: Create ReviewerDashboard template when ready
	// For now, use basic dashboard
	_ = data // Suppress unused variable warning until template is implemented
	err = templates.Dashboard(user, "Reviewer Dashboard").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func (h *DashboardHandlers) renderCommissionDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Commission members have special access via access codes
	// TODO: Implement commission dashboard
	err := templates.Dashboard(user, "Commission Dashboard").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// ================================
// DATA RETRIEVAL METHODS
// ================================

func (h *DashboardHandlers) getStudentDashboardData(email string) (*templates.StudentDashboardData, error) {
	data := &templates.StudentDashboardData{}

	// Get student record
	var studentRecord database.StudentRecord
	err := h.db.Get(&studentRecord,
		"SELECT * FROM student_records WHERE student_email = ?", email)
	if err != nil {
		return nil, err
	}
	data.StudentRecord = &studentRecord

	// Get documents
	var documents []database.Document
	err = h.db.Select(&documents,
		"SELECT * FROM documents WHERE student_record_id = ? ORDER BY uploaded_date DESC",
		studentRecord.ID)
	if err == nil {
		for _, doc := range documents {
			switch doc.DocumentType {
			case "thesis_pdf", "thesis":
				data.HasThesisPDF = true
				data.ThesisDocument = &doc
			case "thesis_source_code", "SOURCE_CODE":
				data.SourceCodeRepository = &doc
				data.SourceCodeStatus = doc.UploadStatus
			case "company_recommendation":
				data.CompanyRecommendation = &doc
			case "video_presentation", "presentation":
				data.VideoPresentation = &doc
			}
		}
	}

	// Get supervisor report
	var supervisorReport database.SupervisorReport
	err = h.db.Get(&supervisorReport,
		"SELECT * FROM supervisor_reports WHERE student_record_id = ? ORDER BY created_date DESC LIMIT 1",
		studentRecord.ID)
	if err == nil {
		data.SupervisorReport = &supervisorReport
	}

	// Get reviewer report
	var reviewerReport database.ReviewerReport
	err = h.db.Get(&reviewerReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ? ORDER BY created_date DESC LIMIT 1",
		studentRecord.ID)
	if err == nil {
		data.ReviewerReport = &reviewerReport
	}

	// Get topic status
	var topicStatus string
	err = h.db.Get(&topicStatus,
		"SELECT status FROM project_topic_registrations WHERE student_record_id = ? ORDER BY created_at DESC LIMIT 1",
		studentRecord.ID)
	if err == nil {
		data.TopicStatus = topicStatus
	} else {
		data.TopicStatus = "not_submitted"
	}

	// Set defense info
	data.DefenseScheduled = studentRecord.DefenseDate != nil
	if data.DefenseScheduled && studentRecord.DefenseDate != nil {
		data.DefenseDate = studentRecord.GetDefenseDateFormatted()
	}

	return data, nil
}

type SupervisorDashboardData struct {
	AssignedStudents []database.StudentRecord
	PendingReports   int
	CompletedReports int
	TotalStudents    int
}

func (h *DashboardHandlers) getSupervisorDashboardData(email string) (*SupervisorDashboardData, error) {
	data := &SupervisorDashboardData{}

	// Get assigned students
	err := h.db.Select(&data.AssignedStudents,
		"SELECT * FROM student_records WHERE supervisor_email = ? ORDER BY student_lastname, student_name",
		email)
	if err != nil {
		return nil, err
	}

	data.TotalStudents = len(data.AssignedStudents)

	// Count pending and completed reports
	for _, student := range data.AssignedStudents {
		var reportCount int
		err := h.db.Get(&reportCount,
			"SELECT COUNT(*) FROM supervisor_reports WHERE student_record_id = ? AND is_signed = true",
			student.ID)
		if err == nil && reportCount > 0 {
			data.CompletedReports++
		} else {
			data.PendingReports++
		}
	}

	return data, nil
}

type DepartmentDashboardData struct {
	TotalStudents     int
	PendingTopics     int
	CompletedDefenses int
	UpcomingDefenses  int
	DepartmentStats   map[string]int
}

func (h *DashboardHandlers) getDepartmentDashboardData(email string) (*DepartmentDashboardData, error) {
	data := &DepartmentDashboardData{
		DepartmentStats: make(map[string]int),
	}

	// Get department head info to find department
	var department string
	err := h.db.Get(&department,
		"SELECT department FROM department_heads WHERE email = ?", email)
	if err != nil {
		return nil, err
	}

	// Get total students in department
	err = h.db.Get(&data.TotalStudents,
		"SELECT COUNT(*) FROM student_records WHERE department LIKE ?",
		"%"+department+"%")
	if err != nil {
		data.TotalStudents = 0
	}

	// Get pending topics
	err = h.db.Get(&data.PendingTopics,
		`SELECT COUNT(*) FROM project_topic_registrations ptr 
         JOIN student_records sr ON ptr.student_record_id = sr.id 
         WHERE sr.department LIKE ? AND ptr.status = 'submitted'`,
		"%"+department+"%")
	if err != nil {
		data.PendingTopics = 0
	}

	// Get upcoming defenses (next 30 days)
	err = h.db.Get(&data.UpcomingDefenses,
		`SELECT COUNT(*) FROM student_records 
         WHERE department LIKE ? AND defense_date IS NOT NULL 
         AND defense_date BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 30 DAY)`,
		"%"+department+"%")
	if err != nil {
		data.UpcomingDefenses = 0
	}

	return data, nil
}

type AdminDashboardData struct {
	TotalUsers       int
	TotalStudents    int
	TotalSupervisors int
	SystemHealth     string
	RecentActivity   []string
}

func (h *DashboardHandlers) getAdminDashboardData() (*AdminDashboardData, error) {
	data := &AdminDashboardData{
		SystemHealth:   "Healthy",
		RecentActivity: []string{},
	}

	// Get total students
	err := h.db.Get(&data.TotalStudents, "SELECT COUNT(*) FROM student_records")
	if err != nil {
		data.TotalStudents = 0
	}

	// Get total supervisors (distinct supervisor emails)
	err = h.db.Get(&data.TotalSupervisors,
		"SELECT COUNT(DISTINCT supervisor_email) FROM student_records WHERE supervisor_email != ''")
	if err != nil {
		data.TotalSupervisors = 0
	}

	// Get department heads count
	var departmentHeads int
	err = h.db.Get(&departmentHeads, "SELECT COUNT(*) FROM department_heads WHERE is_active = true")
	if err != nil {
		departmentHeads = 0
	}

	data.TotalUsers = data.TotalStudents + data.TotalSupervisors + departmentHeads

	return data, nil
}

type ReviewerDashboardData struct {
	AssignedStudents []database.StudentRecord
	PendingReviews   int
	CompletedReviews int
	Invitations      []database.ReviewerInvitation
}

func (h *DashboardHandlers) getReviewerDashboardData(email string) (*ReviewerDashboardData, error) {
	data := &ReviewerDashboardData{}

	// Get assigned students
	err := h.db.Select(&data.AssignedStudents,
		"SELECT * FROM student_records WHERE reviewer_email = ? ORDER BY student_lastname, student_name",
		email)
	if err != nil {
		return nil, err
	}

	// Count pending and completed reviews
	for _, student := range data.AssignedStudents {
		var reportCount int
		err := h.db.Get(&reportCount,
			"SELECT COUNT(*) FROM reviewer_reports WHERE student_record_id = ? AND is_signed = true",
			student.ID)
		if err == nil && reportCount > 0 {
			data.CompletedReviews++
		} else {
			data.PendingReviews++
		}
	}

	// Get active reviewer invitations
	err = h.db.Select(&data.Invitations,
		"SELECT * FROM reviewer_invitations WHERE reviewer_email = ? AND is_active = true ORDER BY created_at DESC",
		email)
	if err != nil {
		data.Invitations = []database.ReviewerInvitation{}
	}

	return data, nil
}

// ================================
// LOCALE HANDLING
// ================================

func (h *DashboardHandlers) getLocale(r *http.Request, w http.ResponseWriter) string {
	// Priority: Query param > Cookie > Default
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = getLocaleFromCookie(r)
	}
	if locale == "" {
		locale = "lt" // default
	}

	// Validate locale
	if locale != "lt" && locale != "en" {
		locale = "lt"
	}

	// Set locale cookie if changed via query param
	if r.URL.Query().Get("locale") != "" {
		setLocaleCookie(w, locale)
	}

	return locale
}

func getLocaleFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("locale")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setLocaleCookie(w http.ResponseWriter, locale string) {
	cookie := &http.Cookie{
		Name:     "locale",
		Value:    locale,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: false,
		Secure:   false, // TODO Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
