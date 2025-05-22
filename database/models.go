package database

import (
	"github.com/jmoiron/sqlx"
)

// DepartmentHead represents the department_heads table
type DepartmentHead struct {
	ID           int    `db:"id" json:"id"`
	Email        string `db:"email" json:"email"`
	Name         string `db:"name" json:"name"`
	SureName     string `db:"sure_name" json:"sure_name"`
	Department   string `db:"department" json:"department"`
	DepartmentEn string `db:"department_en" json:"department_en"`
	JobTitle     string `db:"job_title" json:"job_title"`
	Role         int    `db:"role" json:"role"`
	IsActive     int    `db:"is_active" json:"is_active"`
	CreatedAt    int64  `db:"created_at" json:"created_at"`
}

// CommissionMember represents the commission_members table
type CommissionMember struct {
	ID             int    `db:"id" json:"id"`
	AccessCode     string `db:"access_code" json:"access_code"`
	Department     string `db:"department" json:"department"`
	IsActive       int    `db:"is_active" json:"is_active"`
	ExpiresAt      int64  `db:"expires_at" json:"expires_at"`
	CreatedAt      int64  `db:"created_at" json:"created_at"`
	LastAccessedAt *int64 `db:"last_accessed_at" json:"last_accessed_at"`
}

// StudentRecord represents the student_records table
type StudentRecord struct {
	ID                  int    `db:"id" json:"id"`
	StudentGroup        string `db:"student_group" json:"student_group"`
	FinalProjectTitle   string `db:"final_project_title" json:"final_project_title"`
	FinalProjectTitleEn string `db:"final_project_title_en" json:"final_project_title_en"`
	StudentEmail        string `db:"student_email" json:"student_email"`
	StudentName         string `db:"student_name" json:"student_name"`
	StudentLastname     string `db:"student_lastname" json:"student_lastname"`
	StudentNumber       string `db:"student_number" json:"student_number"`
	SupervisorEmail     string `db:"supervisor_email" json:"supervisor_email"`
	StudyProgram        string `db:"study_program" json:"study_program"`
	Department          string `db:"department" json:"department"`
	ProgramCode         string `db:"program_code" json:"program_code"`
	CurrentYear         int    `db:"current_year" json:"current_year"`
	ReviewerEmail       string `db:"reviewer_email" json:"reviewer_email"`
	ReviewerName        string `db:"reviewer_name" json:"reviewer_name"`
	IsFavorite          int    `db:"is_favorite" json:"is_favorite"`
}

// Document represents the documents table
type Document struct {
	ID              int    `db:"id" json:"id"`
	DocumentType    string `db:"document_type" json:"document_type"`
	FilePath        string `db:"file_path" json:"file_path"`
	UploadedDate    int64  `db:"uploaded_date" json:"uploaded_date"`
	StudentRecordID int    `db:"student_record_id" json:"student_record_id"`
}

// SupervisorReport represents the supervisor_reports table
type SupervisorReport struct {
	ID                  int     `db:"id" json:"id"`
	StudentRecordID     int     `db:"student_record_id" json:"student_record_id"`
	SupervisorComments  string  `db:"supervisor_comments" json:"supervisor_comments"`
	SupervisorName      string  `db:"supervisor_name" json:"supervisor_name"`
	SupervisorPosition  string  `db:"supervisor_position" json:"supervisor_position"`
	SupervisorWorkplace string  `db:"supervisor_workplace" json:"supervisor_workplace"`
	IsPassOrFailed      int     `db:"is_pass_or_failed" json:"is_pass_or_failed"`
	IsSigned            int     `db:"is_signed" json:"is_signed"`
	OtherMatch          float64 `db:"other_match" json:"other_match"`
	OneMatch            float64 `db:"one_match" json:"one_match"`
	OwnMatch            float64 `db:"own_match" json:"own_match"`
	JoinMatch           float64 `db:"join_match" json:"join_match"`
	CreatedDate         int64   `db:"created_date" json:"created_date"`
}

