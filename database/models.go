// database/models.go - All models in a single file
package database

import (
	"database/sql"
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

	// Topic statuses
	TopicStatusDraft              = "draft"
	TopicStatusSubmitted          = "submitted"
	TopicStatusSupervisorApproved = "supervisor_approved"
	TopicStatusApproved           = "approved"
	TopicStatusRejected           = "rejected"
	TopicStatusRevisionRequested  = "revision_requested"
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

	// Commission access levels
	CommissionAccessViewOnly = "view_only"
	CommissionAccessEvaluate = "evaluate"
	CommissionAccessFull     = "full"

	// Commission types
	CommissionTypeDefense    = "defense"
	CommissionTypeReview     = "review"
	CommissionTypeEvaluation = "evaluation"

	// Document access levels
	DocumentAccessPublic     = "public"
	DocumentAccessCommission = "commission"
	DocumentAccessReviewer   = "reviewer"
	DocumentAccessSupervisor = "supervisor"

	// Evaluation statuses
	EvaluationStatusPending   = "pending"
	EvaluationStatusCompleted = "completed"
	EvaluationStatusApproved  = "approved"

	// Academic audit access types
	AuditAccessCommission = "commission"
	AuditAccessReviewer   = "reviewer"
	AuditAccessSupervisor = "supervisor"
	AuditAccessAdmin      = "admin"
	AuditAccessStudent    = "student"
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

// ================================
// REVIEWER INVITATION MODELS
// ================================

// ReviewerInvitation represents a reviewer invitation with secure access
type ReviewerInvitation struct {
	ID              int       `json:"id" db:"id"`
	StudentRecordID int       `json:"student_record_id" db:"student_record_id"`
	ReviewerEmail   string    `json:"reviewer_email" db:"reviewer_email"`
	ReviewerName    string    `json:"reviewer_name" db:"reviewer_name"`
	AccessTokenHash string    `json:"-" db:"access_token_hash"` // Never expose in JSON
	ExpiresAt       int64     `json:"expires_at" db:"expires_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	EmailSentAt     *int64    `json:"email_sent_at" db:"email_sent_at"`
	FirstAccessAt   *int64    `json:"first_access_at" db:"first_access_at"`
	LastAccessAt    *int64    `json:"last_access_at" db:"last_access_at"`
	AccessCount     int       `json:"access_count" db:"access_count"`
	MaxAccesses     int       `json:"max_accesses" db:"max_accesses"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	IPAddress       *string   `json:"ip_address" db:"ip_address"`
	ReviewCompleted bool      `json:"review_completed" db:"review_completed"`
}

// IsExpired checks if the reviewer invitation has expired
func (ri *ReviewerInvitation) IsExpired() bool {
	return time.Now().Unix() > ri.ExpiresAt
}

// IsAccessLimitReached checks if access limit has been reached
func (ri *ReviewerInvitation) IsAccessLimitReached() bool {
	return ri.MaxAccesses > 0 && ri.AccessCount >= ri.MaxAccesses
}

// CanAccess checks if reviewer can access the system
func (ri *ReviewerInvitation) CanAccess() bool {
	return ri.IsActive && !ri.IsExpired() && !ri.IsAccessLimitReached()
}

// HasAccessed checks if reviewer has accessed the system
func (ri *ReviewerInvitation) HasAccessed() bool {
	return ri.FirstAccessAt != nil
}

// GetExpiresAtFormatted returns formatted expiration date
func (ri *ReviewerInvitation) GetExpiresAtFormatted() string {
	return time.Unix(ri.ExpiresAt, 0).Format("2006-01-02 15:04")
}

// GetEmailSentAtFormatted returns formatted email sent date
func (ri *ReviewerInvitation) GetEmailSentAtFormatted() string {
	if ri.EmailSentAt == nil {
		return "Not sent"
	}
	return time.Unix(*ri.EmailSentAt, 0).Format("2006-01-02 15:04")
}

// GetFirstAccessAtFormatted returns formatted first access date
func (ri *ReviewerInvitation) GetFirstAccessAtFormatted() string {
	if ri.FirstAccessAt == nil {
		return "Never"
	}
	return time.Unix(*ri.FirstAccessAt, 0).Format("2006-01-02 15:04")
}

// GetLastAccessAtFormatted returns formatted last access date
func (ri *ReviewerInvitation) GetLastAccessAtFormatted() string {
	if ri.LastAccessAt == nil {
		return "Never"
	}
	return time.Unix(*ri.LastAccessAt, 0).Format("2006-01-02 15:04")
}

// GetStatusDisplay returns user-friendly status
func (ri *ReviewerInvitation) GetStatusDisplay() string {
	if !ri.IsActive {
		return "Deactivated"
	}
	if ri.IsExpired() {
		return "Expired"
	}
	if ri.IsAccessLimitReached() {
		return "Access Limit Reached"
	}
	if ri.ReviewCompleted {
		return "Review Completed"
	}
	if ri.HasAccessed() {
		return "Accessed"
	}
	if ri.EmailSentAt != nil {
		return "Invitation Sent"
	}
	return "Created"
}

// GetStatusColor returns CSS color class for status
func (ri *ReviewerInvitation) GetStatusColor() string {
	if !ri.IsActive {
		return "text-gray-500"
	}
	if ri.IsExpired() {
		return "text-red-600"
	}
	if ri.IsAccessLimitReached() {
		return "text-orange-600"
	}
	if ri.ReviewCompleted {
		return "text-green-600"
	}
	if ri.HasAccessed() {
		return "text-blue-600"
	}
	if ri.EmailSentAt != nil {
		return "text-yellow-600"
	}
	return "text-gray-400"
}

// ================================
// COMMISSION AUDIT AND EVALUATION MODELS
// ================================

// CommissionAccessLog represents commission member access logging
type CommissionAccessLog struct {
	ID                 int       `json:"id" db:"id"`
	CommissionMemberID int       `json:"commission_member_id" db:"commission_member_id"`
	StudentRecordID    *int      `json:"student_record_id" db:"student_record_id"`
	Action             string    `json:"action" db:"action"`
	ResourceAccessed   *string   `json:"resource_accessed" db:"resource_accessed"`
	IPAddress          *string   `json:"ip_address" db:"ip_address"`
	UserAgent          *string   `json:"user_agent" db:"user_agent"`
	AccessTimestamp    time.Time `json:"access_timestamp" db:"access_timestamp"`
	SessionDuration    *int      `json:"session_duration" db:"session_duration"` // Duration in seconds
}

// GetSessionDurationFormatted returns formatted session duration
func (cal *CommissionAccessLog) GetSessionDurationFormatted() string {
	if cal.SessionDuration == nil {
		return "Unknown"
	}
	return FormatDuration(*cal.SessionDuration)
}

// CommissionEvaluation represents commission member's evaluation of a student defense
type CommissionEvaluation struct {
	ID                 int       `json:"id" db:"id"`
	CommissionMemberID int       `json:"commission_member_id" db:"commission_member_id"`
	StudentRecordID    int       `json:"student_record_id" db:"student_record_id"`
	PresentationScore  float64   `json:"presentation_score" db:"presentation_score"`
	DefenseScore       float64   `json:"defense_score" db:"defense_score"`
	AnswersScore       float64   `json:"answers_score" db:"answers_score"`
	OverallScore       float64   `json:"overall_score" db:"overall_score"`
	Comments           string    `json:"comments" db:"comments"`
	QuestionsAsked     string    `json:"questions_asked" db:"questions_asked"`
	EvaluationStatus   string    `json:"evaluation_status" db:"evaluation_status"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// IsCompleted checks if evaluation is completed
func (ce *CommissionEvaluation) IsCompleted() bool {
	return ce.EvaluationStatus == EvaluationStatusCompleted || ce.EvaluationStatus == EvaluationStatusApproved
}

// GetOverallScoreFormatted returns formatted overall score
func (ce *CommissionEvaluation) GetOverallScoreFormatted() string {
	return fmt.Sprintf("%.1f", ce.OverallScore)
}

// GetScoreLevel returns score level description
func (ce *CommissionEvaluation) GetScoreLevel() string {
	if ce.OverallScore >= 9 {
		return "Excellent"
	} else if ce.OverallScore >= 8 {
		return "Very Good"
	} else if ce.OverallScore >= 7 {
		return "Good"
	} else if ce.OverallScore >= 6 {
		return "Satisfactory"
	} else {
		return "Needs Improvement"
	}
}

// GetScoreColor returns CSS color class for score
func (ce *CommissionEvaluation) GetScoreColor() string {
	if ce.OverallScore >= 9 {
		return "text-green-600"
	} else if ce.OverallScore >= 7 {
		return "text-yellow-600"
	} else if ce.OverallScore >= 6 {
		return "text-orange-600"
	} else {
		return "text-red-600"
	}
}

// GetStatusDisplay returns user-friendly status
func (ce *CommissionEvaluation) GetStatusDisplay() string {
	switch ce.EvaluationStatus {
	case EvaluationStatusPending:
		return "Pending"
	case EvaluationStatusCompleted:
		return "Completed"
	case EvaluationStatusApproved:
		return "Approved"
	default:
		return "Unknown"
	}
}

// GetStatusColor returns CSS color class for status
func (ce *CommissionEvaluation) GetStatusColor() string {
	switch ce.EvaluationStatus {
	case EvaluationStatusPending:
		return "text-yellow-600"
	case EvaluationStatusCompleted:
		return "text-blue-600"
	case EvaluationStatusApproved:
		return "text-green-600"
	default:
		return "text-gray-500"
	}
}

// AcademicAuditLog represents comprehensive audit logging for academic integrity
type AcademicAuditLog struct {
	ID               int       `json:"id" db:"id"`
	AccessType       string    `json:"access_type" db:"access_type"`
	AccessIdentifier string    `json:"access_identifier" db:"access_identifier"`
	StudentRecordID  *int      `json:"student_record_id" db:"student_record_id"`
	Action           string    `json:"action" db:"action"`
	ResourceType     string    `json:"resource_type" db:"resource_type"`
	ResourceID       *string   `json:"resource_id" db:"resource_id"`
	IPAddress        string    `json:"ip_address" db:"ip_address"`
	UserAgent        *string   `json:"user_agent" db:"user_agent"`
	SessionID        *string   `json:"session_id" db:"session_id"`
	Success          bool      `json:"success" db:"success"`
	ErrorMessage     *string   `json:"error_message" db:"error_message"`
	Metadata         *string   `json:"metadata" db:"metadata"` // JSON string for additional context
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// GetAccessTypeDisplay returns user-friendly access type
func (aal *AcademicAuditLog) GetAccessTypeDisplay() string {
	switch aal.AccessType {
	case AuditAccessCommission:
		return "Commission Member"
	case AuditAccessReviewer:
		return "Reviewer"
	case AuditAccessSupervisor:
		return "Supervisor"
	case AuditAccessAdmin:
		return "Administrator"
	case AuditAccessStudent:
		return "Student"
	default:
		return "Unknown"
	}
}

// GetStatusDisplay returns user-friendly status
func (aal *AcademicAuditLog) GetStatusDisplay() string {
	if aal.Success {
		return "Success"
	}
	return "Failed"
}

// GetStatusColor returns CSS color class for status
func (aal *AcademicAuditLog) GetStatusColor() string {
	if aal.Success {
		return "text-green-600"
	}
	return "text-red-600"
}

// GetMetadataMap parses metadata JSON string
func (aal *AcademicAuditLog) GetMetadataMap() map[string]interface{} {
	if aal.Metadata == nil {
		return make(map[string]interface{})
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(*aal.Metadata), &metadata); err != nil {
		return make(map[string]interface{})
	}

	return metadata
}

// ================================
// ENHANCED STUDENT AND ACADEMIC MODELS
// ================================

// StudentRecord represents a student in the system (enhanced)
// StudentRecord represents a student in the system (enhanced)
type StudentRecord struct {
	ID                  int            `json:"id" db:"id"`
	StudentGroup        string         `json:"student_group" db:"student_group"`
	FinalProjectTitle   string         `json:"final_project_title" db:"final_project_title"`
	FinalProjectTitleEn sql.NullString `json:"final_project_title_en" db:"final_project_title_en"` // Changed
	StudentEmail        string         `json:"student_email" db:"student_email"`
	StudentName         string         `json:"student_name" db:"student_name"`
	StudentLastname     string         `json:"student_lastname" db:"student_lastname"`
	StudentNumber       string         `json:"student_number" db:"student_number"`
	SupervisorEmail     string         `json:"supervisor_email" db:"supervisor_email"`
	StudyProgram        string         `json:"study_program" db:"study_program"`
	Department          string         `json:"department" db:"department"`
	ProgramCode         string         `json:"program_code" db:"program_code"`
	CurrentYear         int            `json:"current_year" db:"current_year"`
	ReviewerEmail       sql.NullString `json:"reviewer_email" db:"reviewer_email"` // Changed
	ReviewerName        sql.NullString `json:"reviewer_name" db:"reviewer_name"`   // Changed
	IsFavorite          bool           `json:"is_favorite" db:"is_favorite"`
	IsPublicDefense     bool           `json:"is_public_defense" db:"is_public_defense"`
	DefenseDate         sql.NullTime   `json:"defense_date" db:"defense_date"`
	DefenseLocation     sql.NullString `json:"defense_location" db:"defense_location"` // Changed
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// GetDefenseDateFormatted returns formatted defense date
func (s *StudentRecord) GetDefenseDateFormatted() string {
	if !s.DefenseDate.Valid {
		return "Not scheduled"
	}
	return s.DefenseDate.Time.Format("2006-01-02 15:04")
}

// HasDefenseScheduled checks if defense is scheduled
func (s *StudentRecord) HasDefenseScheduled() bool {
	return s.DefenseDate.Valid
}

// IsDefenseUpcoming checks if defense is upcoming (within next 7 days)
func (s *StudentRecord) IsDefenseUpcoming() bool {
	if !s.DefenseDate.Valid {
		return false
	}
	defenseTime := s.DefenseDate.Time
	now := time.Now()
	return defenseTime.After(now) && defenseTime.Before(now.Add(7*24*time.Hour))
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
	if lang == "en" && s.FinalProjectTitleEn.Valid && s.FinalProjectTitleEn.String != "" {
		return s.FinalProjectTitleEn.String
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

// Document represents an uploaded document (enhanced)
type Document struct {
	ID               int       `json:"id" db:"id"`
	DocumentType     string    `json:"document_type" db:"document_type"`
	FilePath         string    `json:"file_path" db:"file_path"`
	UploadedDate     time.Time `json:"uploaded_date" db:"uploaded_date"`
	StudentRecordID  int       `json:"student_record_id" db:"student_record_id"`
	FileSize         *int64    `json:"file_size" db:"file_size"`
	MimeType         *string   `json:"mime_type" db:"mime_type"`
	OriginalFilename *string   `json:"original_filename" db:"original_filename"`
	IsConfidential   bool      `json:"is_confidential" db:"is_confidential"`
	AccessLevel      string    `json:"access_level" db:"access_level"`
	WatermarkApplied bool      `json:"watermark_applied" db:"watermark_applied"`
	RepositoryURL    *string   `json:"repository_url" db:"repository_url"`
	RepositoryID     *string   `json:"repository_id" db:"repository_id"`
	CommitID         *string   `json:"commit_id" db:"commit_id"`
	SubmissionID     *string   `json:"submission_id" db:"submission_id"`
	ValidationStatus string    `json:"validation_status" db:"validation_status"`
	UploadStatus     string    `json:"upload_status" db:"upload_status"`
	ValidationErrors *string   `json:"validation_errors" db:"validation_errors"`
}

// GetFileSizeFormatted returns formatted file size
func (d *Document) GetFileSizeFormatted() string {
	if d.FileSize == nil {
		return "Unknown"
	}
	return FormatFileSize(*d.FileSize)
}

// GetAccessLevelDisplay returns user-friendly access level
func (d *Document) GetAccessLevelDisplay() string {
	switch d.AccessLevel {
	case DocumentAccessPublic:
		return "Public"
	case DocumentAccessCommission:
		return "Commission Only"
	case DocumentAccessReviewer:
		return "Reviewer Only"
	case DocumentAccessSupervisor:
		return "Supervisor Only"
	default:
		return "Unknown"
	}
}

// GetAccessLevelColor returns CSS color class for access level
func (d *Document) GetAccessLevelColor() string {
	switch d.AccessLevel {
	case DocumentAccessPublic:
		return "text-green-600"
	case DocumentAccessCommission:
		return "text-blue-600"
	case DocumentAccessReviewer:
		return "text-yellow-600"
	case DocumentAccessSupervisor:
		return "text-orange-600"
	default:
		return "text-gray-500"
	}
}

// CanBeAccessedBy checks if document can be accessed by specific access type
func (d *Document) CanBeAccessedBy(accessType string) bool {
	switch d.AccessLevel {
	case DocumentAccessPublic:
		return true
	case DocumentAccessCommission:
		return accessType == AuditAccessCommission || accessType == AuditAccessAdmin
	case DocumentAccessReviewer:
		return accessType == AuditAccessReviewer || accessType == AuditAccessAdmin
	case DocumentAccessSupervisor:
		return accessType == AuditAccessSupervisor || accessType == AuditAccessAdmin
	default:
		return accessType == AuditAccessAdmin
	}
}

// ================================
// ENHANCED REPORT MODELS
// ================================

// ReviewerReport represents a reviewer's report (enhanced)
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
	ReviewerInvitationID        *int      `json:"reviewer_invitation_id" db:"reviewer_invitation_id"`
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

// IsFromInvitation checks if report was created from invitation
func (rr *ReviewerReport) IsFromInvitation() bool {
	return rr.ReviewerInvitationID != nil
}

// ================================
// VIEW MODELS FOR ENHANCED FEATURES
// ================================

// StudentDefenseReadiness represents comprehensive view of student defense readiness
type StudentDefenseReadiness struct {
	StudentRecord

	// Topic information
	TopicApproved      bool `json:"topic_approved" db:"topic_approved"`
	SupervisorApproved bool `json:"supervisor_approved" db:"supervisor_approved"`
	ReviewerApproved   bool `json:"reviewer_approved" db:"reviewer_approved"`
	VideoReady         bool `json:"video_ready" db:"video_ready"`
	DocumentsUploaded  bool `json:"documents_uploaded" db:"documents_uploaded"`
	ReadyForDefense    bool `json:"ready_for_defense" db:"ready_for_defense"`

	// Grades
	ReviewerGrade   *float64 `json:"reviewer_grade" db:"reviewer_grade"`
	SupervisorGrade *int     `json:"supervisor_grade" db:"supervisor_grade"`

	// Topic details
	TopicStatus     string  `json:"topic_status" db:"topic_status"`
	TopicApprovedBy *string `json:"topic_approved_by" db:"topic_approved_by"`
	TopicApprovedAt *int64  `json:"topic_approved_at" db:"topic_approved_at"`
}

// GetDefenseReadinessPercentage returns defense readiness percentage
func (sdr *StudentDefenseReadiness) GetDefenseReadinessPercentage() int {
	total := 5 // topic, supervisor, reviewer, video, documents
	completed := 0

	if sdr.TopicApproved {
		completed++
	}
	if sdr.SupervisorApproved {
		completed++
	}
	if sdr.ReviewerApproved {
		completed++
	}
	if sdr.VideoReady {
		completed++
	}
	if sdr.DocumentsUploaded {
		completed++
	}

	return (completed * 100) / total
}

// GetDefenseReadinessStatus returns overall readiness status
func (sdr *StudentDefenseReadiness) GetDefenseReadinessStatus() string {
	if sdr.ReadyForDefense {
		return "Ready for Defense"
	}

	missing := []string{}
	if !sdr.TopicApproved {
		missing = append(missing, "Topic Approval")
	}
	if !sdr.SupervisorApproved {
		missing = append(missing, "Supervisor Report")
	}
	if !sdr.ReviewerApproved {
		missing = append(missing, "Reviewer Report")
	}
	if !sdr.VideoReady {
		missing = append(missing, "Video")
	}
	if !sdr.DocumentsUploaded {
		missing = append(missing, "Documents")
	}

	if len(missing) == 0 {
		return "Ready for Defense"
	}

	return "Missing: " + strings.Join(missing, ", ")
}

// CommissionEvaluationSummary represents summary of commission evaluations for a student
type CommissionEvaluationSummary struct {
	StudentRecordID      int        `json:"student_record_id" db:"student_record_id"`
	StudentName          string     `json:"student_name" db:"student_name"`
	StudentLastname      string     `json:"student_lastname" db:"student_lastname"`
	FinalProjectTitle    string     `json:"final_project_title" db:"final_project_title"`
	StudyProgram         string     `json:"study_program" db:"study_program"`
	Department           string     `json:"department" db:"department"`
	TotalEvaluations     int        `json:"total_evaluations" db:"total_evaluations"`
	CompletedEvaluations int        `json:"completed_evaluations" db:"completed_evaluations"`
	AverageScore         *float64   `json:"average_score" db:"average_score"`
	LastEvaluationUpdate *time.Time `json:"last_evaluation_update" db:"last_evaluation_update"`
}

// GetAverageScoreFormatted returns formatted average score
func (ces *CommissionEvaluationSummary) GetAverageScoreFormatted() string {
	if ces.AverageScore == nil {
		return "Not available"
	}
	return fmt.Sprintf("%.1f", *ces.AverageScore)
}

// GetEvaluationProgress returns evaluation progress
func (ces *CommissionEvaluationSummary) GetEvaluationProgress() string {
	return fmt.Sprintf("%d/%d", ces.CompletedEvaluations, ces.TotalEvaluations)
}

// IsEvaluationComplete checks if all evaluations are complete
func (ces *CommissionEvaluationSummary) IsEvaluationComplete() bool {
	return ces.TotalEvaluations > 0 && ces.CompletedEvaluations == ces.TotalEvaluations
}

// ReviewerInvitationStatus represents reviewer invitation with status details
type ReviewerInvitationStatus struct {
	ReviewerInvitation

	// Student details
	StudentName       string `json:"student_name" db:"student_name"`
	StudentLastname   string `json:"student_lastname" db:"student_lastname"`
	FinalProjectTitle string `json:"final_project_title" db:"final_project_title"`
	StudyProgram      string `json:"study_program" db:"study_program"`
	Department        string `json:"department" db:"department"`

	// Status flags
	IsExpired       bool `json:"is_expired" db:"is_expired"`
	HasAccessed     bool `json:"has_accessed" db:"has_accessed"`
	ReviewSubmitted bool `json:"review_submitted" db:"review_submitted"`
}

// GetStudentFullName returns student's full name
func (ris *ReviewerInvitationStatus) GetStudentFullName() string {
	return ris.StudentName + " " + ris.StudentLastname
}

// GetOverallStatus returns comprehensive status
func (ris *ReviewerInvitationStatus) GetOverallStatus() string {
	if !ris.IsActive {
		return "Deactivated"
	}
	if ris.IsExpired {
		return "Expired"
	}
	if ris.ReviewSubmitted {
		return "Review Completed"
	}
	if ris.HasAccessed {
		return "In Progress"
	}
	if ris.EmailSentAt != nil {
		return "Invitation Sent"
	}
	return "Created"
}

// ================================
// FORM DATA MODELS FOR NEW FEATURES
// ================================

// ReviewerInvitationFormData represents reviewer invitation form data
type ReviewerInvitationFormData struct {
	StudentRecordID int    `json:"student_record_id" validate:"required"`
	ReviewerEmail   string `json:"reviewer_email" validate:"required,email"`
	ReviewerName    string `json:"reviewer_name" validate:"required,min=2"`
	ExpirationDays  int    `json:"expiration_days" validate:"required,min=1,max=90"`
	MaxAccesses     int    `json:"max_accesses" validate:"min=0,max=100"`
	IPRestriction   string `json:"ip_restriction"`
	Message         string `json:"message"`
}

// Validate validates reviewer invitation form data
func (rifd *ReviewerInvitationFormData) Validate() error {
	if rifd.StudentRecordID <= 0 {
		return fmt.Errorf("student record ID is required")
	}
	if !ValidateEmail(rifd.ReviewerEmail) {
		return fmt.Errorf("valid reviewer email is required")
	}
	if len(rifd.ReviewerName) < 2 {
		return fmt.Errorf("reviewer name must be at least 2 characters")
	}
	if rifd.ExpirationDays < 1 || rifd.ExpirationDays > 90 {
		return fmt.Errorf("expiration days must be between 1 and 90")
	}
	if rifd.MaxAccesses < 0 || rifd.MaxAccesses > 100 {
		return fmt.Errorf("max accesses must be between 0 and 100")
	}
	return nil
}

// CommissionMemberFormData represents commission member form data
type CommissionMemberFormData struct {
	Department           string   `json:"department" validate:"required"`
	StudyProgram         string   `json:"study_program"`
	Year                 int      `json:"year"`
	Description          string   `json:"description"`
	ExpirationDays       int      `json:"expiration_days" validate:"required,min=1,max=365"`
	MaxAccess            int      `json:"max_access" validate:"min=0"`
	AllowedStudentGroups []string `json:"allowed_student_groups"`
	AllowedStudyPrograms []string `json:"allowed_study_programs"`
	AccessLevel          string   `json:"access_level" validate:"required"`
	CommissionType       string   `json:"commission_type" validate:"required"`
}

// Validate validates commission member form data
func (cmfd *CommissionMemberFormData) Validate() error {
	if cmfd.Department == "" {
		return fmt.Errorf("department is required")
	}
	if cmfd.ExpirationDays < 1 || cmfd.ExpirationDays > 365 {
		return fmt.Errorf("expiration days must be between 1 and 365")
	}
	if cmfd.MaxAccess < 0 {
		return fmt.Errorf("max access must be non-negative")
	}
	validAccessLevels := []string{CommissionAccessViewOnly, CommissionAccessEvaluate, CommissionAccessFull}
	validAccessLevel := false
	for _, level := range validAccessLevels {
		if cmfd.AccessLevel == level {
			validAccessLevel = true
			break
		}
	}
	if !validAccessLevel {
		return fmt.Errorf("invalid access level")
	}

	validCommissionTypes := []string{CommissionTypeDefense, CommissionTypeReview, CommissionTypeEvaluation}
	validCommissionType := false
	for _, ctype := range validCommissionTypes {
		if cmfd.CommissionType == ctype {
			validCommissionType = true
			break
		}
	}
	if !validCommissionType {
		return fmt.Errorf("invalid commission type")
	}

	return nil
}

// CommissionEvaluationFormData represents commission evaluation form data
type CommissionEvaluationFormData struct {
	CommissionMemberID int     `json:"commission_member_id" validate:"required"`
	StudentRecordID    int     `json:"student_record_id" validate:"required"`
	PresentationScore  float64 `json:"presentation_score" validate:"required,min=0,max=10"`
	DefenseScore       float64 `json:"defense_score" validate:"required,min=0,max=10"`
	AnswersScore       float64 `json:"answers_score" validate:"required,min=0,max=10"`
	OverallScore       float64 `json:"overall_score" validate:"required,min=0,max=10"`
	Comments           string  `json:"comments" validate:"required,min=10"`
	QuestionsAsked     string  `json:"questions_asked"`
}

// Validate validates commission evaluation form data
func (cefd *CommissionEvaluationFormData) Validate() error {
	if cefd.CommissionMemberID <= 0 {
		return fmt.Errorf("commission member ID is required")
	}
	if cefd.StudentRecordID <= 0 {
		return fmt.Errorf("student record ID is required")
	}

	scores := map[string]float64{
		"presentation_score": cefd.PresentationScore,
		"defense_score":      cefd.DefenseScore,
		"answers_score":      cefd.AnswersScore,
		"overall_score":      cefd.OverallScore,
	}

	for name, score := range scores {
		if score < 0 || score > 10 {
			return fmt.Errorf("%s must be between 0 and 10", name)
		}
	}

	if len(cefd.Comments) < 10 {
		return fmt.Errorf("comments must be at least 10 characters")
	}

	return nil
}

// ================================
// REMAINING EXISTING MODELS (keeping unchanged for compatibility)
// ================================

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
	Details      *string   `json:"details" db:"details"` // Changed from JSONMap to string for compatibility
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
	PreferencesJSON      *string   `json:"preferences_json" db:"preferences_json"` // Changed from JSONMap to string
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
	return FormatDuration(*v.Duration)
}

// IsReady checks if video is ready for viewing
func (v *Video) IsReady() bool {
	return v.Status == "ready"
}

// [Continue with all other existing models...]
// (keeping all the existing ProjectTopicRegistration, SupervisorReport, etc. models unchanged)

// ================================
// PROJECT AND TOPIC MODELS (unchanged for compatibility)
// ================================

// ProjectTopicRegistration represents a topic registration
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
	Status          string    `json:"status" db:"status"` // draft, submitted, supervisor_approved, approved, rejected, revision_requested
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	SubmittedAt     *int64    `json:"submitted_at" db:"submitted_at"`
	CurrentVersion  int       `json:"current_version" db:"current_version"`
	ApprovedBy      *string   `json:"approved_by" db:"approved_by"`
	ApprovedAt      *int64    `json:"approved_at" db:"approved_at"`
	RejectionReason *string   `json:"rejection_reason" db:"rejection_reason"`

	// ADD THESE NEW SUPERVISOR APPROVAL FIELDS
	SupervisorApprovedAt      *int64  `json:"supervisor_approved_at" db:"supervisor_approved_at"`
	SupervisorApprovedBy      *string `json:"supervisor_approved_by" db:"supervisor_approved_by"`
	SupervisorRejectionReason *string `json:"supervisor_rejection_reason" db:"supervisor_rejection_reason"`
}

// [Keep all existing methods for ProjectTopicRegistration unchanged...]

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

// [Keep all existing methods for SupervisorReport unchanged...]

// ================================
// UTILITY FUNCTIONS (enhanced)
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

// NullableInt64 converts zero to nil for database
func NullableInt64(i int64) *int64 {
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
	// More lenient email validation that accepts most common formats
	if email == "" {
		return false
	}

	// Basic check: must have @ and at least one dot after @
	atIndex := strings.Index(email, "@")
	if atIndex < 1 {
		return false
	}

	domain := email[atIndex+1:]
	if !strings.Contains(domain, ".") {
		return false
	}

	// Optional: Use regex for more strict validation
	// emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	// return emailRegex.MatchString(email)

	return true
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

// GenerateAccessCode generates a secure access code
func GenerateAccessCode() string {
	const charset = "ABCDEFGHIJKLMNPQRSTUVWXYZ123456789" // Excluded O and 0 for clarity
	const length = 8

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1) // Ensure different timestamps
	}
	return string(result)
}

// ValidateAccessCode validates commission access code format
func ValidateAccessCode(code string) bool {
	if len(code) != 8 {
		return false
	}
	for _, char := range code {
		if !strings.ContainsRune("ABCDEFGHIJKLMNPQRSTUVWXYZ123456789", char) {
			return false
		}
	}
	return true
}

// ================================
// API RESPONSE MODELS (keeping existing for compatibility)
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

// ================================
// PLACEHOLDER DATABASE FUNCTIONS (keeping existing)
// ================================

// CreateAuditLog creates an audit log entry (placeholder function)
func CreateAuditLog(log AuditLog) error {
	fmt.Printf("Audit Log: %+v\n", log)
	return nil
}

// CreateAcademicAuditLog creates an academic audit log entry
func CreateAcademicAuditLog(log AcademicAuditLog) error {
	fmt.Printf("Academic Audit Log: %+v\n", log)
	return nil
}

// GetStudentRecord retrieves a student record by ID (placeholder function)
func GetStudentRecord(id int) (*StudentRecord, error) {
	return &StudentRecord{
		ID:          id,
		StudentName: "Test Student",
	}, nil
}

// GetSupervisorReport retrieves a supervisor report by student ID (placeholder function)
func GetSupervisorReport(studentID int) (*SupervisorReport, error) {
	return nil, nil
}

// SaveSupervisorReport saves a supervisor report to database (placeholder function)
func SaveSupervisorReport(data *SupervisorReportData) error {
	fmt.Printf("Saving supervisor report: %+v\n", data)
	return nil
}

// ================================
// ADDITIONAL FORM AND DATA MODELS
// ================================

// Create a struct to hold filter options
type FilterOptions struct {
	Groups        []string `json:"groups"`
	StudyPrograms []string `json:"study_programs"`
	Years         []int    `json:"years"`
}

// TemplateFilterParams for template filtering
type TemplateFilterParams struct {
	Page         int    `json:"page"` // Add this if missing
	Limit        int    `json:"limit"`
	Group        string `json:"group"`
	StudyProgram string `json:"study_program"`
	TopicStatus  string `json:"topic_status"`
	Year         int    `json:"year"`
	Search       string `json:"search"`
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

// [Keep all remaining existing models and methods unchanged for full compatibility...]

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
	ReviewerInvitationID        *int    `json:"reviewer_invitation_id"`
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
// SUPERVISOR REPORT FORM TYPES (enhanced)
// ================================

// SupervisorReportFormProps represents the props for the supervisor report form
type SupervisorReportFormProps struct {
	// Student and context data
	StudentRecord StudentRecord     `json:"student_record"`
	InitialReport *SupervisorReport `json:"initial_report,omitempty"`

	// Form configuration
	ButtonLabel string `json:"button_label"`
	ModalTitle  string `json:"modal_title"`
	FormVariant string `json:"form_variant"` // "lt" or "en"

	// Form state
	IsModalOpen bool `json:"is_modal_open"`
	IsSaving    bool `json:"is_saving"`
	IsReadOnly  bool `json:"is_read_only"`

	// Validation errors
	ValidationErrors map[string]string `json:"validation_errors,omitempty"`

	// Current user info (supervisor)
	CurrentSupervisorName  string `json:"current_supervisor_name"`
	CurrentSupervisorEmail string `json:"current_supervisor_email"`
}

// SupervisorReportFormData represents the data being edited in the form
type SupervisorReportFormData struct {
	SupervisorComments  string    `json:"supervisor_comments" form:"supervisor_comments"`
	SupervisorWorkplace string    `json:"supervisor_workplace" form:"supervisor_workplace"`
	SupervisorPosition  string    `json:"supervisor_position" form:"supervisor_position"`
	OtherMatch          float64   `json:"other_match" form:"other_match"`
	OneMatch            float64   `json:"one_match" form:"one_match"`
	OwnMatch            float64   `json:"own_match" form:"own_match"`
	JoinMatch           float64   `json:"join_match" form:"join_match"`
	IsPassOrFailed      bool      `json:"is_pass_or_failed" form:"is_pass_or_failed"`
	Grade               *int      `json:"grade" form:"grade"`
	FinalComments       string    `json:"final_comments" form:"final_comments"`
	SubmissionDate      time.Time `json:"submission_date"`
}

// ToSupervisorReportData converts form data to SupervisorReportData
func (f *SupervisorReportFormData) ToSupervisorReportData(studentRecordID int, supervisorName string) *SupervisorReportData {
	return &SupervisorReportData{
		StudentRecordID:     studentRecordID,
		SupervisorComments:  f.SupervisorComments,
		SupervisorName:      supervisorName,
		SupervisorPosition:  f.SupervisorPosition,
		SupervisorWorkplace: f.SupervisorWorkplace,
		IsPassOrFailed:      f.IsPassOrFailed,
		OtherMatch:          f.OtherMatch,
		OneMatch:            f.OneMatch,
		OwnMatch:            f.OwnMatch,
		JoinMatch:           f.JoinMatch,
		Grade:               f.Grade,
		FinalComments:       f.FinalComments,
	}
}

// ToSupervisorReport converts form data to SupervisorReport model
func (f *SupervisorReportFormData) ToSupervisorReport(studentRecordID int, supervisorName string) *SupervisorReport {
	return &SupervisorReport{
		StudentRecordID:     studentRecordID,
		SupervisorComments:  f.SupervisorComments,
		SupervisorName:      supervisorName,
		SupervisorPosition:  f.SupervisorPosition,
		SupervisorWorkplace: f.SupervisorWorkplace,
		IsPassOrFailed:      f.IsPassOrFailed,
		IsSigned:            false,
		OtherMatch:          f.OtherMatch,
		OneMatch:            f.OneMatch,
		OwnMatch:            f.OwnMatch,
		JoinMatch:           f.JoinMatch,
		Grade:               f.Grade,
		FinalComments:       f.FinalComments,
		CreatedDate:         time.Now(),
		UpdatedDate:         time.Now(),
	}
}

// NewSupervisorReportFormData creates form data from existing report or defaults
func NewSupervisorReportFormData(report *SupervisorReport) *SupervisorReportFormData {
	if report == nil {
		return &SupervisorReportFormData{
			IsPassOrFailed: true,
			SubmissionDate: time.Now(),
			OtherMatch:     0.0,
			OneMatch:       0.0,
			OwnMatch:       0.0,
			JoinMatch:      0.0,
		}
	}

	return &SupervisorReportFormData{
		SupervisorComments:  report.SupervisorComments,
		SupervisorWorkplace: report.SupervisorWorkplace,
		SupervisorPosition:  report.SupervisorPosition,
		OtherMatch:          report.OtherMatch,
		OneMatch:            report.OneMatch,
		OwnMatch:            report.OwnMatch,
		JoinMatch:           report.JoinMatch,
		IsPassOrFailed:      report.IsPassOrFailed,
		Grade:               report.Grade,
		FinalComments:       report.FinalComments,
		SubmissionDate:      time.Now(),
	}
}

// GetTotalSimilarity calculates total similarity percentage
func (f *SupervisorReportFormData) GetTotalSimilarity() float64 {
	return f.OtherMatch + f.OneMatch + f.OwnMatch + f.JoinMatch
}

// GetSimilarityStatus returns similarity assessment
func (f *SupervisorReportFormData) GetSimilarityStatus() string {
	total := f.GetTotalSimilarity()
	if total <= 15 {
		return "Low"
	} else if total <= 25 {
		return "Moderate"
	} else {
		return "High"
	}
}

// GetSimilarityColor returns CSS color class for similarity level
func (f *SupervisorReportFormData) GetSimilarityColor() string {
	total := f.GetTotalSimilarity()
	if total <= 15 {
		return "text-green-600"
	} else if total <= 25 {
		return "text-yellow-600"
	} else {
		return "text-red-600"
	}
}

// ================================
// ENHANCED VIEW MODELS
// ================================

// StudentSummaryView represents a comprehensive view of student data (enhanced)
type StudentSummaryView struct {
	StudentRecord

	// From project_topic_registrations
	TopicApproved bool           `json:"topic_approved" db:"topic_approved"` // derived as CASE (1/0)
	TopicStatus   sql.NullString `json:"topic_status" db:"topic_status"`     // ptr.status
	ApprovedBy    sql.NullString `json:"approved_by" db:"approved_by"`       // ptr.approved_by
	ApprovedAt    sql.NullInt64  `json:"approved_at" db:"approved_at"`       // ptr.approved_at

	// From supervisor_reports
	HasSupervisorReport    bool         `json:"has_supervisor_report" db:"has_supervisor_report"`       // derived as CASE (1/0)
	SupervisorReportSigned sql.NullBool `json:"supervisor_report_signed" db:"supervisor_report_signed"` // sup_rep.is_signed

	// From reviewer_reports
	HasReviewerReport    bool            `json:"has_reviewer_report" db:"has_reviewer_report"`       // derived as CASE (1/0)
	ReviewerReportSigned sql.NullBool    `json:"reviewer_report_signed" db:"reviewer_report_signed"` // rev_rep.is_signed
	ReviewerGrade        sql.NullFloat64 `json:"reviewer_grade" db:"reviewer_grade"`                 // rev_rep.grade
	ReviewerQuestions    sql.NullString  `db:"reviewer_questions" json:"reviewer_questions"`

	// From videos
	HasVideo      bool           `json:"has_video" db:"has_video"` // derived as CASE (1/0)
	HasSourceCode bool           `db:"has_source_code"`            // Add this line
	RepositoryURL sql.NullString `db:"repository_url"`             // Optional: if you want the URL too

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
	if !ssv.SupervisorReportSigned.Valid || !ssv.ReviewerReportSigned.Bool {
		return "Reports Pending Signature"
	}
	if ssv.HasDefenseScheduled() {
		return "Defense Scheduled"
	}
	return "Ready for Defense"
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
	if ssv.SupervisorReportSigned.Valid && ssv.ReviewerReportSigned.Bool {
		completed++
	}

	return (completed * 100) / total
}

// CommissionStudentView for public commission access (enhanced)
type CommissionStudentView struct {
	ID                  int          `json:"id" db:"id"`
	StudentName         string       `json:"student_name" db:"student_name"`
	StudentLastname     string       `json:"student_lastname" db:"student_lastname"`
	StudentGroup        string       `json:"student_group" db:"student_group"`
	ProjectTitle        string       `json:"project_title" db:"project_title"`
	ProjectTitleEn      string       `json:"project_title_en" db:"project_title_en"`
	SupervisorName      string       `json:"supervisor_name" db:"supervisor_name"`
	ReviewerName        string       `json:"reviewer_name" db:"reviewer_name"`
	StudyProgram        string       `json:"study_program" db:"study_program"`
	Department          string       `json:"department" db:"department"`
	HasSupervisorReport bool         `json:"has_supervisor_report" db:"has_supervisor_report"`
	HasReviewerReport   bool         `json:"has_reviewer_report" db:"has_reviewer_report"`
	HasVideo            bool         `json:"has_video" db:"has_video"`
	TopicApproved       bool         `json:"topic_approved" db:"topic_approved"`
	ReviewerGrade       *float64     `json:"reviewer_grade" db:"reviewer_grade"`
	DefenseDate         sql.NullTime `json:"defense_date" db:"defense_date"` // Changed from *int64
	DefenseLocation     string       `json:"defense_location" db:"defense_location"`
	IsPublicDefense     bool         `json:"is_public_defense" db:"is_public_defense"`
}

// GetGradeFormatted returns formatted grade
func (csv *CommissionStudentView) GetGradeFormatted() string {
	if csv.ReviewerGrade == nil {
		return "Not graded"
	}
	return fmt.Sprintf("%.1f", *csv.ReviewerGrade)
}

// GetStudentFullName returns full name
func (csv *CommissionStudentView) GetStudentFullName() string {
	return csv.StudentName + " " + csv.StudentLastname
}

// GetDefenseDateFormatted returns formatted defense date
func (csv *CommissionStudentView) GetDefenseDateFormatted() string {
	if !csv.DefenseDate.Valid {
		return "Not scheduled"
	}
	return csv.DefenseDate.Time.Format("2006-01-02 15:04")
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
	if csv.DefenseDate.Valid { // Changed from != nil to .Valid
		badges = append(badges, "defense-scheduled")
	}

	return badges
}

// ================================
// FILTER MODELS (enhanced)
// ================================

// StudentFilter represents filters for student queries (enhanced)
type StudentFilter struct {
	Department      *string `json:"department"`
	StudyProgram    *string `json:"study_program"`
	Year            *int    `json:"year"`
	Group           *string `json:"group"`
	Supervisor      *string `json:"supervisor"`
	Reviewer        *string `json:"reviewer"`
	Status          *string `json:"status"`
	Search          *string `json:"search"`
	TopicStatus     *string `json:"topic_status"`
	DefenseStatus   *string `json:"defense_status"` // scheduled, completed, pending
	IsPublicDefense *bool   `json:"is_public_defense"`
	Page            int     `json:"page"`
	Limit           int     `json:"limit"`
	SortBy          string  `json:"sort_by"`
	SortOrder       string  `json:"sort_order"`
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

// ReportFilter represents filters for report queries (enhanced)
type ReportFilter struct {
	Department   *string `json:"department"`
	StudyProgram *string `json:"study_program"`
	Year         *int    `json:"year"`
	Supervisor   *string `json:"supervisor"`
	Reviewer     *string `json:"reviewer"`
	IsSigned     *bool   `json:"is_signed"`
	HasGrade     *bool   `json:"has_grade"`
	ReportType   *string `json:"report_type"` // supervisor, reviewer, both
	Search       *string `json:"search"`
	Page         int     `json:"page"`
	Limit        int     `json:"limit"`
	SortBy       string  `json:"sort_by"`
	SortOrder    string  `json:"sort_order"`
}

// CommissionFilter represents filters for commission queries
type CommissionFilter struct {
	Department     *string `json:"department"`
	StudyProgram   *string `json:"study_program"`
	Year           *int    `json:"year"`
	CommissionType *string `json:"commission_type"`
	AccessLevel    *string `json:"access_level"`
	IsActive       *bool   `json:"is_active"`
	IsExpired      *bool   `json:"is_expired"`
	Search         *string `json:"search"`
	Page           int     `json:"page"`
	Limit          int     `json:"limit"`
	SortBy         string  `json:"sort_by"`
	SortOrder      string  `json:"sort_order"`
}

// ReviewerInvitationFilter represents filters for reviewer invitation queries
type ReviewerInvitationFilter struct {
	StudentID       *int    `json:"student_id"`
	ReviewerEmail   *string `json:"reviewer_email"`
	IsActive        *bool   `json:"is_active"`
	IsExpired       *bool   `json:"is_expired"`
	HasAccessed     *bool   `json:"has_accessed"`
	ReviewCompleted *bool   `json:"review_completed"`
	Department      *string `json:"department"`
	StudyProgram    *string `json:"study_program"`
	Search          *string `json:"search"`
	Page            int     `json:"page"`
	Limit           int     `json:"limit"`
	SortBy          string  `json:"sort_by"`
	SortOrder       string  `json:"sort_order"`
}

// ================================
// STATISTICS AND DASHBOARD MODELS (enhanced)
// ================================

// DashboardStats represents dashboard statistics (enhanced)
type DashboardStats struct {
	// Student statistics
	TotalStudents        int `json:"total_students"`
	ActiveStudents       int `json:"active_students"`
	StudentsWithTopics   int `json:"students_with_topics"`
	StudentsReadyDefense int `json:"students_ready_defense"`

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
	ActiveCommissions    int `json:"active_commissions"`
	ExpiredCommissions   int `json:"expired_commissions"`
	TotalEvaluations     int `json:"total_evaluations"`
	CompletedEvaluations int `json:"completed_evaluations"`

	// Reviewer invitation statistics
	TotalInvitations  int `json:"total_invitations"`
	ActiveInvitations int `json:"active_invitations"`
	CompletedReviews  int `json:"completed_reviews"`
	PendingReviews    int `json:"pending_reviews"`

	// Grade statistics
	AverageGrade           float64 `json:"average_grade"`
	PassRate               float64 `json:"pass_rate"`
	AverageCommissionScore float64 `json:"average_commission_score"`

	// Defense statistics
	ScheduledDefenses int `json:"scheduled_defenses"`
	CompletedDefenses int `json:"completed_defenses"`
	PublicDefenses    int `json:"public_defenses"`
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

// ReportStatistics represents report statistics (enhanced)
type ReportStatistics struct {
	TotalStudents          int     `json:"total_students"`
	SupervisorReportsCount int     `json:"supervisor_reports_count"`
	ReviewerReportsCount   int     `json:"reviewer_reports_count"`
	SignedReportsCount     int     `json:"signed_reports_count"`
	AverageReviewerGrade   float64 `json:"average_reviewer_grade"`
	AverageSupervisorGrade float64 `json:"average_supervisor_grade"`
	CompletionRate         float64 `json:"completion_rate"`
	OnTimeSubmissionRate   float64 `json:"on_time_submission_rate"`
}

// CommissionStatistics represents commission statistics
type CommissionStatistics struct {
	TotalCommissions     int     `json:"total_commissions"`
	ActiveCommissions    int     `json:"active_commissions"`
	ExpiredCommissions   int     `json:"expired_commissions"`
	TotalAccesses        int     `json:"total_accesses"`
	UniqueAccessors      int     `json:"unique_accessors"`
	AverageSessionTime   float64 `json:"average_session_time"`
	TotalEvaluations     int     `json:"total_evaluations"`
	CompletedEvaluations int     `json:"completed_evaluations"`
	AverageScore         float64 `json:"average_score"`
}

// ReviewerInvitationStatistics represents reviewer invitation statistics
type ReviewerInvitationStatistics struct {
	TotalInvitations      int     `json:"total_invitations"`
	ActiveInvitations     int     `json:"active_invitations"`
	ExpiredInvitations    int     `json:"expired_invitations"`
	AccessedInvitations   int     `json:"accessed_invitations"`
	CompletedReviews      int     `json:"completed_reviews"`
	AverageAccessTime     float64 `json:"average_access_time"`     // Hours from invitation to first access
	AverageCompletionTime float64 `json:"average_completion_time"` // Hours from first access to completion
	ResponseRate          float64 `json:"response_rate"`           // Percentage of invitations accessed
	CompletionRate        float64 `json:"completion_rate"`         // Percentage of accessed invitations completed
}

// ================================
// RESPONSE MODELS (enhanced)
// ================================

// StudentListResponse represents paginated student response (enhanced)
type StudentListResponse struct {
	Students   []StudentSummaryView `json:"students"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
	Statistics *DashboardStats      `json:"statistics,omitempty"`
}

// TopicListResponse represents paginated topic response
type TopicListResponse struct {
	Topics     []TopicWithDetails `json:"topics"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
	Statistics *TopicStatistics   `json:"statistics,omitempty"`
}

// ReportListResponse represents paginated report response
type ReportListResponse struct {
	Reports    []ReportWithDetails `json:"reports"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
	Statistics *ReportStatistics   `json:"statistics,omitempty"`
}

// CommissionListResponse represents paginated commission response
type CommissionListResponse struct {
	Commissions []CommissionMember    `json:"commissions"`
	Total       int                   `json:"total"`
	Page        int                   `json:"page"`
	Limit       int                   `json:"limit"`
	TotalPages  int                   `json:"total_pages"`
	Statistics  *CommissionStatistics `json:"statistics,omitempty"`
}

// ReviewerInvitationListResponse represents paginated reviewer invitation response
type ReviewerInvitationListResponse struct {
	Invitations []ReviewerInvitationStatus    `json:"invitations"`
	Total       int                           `json:"total"`
	Page        int                           `json:"page"`
	Limit       int                           `json:"limit"`
	TotalPages  int                           `json:"total_pages"`
	Statistics  *ReviewerInvitationStatistics `json:"statistics,omitempty"`
}

// CommissionEvaluationListResponse represents paginated commission evaluation response
type CommissionEvaluationListResponse struct {
	Evaluations []CommissionEvaluation        `json:"evaluations"`
	Total       int                           `json:"total"`
	Page        int                           `json:"page"`
	Limit       int                           `json:"limit"`
	TotalPages  int                           `json:"total_pages"`
	Summary     []CommissionEvaluationSummary `json:"summary,omitempty"`
}

// ================================
// REMAINING MODELS (keeping all existing ones unchanged)
// ================================

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
	CommentType         string    `json:"comment_type" db:"comment_type"`
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

// IsEditable checks if the topic can be edited
func (ptr *ProjectTopicRegistration) IsEditable() bool {
	return ptr.Status == "draft" || ptr.Status == "revision_requested"
}

// IsApproved checks if the topic is approved
func (ptr *ProjectTopicRegistration) IsApproved() bool {
	return ptr.Status == "approved"
}

// IsSubmitted checks if the topic is submitted
func (ptr *ProjectTopicRegistration) IsSubmitted() bool {
	return ptr.Status == "submitted" || ptr.Status == "supervisor_approved" || ptr.Status == "approved" || ptr.Status == "rejected"
}

// NEW WORKFLOW STATE METHODS
func (ptr *ProjectTopicRegistration) CanSubmit() bool {
	return ptr.Status == "draft" || ptr.Status == "revision_requested"
}

func (ptr *ProjectTopicRegistration) CanSupervisorReview() bool {
	return ptr.Status == "submitted"
}

func (ptr *ProjectTopicRegistration) CanDepartmentReview() bool {
	return ptr.Status == "supervisor_approved"
}

// UPDATED GetStatusDisplay method with new workflow states
func (ptr *ProjectTopicRegistration) GetStatusDisplay(locale string) string {
	statusMap := map[string]map[string]string{
		"en": {
			"draft":               "Draft",
			"submitted":           "Submitted - Awaiting Supervisor Review",
			"supervisor_approved": "Supervisor Approved - Awaiting Department Review",
			"approved":            "Approved",
			"rejected":            "Rejected",
			"revision_requested":  "Revision Requested",
		},
		"lt": {
			"draft":               "Juodraštis",
			"submitted":           "Pateikta - Laukia vadovo vertinimo",
			"supervisor_approved": "Vadovas patvirtino - Laukia katedros vertinimo",
			"approved":            "Patvirtinta",
			"rejected":            "Atmesta",
			"revision_requested":  "Reikalauja pataisymų",
		},
	}

	if statusMap[locale] != nil && statusMap[locale][ptr.Status] != "" {
		return statusMap[locale][ptr.Status]
	}
	return ptr.Status
}

// UPDATED GetStatusColor method with new workflow states
func (ptr *ProjectTopicRegistration) GetStatusColor() string {
	switch ptr.Status {
	case "draft":
		return "text-gray-500"
	case "submitted":
		return "text-yellow-600"
	case "supervisor_approved":
		return "text-blue-600"
	case "approved":
		return "text-green-600"
	case "rejected":
		return "text-red-600"
	case "revision_requested":
		return "text-orange-600"
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

// GetGradeDisplay returns formatted grade display (keeping existing SupervisorReport methods)
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

type SourceCodeUpload struct {
	ID               int    `json:"id" db:"id"`
	StudentRecordID  int    `json:"student_record_id" db:"student_record_id"`
	SubmissionID     string `json:"submission_id" db:"submission_id"`
	OriginalFilename string `json:"original_filename" db:"original_filename"`
	FileSize         int64  `json:"file_size" db:"file_size"`

	// Azure DevOps fields
	RepositoryURL string  `json:"repository_url" db:"repository_url"`
	RepositoryID  string  `json:"repository_id" db:"repository_id"`
	CommitID      *string `json:"commit_id" db:"commit_id"`

	// Status fields
	ValidationStatus string  `json:"validation_status" db:"validation_status"`
	UploadStatus     string  `json:"upload_status" db:"upload_status"`
	ValidationErrors *string `json:"validation_errors" db:"validation_errors"`

	// Timestamps
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_date"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type StudentInfo struct {
	Name        string `json:"name" form:"name"`
	StudentID   string `json:"student_id" form:"student_id"`
	Email       string `json:"email" form:"email"`
	ThesisTitle string `json:"thesis_title" form:"thesis_title"`
}

type ValidationResult struct {
	Valid     bool     `json:"valid"`
	Warnings  []string `json:"warnings"`
	Errors    []string `json:"errors"`
	FileCount int      `json:"file_count"`
	TotalSize int64    `json:"total_size"`
}

type RepositoryInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	WebURL    string `json:"web_url"`
	CloneURL  string `json:"clone_url"`
	RemoteURL string `json:"remote_url"`
}

type CommitInfo struct {
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	FilesCount int       `json:"files_count"`
	CommitID   string    `json:"commit_id"`
}

// Add to database/models.go if not present
type FilterInfo struct {
	TotalFilesInZip  int   `json:"total_files_in_zip"`
	FilesAfterFilter int   `json:"files_after_filter"`
	FilesSkipped     int   `json:"files_skipped"`
	OriginalSize     int64 `json:"original_size"`
	SizeAfterFilter  int64 `json:"size_after_filter"`
}

// Update SubmissionResult if FilterInfo not included
type SubmissionResult struct {
	Success        bool              `json:"success"`
	Message        string            `json:"message"`
	Error          string            `json:"error,omitempty"`
	RepositoryInfo *RepositoryInfo   `json:"repository_info,omitempty"`
	Validation     *ValidationResult `json:"validation,omitempty"`
	CommitInfo     *CommitInfo       `json:"commit_info,omitempty"`
	SubmissionID   string            `json:"submission_id"`
	DocumentID     int               `json:"document_id,omitempty"`
	FilterInfo     *FilterInfo       `json:"filter_info,omitempty"`
}

func (d *Document) IsSourceCode() bool {
	return d.DocumentType == "thesis_source_code" || d.DocumentType == "SOURCE_CODE"
}

// GetRepositoryDisplayInfo returns repository information for display
func (d *Document) GetRepositoryDisplayInfo() map[string]string {
	if !d.IsSourceCode() || d.RepositoryURL == nil {
		return nil
	}

	info := make(map[string]string)
	info["url"] = *d.RepositoryURL

	if d.RepositoryID != nil {
		info["id"] = *d.RepositoryID
	}

	if d.CommitID != nil {
		info["commit"] = *d.CommitID
	}

	info["status"] = d.UploadStatus
	info["validation"] = d.ValidationStatus

	return info
}

type StudentDashboardData struct {
	// Basic student information
	StudentRecord *StudentRecord `json:"student_record"`

	// Topic registration information
	TopicRegistration *ProjectTopicRegistration  `json:"topic_registration,omitempty"`
	TopicStatus       string                     `json:"topic_status"`
	TopicComments     []TopicRegistrationComment `json:"topic_comments,omitempty"`

	// Reports information
	SupervisorReport *SupervisorReport `json:"supervisor_report,omitempty"`
	ReviewerReport   *ReviewerReport   `json:"reviewer_report,omitempty"`

	// Documents and uploads - using the specific field names your templates expect
	Documents             []Document `json:"documents,omitempty"`
	Videos                []Video    `json:"videos,omitempty"`
	HasThesisPDF          bool       `json:"has_thesis_pdf"`
	ThesisDocument        *Document  `json:"thesis_document,omitempty"`
	CompanyRecommendation *Document  `json:"company_recommendation,omitempty"`
	VideoPresentation     *Video     `json:"video_presentation,omitempty"`

	// Source code repository - this is what your templates are looking for
	SourceCodeRepository *Document `json:"source_code_repository,omitempty"`
	SourceCodeStatus     string    `json:"source_code_status,omitempty"` // ADD THIS FIELD

	// Status flags and progress
	HasTopicApproved    bool `json:"has_topic_approved"`
	HasSupervisorReport bool `json:"has_supervisor_report"`
	HasReviewerReport   bool `json:"has_reviewer_report"`
	HasAllDocuments     bool `json:"has_all_documents"`
	HasVideo            bool `json:"has_video"`
	IsReadyForDefense   bool `json:"is_ready_for_defense"`
	HasSourceCode       bool `json:"has_source_code"`
	TopicCommentCount   int  `json:"topic_comment_count"`
	HasUnreadComments   bool `json:"has_unread_comments"`
	// Defense information
	DefenseScheduled bool   `json:"defense_scheduled"`
	DefenseDate      string `json:"defense_date,omitempty"`
	DefenseLocation  string `json:"defense_location,omitempty"`

	// Progress tracking
	CompletionPercentage int      `json:"completion_percentage"`
	CurrentStage         string   `json:"current_stage"`
	NextActions          []string `json:"next_actions,omitempty"`

	// Notifications and reminders
	Notifications []StudentNotification `json:"notifications,omitempty"`
	Reminders     []StudentReminder     `json:"reminders,omitempty"`

	// Academic year and semester info
	AcademicYear int    `json:"academic_year"`
	Semester     string `json:"semester"`

	// Supervisor and reviewer information
	SupervisorInfo *SupervisorInfo `json:"supervisor_info,omitempty"`
	ReviewerInfo   *ReviewerInfo   `json:"reviewer_info,omitempty"`

	// Deadlines
	Deadlines []StudentDeadline `json:"deadlines,omitempty"`

	// Source code upload information
	SourceCodeUploads []SourceCodeUpload `json:"source_code_uploads,omitempty"`
}

// StudentNotification represents notifications for students
type StudentNotification struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"` // info, warning, success, error
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	ActionURL string    `json:"action_url,omitempty"`
}

// StudentReminder represents reminders for students
type StudentReminder struct {
	ID          int       `json:"id"`
	Type        string    `json:"type"` // deadline, task, meeting
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Priority    string    `json:"priority"` // low, medium, high
	IsCompleted bool      `json:"is_completed"`
}

// StudentDeadline represents important deadlines
type StudentDeadline struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Type        string    `json:"type"` // topic_submission, document_upload, defense
	IsOverdue   bool      `json:"is_overdue"`
	DaysLeft    int       `json:"days_left"`
}

