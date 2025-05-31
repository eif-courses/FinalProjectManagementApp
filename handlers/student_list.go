package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"strconv"
	"strings"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/go-chi/chi/v5"
)

type StudentListHandler struct {
	db *sqlx.DB
}

// NewStudentListHandler creates a new handler instance
func NewStudentListHandler(db *sqlx.DB) *StudentListHandler {
	return &StudentListHandler{
		db: db,
	}
}

// Add missing getLocale function
func getLocale(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}
	return locale
}

// Update your main handler method
func (h *StudentListHandler) StudentTableDisplayHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !canViewStudentList(user.Role) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse filter parameters from URL
	filterParams := parseTemplateFilterParams(r)

	// Get filter options for current user
	filterOptions, err := h.getFilterOptions(user.Role, user.Email)
	if err != nil {
		log.Printf("Error getting filter options: %v", err)
		// Continue with empty options if there's an error
		filterOptions = &database.FilterOptions{}
	}

	// Apply role-based filtering
	var students []database.StudentSummaryView
	var total int

	switch user.Role {
	case auth.RoleSupervisor:
		students, total, err = h.getStudentsForSupervisor(user.Email, filterParams)
	case auth.RoleDepartmentHead:
		students, total, err = h.getStudentsForDepartmentHead(user, filterParams)
	case auth.RoleAdmin:
		students, total, err = h.getAllStudents(filterParams)
	case auth.RoleReviewer:
		students, total, err = h.getStudentsForReviewer(user.Email, filterParams)
	default:
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err != nil {
		log.Printf("Error fetching students: %v", err)
		http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
		return
	}

	// Create pagination
	pagination := database.NewPaginationInfo(filterParams.Page, filterParams.Limit, total)

	// Get locale and search value
	locale := getLocale(r)
	searchValue := r.URL.Query().Get("search")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		err = templates.StudentTableWithPagination(user, students, locale, pagination).Render(r.Context(), w)
	} else {
		// Pass filter options to template
		err = templates.StudentList(user, students, locale, pagination, searchValue, filterParams, filterOptions).Render(r.Context(), w)
	}

	if err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Parse template filter params
func parseTemplateFilterParams(r *http.Request) *database.TemplateFilterParams {
	params := &database.TemplateFilterParams{
		Page:  1,
		Limit: 10,
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}

	// Parse year
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil && year > 0 {
			params.Year = year
		}
	}

	// Parse other filters
	params.Group = r.URL.Query().Get("group")
	params.StudyProgram = r.URL.Query().Get("study_program")
	params.TopicStatus = r.URL.Query().Get("topic_status")
	params.Search = r.URL.Query().Get("search")

	return params
}

// Role-based query methods using your StudentSummaryView model
func (h *StudentListHandler) getStudentsForSupervisor(supervisorEmail string, filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}

	// Use your existing StudentSummaryView model
	baseQuery := `
		SELECT 
			sr.id, sr.student_group, sr.student_name, sr.student_lastname,
			sr.student_email, sr.final_project_title, sr.final_project_title_en,
			sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
			sr.study_program, sr.department, sr.current_year, sr.program_code,
			sr.student_number, sr.is_favorite, sr.is_public_defense, 
			sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
			COALESCE(ptr.status, '') as topic_status, 
			CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
			ptr.approved_by, ptr.approved_at,
			CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			spr.is_signed as supervisor_report_signed,
			CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
			rr.is_signed as reviewer_report_signed,
			rr.grade as reviewer_grade,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
		FROM student_records sr
		LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
		LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
		LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
		LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
		WHERE sr.supervisor_email = ?`

	args = append(args, supervisorEmail)

	// Apply filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
			sr.id, sr.student_group, sr.student_name, sr.student_lastname,
			sr.student_email, sr.final_project_title, sr.final_project_title_en,
			sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
			sr.study_program, sr.department, sr.current_year, sr.program_code,
			sr.student_number, sr.is_favorite, sr.is_public_defense, 
			sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
			COALESCE(ptr.status, '') as topic_status, 
			CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
			ptr.approved_by, ptr.approved_at,
			CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			spr.is_signed as supervisor_report_signed,
			CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
			rr.is_signed as reviewer_report_signed,
			rr.grade as reviewer_grade,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err := h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (h *StudentListHandler) getStudentsForDepartmentHead(user *auth.AuthenticatedUser, filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	// For now, show all students for department head - you can refine this based on department
	return h.getAllStudents(filters)
}

