package handlers

import (
	"FinalProjectManagementApp/auth"
	"github.com/jmoiron/sqlx"
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
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get and set locale
	locale := h.getLocale(r, w)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Route to appropriate dashboard based on role
	switch user.Role {
	case auth.RoleStudent:
		h.renderStudentDashboard(w, r, user, locale)
	case auth.RoleSupervisor:
		h.renderSupervisorDashboard(w, r, user, locale)
	case auth.RoleDepartmentHead:
		h.renderDepartmentDashboard(w, r, user, locale)
	case auth.RoleAdmin:
		h.renderAdminDashboard(w, r, user, locale)
	default:
		http.Error(w, "Unauthorized", http.StatusForbidden)
	}
}

func (h *DashboardHandlers) renderStudentDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get student-specific data
	//data := h.getStudentDashboardData(user.Email)
	//
	//err := templates.StudentDashboard(user, data, locale).Render(r.Context(), w)
	//if err != nil {
	//	http.Error(w, "Failed to render template", http.StatusInternalServerError)
	//}
}

func (h *DashboardHandlers) renderSupervisorDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get supervisor-specific data
	//data := h.getSupervisorDashboardData(user.Email)
	//
	//err := templates.SupervisorDashboard(user, data, locale).Render(r.Context(), w)
	//if err != nil {
	//	http.Error(w, "Failed to render template", http.StatusInternalServerError)
	//}
}

func (h *DashboardHandlers) renderDepartmentDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get department-specific data
	//data := h.getDepartmentDashboardData(user.Email)
	//
	//err := templates.DepartmentDashboard(user, data, locale).Render(r.Context(), w)
	//if err != nil {
	//	http.Error(w, "Failed to render template", http.StatusInternalServerError)
	//}
}

func (h *DashboardHandlers) renderAdminDashboard(w http.ResponseWriter, r *http.Request, user *auth.AuthenticatedUser, locale string) {
	// Get admin-specific data
	//data := h.getAdminDashboardData()
	//
	//err := templates.AdminDashboard(user, data, locale).Render(r.Context(), w)
	//if err != nil {
	//	http.Error(w, "Failed to render template", http.StatusInternalServerError)
	//}
}

// Locale handling
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