// SupervisorInfo represents supervisor information
type SupervisorInfo struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Department  string `json:"department"`
	Position    string `json:"position"`
	Office      string `json:"office,omitempty"`
	Phone       string `json:"phone,omitempty"`
	OfficeHours string `json:"office_hours,omitempty"`
}

// ReviewerInfo represents reviewer information
type ReviewerInfo struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Institution  string `json:"institution,omitempty"`
	Expertise    string `json:"expertise,omitempty"`
	ContactInfo  string `json:"contact_info,omitempty"`
	ReviewStatus string `json:"review_status"` // assigned, in_progress, completed
}

// Methods for StudentDashboardData

// GetCompletionPercentage calculates completion percentage
func (sdd *StudentDashboardData) GetCompletionPercentage() int {
	totalSteps := 5 // topic, supervisor report, reviewer report, documents, video
	completed := 0

	if sdd.HasTopicApproved {
		completed++
	}
	if sdd.HasSupervisorReport {
		completed++
	}
	if sdd.HasReviewerReport {
		completed++
	}
	if sdd.HasAllDocuments {
		completed++
	}
	if sdd.HasVideo {
		completed++
	}

	return (completed * 100) / totalSteps
}

// GetCurrentStage returns the current stage of thesis progress
func (sdd *StudentDashboardData) GetCurrentStage() string {
	if !sdd.HasTopicApproved {
		return "Topic Registration"
	}
	if !sdd.HasSupervisorReport {
		return "Supervisor Evaluation"
	}
	if !sdd.HasReviewerReport {
		return "Reviewer Evaluation"
	}
	if !sdd.HasAllDocuments {
		return "Document Submission"
	}
	if !sdd.HasVideo {
		return "Video Presentation"
	}
	if sdd.DefenseScheduled {
		return "Defense Preparation"
	}
	return "Ready for Defense"
}

