// handlers/dashboard.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"strings"
	"time"
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

		// Option 1: Create empty student data structure
		data = &database.StudentDashboardData{
			StudentRecord: &database.StudentRecord{
				ID:                0,
				StudentNumber:     "N/A",
				StudentGroup:      "N/A",
				StudyProgram:      "N/A",
				FinalProjectTitle: "",
			},
			TopicRegistration:     nil,
			SourceCodeRepository:  nil,
			HasThesisPDF:          false,
			ThesisDocument:        nil,
			CompanyRecommendation: nil,
			VideoPresentation:     nil,
			SupervisorReport:      nil,
			ReviewerReport:        nil,
			TopicCommentCount:     0,
			HasUnreadComments:     false,
		}

		// Still render the student dashboard with empty data
		err = templates.CompactStudentDashboard(user, data, locale).Render(r.Context(), w)
		if err != nil {
			log.Printf("Error rendering compact student dashboard with empty data: %v", err)
			http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
		}
		return
	}

	// Render student dashboard with actual data
	err = templates.CompactStudentDashboard(user, data, locale).Render(r.Context(), w)
	if err != nil {
		log.Printf("Error rendering compact student dashboard: %v", err)
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
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

func (h *DashboardHandlers) getStudentDashboardData(email string) (*database.StudentDashboardData, error) {
	data := &database.StudentDashboardData{}

	// Get student record
	var studentRecord database.StudentRecord
	err := h.db.Get(&studentRecord,
		"SELECT * FROM student_records WHERE student_email = ?", email)
	if err != nil {
		log.Printf("Error getting student record for email %s: %v", email, err)
		return nil, fmt.Errorf("student record not found: %w", err)
	}

	log.Printf("Found student: %s %s, Group: %s, Program: %s",
		studentRecord.StudentName, studentRecord.StudentLastname,
		studentRecord.StudentGroup, studentRecord.StudyProgram)

	data.StudentRecord = &studentRecord

	// Initialize fields to prevent nil errors
	data.Documents = []database.Document{}
	data.Videos = []database.Video{}
	data.TopicCommentCount = 0
	data.HasUnreadComments = false

	// Get topic registration comments count
	err = h.db.Get(&data.TopicCommentCount,
		`SELECT COUNT(*) FROM topic_registration_comments tc
         JOIN project_topic_registrations ptr ON tc.topic_registration_id = ptr.id
         WHERE ptr.student_record_id = ?`, studentRecord.ID)
	if err != nil {
		log.Printf("Error getting comment count: %v", err)
		data.TopicCommentCount = 0
	}

	// Check for unread comments
	var unreadCount int
	err = h.db.Get(&unreadCount,
		`SELECT COUNT(*) FROM topic_registration_comments tc
         JOIN project_topic_registrations ptr ON tc.topic_registration_id = ptr.id
         WHERE ptr.student_record_id = ? AND tc.is_read = FALSE`, studentRecord.ID)
	if err == nil {
		data.HasUnreadComments = unreadCount > 0
	}

	// Get documents with detailed logging
	var documents []database.Document
	err = h.db.Select(&documents,
		"SELECT * FROM documents WHERE student_record_id = ? ORDER BY uploaded_date DESC",
		studentRecord.ID)
	if err != nil {
		log.Printf("Error getting documents: %v", err)
	} else {
		log.Printf("Found %d documents for student %d", len(documents), studentRecord.ID)
		data.Documents = documents

		// Process documents by type
		for i := range documents {
			doc := &documents[i]
			log.Printf("Document type: %s, ID: %d", doc.DocumentType, doc.ID)

			switch strings.ToLower(doc.DocumentType) {
			case "thesis_pdf", "thesis", "final_thesis.pdf":
				data.HasThesisPDF = true
				data.ThesisDocument = doc
			case "thesis_source_code", "source_code":
				data.SourceCodeRepository = doc
				data.HasSourceCode = true
				if doc.UploadStatus != "" {
					data.SourceCodeStatus = doc.UploadStatus
				} else {
					data.SourceCodeStatus = "uploaded"
				}
				log.Printf("Found source code: %s", doc.OriginalFilename)
			case "company_recommendation", "recommendation.pdf":
				data.CompanyRecommendation = doc
			}
		}
	}

	// Get videos
	var videos []database.Video
	err = h.db.Select(&videos,
		"SELECT * FROM videos WHERE student_record_id = ? ORDER BY created_at DESC",
		studentRecord.ID)
	if err != nil {
		log.Printf("Error getting videos: %v", err)
	} else {
		data.Videos = videos
		for i := range videos {
			video := &videos[i]
			if video.Status == "ready" {
				data.VideoPresentation = video
				data.HasVideo = true
				break
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
		data.HasSupervisorReport = true
	} else if err != sql.ErrNoRows {
		log.Printf("Error getting supervisor report: %v", err)
	}

	// Get reviewer report
	var reviewerReport database.ReviewerReport
	err = h.db.Get(&reviewerReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ? ORDER BY created_date DESC LIMIT 1",
		studentRecord.ID)
	if err == nil {
		data.ReviewerReport = &reviewerReport
		data.HasReviewerReport = true
	} else if err != sql.ErrNoRows {
		log.Printf("Error getting reviewer report: %v", err)
	}

	// Get topic registration
	var topicRegistration database.ProjectTopicRegistration
	err = h.db.Get(&topicRegistration,
		"SELECT * FROM project_topic_registrations WHERE student_record_id = ? ORDER BY created_at DESC LIMIT 1",
		studentRecord.ID)
	if err == nil {
		data.TopicRegistration = &topicRegistration
		data.TopicStatus = topicRegistration.Status
		data.HasTopicApproved = topicRegistration.Status == "approved"
	} else {
		if err != sql.ErrNoRows {
			log.Printf("Error getting topic registration: %v", err)
		}
		data.TopicStatus = "not_submitted"
		data.HasTopicApproved = false
	}

	// Handle defense info
	if studentRecord.DefenseDate != nil {
		data.DefenseScheduled = true
		data.DefenseDate = studentRecord.GetDefenseDateFormatted()
		data.DefenseLocation = studentRecord.DefenseLocation
	}

	// Calculate completion status
	data.CompletionPercentage = h.calculateProgress(data)
	data.CurrentStage = h.getCurrentStage(data)
	data.IsReadyForDefense = h.isReadyForDefense(data)

	// Set academic info
	data.AcademicYear = time.Now().Year()
	data.Semester = h.getCurrentSemester()

	return data, nil
}

// Helper methods for student dashboard
func (h *DashboardHandlers) calculateProgress(data *database.StudentDashboardData) int {
	total := 7 // Total requirements
	completed := 0

	// Topic approved
	if data.TopicStatus == "approved" {
		completed++
	}

	// Source code uploaded
	if data.SourceCodeRepository != nil {
		completed++
	}

	// Thesis PDF available
	if data.HasThesisPDF {
		completed++
	}

	// Supervisor report completed and signed
	if data.SupervisorReport != nil && data.SupervisorReport.IsSigned {
		completed++
	}

	// Reviewer report completed and signed
	if data.ReviewerReport != nil && data.ReviewerReport.IsSigned {
		completed++
	}

	// Company recommendation uploaded
	if data.CompanyRecommendation != nil {
		completed++
	}

	// Video presentation uploaded (optional, but counts toward progress)
	if data.VideoPresentation != nil {
		completed++
	}

	return (completed * 100) / total
}

func (h *DashboardHandlers) getCurrentStage(data *database.StudentDashboardData) string {
	if !data.HasTopicApproved {
		return "Topic Registration"
	}
	if !data.HasSupervisorReport {
		return "Supervisor Evaluation"
	}
	if !data.HasReviewerReport {
		return "Reviewer Evaluation"
	}
	if !data.HasThesisPDF {
		return "Document Submission"
	}
	if !data.HasSourceCode {
		return "Source Code Upload"
	}
	if !data.HasVideo {
		return "Video Presentation"
	}
	if data.DefenseScheduled {
		return "Defense Preparation"
	}
	return "Ready for Defense"
}

func (h *DashboardHandlers) isReadyForDefense(data *database.StudentDashboardData) bool {
	return data.HasTopicApproved &&
		data.HasSupervisorReport &&
		data.HasReviewerReport &&
		data.HasThesisPDF &&
		data.HasSourceCode
}

func (h *DashboardHandlers) getCurrentSemester() string {
	month := time.Now().Month()
	if month >= 9 || month <= 1 {
		return "Fall"
	} else if month >= 2 && month <= 6 {
		return "Spring"
	}
	return "Summer"
}

// ================================
// OTHER DASHBOARD DATA METHODS
// ================================

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
         AND defense_date BETWEEN UNIX_TIMESTAMP() AND UNIX_TIMESTAMP() + (30 * 24 * 60 * 60)`,
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
