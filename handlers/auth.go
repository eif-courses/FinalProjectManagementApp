// handlers/auth.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/i18n"
	"html/template"
	"net/http"
)

// HomeHandlerWithI18n handles the home page (shows login if not authenticated, dashboard if authenticated)
func HomeHandlerWithI18n(tmpl *template.Template, localizer *i18n.Localizer, authMiddleware *auth.AuthMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := authMiddleware.GetUserFromSession(r)

		if user == nil {
			// Show login page
			data := localizer.NewTemplateData(
				r.Context(),
				"login",
				nil,
				map[string]interface{}{
					"LoginURL": "/auth/login",
				},
			)
			RenderTemplateWithI18n(w, tmpl, "login.html", data)
			return
		}

		// Redirect to dashboard if already authenticated
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

// DashboardHandlerWithI18n handles the dashboard with authentication
func DashboardHandlerWithI18n(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		data := localizer.NewTemplateData(
			r.Context(),
			"dashboard",
			user,
			map[string]interface{}{
				"Stats": getDashboardStats(user),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "dashboard.html", data)
	}
}

// Helper function for dashboard stats
func getDashboardStats(user *auth.AuthenticatedUser) map[string]interface{} {
	stats := make(map[string]interface{})

	switch user.Role {
	case "student":
		stats["topics"] = 1
		stats["documents"] = 3
		stats["status"] = "In Progress"

	case "supervisor":
		stats["assigned_students"] = len(getStudentsBySupervisor(user.Email))
		stats["pending_reviews"] = 2
		stats["completed_reviews"] = 8

	case "department_head":
		stats["total_students"] = 47
		stats["pending_approvals"] = 12
		stats["approved_topics"] = 23

	default:
		stats["message"] = "Welcome to the system"
	}

	return stats
}
