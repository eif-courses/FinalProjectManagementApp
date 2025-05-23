// auth/entra.go - Authentication service (types moved to auth/types.go)
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

type AuthService struct {
	config       *EntraIDConfig
	oauth2Config *oauth2.Config
	db           *sql.DB
}

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

// getCommissionMemberByEmail retrieves commission member (placeholder)
func (a *AuthService) getCommissionMemberByEmail(ctx context.Context, email string) (*CommissionMember, error) {
	// This is a placeholder - commission_members table uses access_code, not email
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
			PermissionManageCommission,
		}
	case RoleIDDeputyHead:
		return RoleDepartmentHead, []string{
			PermissionViewAllStudents,
			PermissionApproveTopics,
			PermissionGenerateReports,
			PermissionViewDepartmentReports,
			PermissionManageCommission,
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