// ReviewerReport represents the reviewer_reports table
type ReviewerReport struct {
	ID                          int     `db:"id" json:"id"`
	StudentRecordID             int     `db:"student_record_id" json:"student_record_id"`
	ReviewerPersonalDetails     string  `db:"reviewer_personal_details" json:"reviewer_personal_details"`
	Grade                       float64 `db:"grade" json:"grade"`
	ReviewGoals                 string  `db:"review_goals" json:"review_goals"`
	ReviewTheory                string  `db:"review_theory" json:"review_theory"`
	ReviewPractical             string  `db:"review_practical" json:"review_practical"`
	ReviewTheoryPracticalLink   string  `db:"review_theory_practical_link" json:"review_theory_practical_link"`
	ReviewResults               string  `db:"review_results" json:"review_results"`
	ReviewPracticalSignificance *string `db:"review_practical_significance" json:"review_practical_significance"`
	ReviewLanguage              string  `db:"review_language" json:"review_language"`
	ReviewPros                  string  `db:"review_pros" json:"review_pros"`
	ReviewCons                  string  `db:"review_cons" json:"review_cons"`
	ReviewQuestions             string  `db:"review_questions" json:"review_questions"`
	IsSigned                    int     `db:"is_signed" json:"is_signed"`
	CreatedDate                 int64   `db:"created_date" json:"created_date"`
}

// Video represents the videos table
type Video struct {
	ID              int     `db:"id" json:"id"`
	StudentRecordID int     `db:"student_record_id" json:"student_record_id"`
	Key             string  `db:"key" json:"key"`
	Filename        string  `db:"filename" json:"filename"`
	ContentType     string  `db:"content_type" json:"content_type"`
	Size            *int    `db:"size" json:"size"`
	URL             *string `db:"url" json:"url"`
	CreatedAt       string  `db:"created_at" json:"created_at"`
}

// ProjectTopicRegistration represents the project_topic_registrations table
type ProjectTopicRegistration struct {
	ID              int     `db:"id" json:"id"`
	StudentRecordID int     `db:"student_record_id" json:"student_record_id"`
	Title           string  `db:"title" json:"title"`
	TitleEn         string  `db:"title_en" json:"title_en"`
	Problem         string  `db:"problem" json:"problem"`
	Objective       string  `db:"objective" json:"objective"`
	Tasks           string  `db:"tasks" json:"tasks"`
	CompletionDate  *string `db:"completion_date" json:"completion_date"`
	Supervisor      string  `db:"supervisor" json:"supervisor"`
	Status          string  `db:"status" json:"status"`
	CreatedAt       int64   `db:"created_at" json:"created_at"`
	UpdatedAt       int64   `db:"updated_at" json:"updated_at"`
	SubmittedAt     *int64  `db:"submitted_at" json:"submitted_at"`
	CurrentVersion  int     `db:"current_version" json:"current_version"`
}

// TopicRegistrationComment represents the topic_registration_comments table
type TopicRegistrationComment struct {
	ID                  int     `db:"id" json:"id"`
	TopicRegistrationID int     `db:"topic_registration_id" json:"topic_registration_id"`
	FieldName           *string `db:"field_name" json:"field_name"`
	CommentText         string  `db:"comment_text" json:"comment_text"`
	AuthorRole          string  `db:"author_role" json:"author_role"`
	AuthorName          string  `db:"author_name" json:"author_name"`
	CreatedAt           int64   `db:"created_at" json:"created_at"`
	ParentCommentID     *int    `db:"parent_comment_id" json:"parent_comment_id"`
	IsRead              int     `db:"is_read" json:"is_read"`
}

// ProjectTopicRegistrationVersion represents the project_topic_registration_versions table
type ProjectTopicRegistrationVersion struct {
	ID                  int    `db:"id" json:"id"`
	TopicRegistrationID int    `db:"topic_registration_id" json:"topic_registration_id"`
	VersionData         string `db:"version_data" json:"version_data"`
	CreatedBy           string `db:"created_by" json:"created_by"`
	CreatedAt           int64  `db:"created_at" json:"created_at"`
}

