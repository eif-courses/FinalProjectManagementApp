// handlers/topic.go - Complete Enhanced TopicHandlers implementation
package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/go-chi/chi/v5"
)

// Update the struct definition
type TopicHandlers struct {
	db *sqlx.DB // Change from *sql.DB to *sqlx.DB
}

// Update the constructor
func NewTopicHandlers(db *sqlx.DB) *TopicHandlers {
	return &TopicHandlers{db: db}
}

// ShowTopicRegistrationForm displays the topic registration form
func (h *TopicHandlers) ShowTopicRegistrationForm(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}

	// Get existing topic if any
	topic, err := h.getStudentTopic(user.Email)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to load topic", http.StatusInternalServerError)
		return
	}

	// Get comments if topic exists
	var comments []database.TopicRegistrationComment
	var versions []database.ProjectTopicRegistrationVersion // ADD THIS

	if topic != nil {
		comments, err = h.getTopicComments(topic.ID)
		if err != nil {
			comments = []database.TopicRegistrationComment{}
		}

		// GET VERSION HISTORY - ADD THIS
		versions, err = h.getTopicVersions(topic.ID)
		if err != nil {
			versions = []database.ProjectTopicRegistrationVersion{}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// PASS VERSIONS TO THE TEMPLATE - UPDATE THIS
	err = templates.TopicRegistrationModal(user, topic, comments, versions, locale).Render(r.Context(), w)

	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// SaveDraft handles auto-save functionality
func (h *TopicHandlers) SaveDraft(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
	}

	// Save as draft with minimal validation
	topicData := &database.TopicSubmissionData{
		Title:          r.FormValue("title"),
		TitleEn:        r.FormValue("title_en"),
		Problem:        r.FormValue("problem"),
		Objective:      r.FormValue("objective"),
		Tasks:          r.FormValue("tasks"),
		CompletionDate: r.FormValue("completion_date"),
		Supervisor:     r.FormValue("supervisor"),
	}

	// Get or create student record
	studentRecord, err := h.getStudentRecord(user.Email)
	if err != nil {
		h.renderFormError(w, "Student record not found")
		return
	}

	// Check if topic exists
	existingTopic, err := h.getStudentTopic(user.Email)
	if err != nil && err != sql.ErrNoRows {
		h.renderFormError(w, "Database error")
		return
	}

	if existingTopic != nil && existingTopic.IsEditable() {
		err = h.updateTopic(existingTopic.ID, topicData)
	} else if existingTopic == nil {
		_, err = h.createTopic(studentRecord.ID, topicData)
	}

	if err != nil {
		h.renderFormError(w, "Failed to save draft")
		return
	}

	h.renderFormSuccess(w, "Draft saved successfully", 0)
}

// SubmitTopic handles legacy topic submission (for compatibility)
func (h *TopicHandlers) SubmitTopic(w http.ResponseWriter, r *http.Request) {
	// For compatibility, redirect to new submission method
	h.SubmitTopicForReview(w, r)
}

// SubmitTopicForReview handles submitting topic for supervisor review
// Update the SubmitTopicForReview method with better logging
func (h *TopicHandlers) SubmitTopicForReview(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		log.Printf("SubmitTopicForReview: No user in context")
		h.renderFormError(w, "Unauthorized - please login again")
		return
	}

	if user.Role != auth.RoleStudent {
		log.Printf("SubmitTopicForReview: User %s has role %s, not student", user.Email, user.Role)
		h.renderFormError(w, "Only students can submit topics")
		return
	}

	log.Printf("SubmitTopicForReview: User %s (Role: %s) attempting to submit topic", user.Email, user.Role)

	if err := r.ParseForm(); err != nil {
		log.Printf("SubmitTopicForReview: Failed to parse form: %v", err)
		h.renderFormError(w, "Invalid form data")
		return
	}

	// Log all form values
	log.Printf("Form values received:")
	for key, values := range r.Form {
		log.Printf("  %s: %v", key, values)
	}

	// Create topic data
	topicData := &database.TopicSubmissionData{
		Title:          r.FormValue("title"),
		TitleEn:        r.FormValue("title_en"),
		Problem:        r.FormValue("problem"),
		Objective:      r.FormValue("objective"),
		Tasks:          r.FormValue("tasks"),
		CompletionDate: r.FormValue("completion_date"),
		Supervisor:     r.FormValue("supervisor"),
	}

	log.Printf("Topic data created: %+v", topicData)

	// Validate
	if err := topicData.Validate(); err != nil {
		log.Printf("SubmitTopicForReview: Validation failed: %v", err)
		h.renderFormError(w, err.Error())
		return
	}

	log.Printf("Validation passed")

	// Get student record
	studentRecord, err := h.getStudentRecord(user.Email)
	if err != nil {
		log.Printf("SubmitTopicForReview: Failed to get student record for %s: %v", user.Email, err)
		if err == sql.ErrNoRows {
			h.renderFormError(w, "Student record not found. Please contact administrator.")
		} else {
			h.renderFormError(w, "Database error while fetching student record")
		}
		return
	}

	log.Printf("Student record found: ID=%d, Email=%s", studentRecord.ID, studentRecord.StudentEmail)

	// Check if topic already exists
	existingTopic, err := h.getStudentTopic(user.Email)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("SubmitTopicForReview: Database error checking existing topic: %v", err)
		h.renderFormError(w, "Database error")
		return
	}

	var topicID int
	if existingTopic != nil {
		log.Printf("Existing topic found: ID=%d, Status=%s, CanSubmit=%v",
			existingTopic.ID, existingTopic.Status, existingTopic.CanSubmit())

		// Update existing topic and submit
		if !existingTopic.CanSubmit() {
			log.Printf("Topic cannot be submitted - current status: %s", existingTopic.Status)
			h.renderFormError(w, fmt.Sprintf("Topic cannot be submitted in current status: %s", existingTopic.Status))
			return
		}
		topicID = existingTopic.ID
		err = h.updateTopicAndSubmit(topicID, topicData, user)
		if err != nil {
			log.Printf("Failed to update and submit topic: %v", err)
		}
	} else {
		log.Printf("No existing topic, creating new one")
		// Create new topic and submit
		topicID, err = h.createTopicAndSubmit(studentRecord.ID, topicData)
		if err != nil {
			log.Printf("Failed to create and submit topic: %v", err)
		}
	}

	if err != nil {
		log.Printf("SubmitTopicForReview: Failed to submit topic: %v", err)
		h.renderFormError(w, "Failed to submit topic: "+err.Error())
		return
	}

	log.Printf("Topic submitted successfully: ID=%d", topicID)
	h.renderFormSuccess(w, "Topic submitted for review successfully", topicID)
}

