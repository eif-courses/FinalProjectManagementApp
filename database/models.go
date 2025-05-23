// database/models.go - All models in a single file
package database

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ================================
// COMMON TYPES AND UTILITIES
// ================================

// JSONMap is a custom type for handling JSON fields in MySQL
type JSONMap map[string]interface{}

// Scan implements the sql.Scanner interface for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Constants for roles, statuses, etc.
const (
	// User roles
	UserRoleAdmin            = "admin"
	UserRoleDepartmentHead   = "department_head"
	UserRoleSupervisor       = "supervisor"
	UserRoleCommissionMember = "commission_member"
	UserRoleStudent          = "student"

	// Topic statuses
	TopicStatusDraft     = "draft"
	TopicStatusSubmitted = "submitted"
	TopicStatusApproved  = "approved"
	TopicStatusRejected  = "rejected"

	// Document types
	DocumentTypeThesis       = "thesis"
	DocumentTypePresentation = "presentation"
	DocumentTypeCode         = "code"
	DocumentTypeOther        = "other"

	// Video statuses
	VideoStatusPending    = "pending"
	VideoStatusProcessing = "processing"
	VideoStatusReady      = "ready"
	VideoStatusError      = "error"
)

// ================================
// USER AND AUTHENTICATION MODELS
// ================================

// DepartmentHead represents a department head user
type DepartmentHead struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`
	SureName     string    `json:"sure_name" db:"sure_name"`
	Department   string    `json:"department" db:"department"`
	DepartmentEn string    `json:"department_en" db:"department_en"`
	JobTitle     string    `json:"job_title" db:"job_title"`
	Role         int       `json:"role" db:"role"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// GetRoleName returns the role name based on role ID
func (dh *DepartmentHead) GetRoleName() string {
	switch dh.Role {
	case 0:
		return "System Administrator"
	case 1:
		return "Department Head"
	case 2:
		return "Deputy Head"
	case 3:
		return "Secretary"
	case 4:
		return "Coordinator"
	default:
		return "Unknown"
	}
}

// GetLocalizedDepartment returns department name in specified language
func (dh *DepartmentHead) GetLocalizedDepartment(lang string) string {
	if lang == "en" && dh.DepartmentEn != "" {
		return dh.DepartmentEn
	}
	return dh.Department
}

// CommissionMember represents a commission member with access token
type CommissionMember struct {
	ID             int       `json:"id" db:"id"`
	AccessCode     string    `json:"access_code" db:"access_code"`
	Department     string    `json:"department" db:"department"`
	StudyProgram   *string   `json:"study_program" db:"study_program"`
	Year           *int      `json:"year" db:"year"`
	Description    string    `json:"description" db:"description"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	ExpiresAt      int64     `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	LastAccessedAt *int64    `json:"last_accessed_at" db:"last_accessed_at"`
	CreatedBy      string    `json:"created_by" db:"created_by"`
	AccessCount    int       `json:"access_count" db:"access_count"`
	MaxAccess      int       `json:"max_access" db:"max_access"`
}

// IsExpired checks if the commission access has expired
func (cm *CommissionMember) IsExpired() bool {
	return time.Now().Unix() > cm.ExpiresAt
}

// IsAccessLimitReached checks if access limit has been reached
func (cm *CommissionMember) IsAccessLimitReached() bool {
	return cm.MaxAccess > 0 && cm.AccessCount >= cm.MaxAccess
}

// GetExpiresAtFormatted returns formatted expiration date
func (cm *CommissionMember) GetExpiresAtFormatted() string {
	return time.Unix(cm.ExpiresAt, 0).Format("2006-01-02 15:04")
}

// UserSession represents a user session
type UserSession struct {
	ID           int       `json:"id" db:"id"`
	SessionID    string    `json:"session_id" db:"session_id"`
	UserEmail    string    `json:"user_email" db:"user_email"`
	UserData     string    `json:"user_data" db:"user_data"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	LastAccessed time.Time `json:"last_accessed" db:"last_accessed"`
	ExpiresAt    int64     `json:"expires_at" db:"expires_at"`
	IPAddress    *string   `json:"ip_address" db:"ip_address"`
	UserAgent    *string   `json:"user_agent" db:"user_agent"`
	IsActive     bool      `json:"is_active" db:"is_active"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           int       `json:"id" db:"id"`
	UserEmail    string    `json:"user_email" db:"user_email"`
	UserRole     string    `json:"user_role" db:"user_role"`
	Action       string    `json:"action" db:"action"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceID   *string   `json:"resource_id" db:"resource_id"`
	Details      JSONMap   `json:"details" db:"details"`
	IPAddress    *string   `json:"ip_address" db:"ip_address"`
	UserAgent    *string   `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	Success      bool      `json:"success" db:"success"`
}

