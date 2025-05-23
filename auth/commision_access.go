// auth/commission_access.go - Updated for Chi router
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// ... (keep all the existing structs and types from previous version)

// CommissionAccessMiddleware for Chi router - extracts access code from URL
func (cas *CommissionAccessService) CommissionAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract access code from Chi URL parameter or path
		accessCode := chi.URLParam(r, "accessCode")

		// If not found in URL params, try to extract from path
		if accessCode == "" {
			path := strings.TrimPrefix(r.URL.Path, "/commission/")
			parts := strings.Split(path, "/")
			if len(parts) > 0 && parts[0] != "" {
				accessCode = parts[0]
			}
		}

		if accessCode == "" {
			cas.renderAccessError(w, "Access code is required")
			return
		}

		// Validate access code
		access, err := cas.ValidateAccess(r.Context(), accessCode)
		if err != nil {
			cas.renderAccessError(w, "Invalid or expired access code")
			return
		}

		// Add access info to context
		ctx := context.WithValue(r.Context(), "commission_access", access)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// renderAccessError renders an error page for commission access issues
func (cas *CommissionAccessService) renderAccessError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusForbidden)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Access Error</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center">
    <div class="max-w-md mx-auto bg-white rounded-lg shadow-lg p-8 text-center">
        <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 mb-4">
            <svg class="h-6 w-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.99-.833-2.732 0L3.732 16.