// GetNextActions returns list of next actions for the student
func (sdd *StudentDashboardData) GetNextActions() []string {
	var actions []string

	if !sdd.HasTopicApproved {
		if sdd.TopicRegistration == nil {
			actions = append(actions, "Submit topic registration")
		} else if sdd.TopicRegistration.Status == "draft" {
			actions = append(actions, "Complete and submit topic registration")
		} else if sdd.TopicRegistration.Status == "submitted" {
			actions = append(actions, "Wait for topic approval")
		} else if sdd.TopicRegistration.Status == "rejected" {
			actions = append(actions, "Revise and resubmit topic registration")
		}
	}

	if sdd.HasTopicApproved && !sdd.HasSupervisorReport {
		actions = append(actions, "Contact supervisor for evaluation")
	}

	if sdd.HasTopicApproved && !sdd.HasReviewerReport {
		actions = append(actions, "Ensure reviewer has been assigned")
	}

	if sdd.HasTopicApproved && !sdd.HasAllDocuments {
		actions = append(actions, "Upload required documents")
	}

	if sdd.HasTopicApproved && !sdd.HasVideo {
		actions = append(actions, "Record and upload presentation video")
	}

	if sdd.HasTopicApproved && !sdd.HasSourceCode {
		actions = append(actions, "Upload source code")
	}

	if sdd.IsReadyForDefense && !sdd.DefenseScheduled {
		actions = append(actions, "Schedule defense date")
	}

	if len(actions) == 0 {
		actions = append(actions, "All requirements completed")
	}

	return actions
}

