// handlers/topic.go - Complete Enhanced TopicHandlers implementation
package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/go-chi/chi/v5"
)

type TopicHandlers struct {
	db *sql.DB
}

func NewTopicHandlers(db *sql.DB) *TopicHandlers {
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
	if topic != nil {
		comments, err = h.getTopicComments(topic.ID)
		if err != nil {
			comments = []database.TopicRegistrationComment{}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	//err = templates.TopicRegistrationForm(user, topic, comments, locale).Render(r.Context(), w)
	err = templates.TopicRegistrationModal(user, topic, comments, locale).Render(r.Context(), w)

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
func (h *TopicHandlers) SubmitTopicForReview(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, "Invalid form data")
		return
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

	// Validate
	if err := topicData.Validate(); err != nil {
		h.renderFormError(w, err.Error())
		return
	}

	// Get student record
	studentRecord, err := h.getStudentRecord(user.Email)
	if err != nil {
		h.renderFormError(w, "Student record not found")
		return
	}

	// Check if topic already exists
	existingTopic, err := h.getStudentTopic(user.Email)
	if err != nil && err != sql.ErrNoRows {
		h.renderFormError(w, "Database error")
		return
	}

	var topicID int
	if existingTopic != nil {
		// Update existing topic and submit
		if !existingTopic.CanSubmit() {
			h.renderFormError(w, "Topic cannot be submitted in current status")
			return
		}
		topicID = existingTopic.ID
		err = h.updateTopicAndSubmit(topicID, topicData)
	} else {
		// Create new topic and submit
		topicID, err = h.createTopicAndSubmit(studentRecord.ID, topicData)
	}

	if err != nil {
		h.renderFormError(w, "Failed to submit topic")
		return
	}

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
func (h *TopicHandlers) SupervisorRequestRevision(w http.ResponseWriter, r *http.Request) {
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

	revisionReason := r.FormValue("revision_reason")
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

	err = h.updateTopicStatusWithSupervisor(topicID, "revision_requested", user.Email, revisionReason)
	if err != nil {
		h.renderApprovalError(w, "Failed to request revision")
		return
	}

	h.renderApprovalSuccess(w, "Revision requested")
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

func (h *TopicHandlers) updateTopicAndSubmit(topicID int, data *database.TopicSubmissionData) error {
	query := `
        UPDATE project_topic_registrations 
        SET title = ?, title_en = ?, problem = ?, objective = ?, tasks = ?,
            completion_date = ?, supervisor = ?, status = 'submitted', 
            submitted_at = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `

	_, err := h.db.Exec(query,
		data.Title, data.TitleEn, data.Problem, data.Objective,
		data.Tasks, data.CompletionDate, data.Supervisor,
		time.Now().Unix(), topicID)

	return err
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

// Render helper methods
func (h *TopicHandlers) renderFormError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	html := fmt.Sprintf(`
       <div class="rounded-md bg-destructive/15 p-3">
           <div class="flex">
               <div class="ml-3">
                   <h3 class="text-sm font-medium text-destructive">❌ %s</h3>
               </div>
           </div>
       </div>
   `, message)
	w.Write([]byte(html))
}

func (h *TopicHandlers) renderFormSuccess(w http.ResponseWriter, message string, topicID int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`
       <div class="rounded-md bg-green-50 p-3">
           <div class="flex">
               <div class="ml-3">
                   <h3 class="text-sm font-medium text-green-700">✅ %s</h3>
               </div>
           </div>
       </div>
       <script>
           setTimeout(function() {
               window.location.reload();
           }, 2000);
       </script>
   `, message)
	w.Write([]byte(html))
}

func (h *TopicHandlers) renderApprovalError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	html := fmt.Sprintf(`
       <div class="rounded-md bg-destructive/15 p-3">
           <div class="text-destructive text-sm font-medium">❌ %s</div>
       </div>
   `, message)
	w.Write([]byte(html))
}

func (h *TopicHandlers) renderApprovalSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`
       <div class="rounded-md bg-green-50 p-3">
           <div class="text-green-700 text-sm font-medium">✅ %s</div>
       </div>
       <script>
           setTimeout(function() {
               window.location.reload();
           }, 2000);
       </script>
   `, message)
	w.Write([]byte(html))
}

// ShowTopicRegistrationModal displays the topic registration form in a modal
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
	if topic != nil {
		comments, err = h.getTopicComments(topic.ID)
		if err != nil {
			comments = []database.TopicRegistrationComment{}
		}
	}

	// ✅ PASS THE REAL AUTHENTICATED USER (SUPERVISOR), NOT A FAKE STUDENT USER
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = templates.TopicRegistrationModal(user, topic, comments, locale).Render(r.Context(), w)
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