func (h *StudentListHandler) getStudentsForReviewer(reviewerEmail string, filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}

	baseQuery := `
		SELECT 
			sr.id, sr.student_group, sr.student_name, sr.student_lastname,
			sr.student_email, sr.final_project_title, sr.final_project_title_en,
			sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
			sr.study_program, sr.department, sr.current_year, sr.program_code,
			sr.student_number, sr.is_favorite, sr.is_public_defense, 
			sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
			COALESCE(ptr.status, '') as topic_status, 
			CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
			ptr.approved_by, ptr.approved_at,
			CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			spr.is_signed as supervisor_report_signed,
			CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
			rr.is_signed as reviewer_report_signed,
			rr.grade as reviewer_grade,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
		FROM student_records sr
		LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
		LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
		LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
		LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
		WHERE sr.reviewer_email = ?`

	args = append(args, reviewerEmail)

	// Apply filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
			sr.id, sr.student_group, sr.student_name, sr.student_lastname,
			sr.student_email, sr.final_project_title, sr.final_project_title_en,
			sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
			sr.study_program, sr.department, sr.current_year, sr.program_code,
			sr.student_number, sr.is_favorite, sr.is_public_defense, 
			sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
			COALESCE(ptr.status, '') as topic_status, 
			CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
			ptr.approved_by, ptr.approved_at,
			CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			spr.is_signed as supervisor_report_signed,
			CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
			rr.is_signed as reviewer_report_signed,
			rr.grade as reviewer_grade,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err := h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (h *StudentListHandler) getAllStudents(filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}

	baseQuery := `
		SELECT 
			sr.id, sr.student_group, sr.student_name, sr.student_lastname,
			sr.student_email, sr.final_project_title, sr.final_project_title_en,
			sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
			sr.study_program, sr.department, sr.current_year, sr.program_code,
			sr.student_number, sr.is_favorite, sr.is_public_defense, 
			sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
			COALESCE(ptr.status, '') as topic_status, 
			CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
			ptr.approved_by, ptr.approved_at,
			CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			spr.is_signed as supervisor_report_signed,
			CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
			rr.is_signed as reviewer_report_signed,
			rr.grade as reviewer_grade,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
		FROM student_records sr
		LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
		LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
		LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
		LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
		WHERE 1=1`

	// Apply filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
			sr.id, sr.student_group, sr.student_name, sr.student_lastname,
			sr.student_email, sr.final_project_title, sr.final_project_title_en,
			sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
			sr.study_program, sr.department, sr.current_year, sr.program_code,
			sr.student_number, sr.is_favorite, sr.is_public_defense, 
			sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
			COALESCE(ptr.status, '') as topic_status, 
			CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
			ptr.approved_by, ptr.approved_at,
			CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
			spr.is_signed as supervisor_report_signed,
			CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
			rr.is_signed as reviewer_report_signed,
			rr.grade as reviewer_grade,
			CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err := h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

// Helper function to build WHERE clause for filters
func buildWhereClause(filters *database.TemplateFilterParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filters.Group != "" {
		conditions = append(conditions, "sr.student_group = ?")
		args = append(args, filters.Group)
	}

	if filters.StudyProgram != "" {
		conditions = append(conditions, "sr.study_program = ?")
		args = append(args, filters.StudyProgram)
	}

	if filters.Year > 0 {
		conditions = append(conditions, "sr.current_year = ?")
		args = append(args, filters.Year)
	}

	if filters.TopicStatus != "" {
		if filters.TopicStatus == "not_started" {
			conditions = append(conditions, "(ptr.status IS NULL OR ptr.status = '')")
		} else {
			conditions = append(conditions, "ptr.status = ?")
			args = append(args, filters.TopicStatus)
		}
	}

	if filters.Search != "" {
		conditions = append(conditions, `(
            sr.student_name LIKE ? OR 
            sr.student_lastname LIKE ? OR 
            sr.student_email LIKE ? OR 
            sr.final_project_title LIKE ? OR 
            sr.final_project_title_en LIKE ?
        )`)
		searchPattern := "%" + filters.Search + "%"
		for i := 0; i < 5; i++ {
			args = append(args, searchPattern)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " AND "), args
}

// Keep your existing helper functions
func DocumentsAPIHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "id")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Fetch documents for student
	documents, err := getStudentDocuments(studentID)
	if err != nil {
		http.Error(w, "Failed to fetch documents", http.StatusInternalServerError)
		return
	}

	response := database.NewSuccessResponse(documents, "Documents fetched successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func canViewStudentList(role string) bool {
	allowedRoles := []string{"admin", "department_head", "supervisor", "reviewer"}
	for _, allowedRole := range allowedRoles {
		if role == allowedRole {
			return true
		}
	}
	return false
}

