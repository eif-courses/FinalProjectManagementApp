package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/i18n"
)

type AdminHandler struct {
	db        *sqlx.DB
	templates *template.Template
	localizer *i18n.Localizer
}

func NewAdminHandler(db *sqlx.DB, localizer *i18n.Localizer) *AdminHandler {
	funcMap := template.FuncMap{
		"printf": fmt.Sprintf,
		"add":    func(a, b int) int { return a + b },
		"sub":    func(a, b int) int { return a - b },
		"mul":    func(a, b int) int { return a * b },
		"div":    func(a, b int) int { return a / b },
		"formatTime": func(timestamp int64) string {
			return time.Unix(timestamp, 0).Format("2006-01-02 15:04")
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse templates in a specific order to avoid issues
	// First parse layouts
	tmpl, err := tmpl.ParseGlob("templates/layouts/*.html")
	if err != nil {
		fmt.Printf("Error parsing layouts: %v\n", err)
	}

	// Then parse admin templates
	tmpl, err = tmpl.ParseGlob("templates/admin/*.html")
	if err != nil {
		fmt.Printf("Error parsing admin templates: %v\n", err)
	}

	// Then parse components
	tmpl, err = tmpl.ParseGlob("templates/components/*.html")
	if err != nil {
		fmt.Printf("Error parsing components: %v\n", err)
	}

	// Skip shared templates for now to avoid the success.html issue
	// We'll handle them separately if needed

	// Log all loaded templates
	fmt.Println("Loaded admin templates:")
	for _, t := range tmpl.Templates() {
		fmt.Printf("  - %s\n", t.Name())
	}

	return &AdminHandler{
		db:        db,
		templates: tmpl,
		localizer: localizer,
	}
}

// DashboardHandler shows admin dashboard with statistics
func (h *AdminHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	// Get department for department heads
	var department string
	if user.Role == auth.RoleDepartmentHead {
		err := h.db.Get(&department,
			"SELECT department FROM department_heads WHERE email = ?",
			user.Email)
		if err != nil {
			department = ""
		}
	}

	// Get statistics
	stats := h.getDashboardStats(department)

	// Create breadcrumbs
	breadcrumbs := []map[string]string{
		{"Title": h.localizer.T(lang, "navigation.dashboard"), "URL": ""},
	}

	data := map[string]interface{}{
		"Title":       h.localizer.T(lang, "dashboard.admin_dashboard"),
		"User":        user,
		"Lang":        lang,
		"Stats":       stats,
		"CurrentYear": time.Now().Year(),
		"Breadcrumbs": breadcrumbs,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	// Try to execute the dashboard template
	err := h.templates.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		// If template execution fails, log detailed error and try a fallback
		fmt.Printf("Template execution error for dashboard.html: %v\n", err)

		// Try to render a simple HTML response as fallback
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Admin Dashboard</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto p-8">
        <h1 class="text-3xl font-bold mb-4">Admin Dashboard</h1>
        <p>Welcome, %s</p>
        <p class="text-red-600 mt-4">Template error: %v</p>
        <div class="mt-4">
            <a href="/auth/logout" class="bg-red-500 text-white px-4 py-2 rounded">Logout</a>
        </div>
    </div>
</body>
</html>`, user.Name, err)
	}
}

// StudentsTableHandler returns students table HTML for HTMX
func (h *AdminHandler) StudentsTableHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	// Get department filter for department heads
	departmentFilter := ""
	if user.Role == auth.RoleDepartmentHead {
		h.db.Get(&departmentFilter,
			"SELECT department FROM department_heads WHERE email = ?",
			user.Email)
	}

	// Build query
	query := `
		SELECT sr.*, 
			   CASE WHEN ptr.status = 'approved' THEN 1 ELSE 0 END as topic_approved,
			   COALESCE(ptr.status, 'draft') as topic_status,
			   CASE WHEN sup.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			   CASE WHEN rev.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report
		FROM student_records sr
		LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
		LEFT JOIN supervisor_reports sup ON sr.id = sup.student_record_id
		LEFT JOIN reviewer_reports rev ON sr.id = rev.student_record_id
	`

	args := []interface{}{}
	if departmentFilter != "" {
		query += " WHERE sr.department = ?"
		args = append(args, departmentFilter)
	}
	query += " ORDER BY sr.student_group, sr.student_lastname"

	var students []database.StudentSummaryView
	err := h.db.Select(&students, query, args...)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Students": students,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	// Return just the table HTML
	if err := h.templates.ExecuteTemplate(w, "students_table.html", data); err != nil {
		fmt.Printf("Template error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TopicsTableHandler returns topics table HTML for HTMX
func (h *AdminHandler) TopicsTableHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	departmentFilter := ""
	if user.Role == auth.RoleDepartmentHead {
		h.db.Get(&departmentFilter,
			"SELECT department FROM department_heads WHERE email = ?",
			user.Email)
	}

	query := `
		SELECT ptr.*, sr.student_name, sr.student_lastname, sr.student_email, 
			   sr.student_group, sr.study_program
		FROM project_topic_registrations ptr
		JOIN student_records sr ON ptr.student_record_id = sr.id
		WHERE ptr.status = 'submitted'
	`

	args := []interface{}{}
	if departmentFilter != "" {
		query += " AND sr.department = ?"
		args = append(args, departmentFilter)
	}
	query += " ORDER BY ptr.submitted_at DESC"

	var topics []database.TopicWithDetails
	err := h.db.Select(&topics, query, args...)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to load topics", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Topics": topics,
		"User":   user,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "topics_table.html", data); err != nil {
		fmt.Printf("Template error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// CommissionTableHandler returns commission table HTML for HTMX
func (h *AdminHandler) CommissionTableHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	// Get commission accesses
	createdBy := ""
	if user.Role == auth.RoleDepartmentHead {
		createdBy = user.Email
	}

	query := `
		SELECT * FROM commission_members
		WHERE (created_by = ? OR ? = '')
		ORDER BY created_at DESC
	`

	var accesses []auth.CommissionAccess
	err := h.db.Select(&accesses, query, createdBy, createdBy)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		http.Error(w, "Failed to load commission accesses", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Accesses": accesses,
		"BaseURL":  getEnv("BASE_URL", "http://localhost:8080"),
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "commission_table.html", data); err != nil {
		fmt.Printf("Template error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StudentsHandler shows full students page
func (h *AdminHandler) StudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	breadcrumbs := []map[string]string{
		{"Title": h.localizer.T(lang, "navigation.dashboard"), "URL": "/admin"},
		{"Title": h.localizer.T(lang, "navigation.students"), "URL": ""},
	}

	data := map[string]interface{}{
		"Title":       h.localizer.T(lang, "student_management.title"),
		"User":        user,
		"Lang":        lang,
		"CurrentYear": time.Now().Year(),
		"Breadcrumbs": breadcrumbs,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "students.html", data); err != nil {
		fmt.Printf("Template error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StudentsSearchHandler handles student search
func (h *AdminHandler) StudentsSearchHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")

	query := `
		SELECT * FROM student_records 
		WHERE student_name LIKE ? 
		   OR student_lastname LIKE ?
		   OR student_email LIKE ?
		   OR student_number LIKE ?
		ORDER BY student_lastname
		LIMIT 20
	`

	searchParam := "%" + search + "%"
	var students []database.StudentRecord
	err := h.db.Select(&students, query, searchParam, searchParam, searchParam, searchParam)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"students": %d}`, len(students))
}

// TopicsHandler shows topics management page
func (h *AdminHandler) TopicsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	lang := i18n.GetLangFromContext(r.Context())

	data := map[string]interface{}{
		"Title": h.localizer.T(lang, "topic_management.title"),
		"User":  user,
		"Lang":  lang,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "topics.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ApproveTopicHandler handles topic approval
func (h *AdminHandler) ApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	topicID := chi.URLParam(r, "topicID")
	user := auth.GetUserFromContext(r.Context())

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Update topic status
	_, err := h.db.Exec(`
		UPDATE project_topic_registrations 
		SET status = 'approved', 
			approved_by = ?, 
			approved_at = UNIX_TIMESTAMP()
		WHERE id = ?
	`, user.Email, topicID)

	if err != nil {
		http.Error(w, "Failed to approve topic", http.StatusInternalServerError)
		return
	}

	// Return success response for HTMX
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-green-600"><i class="fas fa-check-circle"></i> Approved</span>`))
}

// ReportsHandler shows reports overview
func (h *AdminHandler) ReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	lang := i18n.GetLangFromContext(r.Context())

	breadcrumbs := []map[string]string{
		{"Title": h.localizer.T(lang, "navigation.dashboard"), "URL": "/admin"},
		{"Title": h.localizer.T(lang, "navigation.reports"), "URL": ""},
	}

	data := map[string]interface{}{
		"Title":       h.localizer.T(lang, "reports.title"),
		"User":        user,
		"Lang":        lang,
		"CurrentYear": time.Now().Year(),
		"Breadcrumbs": breadcrumbs,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "reports.html", data); err != nil {
		fmt.Printf("Template error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// CreateCommissionModalHandler returns commission creation modal
func (h *AdminHandler) CreateCommissionModalHandler(w http.ResponseWriter, r *http.Request) {
	lang := i18n.GetLangFromContext(r.Context())

	// Get departments
	var departments []string
	h.db.Select(&departments, "SELECT DISTINCT department FROM student_records ORDER BY department")

	// Get study programs
	var programs []string
	h.db.Select(&programs, "SELECT DISTINCT study_program FROM student_records ORDER BY study_program")

	// Get years
	currentYear := time.Now().Year()
	years := []int{currentYear - 1, currentYear, currentYear + 1}

	data := map[string]interface{}{
		"Departments": departments,
		"Programs":    programs,
		"Years":       years,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "commission_create_modal.html", data); err != nil {
		fmt.Printf("Template error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Helper method to get dashboard statistics
func (h *AdminHandler) getDashboardStats(department string) map[string]interface{} {
	stats := make(map[string]interface{})

	// Base query conditions
	whereClause := ""
	args := []interface{}{}
	if department != "" {
		whereClause = " WHERE sr.department = ?"
		args = append(args, department)
	}

	// Total students
	var totalStudents int
	h.db.Get(&totalStudents,
		"SELECT COUNT(*) FROM student_records sr"+whereClause,
		args...)
	stats["TotalStudents"] = totalStudents

	// Pending topics
	var pendingTopics int
	pendingQuery := "SELECT COUNT(*) FROM project_topic_registrations ptr " +
		"JOIN student_records sr ON ptr.student_record_id = sr.id " +
		"WHERE ptr.status = 'submitted'"
	if department != "" {
		pendingQuery += " AND sr.department = ?"
	}
	h.db.Get(&pendingTopics, pendingQuery, args...)
	stats["PendingTopics"] = pendingTopics

	// Approved topics
	var approvedTopics int
	approvedQuery := "SELECT COUNT(*) FROM project_topic_registrations ptr " +
		"JOIN student_records sr ON ptr.student_record_id = sr.id " +
		"WHERE ptr.status = 'approved'"
	if department != "" {
		approvedQuery += " AND sr.department = ?"
	}
	h.db.Get(&approvedTopics, approvedQuery, args...)
	stats["ApprovedTopics"] = approvedTopics

	// Completed reports
	var completedReports int
	query := `
		SELECT COUNT(DISTINCT sr.id) 
		FROM student_records sr
		JOIN supervisor_reports sup ON sr.id = sup.student_record_id
		JOIN reviewer_reports rev ON sr.id = rev.student_record_id
		WHERE sup.is_signed = 1 AND rev.is_signed = 1
	`
	if department != "" {
		query += " AND sr.department = ?"
	}
	h.db.Get(&completedReports, query, args...)
	stats["CompletedReports"] = completedReports

	// Average grade
	var averageGrade float64
	avgQuery := `
		SELECT COALESCE(AVG(rev.grade), 0)
		FROM reviewer_reports rev
		JOIN student_records sr ON rev.student_record_id = sr.id
		WHERE rev.is_signed = 1
	`
	if department != "" {
		avgQuery += " AND sr.department = ?"
	}
	h.db.Get(&averageGrade, avgQuery, args...)
	stats["AverageGrade"] = averageGrade

	// Additional rate calculations
	if totalStudents > 0 {
		stats["TopicApprovalRate"] = (approvedTopics * 100) / totalStudents

		// Calculate other rates
		var supervisorReports, reviewerReports int
		h.db.Get(&supervisorReports,
			"SELECT COUNT(*) FROM supervisor_reports sup "+
				"JOIN student_records sr ON sup.student_record_id = sr.id"+
				whereClause, args...)
		h.db.Get(&reviewerReports,
			"SELECT COUNT(*) FROM reviewer_reports rev "+
				"JOIN student_records sr ON rev.student_record_id = sr.id"+
				whereClause, args...)

		stats["SupervisorReportRate"] = (supervisorReports * 100) / totalStudents
		stats["ReviewerReportRate"] = (reviewerReports * 100) / totalStudents
	} else {
		stats["TopicApprovalRate"] = 0
		stats["SupervisorReportRate"] = 0
		stats["ReviewerReportRate"] = 0
	}

	// Chart data (placeholder)
	stats["ProgramLabels"] = []string{"PI", "IT", "KS", "MM"}
	stats["ProgramData"] = []int{25, 30, 20, 15}
	stats["GradeData"] = []int{5, 10, 15, 25, 20, 10}

	return stats
}

// Helper function to get environment variable
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
