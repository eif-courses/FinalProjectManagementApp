package services

import (
	"FinalProjectManagementApp/database"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type CommissionService struct {
	db *sqlx.DB
}

func NewCommissionService(db *sqlx.DB) *CommissionService {
	return &CommissionService{db: db}
}

func (s *CommissionService) GenerateAccessCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *CommissionService) CreateAccess(ctx context.Context, params CreateAccessParams) (*database.CommissionMember, error) {
	accessCode, err := s.GenerateAccessCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access code: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(params.DurationDays) * 24 * time.Hour).Unix()

	query := `
		INSERT INTO commission_members (
			access_code, department, study_program, year, description, 
			is_active, expires_at, created_by, max_access, access_count,
			allowed_student_groups, allowed_study_programs, access_level, commission_type
		) VALUES (
			:access_code, :department, :study_program, :year, :description,
			:is_active, :expires_at, :created_by, :max_access, :access_count,
			:allowed_student_groups, :allowed_study_programs, :access_level, :commission_type
		)
	`

	member := &database.CommissionMember{
		AccessCode:     accessCode,
		Department:     params.Department,
		StudyProgram:   sql.NullString{String: params.StudyProgram, Valid: params.StudyProgram != ""},
		Year:           sql.NullInt64{Int64: int64(params.Year), Valid: params.Year > 0},
		Description:    sql.NullString{String: params.Description, Valid: params.Description != ""},
		IsActive:       true,
		ExpiresAt:      expiresAt,
		CreatedBy:      params.CreatedBy,
		MaxAccess:      params.MaxAccess,
		AccessCount:    0,
		AccessLevel:    params.AccessLevel,
		CommissionType: params.CommissionType,
	}

	if len(params.AllowedGroups) > 0 {
		member.AllowedStudentGroups = sql.NullString{
			String: strings.Join(params.AllowedGroups, ","),
			Valid:  true,
		}
	}

	if len(params.AllowedPrograms) > 0 {
		member.AllowedStudyPrograms = sql.NullString{
			String: strings.Join(params.AllowedPrograms, ","),
			Valid:  true,
		}
	}

	result, err := s.db.NamedExecContext(ctx, query, member)
	if err != nil {
		return nil, fmt.Errorf("failed to create commission access: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	member.ID = int(id)
	return member, nil
}

func (s *CommissionService) ValidateAndRecordAccess(ctx context.Context, accessCode string) (*database.CommissionMember, error) {
	var member database.CommissionMember

	query := `SELECT * FROM commission_members WHERE access_code = ?`
	err := s.db.GetContext(ctx, &member, query, accessCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid access code")
		}
		return nil, err
	}

	// Validate access
	if !member.IsActive {
		return nil, fmt.Errorf("access code is deactivated")
	}

	if time.Now().Unix() > member.ExpiresAt {
		return nil, fmt.Errorf("access code has expired")
	}

	if member.MaxAccess > 0 && member.AccessCount >= member.MaxAccess {
		return nil, fmt.Errorf("access limit reached")
	}

	// Update access count and last accessed time
	updateQuery := `
		UPDATE commission_members 
		SET access_count = access_count + 1, last_accessed_at = ?
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, updateQuery, time.Now().Unix(), member.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update access count: %w", err)
	}

	// Log access
	logQuery := `
		INSERT INTO commission_access_logs (commission_member_id, action, resource_accessed, ip_address, user_agent)
		VALUES (?, 'login', 'commission_portal', ?, ?)
	`
	_, _ = s.db.ExecContext(ctx, logQuery, member.ID, "", "") // IP and user agent can be added later

	return &member, nil
}

func (s *CommissionService) GetStudentsForCommission(ctx context.Context, member *database.CommissionMember) ([]database.StudentRecord, error) {
	query := `
		SELECT * FROM student_records 
		WHERE department = ?
	`
	args := []interface{}{member.Department}

	// Add filters based on commission member restrictions
	if member.StudyProgram.Valid && member.StudyProgram.String != "" {
		query += " AND study_program = ?"
		args = append(args, member.StudyProgram.String)
	}

	if member.Year.Valid && member.Year.Int64 > 0 {
		query += " AND current_year = ?"
		args = append(args, member.Year.Int64)
	}

	if member.AllowedStudentGroups.Valid && member.AllowedStudentGroups.String != "" {
		groups := strings.Split(member.AllowedStudentGroups.String, ",")
		placeholders := make([]string, len(groups))
		for i := range groups {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(groups[i]))
		}
		query += fmt.Sprintf(" AND student_group IN (%s)", strings.Join(placeholders, ","))
	}

	query += " ORDER BY student_lastname, student_name"

	var students []database.StudentRecord
	err := s.db.SelectContext(ctx, &students, query, args...)
	return students, err
}

func (s *CommissionService) ListActiveAccess(ctx context.Context, createdBy string) ([]database.CommissionMember, error) {
	query := `
		SELECT * FROM commission_members 
		WHERE (created_by = ? OR ? = '')
		AND is_active = true
		ORDER BY created_at DESC
	`

	var members []database.CommissionMember
	err := s.db.SelectContext(ctx, &members, query, createdBy, createdBy)
	return members, err
}

func (s *CommissionService) DeactivateAccess(ctx context.Context, accessCode string, userEmail string) error {
	query := `
		UPDATE commission_members 
		SET is_active = false 
		WHERE access_code = ? AND created_by = ?
	`

	result, err := s.db.ExecContext(ctx, query, accessCode, userEmail)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("access code not found or unauthorized")
	}

	return nil
}

type CreateAccessParams struct {
	Department      string
	StudyProgram    string
	Year            int
	Description     string
	DurationDays    int
	MaxAccess       int
	CreatedBy       string
	AllowedGroups   []string
	AllowedPrograms []string
	AccessLevel     string
	CommissionType  string
}
