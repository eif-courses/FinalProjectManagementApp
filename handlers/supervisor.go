// handlers/supervisor.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/i18n"
	"html/template"
	"net/http"
)

// SupervisorStudentsHandler shows students assigned to supervisor
func SupervisorStudentsHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		students := getStudentsBySupervisor(user.Email)

		data := localizer.NewTemplateData(
			r.Context(),
			"my_students",
			user,
			map[string]interface{}{
				"Students": students,
			},
		)

		RenderTemplateWithI18n(w, tmpl, "supervisor-students.html", data)
	}
}

// CreateSupervisorReportHandler handles supervisor report creation
func CreateSupervisorReportHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// FIXED: Remove unused user variable
		lang := i18n.GetLangFromContext(r.Context())

		if r.Method == "POST" {
			// Handle report creation
			_ = r.FormValue("student_id") // FIXED: Use underscore for unused variables
			_ = r.FormValue("comments")
			_ = r.FormValue("grade")

			// In real implementation, save to database
			// For now, just return success

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "success", "message": "` + localizer.T(lang, "report_created") + `"}`))
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ViewSupervisorReportHandler shows supervisor report details
func ViewSupervisorReportHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		// Get report ID from URL or form
		reportID := r.URL.Query().Get("id")

		data := localizer.NewTemplateData(
			r.Context(),
			"supervisor_report",
			user,
			map[string]interface{}{
				"Report": getSupervisorReport(reportID),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "supervisor-report.html", data)
	}
}

// Helper functions for supervisor operations
func getSupervisorReport(reportID string) map[string]interface{} {
	// In real implementation, query database
	return map[string]interface{}{
		"ID":             reportID,
		"StudentName":    "Mantas Gzegožveskis",
		"ProjectTitle":   "Thesis Management System",
		"SupervisorName": "Dr. Jonas Petraitis",
		"Comments":       "Excellent work on the project. Shows great understanding of the technologies.",
		"Grade":          10,
		"IsPassOrFailed": 1,
		"IsSigned":       1,
		"CreatedDate":    "2024-01-15",
		"OtherMatch":     0.15,
		"OneMatch":       0.05,
		"OwnMatch":       0.02,
		"JoinMatch":      0.03,
	}
}