// SupervisorApproveTopic handles supervisor approval
func (h *TopicHandlers) SupervisorApproveTopic(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	// Verify user can approve this topic
	topic, err := h.getTopicByID(topicID)
	if err != nil {
		h.renderApprovalError(w, "Topic not found")
		return
	}

	if !topic.CanSupervisorReview() {
		h.renderApprovalError(w, "Topic is not available for supervisor review")
		return
	}

	// TODO: Add supervisor verification logic
	// if !h.isUserSupervisorForTopic(user, topic) {
	//     h.renderApprovalError(w, "You are not authorized to review this topic")
	//     return
	// }

	err = h.updateTopicStatusWithSupervisor(topicID, "supervisor_approved", user.Email, "")
	if err != nil {
		h.renderApprovalError(w, "Failed to approve topic")
		return
	}

	h.renderApprovalSuccess(w, "Topic approved and sent to department head")
}

// SupervisorRequestRevision handles supervisor revision requests
// SupervisorRequestRevision handles supervisor revision requests
func (h *TopicHandlers) SupervisorRequestRevision(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleSupervisor {
		h.renderApprovalError(w, "Unauthorized access")
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		h.renderApprovalError(w, "Invalid topic ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderApprovalError(w, "Invalid form data")
		return
	}

	revisionReason := strings.TrimSpace(r.FormValue("revision_reason"))
	if revisionReason == "" {
		h.renderApprovalError(w, "Revision reason is required")
		return
	}

	// Verify topic can be reviewed
	topic, err := h.getTopicByID(topicID)
	if err != nil {
		h.renderApprovalError(w, "Topic not found")
		return
	}

	if !topic.CanSupervisorReview() {
		h.renderApprovalError(w, "Topic is not available for supervisor review")
		return
	}

	// Begin transaction for version tracking
	tx, err := h.db.Beginx()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		h.renderApprovalError(w, "Database error")
		return
	}
	defer tx.Rollback()

	// Save current version before making changes
	changeSummary := fmt.Sprintf("Supervisor requested revision: %s", revisionReason)
	err = h.saveTopicVersion(tx, topicID, user.Email, changeSummary)
	if err != nil {
		log.Printf("Error saving topic version: %v", err)
	}

	// Update topic status
	query := `
        UPDATE project_topic_registrations 
        SET status = 'revision_requested',
            supervisor_rejection_reason = ?,
            updated_at = CURRENT_TIMESTAMP,
            current_version = current_version + 1
        WHERE id = ?
    `

	_, err = tx.Exec(query, revisionReason, topicID)
	if err != nil {
		log.Printf("Error updating topic status: %v", err)
		h.renderApprovalError(w, "Failed to request revision")
		return
	}

	// Add comment about the revision request
	commentQuery := `
        INSERT INTO topic_registration_comments 
        (topic_registration_id, author_role, author_name, author_email, comment_text, comment_type, is_read)
        VALUES (?, ?, ?, ?, ?, 'revision', true)
    `

	commentText := fmt.Sprintf("Supervisor requested revision: %s", revisionReason)
	_, err = tx.Exec(commentQuery, topicID, user.Role, user.Name, user.Email, commentText)
	if err != nil {
		log.Printf("Error adding comment: %v", err)
		// Don't fail the whole transaction if comment fails
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
		h.renderApprovalError(w, "Failed to save changes")
		return
	}

	// Send success response
	h.renderApprovalSuccess(w, "Revision request sent successfully")
}

// ApproveTopic handles final approval by department head
func (h *TopicHandlers) ApproveTopic(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || (user.Role != auth.RoleDepartmentHead && user.Role != auth.RoleAdmin) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	// Verify topic can be approved
	topic, err := h.getTopicByID(topicID)
	if err != nil {
		h.renderApprovalError(w, "Topic not found")
		return
	}

	if !topic.CanDepartmentReview() && topic.Status != "submitted" {
		h.renderApprovalError(w, "Topic is not available for department approval")
		return
	}

	err = h.updateTopicStatus(topicID, "approved", user.Email, "")
	if err != nil {
		h.renderApprovalError(w, "Failed to approve topic")
		return
	}

	h.renderApprovalSuccess(w, "Topic approved successfully")
}

