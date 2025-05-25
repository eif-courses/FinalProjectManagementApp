// handlers/auth.go
package handlers

import (
	"net/http"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates" // Import your templ components
)

type AuthHandlers struct {
	authMiddleware *auth.AuthMiddleware
}

func NewAuthHandlers(authMiddleware *auth.AuthMiddleware) *AuthHandlers {
	return &AuthHandlers{
		authMiddleware: authMiddleware,
	}
}

func (h *AuthHandlers) ShowLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if user is already logged in
	if user := h.authMiddleware.GetUserFromSession(r); user != nil {
		// Redirect logged-in users to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	// Get error from query params if any
	errorMsg := r.URL.Query().Get("error")

	// Use the Templ component
	component := templates.LoginPage(errorMsg)
	component.Render(r.Context(), w)
}

func (h *AuthHandlers) ShowAccessDeniedPage(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Query().Get("message")
	if message == "" {
		message = "You don't have permission to access this resource."
	}

	component := templates.AccessDeniedPage(message)
	component.Render(r.Context(), w)
}

func (h *AuthHandlers) ShowLogoutPage(w http.ResponseWriter, r *http.Request) {
	component := templates.LogoutConfirmationPage()
	component.Render(r.Context(), w)
}
