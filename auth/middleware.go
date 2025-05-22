// auth/middleware.go
package auth

import (
	"context"
	"encoding/json"
	"net/http"
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
func NewAuthMiddleware(authService *AuthService) *AuthMiddleware {
	// Create session store with secure settings
	store := sessions.NewCookieStore([]byte("your-secret-key-change-this-in-production"))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}

	return &AuthMiddleware{
		authService:  authService,
		sessionStore: store,
	}
}

// RequireAuth middleware that requires user to be authenticated
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := am.GetUserFromSession(r)
		if user == nil {
			// Redirect to login
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware that requires specific role
func (am *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := am.GetUserFromSession(r)
			if user == nil {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			// Check if user has required role
			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				// Removed unused lang variable - just return error directly
				http.Error(w, "Access denied", http.StatusForbidden)
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
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			if !user.HasPermission(permission) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoginHandler handles the login initiation
func (am *AuthMiddleware) LoginHandler(w http.ResponseWriter, r *http.Request) {
	loginURL := am.authService.GenerateLoginURL()
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// CallbackHandler handles the OAuth callback
func (am *AuthMiddleware) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		http.Error(w, "Authorization code not found", http.StatusBadRequest)
		return
	}

	// Exchange code for user info
	user, err := am.authService.HandleCallback(code, state)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save user to session
	if err := am.SaveUserToSession(w, r, user); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect to dashboard
	http.Redirect(w, r, "/", http.StatusFound)
}

// LogoutHandler handles user logout
func (am *AuthMiddleware) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := am.sessionStore.Get(r, SessionName)
	session.Options.MaxAge = -1 // Delete session
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

// SaveUserToSession saves authenticated user to session
func (am *AuthMiddleware) SaveUserToSession(w http.ResponseWriter, r *http.Request, user *AuthenticatedUser) error {
	session, _ := am.sessionStore.Get(r, SessionName)

	userData, err := json.Marshal(user)
	if err != nil {
		return err
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
		return nil
	}

	userData, ok := session.Values["user"].([]byte)
	if !ok {
		return nil
	}

	var user AuthenticatedUser
	if err := json.Unmarshal(userData, &user); err != nil {
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
