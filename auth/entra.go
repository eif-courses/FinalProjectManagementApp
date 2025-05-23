// auth/entra.go (enhanced version)
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type EntraIDConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	RedirectURL  string
}

type UserInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	UserPrincipalName string `json:"userPrincipalName"`
	Mail              string `json:"mail"`
	JobTitle          string `json:"jobTitle"`
	Department        string `json:"department"`
	OfficeLocation    string `json:"officeLocation"`
}

type AuthenticatedUser struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Department  string    `json:"department"`
	JobTitle    string    `json:"job_title"`
	Role        string    `json:"role"`
	RoleID      int       `json:"role_id"` // From database
	Permissions []string  `json:"permissions"`
	LoginTime   time.Time `json:"login_time"`
}

// Database models
type DepartmentHead struct {
	ID           int    `db:"id"`
	Email        string `db:"email"`
	Name         string `db:"name"`
	SureName     string `db:"sure_name"`
	Department   string `db:"department"`
	DepartmentEn string `db:"department_en"`
	JobTitle     string `db:"job_title"`
	Role         int    `db:"role"`
	IsActive     bool   `db:"is_active"`
	CreatedAt    int64  `db:"created_at"`
}

type CommissionMember struct {
	ID             int    `db:"id"`
	AccessCode     string `db:"access_code"`
	Department     string `db:"department"`
	IsActive       bool   `db:"is_active"`
	ExpiresAt      int64  `db:"expires_at"`
	CreatedAt      int64  `db:"created_at"`
	LastAccessedAt *int64 `db:"last_accessed_at"`
}

type AuthService struct {
	config       *EntraIDConfig
	oauth2Config *oauth2.Config
	db           *sql.DB
}

// Role and permission constants
const (
	RoleAdmin            = "admin"
	RoleDepartmentHead   = "department_head"
	RoleSupervisor       = "supervisor"
	RoleCommissionMember = "commission_member"
	RoleStudent          = "student"
	RoleGuest            = "guest"

	// Role IDs from database (department_heads.role column)
	RoleIDSystemAdmin    = 0 // System administrator
	RoleIDDepartmentHead = 1 // Department head
	RoleIDDeputyHead     = 2 // Deputy department head
	RoleIDSecretary      = 3 // Department secretary
	RoleIDCoordinator    = 4 // Program coordinator

	// Permissions
	PermissionFullAccess            = "full_access"
	PermissionViewAllStudents       = "view_all_students"
	PermissionViewAssignedStudents  = "view_assigned_students"
	PermissionApproveTopics         = "approve_topics"
	PermissionManageDepartment      = "manage_department"
	PermissionGenerateReports       = "generate_reports"
	PermissionCreateReports         = "create_reports"
	PermissionReviewSubmissions     = "review_submissions"
	PermissionViewThesis            = "view_thesis"
	PermissionEvaluateDefense       = "evaluate_defense"
	PermissionViewOwnData           = "view_own_data"
	PermissionSubmitTopic           = "submit_topic"
	PermissionUploadDocuments       = "upload_documents"
	PermissionManageUsers           = "manage_users"
	PermissionSystemConfig          = "system_config"
	PermissionManageCommission      = "manage_commission"
	PermissionViewDepartmentReports = "view_department_reports"
)