// RolePermission represents a role permission
type RolePermission struct {
	ID           int       `json:"id" db:"id"`
	RoleName     string    `json:"role_name" db:"role_name"`
	Permission   string    `json:"permission" db:"permission"`
	ResourceType *string   `json:"resource_type" db:"resource_type"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	ID                   int       `json:"id" db:"id"`
	UserEmail            string    `json:"user_email" db:"user_email"`
	Language             string    `json:"language" db:"language"`
	Theme                string    `json:"theme" db:"theme"`
	NotificationsEnabled bool      `json:"notifications_enabled" db:"notifications_enabled"`
	EmailNotifications   bool      `json:"email_notifications" db:"email_notifications"`
	Timezone             string    `json:"timezone" db:"timezone"`
	PreferencesJSON      JSONMap   `json:"preferences_json" db:"preferences_json"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// OAuthState represents OAuth state for security
type OAuthState struct {
	ID         int     `json:"id" db:"id"`
	StateValue string  `json:"state_value" db:"state_value"`
	CreatedAt  int64   `json:"created_at" db:"created_at"`
	ExpiresAt  int64   `json:"expires_at" db:"expires_at"`
	Used       bool    `json:"used" db:"used"`
	IPAddress  *string `json:"ip_address" db:"ip_address"`
}

// ================================
// STUDENT AND ACADEMIC MODELS
// ================================

// StudentRecord represents a student in the system
type StudentRecord struct {
	ID                  int       `json:"id" db:"id"`
	StudentGroup        string    `json:"student_group" db:"student_group"`
	FinalProjectTitle   string    `json:"final_project_title" db:"final_project_title"`
	FinalProjectTitleEn string    `json:"final_project_title_en" db:"final_project_title_en"`
	StudentEmail        string    `json:"student_email" db:"student_email"`
	StudentName         string    `json:"student_name" db:"student_name"`
	StudentLastname     string    `json:"student_lastname" db:"student_lastname"`
	StudentNumber       string    `json:"student_number" db:"student_number"`
	SupervisorEmail     string    `json:"supervisor_email" db:"supervisor_email"`
	StudyProgram        string    `json:"study_program" db:"study_program"`
	Department          string    `json:"department" db:"department"`
	ProgramCode         string    `json:"program_code" db:"program_code"`
	CurrentYear         int       `json:"current_year" db:"current_year"`
	ReviewerEmail       string    `json:"reviewer_email" db:"reviewer_email"`
	ReviewerName        string    `json:"reviewer_name" db:"reviewer_name"`
	IsFavorite          bool      `json:"is_favorite" db:"is_favorite"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// GetFullName returns the student's full name
func (s *StudentRecord) GetFullName() string {
	return s.StudentName + " " + s.StudentLastname
}

// GetDisplayGroup returns formatted group display
func (s *StudentRecord) GetDisplayGroup() string {
	return s.StudentGroup + " (" + s.StudyProgram + ")"
}

// GetLocalizedTitle returns project title in specified language
func (s *StudentRecord) GetLocalizedTitle(lang string) string {
	if lang == "en" && s.FinalProjectTitleEn != "" {
		return s.FinalProjectTitleEn
	}
	return s.FinalProjectTitle
}

// GetDisplayName returns formatted name based on language preference
func (s *StudentRecord) GetDisplayName(lang string) string {
	if lang == "en" {
		return s.StudentName + " " + s.StudentLastname
	}
	return s.StudentLastname + " " + s.StudentName
}

// Document represents an uploaded document
type Document struct {
	ID               int       `json:"id" db:"id"`
	DocumentType     string    `json:"document_type" db:"document_type"`
	FilePath         string    `json:"file_path" db:"file_path"`
	UploadedDate     time.Time `json:"uploaded_date" db:"uploaded_date"`
	StudentRecordID  int       `json:"student_record_id" db:"student_record_id"`
	FileSize         *int64    `json:"file_size" db:"file_size"`
	MimeType         *string   `json:"mime_type" db:"mime_type"`
	OriginalFilename *string   `json:"original_filename" db:"original_filename"`
}

// GetFileSizeFormatted returns formatted file size
func (d *Document) GetFileSizeFormatted() string {
	if d.FileSize == nil {
		return "Unknown"
	}

	size := *d.FileSize
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}

// Video represents a video submission
type Video struct {
	ID              int       `json:"id" db:"id"`
	StudentRecordID int       `json:"student_record_id" db:"student_record_id"`
	Key             string    `json:"key" db:"key"`
	Filename        string    `json:"filename" db:"filename"`
	ContentType     string    `json:"content_type" db:"content_type"`
	Size            *int64    `json:"size" db:"size"`
	URL             *string   `json:"url" db:"url"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	Duration        *int      `json:"duration" db:"duration"` // Duration in seconds
	Status          string    `json:"status" db:"status"`     // pending, processing, ready, error
}

// GetDurationFormatted returns formatted duration
func (v *Video) GetDurationFormatted() string {
	if v.Duration == nil {
		return "Unknown"
	}

	seconds := *v.Duration
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// IsReady checks if video is ready for viewing
func (v *Video) IsReady() bool {
	return v.Status == "ready"
}

// StudentSummaryView represents a comprehensive view of student data
type StudentSummaryView struct {
	StudentRecord

	// Topic information
	TopicApproved bool    `json:"topic_approved" db:"topic_approved"`
	TopicStatus   string  `json:"topic_status" db:"topic_status"`
	ApprovedBy    *string `json:"approved_by" db:"approved_by"`
	ApprovedAt    *int64  `json:"approved_at" db:"approved_at"`

	// Report flags
	HasSupervisorReport bool `json:"has_supervisor_report" db:"has_supervisor_report"`
	HasReviewerReport   bool `json:"has_reviewer_report" db:"has_reviewer_report"`
	HasVideo            bool `json:"has_video" db:"has_video"`

	// Report status
	SupervisorReportSigned bool     `json:"supervisor_report_signed" db:"supervisor_report_signed"`
	ReviewerReportSigned   bool     `json:"reviewer_report_signed" db:"reviewer_report_signed"`
	ReviewerGrade          *float64 `json:"reviewer_grade" db:"reviewer_grade"`
}

// GetCompletionStatus returns overall completion status
func (ssv *StudentSummaryView) GetCompletionStatus() string {
	if !ssv.TopicApproved {
		return "Topic Pending"
	}
	if !ssv.HasSupervisorReport {
		return "Supervisor Report Missing"
	}
	if !ssv.HasReviewerReport {
		return "Reviewer Report Missing"
	}
	if !ssv.SupervisorReportSigned || !ssv.ReviewerReportSigned {
		return "Reports Pending Signature"
	}
	return "Complete"
}

// GetCompletionPercentage returns completion percentage
func (ssv *StudentSummaryView) GetCompletionPercentage() int {
	total := 4 // topic, supervisor report, reviewer report, signatures
	completed := 0

	if ssv.TopicApproved {
		completed++
	}
	if ssv.HasSupervisorReport {
		completed++
	}
	if ssv.HasReviewerReport {
		completed++
	}
	if ssv.SupervisorReportSigned && ssv.ReviewerReportSigned {
		completed++
	}

	return (completed * 100) / total
}

// CommissionStudentView for public commission access
type CommissionStudentView struct {
	ID                  int      `json:"id" db:"id"`
	StudentName         string   `json:"student_name" db:"student_name"`
	StudentGroup        string   `json:"student_group" db:"student_group"`
	ProjectTitle        string   `json:"project_title" db:"project_title"`
	ProjectTitleEn      string   `json:"project_title_en" db:"project_title_en"`
	SupervisorName      string   `json:"supervisor_name" db:"supervisor_name"`
	StudyProgram        string   `json:"study_program" db:"study_program"`
	HasSupervisorReport bool     `json:"has_supervisor_report" db:"has_supervisor_report"`
	HasReviewerReport   bool     `json:"has_reviewer_report" db:"has_reviewer_report"`
	HasVideo            bool     `json:"has_video" db:"has_video"`
	TopicApproved       bool     `json:"topic_approved" db:"topic_approved"`
	ReviewerGrade       *float64 `json:"reviewer_grade" db:"reviewer_grade"`
}

// GetGradeFormatted returns formatted grade
func (csv *CommissionStudentView) GetGradeFormatted() string {
	if csv.ReviewerGrade == nil {
		return "Not graded"
	}
	return fmt.Sprintf("%.1f", *csv.ReviewerGrade)
}

// GetStatusBadges returns status badges for display
func (csv *CommissionStudentView) GetStatusBadges() []string {
	var badges []string

	if csv.TopicApproved {
		badges = append(badges, "topic-approved")
	}
	if csv.HasSupervisorReport {
		badges = append(badges, "supervisor-report")
	}
	if csv.HasReviewerReport {
		badges = append(badges, "reviewer-report")
	}
	if csv.HasVideo {
		badges = append(badges, "video")
	}

	return badges
}

// ================================
// TOPIC AND REGISTRATION MODELS
// ================================

// ProjectTopicRegistration represents a topic registration
type ProjectTopicRegistration struct {
	ID              int       `json:"id" db:"id"`
	StudentRecordID int       `json:"student_record_id" db:"student_record_id"`
	Title           string    `json:"title" db:"title"`
	TitleEn         string    `json:"title_en" db:"title_en"`
	Problem         string    `json:"problem" db:"problem"`
	Objective       string    `json:"objective" db:"objective"`
	Tasks           string    `json:"tasks" db:"tasks"`
	CompletionDate  *string   `json:"completion_date" db:"completion_date"`
	Supervisor      string    `json:"supervisor" db:"supervisor"`
	Status          string    `json:"status" db:"status"` // draft, submitted, approved, rejected
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	SubmittedAt     *int64    `json:"submitted_at" db:"submitted_at"`
	CurrentVersion  int       `json:"current_version" db:"current_version"`
	ApprovedBy      *string   `json:"approved_by" db:"approved_by"`
	ApprovedAt      *int64    `json:"approved_at" db:"approved_at"`
	RejectionReason *string   `json:"rejection_reason" db:"rejection_reason"`
}

// IsEditable checks if the topic can be edited
func (ptr *ProjectTopicRegistration) IsEditable() bool {
	return ptr.Status == "draft" || ptr.Status == "rejected"
}

// IsApproved checks if the topic is approved
func (ptr *ProjectTopicRegistration) IsApproved() bool {
	return ptr.Status == "approved"
}

// IsSubmitted checks if the topic is submitted
func (ptr *ProjectTopicRegistration) IsSubmitted() bool {
	return ptr.Status == "submitted" || ptr.Status == "approved" || ptr.Status == "rejected"
}

// GetStatusDisplay returns user-friendly status
func (ptr *ProjectTopicRegistration) GetStatusDisplay() string {
	switch ptr.Status {
	case "draft":
		return "Draft"
	case "submitted":
		return "Submitted for Review"
	case "approved":
		return "Approved"
	case "rejected":
		return "Rejected"
	default:
		return "Unknown"
	}
}

// GetStatusColor returns CSS color class for status
func (ptr *ProjectTopicRegistration) GetStatusColor() string {
	switch ptr.Status {
	case "draft":
		return "text-gray-500"
	case "submitted":
		return "text-yellow-600"
	case "approved":
		return "text-green-600"
	case "rejected":
		return "text-red-600"
	default:
		return "text-gray-500"
	}
}

// GetSubmittedAtFormatted returns formatted submission date
func (ptr *ProjectTopicRegistration) GetSubmittedAtFormatted() string {
	if ptr.SubmittedAt == nil {
		return "Not submitted"
	}
	return time.Unix(*ptr.SubmittedAt, 0).Format("2006-01-02 15:04")
}

// GetApprovedAtFormatted returns formatted approval date
func (ptr *ProjectTopicRegistration) GetApprovedAtFormatted() string {
	if ptr.ApprovedAt == nil {
		return "Not approved"
	}
	return time.Unix(*ptr.ApprovedAt, 0).Format("2006-01-02 15:04")
}

// GetLocalizedTitle returns topic title in specified language
func (ptr *ProjectTopicRegistration) GetLocalizedTitle(lang string) string {
	if lang == "en" && ptr.TitleEn != "" {
		return ptr.TitleEn
	}
	return ptr.Title
}

// TopicRegistrationComment represents a comment on topic registration
type TopicRegistrationComment struct {
	ID                  int       `json:"id" db:"id"`
	TopicRegistrationID int       `json:"topic_registration_id" db:"topic_registration_id"`
	FieldName           *string   `json:"field_name" db:"field_name"`
	CommentText         string    `json:"comment_text" db:"comment_text"`
	AuthorRole          string    `json:"author_role" db:"author_role"`
	AuthorName          string    `json:"author_name" db:"author_name"`
	AuthorEmail         string    `json:"author_email" db:"author_email"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	ParentCommentID     *int      `json:"parent_comment_id" db:"parent_comment_id"`
	IsRead              bool      `json:"is_read" db:"is_read"`
	CommentType         string    `json:"comment_type" db:"comment_type"` // comment, suggestion, approval, rejection
}

// IsReply checks if this is a reply to another comment
func (trc *TopicRegistrationComment) IsReply() bool {
	return trc.ParentCommentID != nil
}

// GetCommentTypeDisplay returns user-friendly comment type
func (trc *TopicRegistrationComment) GetCommentTypeDisplay() string {
	switch trc.CommentType {
	case "comment":
		return "Comment"
	case "suggestion":
		return "Suggestion"
	case "approval":
		return "Approval"
	case "rejection":
		return "Rejection"
	default:
		return "Comment"
	}
}

// GetCommentTypeColor returns CSS color class for comment type
func (trc *TopicRegistrationComment) GetCommentTypeColor() string {
	switch trc.CommentType {
	case "comment":
		return "text-blue-600"
	case "suggestion":
		return "text-yellow-600"
	case "approval":
		return "text-green-600"
	case "rejection":
		return "text-red-600"
	default:
		return "text-gray-600"
	}
}

// ProjectTopicRegistrationVersion represents a version of topic registration
type ProjectTopicRegistrationVersion struct {
	ID                  int       `json:"id" db:"id"`
	TopicRegistrationID int       `json:"topic_registration_id" db:"topic_registration_id"`
	VersionData         string    `json:"version_data" db:"version_data"`
	CreatedBy           string    `json:"created_by" db:"created_by"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	VersionNumber       int       `json:"version_number" db:"version_number"`
	ChangeSummary       string    `json:"change_summary" db:"change_summary"`
}

// GetVersionData parses and returns the version data
func (ptrv *ProjectTopicRegistrationVersion) GetVersionData() (*ProjectTopicRegistration, error) {
	var topicData ProjectTopicRegistration
	if err := json.Unmarshal([]byte(ptrv.VersionData), &topicData); err != nil {
		return nil, err
	}
	return &topicData, nil
}

// TopicWithDetails represents topic with additional details
type TopicWithDetails struct {
	ProjectTopicRegistration

	// Student information
	StudentName  string `json:"student_name" db:"student_name"`
	StudentEmail string `json:"student_email" db:"student_email"`
	StudentGroup string `json:"student_group" db:"student_group"`
	StudyProgram string `json:"study_program" db:"study_program"`

	// Comments and versions
	CommentCount   int `json:"comment_count" db:"comment_count"`
	VersionCount   int `json:"version_count" db:"version_count"`
	UnreadComments int `json:"unread_comments" db:"unread_comments"`
}

// ================================
// REPORT AND EVALUATION MODELS
// ================================

// SupervisorReport represents a supervisor's report
type SupervisorReport struct {
	ID                  int       `json:"id" db:"id"`
	StudentRecordID     int       `json:"student_record_id" db:"student_record_id"`
	SupervisorComments  string    `json:"supervisor_comments" db:"supervisor_comments"`
	SupervisorName      string    `json:"supervisor_name" db:"supervisor_name"`
	SupervisorPosition  string    `json:"supervisor_position" db:"supervisor_position"`
	SupervisorWorkplace string    `json:"supervisor_workplace" db:"supervisor_workplace"`
	IsPassOrFailed      bool      `json:"is_pass_or_failed" db:"is_pass_or_failed"`
	IsSigned            bool      `json:"is_signed" db:"is_signed"`
	OtherMatch          float64   `json:"other_match" db:"other_match"`
	OneMatch            float64   `json:"one_match" db:"one_match"`
	OwnMatch            float64   `json:"own_match" db:"own_match"`
	JoinMatch           float64   `json:"join_match" db:"join_match"`
	CreatedDate         time.Time `json:"created_date" db:"created_date"`
	UpdatedDate         time.Time `json:"updated_date" db:"updated_date"`
	Grade               *int      `json:"grade" db:"grade"`
	FinalComments       string    `json:"final_comments" db:"final_comments"`
}

// GetGradeDisplay returns formatted grade display
func (sr *SupervisorReport) GetGradeDisplay() string {
	if sr.Grade == nil {
		return "Not graded"
	}

	grade := *sr.Grade
	switch grade {
	case 10:
		return "10 - Excellent"
	case 9:
		return "9 - Very Good"
	case 8:
		return "8 - Good"
	case 7:
		return "7 - Satisfactory"
	case 6:
		return "6 - Weak"
	case 5:
		return "5 - Poor"
	default:
		return fmt.Sprintf("%d", grade)
	}
}

// GetPassFailStatus returns pass/fail status
func (sr *SupervisorReport) GetPassFailStatus() string {
	if sr.IsPassOrFailed {
		return "Pass"
	}
	return "Fail"
}

// GetPassFailColor returns CSS color class for pass/fail status
func (sr *SupervisorReport) GetPassFailColor() string {
	if sr.IsPassOrFailed {
		return "text-green-600"
	}
	return "text-red-600"
}

// GetTotalSimilarity returns total similarity percentage
func (sr *SupervisorReport) GetTotalSimilarity() float64 {
	return sr.OtherMatch + sr.OneMatch + sr.OwnMatch + sr.JoinMatch
}

// GetSimilarityStatus returns similarity status
func (sr *SupervisorReport) GetSimilarityStatus() string {
	total := sr.GetTotalSimilarity()
	if total <= 15 {
		return "Low"
	} else if total <= 25 {
		return "Moderate"
	} else {
		return "High"
	}
}

// GetSimilarityColor returns CSS color class for similarity
func (sr *SupervisorReport) GetSimilarityColor() string {
	total := sr.GetTotalSimilarity()
	if total <= 15 {
		return "text-green-600"
	} else if total <= 25 {
		return "text-yellow-600"
	} else {
		return "text-red-600"
	}
}

// ReviewerReport represents a reviewer's report
type ReviewerReport struct {
	ID                          int       `json:"id" db:"id"`
	StudentRecordID             int       `json:"student_record_id" db:"student_record_id"`
	ReviewerPersonalDetails     string    `json:"reviewer_personal_details" db:"reviewer_personal_details"`
	Grade                       float64   `json:"grade" db:"grade"`
	ReviewGoals                 string    `json:"review_goals" db:"review_goals"`
	ReviewTheory                string    `json:"review_theory" db:"review_theory"`
	ReviewPractical             string    `json:"review_practical" db:"review_practical"`
	ReviewTheoryPracticalLink   string    `json:"review_theory_practical_link" db:"review_theory_practical_link"`
	ReviewResults               string    `json:"review_results" db:"review_results"`
	ReviewPracticalSignificance *string   `json:"review_practical_significance" db:"review_practical_significance"`
	ReviewLanguage              string    `json:"review_language" db:"review_language"`
	ReviewPros                  string    `json:"review_pros" db:"review_pros"`
	ReviewCons                  string    `json:"review_cons" db:"review_cons"`
	ReviewQuestions             string    `json:"review_questions" db:"review_questions"`
	IsSigned                    bool      `json:"is_signed" db:"is_signed"`
	CreatedDate                 time.Time `json:"created_date" db:"created_date"`
	UpdatedDate                 time.Time `json:"updated_date" db:"updated_date"`
}

// GetGradeDisplay returns formatted grade display
func (rr *ReviewerReport) GetGradeDisplay() string {
	if rr.Grade == 0 {
		return "Not graded"
	}
	return fmt.Sprintf("%.1f", rr.Grade)
}

// GetGradeColor returns CSS color class for grade
func (rr *ReviewerReport) GetGradeColor() string {
	if rr.Grade >= 9 {
		return "text-green-600"
	} else if rr.Grade >= 7 {
		return "text-yellow-600"
	} else if rr.Grade >= 6 {
		return "text-orange-600"
	} else {
		return "text-red-600"
	}
}

// GetGradeLevel returns grade level description
func (rr *ReviewerReport) GetGradeLevel() string {
	if rr.Grade >= 9 {
		return "Excellent"
	} else if rr.Grade >= 8 {
		return "Very Good"
	} else if rr.Grade >= 7 {
		return "Good"
	} else if rr.Grade >= 6 {
		return "Satisfactory"
	} else {
		return "Needs Improvement"
	}
}

// IsPositive checks if the review is positive
func (rr *ReviewerReport) IsPositive() bool {
	return rr.Grade >= 6
}

// ReportWithDetails represents a report with student details
type ReportWithDetails struct {
	// Student information
	StudentID    int    `json:"student_id" db:"student_id"`
	StudentName  string `json:"student_name" db:"student_name"`
	StudentEmail string `json:"student_email" db:"student_email"`
	StudentGroup string `json:"student_group" db:"student_group"`
	StudyProgram string `json:"study_program" db:"study_program"`
	ProjectTitle string `json:"project_title" db:"project_title"`

	// Report information
	SupervisorReport *SupervisorReport `json:"supervisor_report,omitempty"`
	ReviewerReport   *ReviewerReport   `json:"reviewer_report,omitempty"`

	// Status flags
	HasSupervisorReport bool `json:"has_supervisor_report"`
	HasReviewerReport   bool `json:"has_reviewer_report"`
	BothReportsSigned   bool `json:"both_reports_signed"`
}

// GetOverallGrade calculates overall grade from both reports
func (rwd *ReportWithDetails) GetOverallGrade() *float64 {
	if rwd.SupervisorReport == nil || rwd.ReviewerReport == nil {
		return nil
	}

	if rwd.SupervisorReport.Grade == nil {
		return nil
	}

	// Calculate weighted average (supervisor 40%, reviewer 60%)
	supervisorGrade := float64(*rwd.SupervisorReport.Grade)
	reviewerGrade := rwd.ReviewerReport.Grade

	overall := (supervisorGrade * 0.4) + (reviewerGrade * 0.6)
	return &overall
}

// GetOverallGradeDisplay returns formatted overall grade
func (rwd *ReportWithDetails) GetOverallGradeDisplay() string {
	grade := rwd.GetOverallGrade()
	if grade == nil {
		return "Not available"
	}
	return fmt.Sprintf("%.1f", *grade)
}

// IsComplete checks if all reports are complete
func (rwd *ReportWithDetails) IsComplete() bool {
	return rwd.HasSupervisorReport &&
		rwd.HasReviewerReport &&
		rwd.BothReportsSigned
}

// ================================
// FILTER AND PAGINATION MODELS
// ================================

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
	NextPage   int  `json:"next_page"`
	PrevPage   int  `json:"prev_page"`
}

// NewPaginationInfo creates pagination info
func NewPaginationInfo(page, limit, total int) *PaginationInfo {
	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginationInfo{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
		NextPage:   page + 1,
		PrevPage:   page - 1,
	}
}

// FilterParams represents common filter parameters
type FilterParams struct {
	Search    string `json:"search" form:"search"`
	Page      int    `json:"page" form:"page"`
	Limit     int    `json:"limit" form:"limit"`
	SortBy    string `json:"sort_by" form:"sort_by"`
	SortOrder string `json:"sort_order" form:"sort_order"`
}

// Normalize normalizes filter parameters
func (fp *FilterParams) Normalize() {
	if fp.Page <= 0 {
		fp.Page = 1
	}
	if fp.Limit <= 0 || fp.Limit > 100 {
		fp.Limit = 20
	}
	if fp.SortOrder != "asc" && fp.SortOrder != "desc" {
		fp.SortOrder = "asc"
	}
}

// GetOffset calculates offset for pagination
func (fp *FilterParams) GetOffset() int {
	return (fp.Page - 1) * fp.Limit
}

// StudentFilter represents filters for student queries
type StudentFilter struct {
	Department   *string `json:"department"`
	StudyProgram *string `json:"study_program"`
	Year         *int    `json:"year"`
	Group        *string `json:"group"`
	Supervisor   *string `json:"supervisor"`
	Status       *string `json:"status"`
	Search       *string `json:"search"`
	Page         int     `json:"page"`
	Limit        int     `json:"limit"`
	SortBy       string  `json:"sort_by"`
	SortOrder    string  `json:"sort_order"`
}

// TopicFilter represents filters for topic queries
type TopicFilter struct {
	Status       *string `json:"status"`
	Supervisor   *string `json:"supervisor"`
	Department   *string `json:"department"`
	StudyProgram *string `json:"study_program"`
	Year         *int    `json:"year"`
	Search       *string `json:"search"`
	Page         int     `json:"page"`
	Limit        int     `json:"limit"`
	SortBy       string  `json:"sort_by"`
	SortOrder    string  `json:"sort_order"`
}

// ReportFilter represents filters for report queries
type ReportFilter struct {
	Department   *string `json:"department"`
	StudyProgram *string `json:"study_program"`
	Year         *int    `json:"year"`
	Supervisor   *string `json:"supervisor"`
	Reviewer     *string `json:"reviewer"`
	IsSigned     *bool   `json:"is_signed"`
	HasGrade     *bool   `json:"has_grade"`
	Search       *string `json:"search"`
	Page         int     `json:"page"`
	Limit        int     `json:"limit"`
	SortBy       string  `json:"sort_by"`
	SortOrder    string  `json:"sort_order"`
}

// ================================
// API RESPONSE MODELS
// ================================

// APIResponse represents a standard API response
type APIResponse struct {
	Success    bool            `json:"success"`
	Message    string          `json:"message,omitempty"`
	Data       interface{}     `json:"data,omitempty"`
	Error      string          `json:"error,omitempty"`
	Pagination *PaginationInfo `json:"pagination,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// NewSuccessResponse creates a success API response
func NewSuccessResponse(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(err error, message string) *APIResponse {
	return &APIResponse{
		Success:   false,
		Message:   message,
		Error:     err.Error(),
		Timestamp: time.Now(),
	}
}

// NewPaginatedResponse creates a paginated API response
func NewPaginatedResponse(data interface{}, pagination *PaginationInfo, message string) *APIResponse {
	return &APIResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
		Timestamp:  time.Now(),
	}
}

// StudentListResponse represents paginated student response
type StudentListResponse struct {
	Students   []StudentSummaryView `json:"students"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

// TopicListResponse represents paginated topic response
type TopicListResponse struct {
	Topics     []TopicWithDetails `json:"topics"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// ReportListResponse represents paginated report response
type ReportListResponse struct {
	Reports    []ReportWithDetails `json:"reports"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

// ================================
// STATISTICS AND DASHBOARD MODELS
// ================================

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	// Student statistics
	TotalStudents      int `json:"total_students"`
	ActiveStudents     int `json:"active_students"`
	StudentsWithTopics int `json:"students_with_topics"`

	// Topic statistics
	TotalTopics    int `json:"total_topics"`
	PendingTopics  int `json:"pending_topics"`
	ApprovedTopics int `json:"approved_topics"`
	RejectedTopics int `json:"rejected_topics"`

	// Report statistics
	SupervisorReports int `json:"supervisor_reports"`
	ReviewerReports   int `json:"reviewer_reports"`
	SignedReports     int `json:"signed_reports"`

	// Commission statistics
	ActiveCommissions  int `json:"active_commissions"`
	ExpiredCommissions int `json:"expired_commissions"`

	// Grade statistics
	AverageGrade float64 `json:"average_grade"`
	PassRate     float64 `json:"pass_rate"`
}

// TopicStatistics represents topic statistics
type TopicStatistics struct {
	TotalTopics     int `json:"total_topics"`
	DraftTopics     int `json:"draft_topics"`
	SubmittedTopics int `json:"submitted_topics"`
	ApprovedTopics  int `json:"approved_topics"`
	RejectedTopics  int `json:"rejected_topics"`
	PendingApproval int `json:"pending_approval"`
}

// ReportStatistics represents report statistics
type ReportStatistics struct {
	TotalStudents          int     `json:"total_students"`
	SupervisorReportsCount int     `json:"supervisor_reports_count"`
	ReviewerReportsCount   int     `json:"reviewer_reports_count"`
	SignedReportsCount     int     `json:"signed_reports_count"`
	AverageGrade           float64 `json:"average_grade"`
	CompletionRate         float64 `json:"completion_rate"`
}

// ================================
// FORM DATA MODELS
// ================================

// TopicSubmissionData represents topic submission data
type TopicSubmissionData struct {
	Title          string `json:"title" validate:"required,min=10,max=200"`
	TitleEn        string `json:"title_en" validate:"required,min=10,max=200"`
	Problem        string `json:"problem" validate:"required,min=50"`
	Objective      string `json:"objective" validate:"required,min=30"`
	Tasks          string `json:"tasks" validate:"required,min=50"`
	CompletionDate string `json:"completion_date"`
	Supervisor     string `json:"supervisor" validate:"required"`
}

// Validate validates the topic submission data
func (tsd *TopicSubmissionData) Validate() error {
	if len(tsd.Title) < 10 || len(tsd.Title) > 200 {
		return fmt.Errorf("title must be between 10 and 200 characters")
	}
	if len(tsd.TitleEn) < 10 || len(tsd.TitleEn) > 200 {
		return fmt.Errorf("english title must be between 10 and 200 characters")
	}
	if len(tsd.Problem) < 50 {
		return fmt.Errorf("problem statement must be at least 50 characters")
	}
	if len(tsd.Objective) < 30 {
		return fmt.Errorf("objective must be at least 30 characters")
	}
	if len(tsd.Tasks) < 50 {
		return fmt.Errorf("tasks must be at least 50 characters")
	}
	if tsd.Supervisor == "" {
		return fmt.Errorf("supervisor is required")
	}
	return nil
}

// SupervisorReportData represents supervisor report submission data
type SupervisorReportData struct {
	StudentRecordID     int     `json:"student_record_id" validate:"required"`
	SupervisorComments  string  `json:"supervisor_comments" validate:"required,min=50"`
	SupervisorName      string  `json:"supervisor_name" validate:"required"`
	SupervisorPosition  string  `json:"supervisor_position" validate:"required"`
	SupervisorWorkplace string  `json:"supervisor_workplace" validate:"required"`
	IsPassOrFailed      bool    `json:"is_pass_or_failed"`
	OtherMatch          float64 `json:"other_match" validate:"min=0,max=100"`
	OneMatch            float64 `json:"one_match" validate:"min=0,max=100"`
	OwnMatch            float64 `json:"own_match" validate:"min=0,max=100"`
	JoinMatch           float64 `json:"join_match" validate:"min=0,max=100"`
	Grade               *int    `json:"grade" validate:"omitempty,min=1,max=10"`
	FinalComments       string  `json:"final_comments"`
}

// Validate validates supervisor report data
func (srd *SupervisorReportData) Validate() error {
	if len(srd.SupervisorComments) < 50 {
		return fmt.Errorf("supervisor comments must be at least 50 characters")
	}
	if srd.SupervisorName == "" {
		return fmt.Errorf("supervisor name is required")
	}
	if srd.SupervisorPosition == "" {
		return fmt.Errorf("supervisor position is required")
	}
	if srd.SupervisorWorkplace == "" {
		return fmt.Errorf("supervisor workplace is required")
	}

	// Validate similarity percentages
	if srd.OtherMatch < 0 || srd.OtherMatch > 100 {
		return fmt.Errorf("other match percentage must be between 0 and 100")
	}
	if srd.OneMatch < 0 || srd.OneMatch > 100 {
		return fmt.Errorf("one match percentage must be between 0 and 100")
	}
	if srd.OwnMatch < 0 || srd.OwnMatch > 100 {
		return fmt.Errorf("own match percentage must be between 0 and 100")
	}
	if srd.JoinMatch < 0 || srd.JoinMatch > 100 {
		return fmt.Errorf("join match percentage must be between 0 and 100")
	}

	if srd.Grade != nil && (*srd.Grade < 1 || *srd.Grade > 10) {
		return fmt.Errorf("grade must be between 1 and 10")
	}

	return nil
}

// ReviewerReportData represents reviewer report submission data
type ReviewerReportData struct {
	StudentRecordID             int     `json:"student_record_id" validate:"required"`
	ReviewerPersonalDetails     string  `json:"reviewer_personal_details" validate:"required"`
	Grade                       float64 `json:"grade" validate:"required,min=1,max=10"`
	ReviewGoals                 string  `json:"review_goals" validate:"required,min=30"`
	ReviewTheory                string  `json:"review_theory" validate:"required,min=30"`
	ReviewPractical             string  `json:"review_practical" validate:"required,min=30"`
	ReviewTheoryPracticalLink   string  `json:"review_theory_practical_link" validate:"required,min=30"`
	ReviewResults               string  `json:"review_results" validate:"required,min=30"`
	ReviewPracticalSignificance *string `json:"review_practical_significance"`
	ReviewLanguage              string  `json:"review_language" validate:"required,min=20"`
	ReviewPros                  string  `json:"review_pros" validate:"required,min=20"`
	ReviewCons                  string  `json:"review_cons" validate:"required,min=20"`
	ReviewQuestions             string  `json:"review_questions" validate:"required,min=20"`
}

// Validate validates reviewer report data
func (rrd *ReviewerReportData) Validate() error {
	if rrd.ReviewerPersonalDetails == "" {
		return fmt.Errorf("reviewer personal details are required")
	}
	if rrd.Grade < 1 || rrd.Grade > 10 {
		return fmt.Errorf("grade must be between 1 and 10")
	}

	// Validate required fields with minimum length
	fields := map[string]string{
		"review_goals":                 rrd.ReviewGoals,
		"review_theory":                rrd.ReviewTheory,
		"review_practical":             rrd.ReviewPractical,
		"review_theory_practical_link": rrd.ReviewTheoryPracticalLink,
		"review_results":               rrd.ReviewResults,
		"review_language":              rrd.ReviewLanguage,
		"review_pros":                  rrd.ReviewPros,
		"review_cons":                  rrd.ReviewCons,
		"review_questions":             rrd.ReviewQuestions,
	}

	for fieldName, fieldValue := range fields {
		if len(fieldValue) < 20 {
			return fmt.Errorf("%s must be at least 20 characters", fieldName)
		}
	}

	return nil
}

// TopicApprovalRequest represents a topic approval request
type TopicApprovalRequest struct {
	TopicID         int    `json:"topic_id"`
	ApproverEmail   string `json:"approver_email"`
	ApproverName    string `json:"approver_name"`
	Comments        string `json:"comments"`
	Approved        bool   `json:"approved"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// ================================
// UTILITY FUNCTIONS
// ================================

// NullableString converts empty string to nil for database
func NullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// NullableInt converts zero to nil for database
func NullableInt(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// StringValue safely returns string value from pointer
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IntValue safely returns int value from pointer
func IntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// Validation helper functions
func ValidateEmail(email string) bool {
	return len(email) > 3 &&
		strings.Contains(email, "@") &&
		strings.Contains(email, ".")
}

func ValidateGrade(grade float64) bool {
	return grade >= 1 && grade <= 10
}

func ValidateYear(year int) bool {
	currentYear := time.Now().Year()
	return year >= currentYear-10 && year <= currentYear+5
}

// FormatFileSize formats file size in human readable format
func FormatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
	}
}

// FormatDuration formats duration in human readable format
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// GetGradeText returns grade text representation
func GetGradeText(grade int) string {
	switch grade {
	case 10:
		return "Excellent"
	case 9:
		return "Very Good"
	case 8:
		return "Good"
	case 7:
		return "Satisfactory"
	case 6:
		return "Weak"
	case 5:
		return "Poor"
	default:
		return "Unknown"
	}
}