// GetOverdueReminders returns overdue reminders
func (sdd *StudentDashboardData) GetOverdueReminders() []StudentReminder {
	var overdue []StudentReminder
	now := time.Now()

	for _, reminder := range sdd.Reminders {
		if !reminder.IsCompleted && reminder.DueDate.Before(now) {
			overdue = append(overdue, reminder)
		}
	}

	return overdue
}

// GetUpcomingDeadlines returns deadlines within next 7 days
func (sdd *StudentDashboardData) GetUpcomingDeadlines() []StudentDeadline {
	var upcoming []StudentDeadline
	now := time.Now()
	weekFromNow := now.Add(7 * 24 * time.Hour)

	for _, deadline := range sdd.Deadlines {
		if deadline.DueDate.After(now) && deadline.DueDate.Before(weekFromNow) {
			upcoming = append(upcoming, deadline)
		}
	}

	return upcoming
}

// GetUnreadNotifications returns unread notifications
func (sdd *StudentDashboardData) GetUnreadNotifications() []StudentNotification {
	var unread []StudentNotification

	for _, notification := range sdd.Notifications {
		if !notification.IsRead {
			unread = append(unread, notification)
		}
	}

	return unread
}

// HasCriticalReminders checks if there are any high priority overdue reminders
func (sdd *StudentDashboardData) HasCriticalReminders() bool {
	overdue := sdd.GetOverdueReminders()
	for _, reminder := range overdue {
		if reminder.Priority == "high" {
			return true
		}
	}
	return false
}