// NewAuthService creates a new authentication service with database
func NewAuthService(db *sql.DB) (*AuthService, error) {
	config := &EntraIDConfig{
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		RedirectURL:  os.Getenv("AZURE_REDIRECT_URI"),
	}

	// Validate required environment variables
	if config.ClientID == "" {
		return nil, fmt.Errorf("AZURE_CLIENT_ID environment variable is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("AZURE_CLIENT_SECRET environment variable is required")
	}
	if config.TenantID == "" {
		return nil, fmt.Errorf("AZURE_TENANT_ID environment variable is required")
	}

	if config.RedirectURL == "" {
		config.RedirectURL = "http://localhost:8080/auth/callback"
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes: []string{
			"openid",
			"profile",
			"email",
			"https://graph.microsoft.com/User.Read",
		},
		Endpoint: microsoft.AzureADEndpoint(config.TenantID),
	}

	return &AuthService{
		config:       config,
		oauth2Config: oauth2Config,
		db:           db,
	}, nil
}

// GenerateLoginURL generates the login URL for Microsoft authentication
func (a *AuthService) GenerateLoginURL() (string, error) {
	state, err := generateRandomState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	return a.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// HandleCallback handles the OAuth callback from Microsoft
func (a *AuthService) HandleCallback(ctx context.Context, code, state string) (*AuthenticatedUser, error) {
	if code == "" {
		return nil, fmt.Errorf("authorization code is required")
	}

	// Exchange code for token
	token, err := a.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from Microsoft Graph
	userInfo, err := a.getUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Determine user role based on database and email/department
	role, roleID, permissions, err := a.determineUserRole(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to determine user role: %w", err)
	}

	authenticatedUser := &AuthenticatedUser{
		ID:          userInfo.ID,
		Name:        userInfo.DisplayName,
		Email:       strings.ToLower(userInfo.Mail),
		Department:  userInfo.Department,
		JobTitle:    userInfo.JobTitle,
		Role:        role,
		RoleID:      roleID,
		Permissions: permissions,
		LoginTime:   time.Now(),
	}

	return authenticatedUser, nil
}

// getUserInfo fetches user information from Microsoft Graph API
func (a *AuthService) getUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Microsoft Graph API returned status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Validate required fields
	if userInfo.Mail == "" {
		return nil, fmt.Errorf("user email is required but not provided by Microsoft Graph")
	}

	return &userInfo, nil
}

// determineUserRole determines user role based on database and email/department
func (a *AuthService) determineUserRole(ctx context.Context, userInfo *UserInfo) (string, int, []string, error) {
	email := strings.ToLower(userInfo.Mail)

	// First, check if user is a department head in the database
	departmentHead, err := a.getDepartmentHead(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return "", 0, nil, fmt.Errorf("failed to check department head: %w", err)
	}

	if departmentHead != nil && departmentHead.IsActive {
		role, permissions := a.getDepartmentHeadPermissions(departmentHead.Role)
		return role, departmentHead.Role, permissions, nil
	}

	// Check if user is a commission member
	commissionMember, err := a.getCommissionMemberByEmail(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return "", 0, nil, fmt.Errorf("failed to check commission member: %w", err)
	}

	if commissionMember != nil && commissionMember.IsActive && time.Now().Unix() < commissionMember.ExpiresAt {
		return RoleCommissionMember, -1, []string{
			PermissionViewThesis,
			PermissionEvaluateDefense,
		}, nil
	}

	// Check if user is a supervisor (academic staff)
	if a.isSupervisor(userInfo) {
		return RoleSupervisor, -1, []string{
			PermissionViewAssignedStudents,
			PermissionCreateReports,
			PermissionReviewSubmissions,
		}, nil
	}

	// Check if user is a student
	if a.isStudentEmail(email) {
		return RoleStudent, -1, []string{
			PermissionViewOwnData,
			PermissionSubmitTopic,
			PermissionUploadDocuments,
		}, nil
	}

	// Default to guest for unknown users
	return RoleGuest, -1, []string{}, nil
}

// getDepartmentHead retrieves department head from database
func (a *AuthService) getDepartmentHead(ctx context.Context, email string) (*DepartmentHead, error) {
	query := `
		SELECT id, email, name, sure_name, department, department_en, 
		       job_title, role, is_active, created_at 
		FROM department_heads 
		WHERE email = ? AND is_active = 1
	`

	var head DepartmentHead
	err := a.db.QueryRowContext(ctx, query, email).Scan(
		&head.ID, &head.Email, &head.Name, &head.SureName,
		&head.Department, &head.DepartmentEn, &head.JobTitle,
		&head.Role, &head.IsActive, &head.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &head, nil
}

// getCommissionMemberByEmail retrieves commission member (this would need adjustment based on your logic)
func (a *AuthService) getCommissionMemberByEmail(ctx context.Context, email string) (*CommissionMember, error) {
	// This is a placeholder - you might need to implement a different logic
	// since commission_members table uses access_code, not email
	// You might want to add an email field or create a mapping table
	return nil, sql.ErrNoRows
}

// getDepartmentHeadPermissions returns permissions based on department head role
func (a *AuthService) getDepartmentHeadPermissions(roleID int) (string, []string) {
	switch roleID {
	case RoleIDSystemAdmin:
		return RoleAdmin, []string{
			PermissionFullAccess,
			PermissionManageUsers,
			PermissionSystemConfig,
		}
	case RoleIDDepartmentHead:
		return RoleDepartmentHead, []string{
			PermissionViewAllStudents,
			PermissionApproveTopics,
			PermissionManageDepartment,
			PermissionGenerateReports,
			PermissionViewDepartmentReports,
		}
	case RoleIDDeputyHead:
		return RoleDepartmentHead, []string{
			PermissionViewAllStudents,
			PermissionApproveTopics,
			PermissionGenerateReports,
			PermissionViewDepartmentReports,
		}
	case RoleIDSecretary:
		return RoleDepartmentHead, []string{
			PermissionViewAllStudents,
			PermissionGenerateReports,
			PermissionViewDepartmentReports,
		}
	case RoleIDCoordinator:
		return RoleDepartmentHead, []string{
			PermissionViewAllStudents,
			PermissionApproveTopics,
			PermissionGenerateReports,
		}
	default:
		return RoleDepartmentHead, []string{
			PermissionViewAllStudents,
			PermissionApproveTopics,
			PermissionManageDepartment,
			PermissionGenerateReports,
		}
	}
}

// isSupervisor checks if user is academic staff
func (a *AuthService) isSupervisor(userInfo *UserInfo) bool {
	email := strings.ToLower(userInfo.Mail)
	jobTitle := strings.ToLower(userInfo.JobTitle)

	return strings.Contains(jobTitle, "professor") ||
		strings.Contains(jobTitle, "lecturer") ||
		strings.Contains(jobTitle, "dėstytojas") ||
		strings.Contains(jobTitle, "profesorius") ||
		strings.Contains(jobTitle, "docentas") ||
		strings.Contains(jobTitle, "dr.") ||
		(strings.Contains(email, "@viko.lt") && !a.isStudentEmail(email))
}

// isStudentEmail checks if email belongs to a student
func (a *AuthService) isStudentEmail(email string) bool {
	studentPatterns := []string{
		"@stud.viko.lt",
		"@student.viko.lt",
	}

	for _, pattern := range studentPatterns {
		if strings.Contains(email, pattern) {
			return true
		}
	}

	// Check for student ID patterns (numbers at start)
	if strings.Contains(email, "@viko.lt") {
		parts := strings.Split(email, "@")
		if len(parts) > 0 {
			localPart := parts[0]
			if len(localPart) > 0 && localPart[0] >= '0' && localPart[0] <= '9' {
				return true
			}
		}
	}

	return false
}

// generateRandomState generates a random state for OAuth security
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// User permission checking methods (enhanced)

// HasPermission checks if user has a specific permission
func (u *AuthenticatedUser) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission || p == PermissionFullAccess {
			return true
		}
	}
	return false
}

