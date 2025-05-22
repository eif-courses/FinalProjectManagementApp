// auth/entra.go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
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
	ID          string
	Name        string
	Email       string
	Department  string
	JobTitle    string
	Role        string // Will be determined from email/department
	Permissions []string
}

type AuthService struct {
	config       *EntraIDConfig
	oauth2Config *oauth2.Config
}

// NewAuthService creates a new authentication service
func NewAuthService() *AuthService {
	config := &EntraIDConfig{
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		RedirectURL:  os.Getenv("AZURE_REDIRECT_URI"),
	}

	// Validate required environment variables
	if config.ClientID == "" || config.ClientSecret == "" || config.TenantID == "" {
		panic("Missing required Azure AD environment variables. Please set AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, and AZURE_TENANT_ID")
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
	}
}

// GenerateLoginURL generates the login URL for Microsoft authentication
func (a *AuthService) GenerateLoginURL() string {
	state := generateRandomState()
	return a.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// HandleCallback handles the OAuth callback from Microsoft
func (a *AuthService) HandleCallback(code, state string) (*AuthenticatedUser, error) {
	// Exchange code for token
	token, err := a.oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Get user info from Microsoft Graph
	userInfo, err := a.getUserInfo(token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Determine user role based on email/department
	role, permissions := a.determineUserRole(userInfo)

	authenticatedUser := &AuthenticatedUser{
		ID:          userInfo.ID,
		Name:        userInfo.DisplayName,
		Email:       userInfo.Mail,
		Department:  userInfo.Department,
		JobTitle:    userInfo.JobTitle,
		Role:        role,
		Permissions: permissions,
	}

	return authenticatedUser, nil
}

// getUserInfo fetches user information from Microsoft Graph API
func (a *AuthService) getUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info from Microsoft Graph: %s", string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// determineUserRole determines user role based on email domain, department, or other criteria
func (a *AuthService) determineUserRole(userInfo *UserInfo) (string, []string) {
	email := strings.ToLower(userInfo.Mail)
	department := strings.ToLower(userInfo.Department)
	jobTitle := strings.ToLower(userInfo.JobTitle)

	// Define role determination logic
	switch {
	// Department heads (based on job title or specific emails)
	case strings.Contains(jobTitle, "head") ||
		strings.Contains(jobTitle, "director") ||
		strings.Contains(jobTitle, "dean") ||
		isDepartmentHead(email):
		return "department_head", []string{"view_all_students", "approve_topics", "manage_department", "generate_reports"}

	// Supervisors (lecturers, professors)
	case strings.Contains(jobTitle, "professor") ||
		strings.Contains(jobTitle, "lecturer") ||
		strings.Contains(jobTitle, "dr.") ||
		strings.Contains(department, "academic"):
		return "supervisor", []string{"view_assigned_students", "create_reports", "review_submissions"}

	// Commission members (for thesis defense)
	case strings.Contains(jobTitle, "commission") ||
		isCommissionMember(email):
		return "commission_member", []string{"view_thesis", "evaluate_defense"}

	// Students (default for student email domains or specific patterns)
	case strings.Contains(email, "student") ||
		strings.HasSuffix(email, "@stud.viko.lt") ||
		isStudentEmail(email):
		return "student", []string{"view_own_data", "submit_topic", "upload_documents"}

	// Admin (specific admin emails)
	case isSystemAdmin(email):
		return "admin", []string{"full_access", "manage_users", "system_config"}

	default:
		// Default to student role for unknown users
		return "student", []string{"view_own_data", "submit_topic", "upload_documents"}
	}
}

// Helper functions for role determination
func isDepartmentHead(email string) bool {
	departmentHeads := []string{
		"j.petraitis@viko.lt",
		"r.kazlauskiene@viko.lt",
		// Add more department head emails
	}

	for _, head := range departmentHeads {
		if email == head {
			return true
		}
	}
	return false
}

func isCommissionMember(email string) bool {
	// Check if user is in commission members list
	// This could be fetched from database in real implementation
	return strings.Contains(email, "commission") ||
		strings.Contains(email, "committee")
}

func isStudentEmail(email string) bool {
	// Check student email patterns
	studentPatterns := []string{
		"@stud.viko.lt",
		"@student.viko.lt",
		// Add more student email patterns
	}

	for _, pattern := range studentPatterns {
		if strings.Contains(email, pattern) {
			return true
		}
	}
	return false
}

func isSystemAdmin(email string) bool {
	admins := []string{
		"admin@viko.lt",
		"system@viko.lt",
		"m.gzegozevskis@eif.viko.lt", // Your admin email
		// Add more admin emails
	}

	for _, admin := range admins {
		if email == admin {
			return true
		}
	}
	return false
}

// generateRandomState generates a random state for OAuth security
func generateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// HasPermission checks if user has a specific permission
func (u *AuthenticatedUser) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission || p == "full_access" {
			return true
		}
	}
	return false
}

// CanAccessStudents checks if user can access student management
func (u *AuthenticatedUser) CanAccessStudents() bool {
	return u.HasPermission("view_all_students") ||
		u.HasPermission("view_assigned_students") ||
		u.HasPermission("full_access")
}

// CanApproveTopics checks if user can approve topics
func (u *AuthenticatedUser) CanApproveTopics() bool {
	return u.HasPermission("approve_topics") || u.HasPermission("full_access")
}

// IsStudent checks if user is a student
func (u *AuthenticatedUser) IsStudent() bool {
	return u.Role == "student"
}

// IsSupervisor checks if user is a supervisor
func (u *AuthenticatedUser) IsSupervisor() bool {
	return u.Role == "supervisor"
}

// IsDepartmentHead checks if user is a department head
func (u *AuthenticatedUser) IsDepartmentHead() bool {
	return u.Role == "department_head"
}