// Extended structs for complex queries with joins
type StudentWithDetails struct {
	StudentRecord
	SupervisorReportExists bool `db:"supervisor_report_exists"`
	TopicApproved          bool `db:"topic_approved"`
	HasVideo               bool `db:"has_video"`
	HasDocuments           bool `db:"has_documents"`
}

// Database operations
type ThesisDB struct {
	*sqlx.DB
}

func NewThesisDB(db *sqlx.DB) *ThesisDB {
	return &ThesisDB{DB: db}
}

// Student operations
func (db *ThesisDB) GetAllStudents() ([]StudentRecord, error) {
	var students []StudentRecord
	query := `SELECT * FROM student_records ORDER BY student_group, student_name`
	err := db.Select(&students, query)
	return students, err
}

func (db *ThesisDB) GetStudentByID(id int) (*StudentRecord, error) {
	var student StudentRecord
	query := `SELECT * FROM student_records WHERE id = ?`
	err := db.Get(&student, query, id)
	return &student, err
}

func (db *ThesisDB) GetStudentsWithDetails() ([]StudentWithDetails, error) {
	var students []StudentWithDetails
	query := `
        SELECT 
            sr.*,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as supervisor_report_exists,
            CASE WHEN ptr.status = 'approved' THEN 1 ELSE 0 END as topic_approved,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video,
            CASE WHEN d.id IS NOT NULL THEN 1 ELSE 0 END as has_documents
        FROM student_records sr
        LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
        LEFT JOIN videos v ON sr.id = v.student_record_id
        LEFT JOIN documents d ON sr.id = d.student_record_id
        GROUP BY sr.id
        ORDER BY sr.student_group, sr.student_name
    `
	err := db.Select(&students, query)
	return students, err
}

// Department Head operations
func (db *ThesisDB) GetDepartmentHeads() ([]DepartmentHead, error) {
	var heads []DepartmentHead
	query := `SELECT * FROM department_heads WHERE is_active = 1 ORDER BY department, name`
	err := db.Select(&heads, query)
	return heads, err
}

func (db *ThesisDB) GetDepartmentHeadByEmail(email string) (*DepartmentHead, error) {
	var head DepartmentHead
	query := `SELECT * FROM department_heads WHERE email = ? AND is_active = 1`
	err := db.Get(&head, query, email)
	return &head, err
}

// Supervisor Report operations
func (db *ThesisDB) CreateSupervisorReport(report *SupervisorReport) error {
	query := `
        INSERT INTO supervisor_reports (
            student_record_id, supervisor_comments, supervisor_name,
            supervisor_position, supervisor_workplace, is_pass_or_failed,
            is_signed, other_match, one_match, own_match, join_match
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	_, err := db.Exec(query,
		report.StudentRecordID, report.SupervisorComments, report.SupervisorName,
		report.SupervisorPosition, report.SupervisorWorkplace, report.IsPassOrFailed,
		report.IsSigned, report.OtherMatch, report.OneMatch, report.OwnMatch, report.JoinMatch,
	)
	return err
}

func (db *ThesisDB) GetSupervisorReportByStudentID(studentID int) (*SupervisorReport, error) {
	var report SupervisorReport
	query := `SELECT * FROM supervisor_reports WHERE student_record_id = ?`
	err := db.Get(&report, query, studentID)
	return &report, err
}

// Project Topic Registration operations
func (db *ThesisDB) CreateTopicRegistration(registration *ProjectTopicRegistration) error {
	query := `
        INSERT INTO project_topic_registrations (
            student_record_id, title, title_en, problem, objective,
            tasks, completion_date, supervisor, status
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	_, err := db.Exec(query,
		registration.StudentRecordID, registration.Title, registration.TitleEn,
		registration.Problem, registration.Objective, registration.Tasks,
		registration.CompletionDate, registration.Supervisor, registration.Status,
	)
	return err
}

func (db *ThesisDB) UpdateTopicRegistrationStatus(id int, status string) error {
	query := `UPDATE project_topic_registrations SET status = ?, updated_at = strftime('%s', 'now') WHERE id = ?`
	_, err := db.Exec(query, status, id)
	return err
}