// GetRecentSourceCodeUpload returns the most recent source code upload
func (sdd *StudentDashboardData) GetRecentSourceCodeUpload() *SourceCodeUpload {
	if len(sdd.SourceCodeUploads) == 0 {
		return nil
	}

	// Assuming they're sorted by date, return the first one
	// You might want to sort them in the query instead
	return &sdd.SourceCodeUploads[0]
}

// GetTopicApprovalStatus returns topic approval status with details
func (sdd *StudentDashboardData) GetTopicApprovalStatus() map[string]interface{} {
	status := make(map[string]interface{})

	if sdd.TopicRegistration == nil {
		status["status"] = "not_submitted"
		status["message"] = "Topic not yet submitted"
		status["color"] = "gray"
		return status
	}

	switch sdd.TopicRegistration.Status {
	case "draft":
		status["status"] = "draft"
		status["message"] = "Topic saved as draft"
		status["color"] = "gray"
	case "submitted":
		status["status"] = "pending"
		status["message"] = "Topic submitted for review"
		status["color"] = "yellow"
	case "approved":
		status["status"] = "approved"
		status["message"] = "Topic approved"
		status["color"] = "green"
	case "rejected":
		status["status"] = "rejected"
		status["message"] = "Topic rejected - revisions needed"
		status["color"] = "red"
	default:
		status["status"] = "unknown"
		status["message"] = "Unknown status"
		status["color"] = "gray"
	}

	return status
}

