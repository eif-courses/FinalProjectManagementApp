package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

type ReviewerAccessHandler struct {
	db *sqlx.DB
}

func NewReviewerAccessHandler(db *sqlx.DB) *ReviewerAccessHandler {
	return &ReviewerAccessHandler{db: db}
}

// Show management page for admins
func (h *ReviewerAccessHandler) ShowManagementPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || (user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	tokens, err := h.getActiveReviewerTokens()
	if err != nil {
		http.Error(w, "Failed to load tokens", http.StatusInternalServerError)
		return
	}

	reviewers, err := h.getAllReviewers()
	if err != nil {
		http.Error(w, "Failed to load reviewers", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = templates.ReviewerAccessManagement(tokens, reviewers, user).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// Create reviewer access token
func (h *ReviewerAccessHandler) CreateReviewerAccess(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || (user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	reviewerEmail := r.FormValue("reviewer_email")
	reviewerName := r.FormValue("reviewer_name")
	department := r.FormValue("department")
	daysValidStr := r.FormValue("days_valid")

	if reviewerEmail == "" || reviewerName == "" {
		http.Error(w, "Reviewer email and name are required", http.StatusBadRequest)
		return
	}

	daysValid, err := strconv.Atoi(daysValidStr)
	if err != nil || daysValid <= 0 {
		daysValid = 30 // Default 30 days
	}

	// Generate access token
	accessToken, err := h.generateAccessToken()
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// Create reviewer access token
	token := &database.ReviewerAccessToken{
		ReviewerEmail: reviewerEmail,
		ReviewerName:  reviewerName,
		AccessToken:   accessToken,
		Department:    department,
		CreatedAt:     time.Now().Unix(),
		ExpiresAt:     time.Now().AddDate(0, 0, daysValid).Unix(),
		IsActive:      true,
		CreatedBy:     user.Email,
	}

	err = h.createReviewerToken(token)
	if err != nil {
		http.Error(w, "Failed to create reviewer access", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"message":      "Reviewer access created successfully",
		"access_token": accessToken,
		"access_url":   fmt.Sprintf("/reviewer/%s", accessToken),
	})
}

// DeactivateReviewerAccess
func (h *ReviewerAccessHandler) DeactivateReviewerAccess(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || (user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	accessToken := chi.URLParam(r, "accessToken")

	query := `UPDATE reviewer_access_tokens SET is_active = false WHERE access_token = ?`
	_, err := h.db.Exec(query, accessToken)
	if err != nil {
		http.Error(w, "Failed to deactivate access", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Reviewer access deactivated successfully",
	})
}

// Database helper methods
func (h *ReviewerAccessHandler) generateAccessToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (h *ReviewerAccessHandler) createReviewerToken(token *database.ReviewerAccessToken) error {
	query := `
        INSERT INTO reviewer_access_tokens (
            reviewer_email, reviewer_name, access_token, department,
            created_at, expires_at, max_access, is_active, created_by
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	_, err := h.db.Exec(query,
		token.ReviewerEmail, token.ReviewerName, token.AccessToken,
		token.Department, token.CreatedAt, token.ExpiresAt,
		token.MaxAccess, token.IsActive, token.CreatedBy)

	return err
}

func (h *ReviewerAccessHandler) getActiveReviewerTokens() ([]database.ReviewerAccessToken, error) {
	query := `SELECT * FROM reviewer_access_tokens WHERE is_active = true ORDER BY created_at DESC`

	var tokens []database.ReviewerAccessToken
	err := h.db.Select(&tokens, query)
	return tokens, err
}

func (h *ReviewerAccessHandler) getAllReviewers() ([]string, error) {
	query := `SELECT DISTINCT reviewer_email FROM student_records WHERE reviewer_email IS NOT NULL AND reviewer_email != '' ORDER BY reviewer_email`

	var reviewers []string
	err := h.db.Select(&reviewers, query)
	return reviewers, err
}
