// handlers/commission_handler.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CommissionHandler struct {
	db *sqlx.DB
}

func NewCommissionHandler(db *sqlx.DB) *CommissionHandler {
	return &CommissionHandler{
		db: db,
	}
}

// Generate random access code
func (h *CommissionHandler) generateAccessCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Helper to get user from context
func (h *CommissionHandler) getUserFromContext(r *http.Request) *auth.AuthenticatedUser {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		return nil
	}
	return user
}

// Show management page
func (h *CommissionHandler) ShowManagementPage(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Check permission
	if user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get user's department from the database or use default
	department := h.getUserDepartment(user)

	// Debug: Let's see what departments we have
	var allDepartments []string
	debugQuery := `SELECT DISTINCT department FROM student_records WHERE department IS NOT NULL AND department != ''`
	h.db.Select(&allDepartments, debugQuery)
	log.Printf("Available departments in database: %v", allDepartments)

	// Get active access codes for this department
	var accessCodes []database.CommissionMember
	query := `
		SELECT * FROM commission_members 
		WHERE department = ?
		AND is_active = true
		ORDER BY created_at DESC
	`
	h.db.Select(&accessCodes, query, department)

	// Get study programs for ALL records (for debugging)
	var allStudyPrograms []string
	allProgramsQuery := `
		SELECT DISTINCT study_program 
		FROM student_records 
		WHERE study_program IS NOT NULL
		AND study_program != ''
		ORDER BY study_program
	`
	h.db.Select(&allStudyPrograms, allProgramsQuery)
	log.Printf("All study programs in database: %v", allStudyPrograms)

	// Get study programs for this department
	var studyPrograms []string
	programQuery := `
		SELECT DISTINCT study_program 
		FROM student_records 
		WHERE department = ?
		AND study_program IS NOT NULL
		AND study_program != ''
		ORDER BY study_program
	`
	err := h.db.Select(&studyPrograms, programQuery, department)
	if err != nil {
		log.Printf("Error getting study programs: %v", err)
	}

	log.Printf("Department: %s, Study programs found: %v", department, studyPrograms)

	// If no programs found for department, get all programs
	if len(studyPrograms) == 0 {
		log.Println("No programs found for department, using all programs")
		studyPrograms = allStudyPrograms
	}

	component := templates.CommissionManagement(user, "lt", templates.CommissionManagementData{
		AccessCodes:   accessCodes,
		StudyPrograms: studyPrograms,
		CurrentYear:   time.Now().Year(),
		Department:    department,
	})

	component.Render(r.Context(), w)
}

// Helper function to get user's department
func (h *CommissionHandler) getUserDepartment(user *auth.AuthenticatedUser) string {
	// First, check if user has department set
	if user.Department != "" {
		return user.Department
	}

	// If admin, try to get the first available department
	if user.Role == auth.RoleAdmin {
		var dept string
		query := `SELECT DISTINCT department FROM student_records WHERE department IS NOT NULL AND department != '' LIMIT 1`
		err := h.db.Get(&dept, query)
		if err == nil && dept != "" {
			return dept
		}
	}

	// Try to get department from department_heads table
	var dept string
	query := `SELECT department FROM department_heads WHERE email = ?`
	err := h.db.Get(&dept, query, user.Email)
	if err == nil && dept != "" {
		return dept
	}

	// Default fallback
	return "Informacijos technologijų katedra"
}

// Create new access token
func (h *CommissionHandler) CreateAccess(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get form values
	durationDays, _ := strconv.Atoi(r.FormValue("duration_days"))
	if durationDays <= 0 {
		durationDays = 30
	}
	maxAccess, _ := strconv.Atoi(r.FormValue("max_access"))

	// Generate access code
	accessCode, err := h.generateAccessCode()
	if err != nil {
		http.Error(w, "Failed to generate access code", http.StatusInternalServerError)
		return
	}

	// Use user's department
	department := h.getUserDepartment(user)
	studyProgram := r.FormValue("study_program")

	log.Printf("Creating access token - Department: %s, Study Program: %s", department, studyProgram)

	// Calculate expiration
	expiresAt := time.Now().Add(time.Duration(durationDays) * 24 * time.Hour).Unix()

	// Insert into database
	query := `
		INSERT INTO commission_members (
			access_code, department, study_program, year, 
			is_active, expires_at, created_by, max_access, 
			access_level, commission_type, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`

	result, err := h.db.Exec(query,
		accessCode,
		department,
		studyProgram,
		time.Now().Year(),
		true,
		expiresAt,
		user.Email,
		maxAccess,
		"view_only",
		"defense",
	)

	if err != nil {
		log.Printf("Error creating access token: %v", err)
		http.Error(w, "Failed to create access token", http.StatusInternalServerError)
		return
	}

	// Get the inserted ID
	id, _ := result.LastInsertId()

	// Create member object for response
	member := &database.CommissionMember{
		ID:           int(id),
		AccessCode:   accessCode,
		Department:   department,
		StudyProgram: sql.NullString{String: studyProgram, Valid: true},
		Year:         sql.NullInt64{Int64: int64(time.Now().Year()), Valid: true},
		IsActive:     true,
		ExpiresAt:    expiresAt,
		CreatedBy:    user.Email,
		MaxAccess:    maxAccess,
		AccessCount:  0,
		AccessLevel:  "view_only",
	}

	// Return the new row
	component := templates.SimpleAccessCodeRow(member, "lt")
	component.Render(r.Context(), w)
}