// CanAccessStudents checks if user can access student management
func (u *AuthenticatedUser) CanAccessStudents() bool {
	return u.HasPermission(PermissionViewAllStudents) ||
		u.HasPermission(PermissionViewAssignedStudents) ||
		u.HasPermission(PermissionFullAccess)
}

// CanApproveTopics checks if user can approve topics
func (u *AuthenticatedUser) CanApproveTopics() bool {
	return u.HasPermission(PermissionApproveTopics) || u.HasPermission(PermissionFullAccess)
}

// CanManageDepartment checks if user can manage department
func (u *AuthenticatedUser) CanManageDepartment() bool {
	return u.HasPermission(PermissionManageDepartment) || u.HasPermission(PermissionFullAccess)
}

// IsStudent checks if user is a student
func (u *AuthenticatedUser) IsStudent() bool {
	return u.Role == RoleStudent
}

// IsSupervisor checks if user is a supervisor
func (u *AuthenticatedUser) IsSupervisor() bool {
	return u.Role == RoleSupervisor
}

// IsDepartmentHead checks if user is a department head
func (u *AuthenticatedUser) IsDepartmentHead() bool {
	return u.Role == RoleDepartmentHead
}

// IsAdmin checks if user is an admin
func (u *AuthenticatedUser) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// GetDisplayRole returns user-friendly role name
func (u *AuthenticatedUser) GetDisplayRole() string {
	switch u.Role {
	case RoleAdmin:
		return "Administrator"
	case RoleDepartmentHead:
		// Return more specific role based on RoleID
		switch u.RoleID {
		case RoleIDDepartmentHead:
			return "Department Head"
		case RoleIDDeputyHead:
			return "Deputy Department Head"
		case RoleIDSecretary:
			return "Department Secretary"
		case RoleIDCoordinator:
			return "Program Coordinator"
		default:
			return "Department Head"
		}
	case RoleSupervisor:
		return "Supervisor"
	case RoleCommissionMember:
		return "Commission Member"
	case RoleStudent:
		return "Student"
	default:
		return "Guest"
	}
}

// Helper methods for database operations

// AddDepartmentHead adds a new department head to the database
func (a *AuthService) AddDepartmentHead(ctx context.Context, head *DepartmentHead) error {
	query := `
		INSERT INTO department_heads (email, name, sure_name, department, department_en, job_title, role, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := a.db.ExecContext(ctx, query,
		head.Email, head.Name, head.SureName, head.Department,
		head.DepartmentEn, head.JobTitle, head.Role, head.IsActive)

	return err
}

// UpdateDepartmentHeadRole updates the role of a department head
func (a *AuthService) UpdateDepartmentHeadRole(ctx context.Context, email string, roleID int) error {
	query := `UPDATE department_heads SET role = ? WHERE email = ?`
	_, err := a.db.ExecContext(ctx, query, roleID, email)
	return err
}

// DeactivateDepartmentHead deactivates a department head
func (a *AuthService) DeactivateDepartmentHead(ctx context.Context, email string) error {
	query := `UPDATE department_heads SET is_active = 0 WHERE email = ?`
	_, err := a.db.ExecContext(ctx, query, email)
	return err
}
