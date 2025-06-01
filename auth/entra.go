// auth/entra.go - Authentication service with reviewer role support
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type AuthService struct {
	config         *EntraIDConfig
	oauth2Config   *oauth2.Config
	appGraphClient *msgraphsdk.GraphServiceClient
	db             *sqlx.DB
}

// NewAuthService creates a new authentication service with database
func NewAuthService(db *sqlx.DB) (*AuthService, error) {
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

	// Create application credentials for system notifications
	var appGraphClient *msgraphsdk.GraphServiceClient
	appCred, err := azidentity.NewClientSecretCredential(
		config.TenantID,
		config.ClientID,
		config.ClientSecret,
		nil,
	)
	if err != nil {
		log.Printf("Warning: Failed to create app credentials for notifications: %v", err)
		log.Println("Notification service will be disabled")
		appGraphClient = nil
	} else {
		// Create Graph client with application permissions
		appGraphClient, err = msgraphsdk.NewGraphServiceClientWithCredentials(
			appCred, []string{"https://graph.microsoft.com/.default"})
		if err != nil {
			log.Printf("Warning: Failed to create app graph client: %v", err)
			log.Println("Notification service will be disabled")
			appGraphClient = nil
		} else {
			log.Println("Application Graph client initialized successfully")
		}
	}

	return &AuthService{
		config:         config,
		oauth2Config:   oauth2Config,
		appGraphClient: appGraphClient,
		db:             db,
	}, nil
}

// GetAppGraphClient returns the application Graph client for system operations
func (a *AuthService) GetAppGraphClient() *msgraphsdk.GraphServiceClient {
	return a.appGraphClient
}

// GenerateLoginURL generates the login URL for Microsoft authentication
func (a *AuthService) GenerateLoginURL() (string, error) {
	state, err := generateRandomState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Force fresh authentication by adding prompt=login
	return a.oauth2Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "login"),
	), nil
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("DEBUG: Microsoft Graph response: %s", string(body))

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// DEBUG: Let's see what fields we actually got
	log.Printf("DEBUG: Parsed UserInfo:")
	log.Printf("  ID: %s", userInfo.ID)
	log.Printf("  DisplayName: %s", userInfo.DisplayName)
	log.Printf("  Mail: %s", userInfo.Mail)
	log.Printf("  UserPrincipalName: %s", userInfo.UserPrincipalName)
	log.Printf("  Department: %s", userInfo.Department)
	log.Printf("  JobTitle: %s", userInfo.JobTitle)

	// Try multiple email fields
	email := userInfo.Mail
	if email == "" {
		email = userInfo.UserPrincipalName
	}
	if email == "" {
		log.Printf("DEBUG: No email found in Mail or UserPrincipalName fields")
		return nil, fmt.Errorf("user email is required but not provided by Microsoft Graph. Available fields: ID=%s, DisplayName=%s", userInfo.ID, userInfo.DisplayName)
	}

	// Update the Mail field with whatever email we found
	userInfo.Mail = email
	log.Printf("DEBUG: Using email: %s", email)

	return &userInfo, nil
}