// GetDefenseReadiness returns defense readiness information
func (sdd *StudentDashboardData) GetDefenseReadiness() map[string]interface{} {
	readiness := make(map[string]interface{})

	readiness["percentage"] = sdd.GetCompletionPercentage()
	readiness["is_ready"] = sdd.IsReadyForDefense
	readiness["current_stage"] = sdd.GetCurrentStage()
	readiness["next_actions"] = sdd.GetNextActions()

	if sdd.IsReadyForDefense {
		readiness["status"] = "ready"
		readiness["message"] = "Ready for defense scheduling"
		readiness["color"] = "green"
	} else {
		readiness["status"] = "in_progress"
		readiness["message"] = fmt.Sprintf("Progress: %d%% complete", sdd.GetCompletionPercentage())
		readiness["color"] = "blue"
	}

	return readiness
}

// NewStudentDashboardData creates a new StudentDashboardData instance
func NewStudentDashboardData(student *StudentRecord) *StudentDashboardData {
	return &StudentDashboardData{
		StudentRecord:        student,
		TopicStatus:          "not_submitted",
		CompletionPercentage: 0,
		CurrentStage:         "Topic Registration",
		NextActions:          []string{"Submit topic registration"},
		AcademicYear:         time.Now().Year(),
		Semester:             "Spring", // You might want to calculate this
		Notifications:        []StudentNotification{},
		Reminders:            []StudentReminder{},
		Deadlines:            []StudentDeadline{},
		SourceCodeUploads:    []SourceCodeUpload{},
	}
}