// RejectTopic handles topic rejection by department head
func (h *TopicHandlers) RejectTopic(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || (user.Role != auth.RoleDepartmentHead && user.Role != auth.RoleAdmin) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	rejectionReason := r.FormValue("rejection_reason")
	if rejectionReason == "" {
		h.renderApprovalError(w, "Rejection reason is required")
		return
	}

	err = h.updateTopicStatus(topicID, "rejected", user.Email, rejectionReason)
	if err != nil {
		h.renderApprovalError(w, "Failed to reject topic")
		return
	}

	h.renderApprovalSuccess(w, "Topic rejected")
}

// AddComment handles adding comments to topics
func (h *TopicHandlers) AddComment(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	comment := database.TopicRegistrationComment{
		TopicRegistrationID: topicID,
		CommentText:         r.FormValue("comment"),
		AuthorRole:          user.Role,
		AuthorName:          user.Name,
		AuthorEmail:         user.Email,
		CommentType:         r.FormValue("comment_type"),
		IsRead:              true,
		CreatedAt:           time.Now(),
	}

	if comment.CommentType == "" {
		comment.CommentType = "comment"
	}

	err = h.addTopicComment(&comment)
	if err != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	// Return the new comment HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CommentCard(comment, "lt").Render(r.Context(), w)
}

