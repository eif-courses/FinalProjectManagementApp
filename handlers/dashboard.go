package handlers

import (
	"FinalProjectManagementApp/components/templates"
	"net/http"

	"FinalProjectManagementApp/auth"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get locale from query parameter or cookie
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = getLocaleFromCookie(r)
	}
	if locale == "" {
		locale = "lt" // default
	}

	// Set locale cookie if changed
	if r.URL.Query().Get("locale") != "" {
		setLocaleCookie(w, locale)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := templates.Layout(user, locale, "Dashboard").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
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
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// Placeholder handlers for other routes
func StudentProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Profile - Coming Soon"))
}

func StudentTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Topic - Coming Soon"))
}

func SubmitTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Submit Topic - Coming Soon"))
}

func SupervisorDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Supervisor Dashboard - Coming Soon"))
}

func SupervisorStudentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Supervisor Students - Coming Soon"))
}

func SupervisorReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Supervisor Reports - Coming Soon"))
}

func DepartmentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Department Dashboard - Coming Soon"))
}

func DepartmentStudentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Department Students - Coming Soon"))
}

func PendingTopicsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pending Topics - Coming Soon"))
}

func ApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Approve Topic - Coming Soon"))
}

func RejectTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Reject Topic - Coming Soon"))
}

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Admin Dashboard - Coming Soon"))
}

func AdminUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Admin Users - Coming Soon"))
}

func AuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Audit Logs - Coming Soon"))
}
