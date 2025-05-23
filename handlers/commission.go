// handlers/commission.go - Fixed commission handlers
package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
)

// CommissionHandler handles commission-related requests
type CommissionHandler struct {
	commissionService *auth.CommissionAccessService
	authMiddleware    *auth.AuthMiddleware
	baseURL           string
}

// NewCommissionHandler creates a new commission handler
func NewCommissionHandler(commissionService *auth.CommissionAccessService, authMiddleware *auth.AuthMiddleware, baseURL string) *CommissionHandler {
	return &CommissionHandler{
		commissionService: commissionService,
		authMiddleware:    authMiddleware,
		baseURL:           baseURL,
	}
}

// ListCommissionAccessesHandler shows all commission accesses
func (ch *CommissionHandler) ListCommissionAccessesHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	// Get commission accesses (empty string means get all for admins)
	createdBy := ""
	if !user.IsAdmin() {
		createdBy = user.Email
	}

	accesses, err := ch.commissionService.ListAccesses(r.Context(), createdBy)
	if err != nil {
		http.Error(w, "Failed to load commission accesses", http.StatusInternalServerError)
		return
	}

	// Render template
	ch.renderTemplate(w, "commission_list.html", map[string]interface{}{
		"Title":    "Commission Access Management",
		"User":     user,
		"Accesses": accesses,
		"BaseURL":  ch.baseURL,
	})
}

// CreateCommissionAccessHandler handles creation of new commission access
func (ch *CommissionHandler) CreateCommissionAccessHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Show creation form
		ch.renderTemplate(w, "commission_create.html", map[string]interface{}{
			"Title": "Create Commission Access",
			"User":  user,
		})
		return
	}

	// Handle POST request
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Extract form data
	department := r.FormValue("department")
	studyProgram := r.FormValue("study_program")
	description := r.FormValue("description")
	yearStr := r.FormValue("year")
	expiresStr := r.FormValue("expires_at")
	maxAccessStr := r.FormValue("max_access")

	// Validate required fields
	if department == "" {
		http.Error(w, "Department is required", http.StatusBadRequest)
		return
	}

	// Parse year
	var year int
	if yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year", http.StatusBadRequest)
			return
		}
	}

	// Parse expiration date
	var expiresAt int64
	if expiresStr != "" {
		expiresTime, err := time.Parse("2006-01-02T15:04", expiresStr)
		if err != nil {
			http.Error(w, "Invalid expiration date", http.StatusBadRequest)
			return
		}
		expiresAt = expiresTime.Unix()
	} else {
		// Default to 30 days from now
		expiresAt = time.Now().AddDate(0, 0, 30).Unix()
	}

	// Parse max access
	var maxAccess int
	if maxAccessStr != "" {
		var err error
		maxAccess, err = strconv.Atoi(maxAccessStr)
		if err != nil {
			http.Error(w, "Invalid max access count", http.StatusBadRequest)
			return
		}
	}

	// Create access
	access, err := ch.commissionService.CreateAccess(
		r.Context(),
		department,
		studyProgram,
		description,
		user.Email,
		year,
		expiresAt,
		maxAccess,
	)
	if err != nil {
		http.Error(w, "Failed to create commission access: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response for HTMX or redirect for regular form
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"access_code": access.AccessCode,
			"access_url":  ch.baseURL + "/commission/" + access.AccessCode,
		})
		return
	}

	// Regular form submission - redirect to list
	http.Redirect(w, r, "/admin/commission?created="+access.AccessCode, http.StatusFound)
}

// DeactivateAccessHandler deactivates a commission access
func (ch *CommissionHandler) DeactivateAccessHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accessCode := chi.URLParam(r, "accessCode")
	if accessCode == "" {
		http.Error(w, "Access code is required", http.StatusBadRequest)
		return
	}

	// Check if user can deactivate this access
	access, err := ch.commissionService.GetAccess(r.Context(), accessCode)
	if err != nil {
		http.Error(w, "Access not found", http.StatusNotFound)
		return
	}

	// Only allow admin or creator to deactivate
	if !user.IsAdmin() && access.CreatedBy != user.Email {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Deactivate access
	if err := ch.commissionService.DeactivateAccess(r.Context(), accessCode); err != nil {
		http.Error(w, "Failed to deactivate access", http.StatusInternalServerError)
		return
	}

	// Return appropriate response
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Access deactivated successfully",
		})
		return
	}

	http.Redirect(w, r, "/admin/commission?deactivated="+accessCode, http.StatusFound)
}