// New methods for role-specific views
func (h *TopicHandlers) ShowSupervisorTopics(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleSupervisor {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// TODO: Implement supervisor topics view
	// This should show all topics where the current user is the supervisor
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Supervisor Topics - Coming Soon</h1>"))
}

func (h *TopicHandlers) ShowPendingSupervisorTopics(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleSupervisor {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// TODO: Implement pending supervisor topics view
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Pending Supervisor Topics - Coming Soon</h1>"))
}

func (h *TopicHandlers) ShowDepartmentTopics(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleDepartmentHead {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// TODO: Implement department topics view
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Department Topics - Coming Soon</h1>"))
}

func (h *TopicHandlers) ShowPendingDepartmentTopics(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleDepartmentHead {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// TODO: Implement pending department topics view
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Pending Department Topics - Coming Soon</h1>"))
}

func (h *TopicHandlers) ShowAllTopics(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleAdmin {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// TODO: Implement admin all topics view
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>All Topics (Admin) - Coming Soon</h1>"))
}

func (h *TopicHandlers) ShowTopicAnalytics(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleAdmin {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// TODO: Implement topic analytics view
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Topic Analytics - Coming Soon</h1>"))
}

func (h *TopicHandlers) ResetTopicWorkflow(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || user.Role != auth.RoleAdmin {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	// Reset topic to draft status
	err = h.updateTopicStatus(topicID, "draft", user.Email, "")
	if err != nil {
		h.renderApprovalError(w, "Failed to reset topic workflow")
		return
	}

	h.renderApprovalSuccess(w, "Topic workflow reset to draft")
}

// Database helper methods
func (h *TopicHandlers) getStudentTopic(studentEmail string) (*database.ProjectTopicRegistration, error) {
	query := `
        SELECT ptr.id, ptr.student_record_id, ptr.title, ptr.title_en, ptr.problem, 
               ptr.objective, ptr.tasks, ptr.completion_date, ptr.supervisor, ptr.status, 
               ptr.created_at, ptr.updated_at, ptr.submitted_at, ptr.current_version, 
               ptr.approved_by, ptr.approved_at, ptr.rejection_reason,
               ptr.supervisor_approved_at, ptr.supervisor_approved_by, ptr.supervisor_rejection_reason
        FROM project_topic_registrations ptr
        JOIN student_records sr ON ptr.student_record_id = sr.id
        WHERE sr.student_email = ?
        ORDER BY ptr.created_at DESC
        LIMIT 1
    `

	var topic database.ProjectTopicRegistration
	var completionDate, approvedBy, rejectionReason, supervisorApprovedBy, supervisorRejectionReason sql.NullString
	var submittedAt, approvedAt, supervisorApprovedAt sql.NullInt64

	err := h.db.QueryRow(query, studentEmail).Scan(
		&topic.ID, &topic.StudentRecordID, &topic.Title, &topic.TitleEn,
		&topic.Problem, &topic.Objective, &topic.Tasks, &completionDate,
		&topic.Supervisor, &topic.Status, &topic.CreatedAt, &topic.UpdatedAt,
		&submittedAt, &topic.CurrentVersion, &approvedBy, &approvedAt,
		&rejectionReason, &supervisorApprovedAt, &supervisorApprovedBy, &supervisorRejectionReason,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if completionDate.Valid {
		topic.CompletionDate = &completionDate.String
	}
	if submittedAt.Valid {
		topic.SubmittedAt = &submittedAt.Int64
	}
	if approvedBy.Valid {
		topic.ApprovedBy = &approvedBy.String
	}
	if approvedAt.Valid {
		topic.ApprovedAt = &approvedAt.Int64
	}
	if rejectionReason.Valid {
		topic.RejectionReason = &rejectionReason.String
	}
	if supervisorApprovedAt.Valid {
		topic.SupervisorApprovedAt = &supervisorApprovedAt.Int64
	}
	if supervisorApprovedBy.Valid {
		topic.SupervisorApprovedBy = &supervisorApprovedBy.String
	}
	if supervisorRejectionReason.Valid {
		topic.SupervisorRejectionReason = &supervisorRejectionReason.String
	}

	return &topic, nil
}

func (h *TopicHandlers) getStudentRecord(studentEmail string) (*database.StudentRecord, error) {
	query := `
        SELECT id, student_group, final_project_title, final_project_title_en,
               student_email, student_name, student_lastname, student_number,
               supervisor_email, study_program, department, program_code,
               current_year, reviewer_email, reviewer_name, is_favorite,
               created_at, updated_at
        FROM student_records
        WHERE student_email = ?
    `

	var record database.StudentRecord
	err := h.db.QueryRow(query, studentEmail).Scan(
		&record.ID, &record.StudentGroup, &record.FinalProjectTitle,
		&record.FinalProjectTitleEn, &record.StudentEmail, &record.StudentName,
		&record.StudentLastname, &record.StudentNumber, &record.SupervisorEmail,
		&record.StudyProgram, &record.Department, &record.ProgramCode,
		&record.CurrentYear, &record.ReviewerEmail, &record.ReviewerName,
		&record.IsFavorite, &record.CreatedAt, &record.UpdatedAt,
	)

	return &record, err
}

func (h *TopicHandlers) getTopicByID(topicID int) (*database.ProjectTopicRegistration, error) {
	query := `
        SELECT id, student_record_id, title, title_en, problem, 
               objective, tasks, completion_date, supervisor, status, 
               created_at, updated_at, submitted_at, current_version, 
               approved_by, approved_at, rejection_reason,
               supervisor_approved_at, supervisor_approved_by, supervisor_rejection_reason
        FROM project_topic_registrations
        WHERE id = ?
    `

	var topic database.ProjectTopicRegistration
	var completionDate, approvedBy, rejectionReason, supervisorApprovedBy, supervisorRejectionReason sql.NullString
	var submittedAt, approvedAt, supervisorApprovedAt sql.NullInt64

	err := h.db.QueryRow(query, topicID).Scan(
		&topic.ID, &topic.StudentRecordID, &topic.Title, &topic.TitleEn,
		&topic.Problem, &topic.Objective, &topic.Tasks, &completionDate,
		&topic.Supervisor, &topic.Status, &topic.CreatedAt, &topic.UpdatedAt,
		&submittedAt, &topic.CurrentVersion, &approvedBy, &approvedAt,
		&rejectionReason, &supervisorApprovedAt, &supervisorApprovedBy, &supervisorRejectionReason,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if completionDate.Valid {
		topic.CompletionDate = &completionDate.String
	}
	if submittedAt.Valid {
		topic.SubmittedAt = &submittedAt.Int64
	}
	if approvedBy.Valid {
		topic.ApprovedBy = &approvedBy.String
	}
	if approvedAt.Valid {
		topic.ApprovedAt = &approvedAt.Int64
	}
	if rejectionReason.Valid {
		topic.RejectionReason = &rejectionReason.String
	}
	if supervisorApprovedAt.Valid {
		topic.SupervisorApprovedAt = &supervisorApprovedAt.Int64
	}
	if supervisorApprovedBy.Valid {
		topic.SupervisorApprovedBy = &supervisorApprovedBy.String
	}
	if supervisorRejectionReason.Valid {
		topic.SupervisorRejectionReason = &supervisorRejectionReason.String
	}

	return &topic, nil
}

func (h *TopicHandlers) createTopic(studentRecordID int, data *database.TopicSubmissionData) (int, error) {
	query := `
        INSERT INTO project_topic_registrations (
            student_record_id, title, title_en, problem, objective, tasks,
            completion_date, supervisor, status, current_version
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'draft', 1)
    `

	result, err := h.db.Exec(query,
		studentRecordID, data.Title, data.TitleEn, data.Problem,
		data.Objective, data.Tasks, data.CompletionDate, data.Supervisor)

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (h *TopicHandlers) createTopicAndSubmit(studentRecordID int, data *database.TopicSubmissionData) (int, error) {
	query := `
        INSERT INTO project_topic_registrations (
            student_record_id, title, title_en, problem, objective, tasks,
            completion_date, supervisor, status, current_version, submitted_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'submitted', 1, ?)
    `

	result, err := h.db.Exec(query,
		studentRecordID, data.Title, data.TitleEn, data.Problem,
		data.Objective, data.Tasks, data.CompletionDate, data.Supervisor,
		time.Now().Unix())

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (h *TopicHandlers) updateTopic(topicID int, data *database.TopicSubmissionData) error {
	query := `
        UPDATE project_topic_registrations 
        SET title = ?, title_en = ?, problem = ?, objective = ?, tasks = ?,
            completion_date = ?, supervisor = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `

	_, err := h.db.Exec(query,
		data.Title, data.TitleEn, data.Problem, data.Objective,
		data.Tasks, data.CompletionDate, data.Supervisor, topicID)

	return err
}

// Update the updateTopicAndSubmit method
func (h *TopicHandlers) updateTopicAndSubmit(topicID int, data *database.TopicSubmissionData, user *auth.AuthenticatedUser) error {
	tx, err := h.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the original topic BEFORE any changes
	var originalTopic database.ProjectTopicRegistration
	err = tx.Get(&originalTopic, "SELECT * FROM project_topic_registrations WHERE id = ?", topicID)
	if err != nil {
		return err
	}

	// Detect what will change
	changes := h.detectChanges(&originalTopic, data)
	changeSummary := h.buildChangeSummary(changes)

	// If there are actual changes, save the current version
	if len(changes) > 0 {
		// Save the CURRENT state before updating
		err = h.saveTopicVersion(tx, topicID, user.Email, changeSummary)
		if err != nil {
			return err
		}
	}

	// Now perform the update
	query := `
        UPDATE project_topic_registrations 
        SET title = ?, title_en = ?, problem = ?, objective = ?, tasks = ?,
            completion_date = ?, supervisor = ?, status = 'submitted', 
            submitted_at = ?, updated_at = CURRENT_TIMESTAMP,
            current_version = current_version + 1
        WHERE id = ?
    `
	_, err = tx.Exec(query,
		data.Title, data.TitleEn, data.Problem, data.Objective,
		data.Tasks, data.CompletionDate, data.Supervisor,
		time.Now().Unix(), topicID)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// Helper method to detect changes
func (h *TopicHandlers) detectChanges(original *database.ProjectTopicRegistration, new *database.TopicSubmissionData) map[string][]string {
	changes := make(map[string][]string)

	if original.Title != new.Title {
		changes["Title"] = []string{original.Title, new.Title}
	}
	if original.TitleEn != new.TitleEn {
		changes["Title (English)"] = []string{original.TitleEn, new.TitleEn}
	}
	if original.Problem != new.Problem {
		changes["Problem"] = []string{original.Problem, new.Problem}
	}
	if original.Objective != new.Objective {
		changes["Objective"] = []string{original.Objective, new.Objective}
	}
	if original.Tasks != new.Tasks {
		changes["Tasks"] = []string{original.Tasks, new.Tasks}
	}
	if original.Supervisor != new.Supervisor {
		changes["Supervisor"] = []string{original.Supervisor, new.Supervisor}
	}

	return changes
}

// Helper to build change summary
func (h *TopicHandlers) buildChangeSummary(changes map[string][]string) string {
	if len(changes) == 0 {
		return "No changes"
	}

	var parts []string
	for field := range changes {
		parts = append(parts, field)
	}

	return fmt.Sprintf("Updated: %s", strings.Join(parts, ", "))
}

// Add this to TopicHandlers
func (h *TopicHandlers) ShowTopicVersionHistory(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	// Get the topic first
	topic, err := h.getTopicByID(topicID)
	if err != nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	versions, err := h.getTopicVersions(topicID)
	if err != nil {
		http.Error(w, "Failed to load versions", http.StatusInternalServerError)
		return
	}

	// Render version history component with topic
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TopicVersionHistory(topic, versions, getLocale(r)).Render(r.Context(), w)
}

// Add this method to TopicHandlers in topic.go
//func (h *TopicHandlers) ShowVersionDiff(w http.ResponseWriter, r *http.Request) {
//	user := auth.GetUserFromContext(r.Context())
//	if user == nil {
//		http.Error(w, "Unauthorized", http.StatusUnauthorized)
//		return
//	}
//
//	topicIDStr := chi.URLParam(r, "id")
//	versionIDStr := chi.URLParam(r, "versionId")
//
//	topicID, err := strconv.Atoi(topicIDStr)
//	if err != nil {
//		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
//		return
//	}
//
//	versionNumber, err := strconv.Atoi(versionIDStr)
//	if err != nil {
//		http.Error(w, "Invalid version ID", http.StatusBadRequest)
//		return
//	}
//
//	// Get the requested version
//	var requestedVersion database.ProjectTopicRegistrationVersion
//	query := `
//		SELECT * FROM project_topic_registration_versions
//		WHERE topic_registration_id = ? AND version_number = ?
//	`
//	err = h.db.Get(&requestedVersion, query, topicID, versionNumber)
//	if err != nil {
//		http.Error(w, "Version not found", http.StatusNotFound)
//		return
//	}
//
//	// Parse the version data
//	var versionData database.ProjectTopicRegistration
//	err = json.Unmarshal([]byte(requestedVersion.VersionData), &versionData)
//	if err != nil {
//		http.Error(w, "Failed to parse version data", http.StatusInternalServerError)
//		return
//	}
//
//	// Get the previous version for comparison (if exists)
//	var previousVersionData *database.ProjectTopicRegistration
//	changes := make(map[string][]string)
//
//	if versionNumber > 1 {
//		var previousVersion database.ProjectTopicRegistrationVersion
//		err = h.db.Get(&previousVersion, query, topicID, versionNumber-1)
//		if err == nil {
//			var prevData database.ProjectTopicRegistration
//			if err := json.Unmarshal([]byte(previousVersion.VersionData), &prevData); err == nil {
//				previousVersionData = &prevData
//				changes = h.compareVersions(&prevData, &versionData)
//			}
//		}
//	} else {
//		// For version 1, compare with empty state
//		emptyTopic := &database.ProjectTopicRegistration{}
//		changes = h.compareVersions(emptyTopic, &versionData)
//	}
//
//	// Get locale
//	locale := r.URL.Query().Get("locale")
//	if locale == "" {
//		locale = "lt"
//	}
//
//	// Render the diff modal
//	w.Header().Set("Content-Type", "text/html; charset=utf-8")
//	templates.VersionDiffModal(previousVersionData, &versionData, changes, locale).Render(r.Context(), w)
//}

// Helper method to compare two versions
func (h *TopicHandlers) compareVersions(old, new *database.ProjectTopicRegistration) map[string][]string {
	changes := make(map[string][]string)

	// Compare title
	if old.Title != new.Title {
		changes["Tema (Lietuvių k.) / Title (Lithuanian)"] = []string{old.Title, new.Title}
	}

	// Compare English title
	if old.TitleEn != new.TitleEn {
		changes["Tema (Anglų k.) / Title (English)"] = []string{old.TitleEn, new.TitleEn}
	}

	// Compare problem
	if old.Problem != new.Problem {
		changes["Problemos aprašymas / Problem Description"] = []string{old.Problem, new.Problem}
	}

	// Compare objective
	if old.Objective != new.Objective {
		changes["Tikslas / Objective"] = []string{old.Objective, new.Objective}
	}

	// Compare tasks
	if old.Tasks != new.Tasks {
		changes["Uždaviniai / Tasks"] = []string{old.Tasks, new.Tasks}
	}

	// Compare supervisor
	if old.Supervisor != new.Supervisor {
		changes["Vadovas / Supervisor"] = []string{old.Supervisor, new.Supervisor}
	}

	// Compare completion date
	oldDate := ""
	newDate := ""
	if old.CompletionDate != nil {
		oldDate = *old.CompletionDate
	}
	if new.CompletionDate != nil {
		newDate = *new.CompletionDate
	}
	if oldDate != newDate {
		changes["Užbaigimo data / Completion Date"] = []string{oldDate, newDate}
	}

	// Compare status
	if old.Status != new.Status {
		changes["Būsena / Status"] = []string{
			h.getStatusDisplayForDiff(old.Status),
			h.getStatusDisplayForDiff(new.Status),
		}
	}

	return changes
}

// Helper to get status display for diff
func (h *TopicHandlers) getStatusDisplayForDiff(status string) string {
	statusMap := map[string]string{
		"draft":               "Juodraštis / Draft",
		"submitted":           "Pateikta / Submitted",
		"supervisor_approved": "Vadovas patvirtino / Supervisor Approved",
		"approved":            "Patvirtinta / Approved",
		"rejected":            "Atmesta / Rejected",
		"revision_requested":  "Prašoma pataisymų / Revision Requested",
	}

	if display, ok := statusMap[status]; ok {
		return display
	}
	return status
}

func (h *TopicHandlers) updateTopicStatusWithSupervisor(topicID int, status, supervisorEmail, reason string) error {
	if status == "supervisor_approved" {
		query := `
            UPDATE project_topic_registrations 
            SET status = ?, supervisor_approved_by = ?, supervisor_approved_at = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `
		_, err := h.db.Exec(query, status, supervisorEmail, time.Now().Unix(), topicID)
		return err
	} else if status == "revision_requested" {
		query := `
            UPDATE project_topic_registrations 
            SET status = ?, supervisor_rejection_reason = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `
		_, err := h.db.Exec(query, status, reason, topicID)
		return err
	}

	// For other statuses
	query := `UPDATE project_topic_registrations SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := h.db.Exec(query, status, topicID)
	return err
}

func (h *TopicHandlers) updateTopicStatus(topicID int, status, approverEmail, rejectionReason string) error {
	if status == "approved" {
		query := `
            UPDATE project_topic_registrations 
            SET status = ?, approved_by = ?, approved_at = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `
		_, err := h.db.Exec(query, status, approverEmail, time.Now().Unix(), topicID)
		return err
	} else if status == "rejected" {
		query := `
            UPDATE project_topic_registrations 
            SET status = ?, rejection_reason = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `
		_, err := h.db.Exec(query, status, rejectionReason, topicID)
		return err
	}

	query := `UPDATE project_topic_registrations SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := h.db.Exec(query, status, topicID)
	return err
}

func (h *TopicHandlers) getTopicComments(topicID int) ([]database.TopicRegistrationComment, error) {
	query := `
        SELECT id, topic_registration_id, field_name, comment_text, author_role,
               author_name, author_email, created_at, parent_comment_id, is_read,
               comment_type
        FROM topic_registration_comments
        WHERE topic_registration_id = ?
        ORDER BY created_at ASC
    `

	rows, err := h.db.Query(query, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []database.TopicRegistrationComment
	for rows.Next() {
		var comment database.TopicRegistrationComment
		var fieldName sql.NullString
		var parentCommentID sql.NullInt32

		err := rows.Scan(
			&comment.ID, &comment.TopicRegistrationID, &fieldName,
			&comment.CommentText, &comment.AuthorRole, &comment.AuthorName,
			&comment.AuthorEmail, &comment.CreatedAt, &parentCommentID,
			&comment.IsRead, &comment.CommentType,
		)
		if err != nil {
			continue
		}

		if fieldName.Valid {
			comment.FieldName = &fieldName.String
		}

		if parentCommentID.Valid {
			parentID := int(parentCommentID.Int32)
			comment.ParentCommentID = &parentID
		} else {
			comment.ParentCommentID = nil
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (h *TopicHandlers) addTopicComment(comment *database.TopicRegistrationComment) error {
	query := `
       INSERT INTO topic_registration_comments (
           topic_registration_id, comment_text, author_role, author_name,
           author_email, comment_type, is_read
       ) VALUES (?, ?, ?, ?, ?, ?, ?)
   `

	_, err := h.db.Exec(query,
		comment.TopicRegistrationID, comment.CommentText, comment.AuthorRole,
		comment.AuthorName, comment.AuthorEmail, comment.CommentType, comment.IsRead)

	return err
}

// ShowTopicRegistrationModal displays the topic registration form in a modal
// Update ShowTopicRegistrationModal to include versions
func (h *TopicHandlers) ShowTopicRegistrationModal(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "studentId")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "view"
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}

	// Get student record to find email
	studentRecord, err := h.getStudentRecordByID(studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusInternalServerError)
		return
	}

	// Get existing topic if any
	topic, err := h.getStudentTopic(studentRecord.StudentEmail)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Failed to load topic", http.StatusInternalServerError)
		return
	}

	// Get comments if topic exists
	var comments []database.TopicRegistrationComment
	var versions []database.ProjectTopicRegistrationVersion // ADD THIS

	if topic != nil {
		comments, err = h.getTopicComments(topic.ID)
		if err != nil {
			comments = []database.TopicRegistrationComment{}
		}

		// GET VERSION HISTORY - ADD THIS
		versions, err = h.getTopicVersions(topic.ID)
		if err != nil {
			versions = []database.ProjectTopicRegistrationVersion{}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// PASS VERSIONS TO THE TEMPLATE - UPDATE THIS
	err = templates.TopicRegistrationModal(user, topic, comments, versions, locale).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}
func (h *TopicHandlers) getStudentRecordByID(studentID int) (*database.StudentRecord, error) {
	query := `
        SELECT id, student_group, final_project_title, final_project_title_en,
               student_email, student_name, student_lastname, student_number,
               supervisor_email, study_program, department, program_code,
               current_year, reviewer_email, reviewer_name, is_favorite,
               created_at, updated_at
        FROM student_records
        WHERE id = ?
    `

	var record database.StudentRecord
	err := h.db.QueryRow(query, studentID).Scan(
		&record.ID, &record.StudentGroup, &record.FinalProjectTitle,
		&record.FinalProjectTitleEn, &record.StudentEmail, &record.StudentName,
		&record.StudentLastname, &record.StudentNumber, &record.SupervisorEmail,
		&record.StudyProgram, &record.Department, &record.ProgramCode,
		&record.CurrentYear, &record.ReviewerEmail, &record.ReviewerName,
		&record.IsFavorite, &record.CreatedAt, &record.UpdatedAt,
	)

	return &record, err
}

// Add this method to save versions when topics are updated
func (h *TopicHandlers) saveTopicVersion(tx *sqlx.Tx, topicID int, changedBy string, changeSummary string) error {
	// Get current topic data
	var topic database.ProjectTopicRegistration
	query := `
		SELECT * FROM project_topic_registrations WHERE id = ?
	`
	err := tx.Get(&topic, query, topicID)
	if err != nil {
		return err
	}

	// Serialize topic data to JSON
	topicJSON, err := json.Marshal(topic)
	if err != nil {
		return err
	}

	// Get current version number
	var currentVersion int
	err = tx.Get(&currentVersion,
		"SELECT COALESCE(MAX(version_number), 0) FROM project_topic_registration_versions WHERE topic_registration_id = ?",
		topicID)
	if err != nil {
		return err
	}

	// Insert new version
	insertQuery := `
		INSERT INTO project_topic_registration_versions 
		(topic_registration_id, version_data, created_by, version_number, change_summary)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertQuery, topicID, string(topicJSON), changedBy, currentVersion+1, changeSummary)
	return err
}

// Add this method to get version history
func (h *TopicHandlers) getTopicVersions(topicID int) ([]database.ProjectTopicRegistrationVersion, error) {
	query := `
		SELECT * FROM project_topic_registration_versions 
		WHERE topic_registration_id = ? 
		ORDER BY version_number DESC
	`
	var versions []database.ProjectTopicRegistrationVersion
	err := h.db.Select(&versions, query, topicID)
	return versions, err
}

// DepartmentRequestRevision handles department head revision requests
func (h *TopicHandlers) DepartmentRequestRevision(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil || (user.Role != auth.RoleDepartmentHead && user.Role != auth.RoleAdmin) {
		h.renderApprovalError(w, "Unauthorized access")
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		h.renderApprovalError(w, "Invalid topic ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderApprovalError(w, "Invalid form data")
		return
	}

	revisionReason := strings.TrimSpace(r.FormValue("revision_reason"))
	if revisionReason == "" {
		h.renderApprovalError(w, "Revision reason is required")
		return
	}

	// Verify topic can be reviewed by department
	topic, err := h.getTopicByID(topicID)
	if err != nil {
		h.renderApprovalError(w, "Topic not found")
		return
	}

	// Department can request revision on supervisor_approved or submitted topics
	if topic.Status != "supervisor_approved" && topic.Status != "submitted" {
		h.renderApprovalError(w, "Topic is not available for department review")
		return
	}

	// Begin transaction for version tracking
	tx, err := h.db.Beginx()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		h.renderApprovalError(w, "Database error")
		return
	}
	defer tx.Rollback()

	// Save current version before making changes
	changeSummary := fmt.Sprintf("Department requested revision: %s", revisionReason)
	err = h.saveTopicVersion(tx, topicID, user.Email, changeSummary)
	if err != nil {
		log.Printf("Error saving topic version: %v", err)
	}

	// Update topic status to revision_requested
	query := `
        UPDATE project_topic_registrations 
        SET status = 'revision_requested',
            rejection_reason = ?,
            updated_at = CURRENT_TIMESTAMP,
            current_version = current_version + 1
        WHERE id = ?
    `

	_, err = tx.Exec(query, revisionReason, topicID)
	if err != nil {
		log.Printf("Error updating topic status: %v", err)
		h.renderApprovalError(w, "Failed to request revision")
		return
	}

	// Add comment about the revision request
	commentQuery := `
        INSERT INTO topic_registration_comments 
        (topic_registration_id, author_role, author_name, author_email, comment_text, comment_type, is_read)
        VALUES (?, ?, ?, ?, ?, 'revision', true)
    `

	commentText := fmt.Sprintf("Department requested revision: %s", revisionReason)
	_, err = tx.Exec(commentQuery, topicID, user.Role, user.Name, user.Email, commentText)
	if err != nil {
		log.Printf("Error adding comment: %v", err)
		// Don't fail the whole transaction if comment fails
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
		h.renderApprovalError(w, "Failed to save changes")
		return
	}

	// Send success response
	h.renderApprovalSuccess(w, "Revision request sent successfully")
}

// renderApprovalSuccess renders a success message for HTMX responses
func (h *TopicHandlers) renderApprovalSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	// Return clean HTML without any scripts
	html := fmt.Sprintf(`<div class="bg-green-50 border border-green-200 text-green-800 px-3 py-2 rounded text-sm">
        ✓ %s
    </div>`, message)

	w.Write([]byte(html))
}

// renderApprovalError renders an error message for HTMX responses
func (h *TopicHandlers) renderApprovalError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK) // Use 200 for HTMX to process the response

	// Return clean HTML without any scripts
	html := fmt.Sprintf(`<div class="bg-red-50 border border-red-200 text-red-800 px-3 py-2 rounded text-sm">
        ❌ %s
    </div>`, message)

	w.Write([]byte(html))
}

// renderFormError renders a form error message for HTMX responses
func (h *TopicHandlers) renderFormError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK) // Use 200 for HTMX to process the response

	html := fmt.Sprintf(`<div class="bg-red-50 border border-red-200 text-red-800 px-3 py-2 rounded text-sm">
        ❌ %s
    </div>`, message)

	w.Write([]byte(html))
}

// renderFormSuccess renders a form success message for HTMX responses
func (h *TopicHandlers) renderFormSuccess(w http.ResponseWriter, message string, topicID int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	html := fmt.Sprintf(`<div class="bg-green-50 border border-green-200 text-green-800 px-3 py-2 rounded text-sm">
        ✓ %s
    </div>`, message)

	w.Write([]byte(html))
}

// ShowVersionChanges shows inline version changes
func (h *TopicHandlers) ShowVersionChanges(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	versionIDStr := chi.URLParam(r, "versionId")

	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	versionNumber, err := strconv.Atoi(versionIDStr)
	if err != nil {
		http.Error(w, "Invalid version ID", http.StatusBadRequest)
		return
	}

	// Get the requested version
	var requestedVersion database.ProjectTopicRegistrationVersion
	query := `
		SELECT * FROM project_topic_registration_versions 
		WHERE topic_registration_id = ? AND version_number = ?
	`
	err = h.db.Get(&requestedVersion, query, topicID, versionNumber)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Parse the version data
	var versionData database.ProjectTopicRegistration
	err = json.Unmarshal([]byte(requestedVersion.VersionData), &versionData)
	if err != nil {
		http.Error(w, "Failed to parse version data", http.StatusInternalServerError)
		return
	}

	// Get the previous version for comparison (if exists)
	changes := make(map[string][]string)

	if versionNumber > 1 {
		var previousVersion database.ProjectTopicRegistrationVersion
		err = h.db.Get(&previousVersion, query, topicID, versionNumber-1)
		if err == nil {
			var prevData database.ProjectTopicRegistration
			if err := json.Unmarshal([]byte(previousVersion.VersionData), &prevData); err == nil {
				changes = h.compareVersions(&prevData, &versionData)
			}
		}
	} else {
		// For version 1, compare with empty state
		emptyTopic := &database.ProjectTopicRegistration{}
		changes = h.compareVersions(emptyTopic, &versionData)
	}

	// Get locale
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}

	// Render the inline comparison
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.VersionInlineComparison(changes, locale).Render(r.Context(), w)
}

// GetTopicContent returns current topic content
func (h *TopicHandlers) GetTopicContent(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	topic, err := h.getTopicByID(topicID)
	if err != nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TopicContentDisplay(topic, nil, false, locale).Render(r.Context(), w)
}

// CompareTopicVersions returns comparison view
func (h *TopicHandlers) CompareTopicVersions(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	topicIDStr := chi.URLParam(r, "id")
	versionIDStr := chi.URLParam(r, "versionId")

	topicID, err := strconv.Atoi(topicIDStr)
	if err != nil {
		http.Error(w, "Invalid topic ID", http.StatusBadRequest)
		return
	}

	versionNumber, err := strconv.Atoi(versionIDStr)
	if err != nil {
		http.Error(w, "Invalid version ID", http.StatusBadRequest)
		return
	}

	// Get current topic
	currentTopic, err := h.getTopicByID(topicID)
	if err != nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	// Get comparison version
	var comparisonVersion database.ProjectTopicRegistrationVersion
	query := `
		SELECT * FROM project_topic_registration_versions 
		WHERE topic_registration_id = ? AND version_number = ?
	`
	err = h.db.Get(&comparisonVersion, query, topicID, versionNumber)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	// Parse comparison version data
	var comparisonTopic database.ProjectTopicRegistration
	err = json.Unmarshal([]byte(comparisonVersion.VersionData), &comparisonTopic)
	if err != nil {
		http.Error(w, "Failed to parse version data", http.StatusInternalServerError)
		return
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.TopicContentDisplay(currentTopic, &comparisonTopic, true, locale).Render(r.Context(), w)
}
