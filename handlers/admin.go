// handlers/admin.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/i18n"
	"html/template"
	"net/http"
)

// UserManagementHandler handles user management for admins
func UserManagementHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		data := localizer.NewTemplateData(
			r.Context(),
			"user_management",
			user,
			map[string]interface{}{
				"Users": getAllUsers(),
				"Roles": []string{"student", "supervisor", "department_head", "admin"},
			},
		)

		RenderTemplateWithI18n(w, tmpl, "user-management.html", data)
	}
}

// ApproveTopicHandler handles topic approval for department heads
func ApproveTopicHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		lang := i18n.GetLangFromContext(r.Context())

		if !user.CanApproveTopics() {
			http.Error(w, localizer.T(lang, "access_denied"), http.StatusForbidden)
			return
		}

		// FIXED: Use the variables or remove them
		_ = r.FormValue("topic_id")     // Use underscore to indicate intentionally unused
		action := r.FormValue("action") // "approve" or "reject"

		// In real implementation, update database
		// For now, just return success

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "message": "` + localizer.T(lang, "topic_"+action+"d") + `"}`))
	}
}

// SystemSettingsHandler handles system settings for admins
func SystemSettingsHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		data := localizer.NewTemplateData(
			r.Context(),
			"system_settings",
			user,
			map[string]interface{}{
				"Settings": getSystemSettings(),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "system-settings.html", data)
	}
}

// ReportsHandler generates various reports for admins
func ReportsHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		reportType := r.URL.Query().Get("type")

		data := localizer.NewTemplateData(
			r.Context(),
			"reports",
			user,
			map[string]interface{}{
				"ReportType": reportType,
				"ReportData": generateReport(reportType),
				"AvailableReports": []string{
					"student_progress",
					"supervisor_workload",
					"department_statistics",
					"thesis_submissions",
				},
			},
		)

		RenderTemplateWithI18n(w, tmpl, "reports.html", data)
	}
}

// Helper functions for admin operations
func getAllUsers() []map[string]interface{} {
	// In real implementation, query database
	return []map[string]interface{}{
		{
			"ID":         1,
			"Name":       "Dr. Jonas Petraitis",
			"Email":      "j.petraitis@viko.lt",
			"Role":       "department_head",
			"Department": "Information Systems",
			"Active":     true,
			"LastLogin":  "2024-01-20",
		},
		{
			"ID":         2,
			"Name":       "Mantas Gzegožveskis",
			"Email":      "m.gzegozevskis@eif.viko.lt",
			"Role":       "student",
			"Department": "Information Systems",
			"Active":     true,
			"LastLogin":  "2024-01-22",
		},
		{
			"ID":         3,
			"Name":       "Prof. Rima Kazlauskienė",
			"Email":      "r.kazlauskiene@viko.lt",
			"Role":       "supervisor",
			"Department": "Information Systems",
			"Active":     true,
			"LastLogin":  "2024-01-21",
		},
	}
}

func getSystemSettings() map[string]interface{} {
	// In real implementation, query database
	return map[string]interface{}{
		"MaxFileSize":        "10MB",
		"AllowedFileTypes":   []string{"pdf", "doc", "docx"},
		"SessionTimeout":     "7 days",
		"EnableRegistration": false,
		"RequireApproval":    true,
		"DefaultLanguage":    "lt",
		"SupportedLanguages": []string{"lt", "en"},
		"MaintenanceMode":    false,
	}
}

func generateReport(reportType string) map[string]interface{} {
	// In real implementation, query database and generate actual report
	switch reportType {
	case "student_progress":
		return map[string]interface{}{
			"TotalStudents":      47,
			"CompletedThesis":    8,
			"InProgress":         23,
			"NotStarted":         16,
			"OverdueSubmissions": 3,
		}
	case "supervisor_workload":
		return map[string]interface{}{
			"TotalSupervisors":             12,
			"AverageStudentsPerSupervisor": 4,
			"MaxStudentsPerSupervisor":     8,
			"MinStudentsPerSupervisor":     1,
		}
	case "department_statistics":
		return map[string]interface{}{
			"Departments": []map[string]interface{}{
				{"Name": "Information Systems", "Students": 25, "Completed": 5},
				{"Name": "Computer Networks", "Students": 15, "Completed": 2},
				{"Name": "Software Systems", "Students": 7, "Completed": 1},
			},
		}
	default:
		return map[string]interface{}{
			"Message": "Report type not found",
		}
	}
}