// determineUserRole determines user role based on database and email/department - FIXED ORDER
func (a *AuthService) determineUserRole(ctx context.Context, userInfo *UserInfo) (string, int, []string, error) {
	email := strings.ToLower(userInfo.Mail)

	log.Printf("DEBUG: Determining role for user: %s", email)

	// 1. First, check if user is a department head in the database
	departmentHead, err := a.getDepartmentHead(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return "", 0, nil, fmt.Errorf("failed to check department head: %w", err)
	}

	if departmentHead != nil && departmentHead.IsActive {
		log.Printf("DEBUG: User is department head with role %d", departmentHead.Role)
		role, permissions := a.getDepartmentHeadPermissions(departmentHead.Role)
		return role, departmentHead.Role, permissions, nil
	}

	// 2. Check if user is a supervisor in the database
	isSupervisorInDB, err := a.isSupervisorInDatabase(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return "", 0, nil, fmt.Errorf("failed to check supervisor in database: %w", err)
	}

	log.Printf("DEBUG: isSupervisorInDatabase result for %s: %v (error: %v)", email, isSupervisorInDB, err)

	if isSupervisorInDB {
		log.Printf("DEBUG: User found as supervisor in database")
		return RoleSupervisor, -1, []string{
			PermissionViewAssignedStudents,
			PermissionCreateReports,
			PermissionReviewSubmissions,
		}, nil
	}

	// 3. Check if user is a supervisor (academic staff) based on email/job title
	if a.isSupervisor(userInfo) {
		log.Printf("DEBUG: User detected as supervisor by email/title patterns")
		return RoleSupervisor, -1, []string{
			PermissionViewAssignedStudents,
			PermissionCreateReports,
			PermissionReviewSubmissions,
		}, nil
	}

	// 4. Check if user is a reviewer (assigned as reviewer in student_records)
	isReviewer, err := a.isReviewer(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return "", 0, nil, fmt.Errorf("failed to check reviewer status: %w", err)
	}

	if isReviewer {
		log.Printf("DEBUG: User is reviewer")
		return RoleReviewer, -1, []string{
			PermissionViewAssignedStudents,
			PermissionCreateReports,
			PermissionReviewSubmissions,
			PermissionViewThesis,
		}, nil
	}

	// 5. Check if user is a commission member
	commissionMember, err := a.getCommissionMemberByEmail(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return "", 0, nil, fmt.Errorf("failed to check commission member: %w", err)
	}

	if commissionMember != nil && commissionMember.IsActive && time.Now().Unix() < commissionMember.ExpiresAt {
		log.Printf("DEBUG: User is commission member")
		return RoleCommissionMember, -1, []string{
			PermissionViewThesis,
			PermissionEvaluateDefense,
		}, nil
	}

	// 6. LAST: Check if user is a student (only after all other checks fail)
	if a.isStudentEmail(email) {
		log.Printf("DEBUG: User detected as student")
		return RoleStudent, -1, []string{
			PermissionViewOwnData,
			PermissionSubmitTopic,
			PermissionUploadDocuments,
		}, nil
	}

	// Default to guest for unknown users
	log.Printf("DEBUG: User defaulted to guest role")
	return RoleGuest, -1, []string{}, nil
}

// getDepartmentHead retrieves department head from database using sqlx
func (a *AuthService) getDepartmentHead(ctx context.Context, email string) (*DepartmentHead, error) {
	query := `
		SELECT id, email, name, sure_name, department, department_en, 
		       job_title, role, is_active, created_at 
		FROM department_heads 
		WHERE email = ? AND is_active = 1
	`

	var head DepartmentHead
	err := a.db.GetContext(ctx, &head, query, email)
	if err != nil {
		return nil, err
	}

	return &head, nil
}

// getCommissionMemberByEmail retrieves commission member using sqlx
func (a *AuthService) getCommissionMemberByEmail(ctx context.Context, email string) (*CommissionMember, error) {
	// This is a placeholder since commission_members table uses access_code, not email
	// You might want to add an email field or create a mapping table
	return nil, sql.ErrNoRows
}

