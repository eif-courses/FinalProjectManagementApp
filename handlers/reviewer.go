package handlers

import (
	"fmt"
	"net/http"
)

// handlers/reviewer.go
func (h *StudentListHandler) ReviewerStudentsList(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL
	//token := r.URL.Query().Get("token")
	//if token == "" {
	//	http.Error(w, "Access token required", http.StatusUnauthorized)
	//	return
	//}
	//
	//// Validate token and get reviewer email
	//reviewerEmail, err := h.validateReviewerToken(token)
	//if err != nil {
	//	http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
	//	return
	//}
	//
	//// Create a mock user for template rendering
	//user := &auth.AuthenticatedUser{
	//	Email: reviewerEmail,
	//	Role:  auth.RoleReviewer,
	//	Name:  "Reviewer", // You could fetch actual name from database
	//}

	// Get students for this reviewer
	//filters := parseFilters(r)
	//students, pagination, err := h.getStudentsForReviewer(reviewerEmail, filters)
	//if err != nil {
	//	http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
	//	return
	//}
	//
	//// Render template with reviewer-specific view
	//locale := getLocale(r)
	//searchValue := r.URL.Query().Get("search")
	//
	//component := templates.ReviewerStudentList(user, students, locale, pagination, searchValue, filters)
	//component.Render(r.Context(), w)
}

func (h *StudentListHandler) validateReviewerToken(token string) (string, error) {
	// Implement your token validation logic here
	// This could be JWT, database lookup, etc.
	// Return the reviewer email if valid
	return "", fmt.Errorf("not implemented")
}