// CommissionViewHandler handles public commission access views
func (ch *CommissionHandler) CommissionViewHandler(w http.ResponseWriter, r *http.Request) {
	// Get access from context (set by middleware)
	access := auth.GetCommissionAccessFromContext(r.Context())
	if access == nil {
		http.Error(w, "Invalid access", http.StatusForbidden)
		return
	}

	// Get URL path to determine what to show
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/commission/"+access.AccessCode), "/")

	// Default to student list if no specific path
	if len(pathParts) <= 1 || pathParts[1] == "" {
		ch.showCommissionStudentList(w, r, access)
		return
	}

	// Handle different views
	switch pathParts[1] {
	case "student":
		if len(pathParts) > 2 {
			ch.showCommissionStudentDetails(w, r, access, pathParts[2])
		} else {
			ch.showCommissionStudentList(w, r, access)
		}
	case "reports":
		ch.showCommissionReports(w, r, access)
	case "statistics":
		ch.showCommissionStatistics(w, r, access)
	default:
		ch.showCommissionStudentList(w, r, access)
	}
}

// showCommissionStudentList shows list of students for commission
func (ch *CommissionHandler) showCommissionStudentList(w http.ResponseWriter, r *http.Request, access *auth.CommissionAccess) {
	// Here you would fetch students based on the commission access criteria
	// For now, we'll create a mock response
	students := ch.getStudentsForCommission(r.Context(), access)

	ch.renderCommissionTemplate(w, "commission_students.html", map[string]interface{}{
		"Title":    "Defense Commission - Student List",
		"Access":   access,
		"Students": students,
	})
}

// showCommissionStudentDetails shows details for a specific student
func (ch *CommissionHandler) showCommissionStudentDetails(w http.ResponseWriter, r *http.Request, access *auth.CommissionAccess, studentID string) {
	// Fetch student details
	student := ch.getStudentDetails(r.Context(), studentID, access)
	if student == nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	ch.renderCommissionTemplate(w, "commission_student_details.html", map[string]interface{}{
		"Title":   "Defense Commission - Student Details",
		"Access":  access,
		"Student": student,
	})
}

// showCommissionReports shows reports for commission
func (ch *CommissionHandler) showCommissionReports(w http.ResponseWriter, r *http.Request, access *auth.CommissionAccess) {
	reports := ch.getReportsForCommission(r.Context(), access)

	ch.renderCommissionTemplate(w, "commission_reports.html", map[string]interface{}{
		"Title":   "Defense Commission - Reports",
		"Access":  access,
		"Reports": reports,
	})
}

// showCommissionStatistics shows statistics for commission
func (ch *CommissionHandler) showCommissionStatistics(w http.ResponseWriter, r *http.Request, access *auth.CommissionAccess) {
	stats := ch.getStatisticsForCommission(r.Context(), access)

	ch.renderCommissionTemplate(w, "commission_statistics.html", map[string]interface{}{
		"Title":      "Defense Commission - Statistics",
		"Access":     access,
		"Statistics": stats,
	})
}

// Helper methods for data retrieval

// getStudentsForCommission retrieves students based on commission access criteria
func (ch *CommissionHandler) getStudentsForCommission(ctx context.Context, access *auth.CommissionAccess) []database.CommissionStudentView {
	// This is a placeholder - implement actual database query
	// You would typically filter by department, study program, year, etc.
	return []database.CommissionStudentView{
		{
			ID:                  1,
			StudentName:         "Jonas Petraitis",
			StudentGroup:        "PIVTfm-22",
			ProjectTitle:        "Web aplikacijos kūrimas naudojant Go",
			ProjectTitleEn:      "Web Application Development using Go",
			SupervisorName:      "Dr. Rasa Kazlauskienė",
			StudyProgram:        "Programų inžinerija",
			HasSupervisorReport: true,
			HasReviewerReport:   true,
			HasVideo:            true,
			TopicApproved:       true,
			ReviewerGrade:       floatPtr(8.5),
		},
		{
			ID:                  2,
			StudentName:         "Marija Jonaitienė",
			StudentGroup:        "PIVTfm-22",
			ProjectTitle:        "Mobiliosios aplikacijos duomenų saugumas",
			ProjectTitleEn:      "Mobile Application Data Security",
			SupervisorName:      "Prof. Maksim Gžegožewski",
			StudyProgram:        "Programų inžinerija",
			HasSupervisorReport: true,
			HasReviewerReport:   false,
			HasVideo:            true,
			TopicApproved:       true,
			ReviewerGrade:       nil,
		},
	}
}