// Workflow progress percentage
func (ptr *ProjectTopicRegistration) GetWorkflowProgress() int {
	switch ptr.Status {
	case "draft":
		return 0
	case "submitted":
		return 33
	case "supervisor_approved":
		return 66
	case "approved":
		return 100
	case "rejected", "revision_requested":
		return 0
	default:
		return 0
	}
}

// Check if topic is in final state
func (ptr *ProjectTopicRegistration) IsFinalState() bool {
	return ptr.Status == "approved" || ptr.Status == "rejected"
}

// Get next action for student
func (ptr *ProjectTopicRegistration) GetNextAction(locale string) string {
	actionMap := map[string]map[string]string{
		"en": {
			"draft":               "Complete and submit for review",
			"submitted":           "Wait for supervisor review",
			"supervisor_approved": "Wait for department approval",
			"approved":            "Topic approved - proceed to next phase",
			"rejected":            "Review feedback and resubmit",
			"revision_requested":  "Address feedback and resubmit",
		},
		"lt": {
			"draft":               "Užpildyti ir pateikti vertinimui",
			"submitted":           "Laukti vadovo vertinimo",
			"supervisor_approved": "Laukti katedros patvirtinimo",
			"approved":            "Tema patvirtinta - tęsti toliau",
			"rejected":            "Peržiūrėti atsiliepimus ir pateikti iš naujo",
			"revision_requested":  "Atsižvelgti į pastabas ir pateikti iš naujo",
		},
	}

	if actionMap[locale] != nil && actionMap[locale][ptr.Status] != "" {
		return actionMap[locale][ptr.Status]
	}
	return "Unknown status"
}

