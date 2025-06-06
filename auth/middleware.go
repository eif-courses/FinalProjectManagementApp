// auth/middleware.go - Fixed version with HTMX support
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
	SessionName               = "thesis-session"
)

type AuthMiddleware struct {
	authService  *AuthService
	sessionStore sessions.Store
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *AuthService) (*AuthMiddleware, error) {
	// Get session secret from environment
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "your-secret-key-change-this-in-production"
	}

	// Create session store with secure settings
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production", // Only secure in production
		SameSite: http.SameSiteLaxMode,
	}

	return &AuthMiddleware{
		authService:  authService,
		sessionStore: store,
	}, nil
}

// RequireAuth middleware that requires user to be authenticated
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("RequireAuth middleware called for: %s %s", r.Method, r.URL.Path)

		user := am.GetUserFromSession(r)
		if user == nil {
			log.Printf("No user found in session for: %s", r.URL.Path)

			// Handle HTMX requests differently
			if r.Header.Get("HX-Request") == "true" {
				log.Printf("HTMX request detected, sending HX-Redirect header")
				w.Header().Set("HX-Redirect", "/auth/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Store the original URL for redirect after login
			if r.URL.Path != "/auth/login" {
				session, _ := am.sessionStore.Get(r, SessionName)
				session.Values["redirect_url"] = r.URL.String()
				session.Save(r, w)
			}

			// Redirect to login
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		log.Printf("User authenticated: %s (%s)", user.Email, user.Role)

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware that requires specific roles
func (am *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := am.GetUserFromSession(r)
			if user == nil {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				log.Printf("User %s does not have required role. Has: %s, Required: %v", user.Email, user.Role, roles)
				if r.Header.Get("HX-Request") == "true" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Access denied: insufficient permissions"))
					return
				}
				http.Error(w, "Access denied: insufficient permissions", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission middleware that requires specific permission
func (am *AuthMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := am.GetUserFromSession(r)
			if user == nil {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			if !user.HasPermission(permission) {
				if r.Header.Get("HX-Request") == "true" {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Access denied: insufficient permissions"))
					return
				}
				http.Error(w, "Access denied: insufficient permissions", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Rest of your existing methods remain the same...
func (am *AuthMiddleware) LoginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("LoginHandler called")

	// Check if user is already logged in
	if user := am.GetUserFromSession(r); user != nil {
		log.Printf("User already logged in: %s", user.Email)
		// Redirect to dashboard instead of home page
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	log.Printf("Generating login URL...")
	loginURL, err := am.authService.GenerateLoginURL()
	if err != nil {
		log.Printf("Failed to generate login URL: %v", err)
		http.Redirect(w, r, "/?error=Failed to generate login URL", http.StatusSeeOther)
		return
	}

	log.Printf("Generated login URL: %s", loginURL)
	log.Printf("Redirecting to Microsoft OAuth...")
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// CallbackHandler handles the OAuth callback
func (am *AuthMiddleware) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	// Check for OAuth error
	if errorParam != "" {
		errorDescription := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("OAuth error: %s - %s", errorParam, errorDescription), http.StatusBadRequest)
		return
	}

	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	// Exchange code for user info
	user, err := am.authService.HandleCallback(r.Context(), code, state)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save user to session
	if err := am.SaveUserToSession(w, r, user); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Get redirect URL and clear it from session
	redirectURL := am.getRedirectURL(r)
	am.clearRedirectURL(w, r)

	// Redirect to dashboard or original URL
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// LogoutHandler handles user logout
func (am *AuthMiddleware) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := am.sessionStore.Get(r, SessionName)

	// Clear all session values
	for key := range session.Values {
		delete(session.Values, key)
	}

	// Set session to expire immediately
	session.Options.MaxAge = -1
	session.Save(r, w)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusFound)
}

// SaveUserToSession saves authenticated user to session
func (am *AuthMiddleware) SaveUserToSession(w http.ResponseWriter, r *http.Request, user *AuthenticatedUser) error {
	session, err := am.sessionStore.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	session.Values["user"] = userData
	session.Values["authenticated"] = true
	session.Values["login_time"] = time.Now().Unix()

	return session.Save(r, w)
}

// GetUserFromSession retrieves user from session
func (am *AuthMiddleware) GetUserFromSession(r *http.Request) *AuthenticatedUser {
	session, err := am.sessionStore.Get(r, SessionName)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		return nil
	}

	userData, ok := session.Values["user"].([]byte)
	if !ok {
		log.Printf("No user data in session")
		return nil
	}

	var user AuthenticatedUser
	if err := json.Unmarshal(userData, &user); err != nil {
		log.Printf("Failed to unmarshal user data: %v", err)
		return nil
	}

	return &user
}

// GetUserFromContext gets user from request context
func GetUserFromContext(ctx context.Context) *AuthenticatedUser {
	if user, ok := ctx.Value(UserContextKey).(*AuthenticatedUser); ok {
		return user
	}
	return nil
}

// IsAuthenticated checks if request has authenticated user
func (am *AuthMiddleware) IsAuthenticated(r *http.Request) bool {
	return am.GetUserFromSession(r) != nil
}

// UserInfoHandler returns user info as JSON (for API calls)
func (am *AuthMiddleware) UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	user := am.GetUserFromSession(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Helper methods
func (am *AuthMiddleware) getRedirectURL(r *http.Request) string {
	session, err := am.sessionStore.Get(r, SessionName)
	if err != nil {
		return "/dashboard"
	}

	if redirectURL, ok := session.Values["redirect_url"].(string); ok && redirectURL != "" {
		return redirectURL
	}

	return "/dashboard"
}

func (am *AuthMiddleware) clearRedirectURL(w http.ResponseWriter, r *http.Request) {
	session, _ := am.sessionStore.Get(r, SessionName)
	delete(session.Values, "redirect_url")
	session.Save(r, w)
}