// getStudentDetails retrieves detailed information for a specific student
func (ch *CommissionHandler) getStudentDetails(ctx context.Context, studentID string, access *auth.CommissionAccess) *database.StudentSummaryView {
	// This is a placeholder - implement actual database query
	return &database.StudentSummaryView{
		StudentRecord: database.StudentRecord{
			ID:                  1,
			StudentGroup:        "PIVTfm-22",
			FinalProjectTitle:   "Web aplikacijos kūrimas naudojant Go",
			FinalProjectTitleEn: "Web Application Development using Go",
			StudentEmail:        "jonas.petraitis@stud.viko.lt",
			StudentName:         "Jonas",
			StudentLastname:     "Petraitis",
			StudentNumber:       "20220001",
			SupervisorEmail:     "r.kazlauskiene@viko.lt",
			StudyProgram:        "Programų inžinerija",
			Department:          "Informacijos technologijų katedra",
			ProgramCode:         "PIVT",
			CurrentYear:         2024,
			ReviewerEmail:       "reviewer@viko.lt",
			ReviewerName:        "Dr. Antanas Petrauskas",
		},
		TopicApproved:          true,
		TopicStatus:            "approved",
		HasSupervisorReport:    true,
		HasReviewerReport:      true,
		HasVideo:               true,
		SupervisorReportSigned: true,
		ReviewerReportSigned:   true,
		ReviewerGrade:          floatPtr(8.5),
	}
}

// getReportsForCommission retrieves reports for commission access
func (ch *CommissionHandler) getReportsForCommission(ctx context.Context, access *auth.CommissionAccess) []database.ReportWithDetails {
	// This is a placeholder - implement actual database query
	return []database.ReportWithDetails{
		{
			StudentID:           1,
			StudentName:         "Jonas Petraitis",
			StudentEmail:        "jonas.petraitis@stud.viko.lt",
			StudentGroup:        "PIVTfm-22",
			StudyProgram:        "Programų inžinerija",
			ProjectTitle:        "Web aplikacijos kūrimas naudojant Go",
			HasSupervisorReport: true,
			HasReviewerReport:   true,
			BothReportsSigned:   true,
		},
	}
}

// getStatisticsForCommission retrieves statistics for commission
func (ch *CommissionHandler) getStatisticsForCommission(ctx context.Context, access *auth.CommissionAccess) database.DashboardStats {
	// This is a placeholder - implement actual database query
	return database.DashboardStats{
		TotalStudents:      15,
		ActiveStudents:     15,
		StudentsWithTopics: 15,
		TotalTopics:        15,
		PendingTopics:      0,
		ApprovedTopics:     15,
		RejectedTopics:     0,
		SupervisorReports:  14,
		ReviewerReports:    12,
		SignedReports:      10,
		AverageGrade:       7.8,
		PassRate:           93.3,
	}
}

// Template rendering methods

// renderTemplate renders a regular template
func (ch *CommissionHandler) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	// This is a placeholder implementation
	// In production, you'd use your actual template engine
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Simple template rendering for demonstration
	tmplStr := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <h1 class="text-3xl font-bold mb-6">{{.Title}}</h1>
        <div class="bg-white rounded-lg shadow p-6">
            <pre>{{printf "%+v" .}}</pre>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New(templateName).Parse(tmplStr)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