type ImportResult struct {
	SuccessCount int             `json:"success_count"`
	ErrorCount   int             `json:"error_count"`
	Errors       []ImportError   `json:"errors"`
	Duplicates   []string        `json:"duplicates"`
	NewStudents  []StudentRecord `json:"new_students"`
}

type ImportError struct {
	Row     int               `json:"row"`
	Field   string            `json:"field"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

type ImportOptions struct {
	OverwriteExisting bool   `json:"overwrite_existing"`
	ValidateEmails    bool   `json:"validate_emails"`
	SendNotifications bool   `json:"send_notifications"`
	ImportedByEmail   string `json:"imported_by_email"`
	AllowedDepartment string `json:"allowed_department"`
	UserRole          string `json:"user_role"`
}

// ReviewerReportFormProps for the template
type ReviewerReportFormProps struct {
	StudentRecord *StudentRecord
	IsReadOnly    bool
	FormVariant   string // "en" or "lt"
	ReviewerName  string
	AccessToken   string // Add this field
}

// ReviewerReportFormData for form data
type ReviewerReportFormData struct {
	ReviewerPersonalDetails     string
	Grade                       float64
	ReviewGoals                 string
	ReviewTheory                string
	ReviewPractical             string
	ReviewTheoryPracticalLink   string
	ReviewResults               string
	ReviewPracticalSignificance string
	ReviewLanguage              string
	ReviewPros                  string
	ReviewCons                  string
	ReviewQuestions             string
}

// COMMISION

type CommissionMember struct {
	ID                   int            `db:"id"`
	AccessCode           string         `db:"access_code"`
	Department           string         `db:"department"`
	StudyProgram         sql.NullString `db:"study_program"`
	Year                 sql.NullInt64  `db:"year"`
	Description          sql.NullString `db:"description"`
	IsActive             bool           `db:"is_active"`
	ExpiresAt            int64          `db:"expires_at"`
	CreatedAt            time.Time      `db:"created_at"`
	LastAccessedAt       sql.NullInt64  `db:"last_accessed_at"`
	CreatedBy            string         `db:"created_by"`
	AccessCount          int            `db:"access_count"`
	MaxAccess            int            `db:"max_access"`
	AllowedStudentGroups sql.NullString `db:"allowed_student_groups"`
	AllowedStudyPrograms sql.NullString `db:"allowed_study_programs"`
	AccessLevel          string         `db:"access_level"`
	CommissionType       string         `db:"commission_type"`
}

// REVIEWER TYPES
type ReviewerAccessToken struct {
	ID             int    `db:"id" json:"id"`
	ReviewerEmail  string `db:"reviewer_email" json:"reviewer_email"`
	ReviewerName   string `db:"reviewer_name" json:"reviewer_name"`
	AccessToken    string `db:"access_token" json:"access_token"`
	Department     string `db:"department" json:"department"`
	CreatedAt      int64  `db:"created_at" json:"created_at"`
	ExpiresAt      int64  `db:"expires_at" json:"expires_at"`
	MaxAccess      int    `db:"max_access" json:"max_access"`
	AccessCount    int    `db:"access_count" json:"access_count"`
	LastAccessedAt *int64 `db:"last_accessed_at" json:"last_accessed_at"`
	IsActive       bool   `db:"is_active" json:"is_active"`
	CreatedBy      string `db:"created_by" json:"created_by"`
}

func (rat *ReviewerAccessToken) IsExpired() bool {
	return time.Now().Unix() > rat.ExpiresAt
}

func (rat *ReviewerAccessToken) CanAccess() bool {
	return rat.IsActive && !rat.IsExpired() &&
		(rat.MaxAccess == 0 || rat.AccessCount < rat.MaxAccess)
}