// isReviewer checks if user is assigned as a reviewer for any students using sqlx
func (a *AuthService) isReviewer(ctx context.Context, email string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM student_records WHERE reviewer_email = ?`
	err := a.db.GetContext(ctx, &count, query, email)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// isSupervisorInDatabase checks if user is listed as a supervisor in student_records using sqlx
func (a *AuthService) isSupervisorInDatabase(ctx context.Context, email string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM student_records WHERE supervisor_email = ? AND supervisor_email != ''`

	err := a.db.GetContext(ctx, &count, query, email)
	if err != nil {
		log.Printf("ERROR: Failed to check supervisor in database for %s: %v", email, err)
		return false, err
	}

	log.Printf("DEBUG: Found %d student records for supervisor %s", count, email)
	return count > 0, nil
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

// isSupervisor checks if user is academic staff - IMPROVED
func (a *AuthService) isSupervisor(userInfo *UserInfo) bool {
	email := strings.ToLower(userInfo.Mail)
	jobTitle := strings.ToLower(userInfo.JobTitle)

	log.Printf("DEBUG: isSupervisor checks for %s:", email)
	log.Printf("  Job Title: '%s'", jobTitle)

	// Don't classify students as supervisors
	if a.isStudentEmail(email) {
		log.Printf("DEBUG: Rejected - is student email")
		return false
	}

	// Check job title patterns
	supervisorTitles := []string{
		"professor", "lecturer", "dėstytojas", "profesorius",
		"docentas", "dr.", "assistant", "associate", "vadovas",
		"kompetencij", // for emails like personalokompetencijos
	}

	for _, title := range supervisorTitles {
		if strings.Contains(jobTitle, title) || strings.Contains(email, title) {
			log.Printf("DEBUG: Matched supervisor pattern: %s", title)
			return true
		}
	}

	// Check email domain patterns for staff - IMPROVED LOGIC
	if strings.Contains(email, "@viko.lt") && !a.isStudentEmail(email) {
		log.Printf("DEBUG: Staff email at @viko.lt domain (not a student)")
		return true
	}

	log.Printf("DEBUG: No supervisor patterns matched")
	return false
}

// isStudentEmail checks if email belongs to a student
func (a *AuthService) isStudentEmail(email string) bool {
	log.Printf("DEBUG: Checking if %s is student email", email)

	// Specific student emails
	specificStudents := []string{
		"penworld@eif.viko.lt",
		// Add other specific student emails here
	}

	for _, studentEmail := range specificStudents {
		if email == studentEmail {
			log.Printf("DEBUG: Matched specific student email: %s", studentEmail)
			return true
		}
	}

	// General student patterns
	studentPatterns := []string{
		"@stud.viko.lt",
		"@student.viko.lt",
	}

	for _, pattern := range studentPatterns {
		if strings.Contains(email, pattern) {
			log.Printf("DEBUG: Matched student pattern: %s", pattern)
			return true
		}
	}

	// Check for student ID patterns (numbers at start)
	if strings.Contains(email, "@viko.lt") {
		parts := strings.Split(email, "@")
		if len(parts) > 0 {
			localPart := parts[0]
			if len(localPart) > 0 && localPart[0] >= '0' && localPart[0] <= '9' {
				log.Printf("DEBUG: Matched student ID pattern (starts with number): %s", localPart)
				return true
			}
		}
	}

	log.Printf("DEBUG: No student patterns matched for %s", email)
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

// Helper methods for database operations using sqlx

// AddDepartmentHead adds a new department head to the database using sqlx
func (a *AuthService) AddDepartmentHead(ctx context.Context, head *DepartmentHead) error {
	query := `
		INSERT INTO department_heads (email, name, sure_name, department, department_en, job_title, role, is_active)
		VALUES (:email, :name, :sure_name, :department, :department_en, :job_title, :role, :is_active)
	`

	_, err := a.db.NamedExecContext(ctx, query, head)
	return err
}

// UpdateDepartmentHeadRole updates the role of a department head using sqlx
func (a *AuthService) UpdateDepartmentHeadRole(ctx context.Context, email string, roleID int) error {
	query := `UPDATE department_heads SET role = ? WHERE email = ?`
	_, err := a.db.ExecContext(ctx, query, roleID, email)
	return err
}

// DeactivateDepartmentHead deactivates a department head using sqlx
func (a *AuthService) DeactivateDepartmentHead(ctx context.Context, email string) error {
	query := `UPDATE department_heads SET is_active = 0 WHERE email = ?`
	_, err := a.db.ExecContext(ctx, query, email)
	return err
}

// GetDepartmentHeads retrieves all department heads using sqlx
func (a *AuthService) GetDepartmentHeads(ctx context.Context) ([]DepartmentHead, error) {
	query := `
		SELECT id, email, name, sure_name, department, department_en, 
		       job_title, role, is_active, created_at 
		FROM department_heads 
		WHERE is_active = 1 
		ORDER BY department, name
	`

	var heads []DepartmentHead
	err := a.db.SelectContext(ctx, &heads, query)
	return heads, err
}

// GetDepartmentHeadByID retrieves a department head by ID using sqlx
func (a *AuthService) GetDepartmentHeadByID(ctx context.Context, id int) (*DepartmentHead, error) {
	query := `
		SELECT id, email, name, sure_name, department, department_en, 
		       job_title, role, is_active, created_at 
		FROM department_heads 
		WHERE id = ?
	`

	var head DepartmentHead
	err := a.db.GetContext(ctx, &head, query, id)
	if err != nil {
		return nil, err
	}

	return &head, nil
}

// UpdateDepartmentHead updates a department head using sqlx
func (a *AuthService) UpdateDepartmentHead(ctx context.Context, head *DepartmentHead) error {
	query := `
		UPDATE department_heads 
		SET name = :name, sure_name = :sure_name, department = :department, 
		    department_en = :department_en, job_title = :job_title, role = :role, is_active = :is_active
		WHERE id = :id
	`

	_, err := a.db.NamedExecContext(ctx, query, head)
	return err
}