// Delete access token
func (h *CommissionHandler) DeactivateAccess(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	accessCode := chi.URLParam(r, "accessCode")

	query := `
		UPDATE commission_members 
		SET is_active = false 
		WHERE access_code = ?
	`

	_, err := h.db.Exec(query, accessCode)
	if err != nil {
		http.Error(w, "Failed to deactivate token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Public access - show student list with filters and pagination
func (h *CommissionHandler) ShowStudentList(w http.ResponseWriter, r *http.Request) {
	accessCode := chi.URLParam(r, "accessCode")

	// Get and validate commission member
	var member database.CommissionMember
	query := `SELECT * FROM commission_members WHERE access_code = ?`
	err := h.db.Get(&member, query, accessCode)
	if err != nil {
		log.Printf("Invalid access token: %s", accessCode)
		http.Error(w, "Invalid access token", http.StatusUnauthorized)
		return
	}

	// Check if active
	if !member.IsActive {
		http.Error(w, "Access token is deactivated", http.StatusUnauthorized)
		return
	}

	// Check if expired
	if time.Now().Unix() > member.ExpiresAt {
		http.Error(w, "Access token has expired", http.StatusUnauthorized)
		return
	}

	// Check access limit
	if member.MaxAccess > 0 && member.AccessCount >= member.MaxAccess {
		http.Error(w, "Access limit reached", http.StatusUnauthorized)
		return
	}

	// Update access count and last accessed time
	updateQuery := `
		UPDATE commission_members 
		SET access_count = access_count + 1, last_accessed_at = ?
		WHERE id = ?
	`
	_, err = h.db.Exec(updateQuery, time.Now().Unix(), member.ID)
	if err != nil {
		log.Printf("Failed to update access count: %v", err)
	}

	// Parse query parameters for filters and pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	searchValue := strings.TrimSpace(r.URL.Query().Get("search"))
	groupFilter := strings.TrimSpace(r.URL.Query().Get("group"))
	topicStatusFilter := strings.TrimSpace(r.URL.Query().Get("topic_status"))

	// Log the request parameters
	log.Printf("Commission access - Code: %s, Department: %s, Program: %s, Year: %d, Page: %d, Limit: %d, Search: %s",
		accessCode, member.Department, member.StudyProgram.String, member.Year.Int64, page, limit, searchValue)

	// Build base query conditions
	whereConditions := []string{
		"sr.department = ?",
		"sr.study_program = ?",
		"sr.current_year = ?",
	}
	args := []interface{}{
		member.Department,
		member.StudyProgram.String,
		member.Year.Int64,
	}

	// Add search filter if provided
	if searchValue != "" {
		searchCondition := `(
			sr.student_name LIKE ? OR 
			sr.student_lastname LIKE ? OR 
			sr.student_email LIKE ? OR 
			sr.final_project_title LIKE ? OR
			sr.final_project_title_en LIKE ? OR
			sr.student_number LIKE ?
		)`
		whereConditions = append(whereConditions, searchCondition)
		searchPattern := "%" + searchValue + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Add group filter if provided
	if groupFilter != "" {
		whereConditions = append(whereConditions, "sr.student_group = ?")
		args = append(args, groupFilter)
	}

	// Add topic status filter if provided
	if topicStatusFilter != "" {
		if topicStatusFilter == "not_started" {
			whereConditions = append(whereConditions, "(ptr.id IS NULL OR ptr.status IS NULL)")
		} else {
			whereConditions = append(whereConditions, "ptr.status = ?")
			args = append(args, topicStatusFilter)
		}
	}

	// Build WHERE clause
	whereClause := " WHERE " + strings.Join(whereConditions, " AND ")

	// First, get the total count for pagination
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT sr.id)
		FROM student_records sr
		LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
		LEFT JOIN supervisor_reports sup_rep ON sr.id = sup_rep.student_record_id
		LEFT JOIN reviewer_reports rev_rep ON sr.id = rev_rep.student_record_id
		` + whereClause

	err = h.db.Get(&total, countQuery, args...)
	if err != nil {
		log.Printf("Failed to count students: %v", err)
		http.Error(w, "Failed to count students", http.StatusInternalServerError)
		return
	}

	// Calculate pagination
	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Calculate offset for pagination
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// Add pagination parameters to args
	paginatedArgs := append(args, limit, offset)

	// Build the main query to get students with all related data
	studentQuery := `
    SELECT 
        sr.id,
        sr.student_group,
        sr.final_project_title,
        sr.final_project_title_en,
        sr.student_email,
        sr.student_name,
        sr.student_lastname,
        sr.student_number,
        sr.supervisor_email,
        sr.study_program,
        sr.department,
        sr.program_code,
        sr.current_year,
        sr.reviewer_email,
        sr.reviewer_name,
        sr.is_favorite,
        sr.is_public_defense,
        sr.defense_date,
        sr.defense_location,
        sr.created_at,
        sr.updated_at,
        CASE 
            WHEN ptr.status = 'approved' THEN 1 
            ELSE 0 
        END as topic_approved,
        ptr.status as topic_status,
        ptr.approved_by,
        ptr.approved_at,
        CASE 
            WHEN sup_rep.id IS NOT NULL THEN 1 
            ELSE 0 
        END as has_supervisor_report,
        sup_rep.is_signed as supervisor_report_signed,
        CASE 
            WHEN rev_rep.id IS NOT NULL THEN 1 
            ELSE 0 
        END as has_reviewer_report,
        rev_rep.is_signed as reviewer_report_signed,
        rev_rep.grade as reviewer_grade,
        rev_rep.review_questions as reviewer_questions,
        CASE 
            WHEN v.id IS NOT NULL THEN 1 
            ELSE 0 
        END as has_video,
        CASE 
            WHEN EXISTS (
                SELECT 1 FROM documents d 
                WHERE d.student_record_id = sr.id 
                AND d.document_type IN ('thesis_source_code', 'SOURCE_CODE')
                LIMIT 1
            ) THEN 1 
            ELSE 0 
        END as has_source_code,
        (
            SELECT d2.repository_url 
            FROM documents d2 
            WHERE d2.student_record_id = sr.id 
            AND d2.document_type IN ('thesis_source_code', 'SOURCE_CODE')
            ORDER BY d2.uploaded_date DESC
            LIMIT 1
        ) as repository_url
    FROM student_records sr
    LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
    LEFT JOIN supervisor_reports sup_rep ON sr.id = sup_rep.student_record_id
    LEFT JOIN reviewer_reports rev_rep ON sr.id = rev_rep.student_record_id
    LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
    ` + whereClause + `
    GROUP BY sr.id
    ORDER BY sr.student_lastname ASC, sr.student_name ASC
    LIMIT ? OFFSET ?
	`

	// Execute the query
	var students []database.StudentSummaryView
	err = h.db.Select(&students, studentQuery, paginatedArgs...)
	if err != nil {
		log.Printf("Failed to load students: %v", err)
		log.Printf("Query: %s", studentQuery)
		log.Printf("Args: %v", paginatedArgs)
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	log.Printf("Found %d students (page %d of %d)", len(students), page, totalPages)

	// Create pagination info
	pagination := &database.PaginationInfo{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
		NextPage:   page + 1,
		PrevPage:   page - 1,
	}

	// Create filters object for template
	filters := &database.TemplateFilterParams{
		Page:        page,
		Limit:       limit,
		Group:       groupFilter,
		TopicStatus: topicStatusFilter,
		Search:      searchValue,
	}

	// Check if this is an HTMX request (partial update)
	if r.Header.Get("HX-Request") == "true" {
		// Return only the table component for HTMX updates
		log.Printf("HTMX request detected, returning partial update")
		component := templates.CommissionStudentTable(students, pagination, accessCode) // ADD accessCode here

		err = component.Render(r.Context(), w)
		if err != nil {
			log.Printf("Failed to render partial table: %v", err)
			http.Error(w, "Failed to render table", http.StatusInternalServerError)
		}
		return
	}

	// Return full page for regular requests
	component := templates.CommissionStudentList(
		accessCode,
		students,
		member.StudyProgram.String,
		pagination,
		searchValue,
		filters,
	)

	err = component.Render(r.Context(), w)
	if err != nil {
		log.Printf("Failed to render full page: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// API endpoint to refresh access codes list
func (h *CommissionHandler) ListActiveAccess(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	department := h.getUserDepartment(user)

	var accessCodes []database.CommissionMember
	query := `
		SELECT * FROM commission_members 
		WHERE department = ?
		AND is_active = true
		ORDER BY created_at DESC
	`
	h.db.Select(&accessCodes, query, department)

	// Render just the table rows
	for _, code := range accessCodes {
		templates.SimpleAccessCodeRow(&code, "lt").Render(r.Context(), w)
	}
}