// renderCommissionTemplate renders a commission-specific template
func (ch *CommissionHandler) renderCommissionTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Commission-specific template with navigation
	tmplStr := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{.Title}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <!-- Commission Navigation -->
    <nav class="bg-blue-600 text-white p-4">
        <div class="container mx-auto flex justify-between items-center">
            <h1 class="text-xl font-bold">Defense Commission Portal</h1>
            <div class="space-x-4">
                <a href="/commission/{{.Access.AccessCode}}" class="hover:underline">Students</a>
                <a href="/commission/{{.Access.AccessCode}}/reports" class="hover:underline">Reports</a>
                <a href="/commission/{{.Access.AccessCode}}/statistics" class="hover:underline">Statistics</a>
            </div>
        </div>
    </nav>

    <!-- Commission Info -->
    <div class="bg-white shadow-sm border-b">
        <div class="container mx-auto px-4 py-3">
            <div class="flex justify-between items-center text-sm text-gray-600">
                <div>
                    <span class="font-medium">Department:</span> {{.Access.Department}}
                    {{if .Access.StudyProgram}}
                    <span class="ml-4"><span class="font-medium">Program:</span> {{.Access.StudyProgram}}</span>
                    {{end}}
                    {{if .Access.Year}}
                    <span class="ml-4"><span class="font-medium">Year:</span> {{.Access.Year}}</span>
                    {{end}}
                </div>
                <div>
                    <span class="font-medium">Expires:</span> 
                    <time>{{formatTime .Access.ExpiresAt}}</time>
                </div>
            </div>
        </div>
    </div>

    <!-- Content -->
    <div class="container mx-auto px-4 py-8">
        <h1 class="text-3xl font-bold mb-6">{{.Title}}</h1>
        <div class="bg-white rounded-lg shadow p-6">
            {{if eq .Title "Defense Commission - Student List"}}
                {{template "studentList" .}}
            {{else if eq .Title "Defense Commission - Student Details"}}
                {{template "studentDetails" .}}
            {{else if eq .Title "Defense Commission - Reports"}}
                {{template "reports" .}}
            {{else if eq .Title "Defense Commission - Statistics"}}
                {{template "statistics" .}}
            {{else}}
                <pre>{{printf "%+v" .}}</pre>
            {{end}}
        </div>
    </div>
</body>
</html>

{{define "studentList"}}
<div class="overflow-x-auto">
    <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
            <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Student</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Project Title</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Supervisor</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Grade</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
            {{range .Students}}
            <tr>
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-sm font-medium text-gray-900">{{.StudentName}}</div>
                    <div class="text-sm text-gray-500">{{.StudentGroup}}</div>
                </td>
                <td class="px-6 py-4">
                    <div class="text-sm text-gray-900">{{.ProjectTitle}}</div>
                    <div class="text-sm text-gray-500">{{.ProjectTitleEn}}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.SupervisorName}}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="flex space-x-1">
                        {{if .TopicApproved}}<span class="px-2 py-1 text-xs bg-green-100 text-green-800 rounded">Topic Approved</span>{{end}}
                        {{if .HasSupervisorReport}}<span class="px-2 py-1 text-xs bg-blue-100 text-blue-800 rounded">Supervisor Report</span>{{end}}
                        {{if .HasReviewerReport}}<span class="px-2 py-1 text-xs bg-purple-100 text-purple-800 rounded">Reviewer Report</span>{{end}}
                        {{if .HasVideo}}<span class="px-2 py-1 text-xs bg-yellow-100 text-yellow-800 rounded">Video</span>{{end}}
                    </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {{if .ReviewerGrade}}{{printf "%.1f" .ReviewerGrade}}{{else}}-{{end}}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <a href="/commission/{{$.Access.AccessCode}}/student/{{.ID}}" class="text-indigo-600 hover:text-indigo-900">View Details</a>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>
{{end}}

