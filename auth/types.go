// auth/types.go - All authentication and authorization types
package auth

import "time"

// ================================
// CONFIG AND USER INFO TYPES
// ================================

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

// ================================
// DATABASE MODEL TYPES (FIXED FOR MYSQL)
// ================================

// DepartmentHead represents a department head user - FIXED types for MySQL
type DepartmentHead struct {
	ID           int       `db:"id"`
	Email        string    `db:"email"`
	Name         string    `db:"name"`
	SureName     string    `db:"sure_name"`
	Department   string    `db:"department"`
	DepartmentEn string    `db:"department_en"`
	JobTitle     string    `db:"job_title"`
	Role         int       `db:"role"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"` // ✅ FIXED: time.Time for MySQL TIMESTAMP
}

// CommissionMember represents a commission member with access token - FIXED types
type CommissionMember struct {
	ID             int       `db:"id"`
	AccessCode     string    `db:"access_code"`
	Department     string    `db:"department"`
	StudyProgram   *string   `db:"study_program"`
	Year           *int      `db:"year"`
	Description    string    `db:"description"`
	IsActive       bool      `db:"is_active"`
	ExpiresAt      int64     `db:"expires_at"`       // BIGINT stays as int64
	CreatedAt      time.Time `db:"created_at"`       // ✅ FIXED: time.Time for MySQL TIMESTAMP
	LastAccessedAt *int64    `db:"last_accessed_at"` // BIGINT NULL stays as *int64
	CreatedBy      string    `db:"created_by"`
	AccessCount    int       `db:"access_count"`
	MaxAccess      int       `db:"max_access"`
}

// ================================
// ROLE AND PERMISSION CONSTANTS
// ================================

const (
	// User roles
	RoleAdmin            = "admin"
	RoleDepartmentHead   = "department_head"
	RoleSupervisor       = "supervisor"
	RoleReviewer         = "reviewer" // ✅ ADDED: Reviewer role
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

// ================================
// HELPER METHODS FOR DEPARTMENT HEAD
// ================================

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
// HELPER METHODS FOR COMMISSION MEMBER
// ================================

// IsExpired checks if the commission access has expired
func (cm *CommissionMember) IsExpired() bool {
	return time.Now().Unix() > cm.ExpiresAt
}

// IsAccessLimitReached checks if access limit has been reached
func (cm *CommissionMember) IsAccessLimitReached() bool {
	return cm.MaxAccess > 0 && cm.AccessCount >= cm.MaxAccess
}

// IsValid checks if access is valid (active, not expired, not over limit)
func (cm *CommissionMember) IsValid() bool {
	return cm.IsActive && !cm.IsExpired() && !cm.IsAccessLimitReached()
}

// GetExpiresAtFormatted returns formatted expiration date
func (cm *CommissionMember) GetExpiresAtFormatted() string {
	return time.Unix(cm.ExpiresAt, 0).Format("2006-01-02 15:04")
}

// ================================
// HELPER METHODS FOR AUTHENTICATED USER
// ================================

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

// IsReviewer checks if user is a reviewer
func (u *AuthenticatedUser) IsReviewer() bool {
	return u.Role == RoleReviewer
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
	case RoleReviewer:
		return "Reviewer"
	case RoleCommissionMember:
		return "Commission Member"
	case RoleStudent:
		return "Student"
	default:
		return "Guest"
	}
}