func getStudentDocuments(studentID int) ([]database.Document, error) {
	documents := []database.Document{
		{
			ID:               1,
			DocumentType:     "thesis",
			OriginalFilename: database.NullableString("Baigiamasis_darbas.pdf"),
			FileSize:         database.NullableInt64(1024000),
		},
		{
			ID:               2,
			DocumentType:     "video",
			OriginalFilename: database.NullableString("Gynyba_video.mp4"),
			FileSize:         database.NullableInt64(50240000),
		},
	}
	return documents, nil
}

func getTopicStatus(status sql.NullString) string {
	if status.Valid {
		return status.String
	}
	return ""
}

// Get available groups for current user
func (h *StudentListHandler) getAvailableGroups(userRole, userEmail string) ([]string, error) {
	var groups []string
	var query string
	var args []interface{}

	switch userRole {
	case auth.RoleSupervisor:
		query = `SELECT DISTINCT student_group FROM student_records WHERE supervisor_email = ? ORDER BY student_group`
		args = []interface{}{userEmail}
	case auth.RoleReviewer:
		query = `SELECT DISTINCT student_group FROM student_records WHERE reviewer_email = ? ORDER BY student_group`
		args = []interface{}{userEmail}
	default:
		query = `SELECT DISTINCT student_group FROM student_records ORDER BY student_group`
		args = []interface{}{}
	}

	err := h.db.Select(&groups, query, args...)
	return groups, err
}

// Get available study programs for current user
func (h *StudentListHandler) getAvailableStudyPrograms(userRole, userEmail string) ([]string, error) {
	var programs []string
	var query string
	var args []interface{}

	switch userRole {
	case auth.RoleSupervisor:
		query = `SELECT DISTINCT study_program FROM student_records WHERE supervisor_email = ? ORDER BY study_program`
		args = []interface{}{userEmail}
	case auth.RoleReviewer:
		query = `SELECT DISTINCT study_program FROM student_records WHERE reviewer_email = ? ORDER BY study_program`
		args = []interface{}{userEmail}
	default:
		query = `SELECT DISTINCT study_program FROM student_records ORDER BY study_program`
		args = []interface{}{}
	}

	err := h.db.Select(&programs, query, args...)
	return programs, err
}

// Get available years for current user
func (h *StudentListHandler) getAvailableYears(userRole, userEmail string) ([]int, error) {
	var years []int
	var query string
	var args []interface{}

	switch userRole {
	case auth.RoleSupervisor:
		query = `SELECT DISTINCT current_year FROM student_records WHERE supervisor_email = ? ORDER BY current_year DESC`
		args = []interface{}{userEmail}
	case auth.RoleReviewer:
		query = `SELECT DISTINCT current_year FROM student_records WHERE reviewer_email = ? ORDER BY current_year DESC`
		args = []interface{}{userEmail}
	default:
		query = `SELECT DISTINCT current_year FROM student_records ORDER BY current_year DESC`
		args = []interface{}{}
	}

	err := h.db.Select(&years, query, args...)
	return years, err
}

// Get all filter options for current user
func (h *StudentListHandler) getFilterOptions(userRole, userEmail string) (*database.FilterOptions, error) {
	groups, err := h.getAvailableGroups(userRole, userEmail)
	if err != nil {
		return nil, err
	}

	programs, err := h.getAvailableStudyPrograms(userRole, userEmail)
	if err != nil {
		return nil, err
	}

	years, err := h.getAvailableYears(userRole, userEmail)
	if err != nil {
		return nil, err
	}

	return &database.FilterOptions{
		Groups:        groups,
		StudyPrograms: programs,
		Years:         years,
	}, nil
}