{{define "studentDetails"}}
<div class="space-y-6">
    <div class="bg-gray-50 p-4 rounded-lg">
        <h3 class="text-lg font-medium mb-2">Student Information</h3>
        <div class="grid grid-cols-2 gap-4 text-sm">
            <div><span class="font-medium">Name:</span> {{.Student.GetFullName}}</div>
            <div><span class="font-medium">Email:</span> {{.Student.StudentEmail}}</div>
            <div><span class="font-medium">Student Number:</span> {{.Student.StudentNumber}}</div>
            <div><span class="font-medium">Group:</span> {{.Student.StudentGroup}}</div>
            <div><span class="font-medium">Program:</span> {{.Student.StudyProgram}}</div>
            <div><span class="font-medium">Department:</span> {{.Student.Department}}</div>
        </div>
    </div>

    <div class="bg-gray-50 p-4 rounded-lg">
        <h3 class="text-lg font-medium mb-2">Project Information</h3>
        <div class="space-y-2 text-sm">
            <div><span class="font-medium">Title (LT):</span> {{.Student.FinalProjectTitle}}</div>
            <div><span class="font-medium">Title (EN):</span> {{.Student.FinalProjectTitleEn}}</div>
            <div><span class="font-medium">Supervisor:</span> {{.Student.SupervisorEmail}}</div>
            <div><span class="font-medium">Reviewer:</span> {{.Student.ReviewerName}} ({{.Student.ReviewerEmail}})</div>
        </div>
    </div>

    <div class="bg-gray-50 p-4 rounded-lg">
        <h3 class="text-lg font-medium mb-2">Status</h3>
        <div class="grid grid-cols-2 gap-4 text-sm">
            <div><span class="font-medium">Topic Status:</span> 
                <span class="{{if .Student.TopicApproved}}text-green-600{{else}}text-red-600{{end}}">
                    {{if .Student.TopicApproved}}Approved{{else}}Not Approved{{end}}
                </span>
            </div>
            <div><span class="font-medium">Completion:</span> {{.Student.GetCompletionPercentage}}%</div>
            <div><span class="font-medium">Supervisor Report:</span> 
                <span class="{{if .Student.HasSupervisorReport}}text-green-600{{else}}text-red-600{{end}}">
                    {{if .Student.HasSupervisorReport}}Submitted{{else}}Missing{{end}}
                </span>
            </div>
            <div><span class="font-medium">Reviewer Report:</span> 
                <span class="{{if .Student.HasReviewerReport}}text-green-600{{else}}text-red-600{{end}}">
                    {{if .Student.HasReviewerReport}}Submitted{{else}}Missing{{end}}
                </span>
            </div>
            <div><span class="font-medium">Video Presentation:</span> 
                <span class="{{if .Student.HasVideo}}text-green-600{{else}}text-red-600{{end}}">
                    {{if .Student.HasVideo}}Available{{else}}Missing{{end}}
                </span>
            </div>
            <div><span class="font-medium">Reviewer Grade:</span> 
                {{if .Student.ReviewerGrade}}{{printf "%.1f" .Student.ReviewerGrade}}{{else}}-{{end}}
            </div>
        </div>
    </div>
</div>
{{end}}

{{define "reports"}}
<p>Reports view - implement based on your needs</p>
{{end}}

{{define "statistics"}}
<div class="grid grid-cols-2 md:grid-cols-4 gap-6">
    <div class="bg-blue-50 p-4 rounded-lg">
        <div class="text-2xl font-bold text-blue-600">{{.Statistics.TotalStudents}}</div>
        <div class="text-sm text-gray-600">Total Students</div>
    </div>
    <div class="bg-green-50 p-4 rounded-lg">
        <div class="text-2xl font-bold text-green-600">{{.Statistics.ApprovedTopics}}</div>
        <div class="text-sm text-gray-600">Approved Topics</div>
    </div>
    <div class="bg-purple-50 p-4 rounded-lg">
        <div class="text-2xl font-bold text-purple-600">{{.Statistics.SignedReports}}</div>
        <div class="text-sm text-gray-600">Signed Reports</div>
    </div>
    <div class="bg-yellow-50 p-4 rounded-lg">
        <div class="text-2xl font-bold text-yellow-600">{{printf "%.1f" .Statistics.AverageGrade}}</div>
        <div class="text-sm text-gray-600">Average Grade</div>
    </div>
</div>
{{end}}`

	// Add helper functions
	funcMap := template.FuncMap{
		"formatTime": func(timestamp int64) string {
			return time.Unix(timestamp, 0).Format("2006-01-02 15:04")
		},
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		return
	}
}

// Helper function to create float pointer
func floatPtr(f float64) *float64 {
	return &f
}
