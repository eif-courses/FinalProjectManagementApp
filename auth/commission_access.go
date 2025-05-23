// auth/commission_access.go - Complete fixed version
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// CommissionAccess represents commission access data
type CommissionAccess struct {
	ID             int    `json:"id" db:"id"`
	AccessCode     string `json:"access_code" db:"access_code"`
	Department     string `json:"department" db:"department"`
	StudyProgram   string `json:"study_program" db:"study_program"`
	Year           int    `json:"year" db:"year"`
	Description    string `json:"description" db:"description"`
	IsActive       bool   `json:"is_active" db:"is_active"`
	ExpiresAt      int64  `json:"expires_at" db:"expires_at"`
	CreatedAt      int64  `json:"created_at" db:"created_at"`
	LastAccessedAt *int64 `json:"last_accessed_at" db:"last_accessed_at"`
	CreatedBy      string `json:"created_by" db:"created_by"`
	AccessCount    int    `json:"access_count" db:"access_count"`
	MaxAccess      int    `json:"max_access" db:"max_access"`
}

// IsExpired checks if access has expired
func (ca *CommissionAccess) IsExpired() bool {
	return time.Now().Unix() > ca.ExpiresAt
}

// IsAccessLimitReached checks if access limit has been reached
func (ca *CommissionAccess) IsAccessLimitReached() bool {
	return ca.MaxAccess > 0 && ca.AccessCount >= ca.MaxAccess
}

// IsValid checks if access is valid (active, not expired, not over limit)
func (ca *CommissionAccess) IsValid() bool {
	return ca.IsActive && !ca.IsExpired() && !ca.IsAccessLimitReached()
}

// CommissionAccessService handles commission access logic
type CommissionAccessService struct {
	db *sql.DB
}

// NewCommissionAccessService creates a new commission access service
func NewCommissionAccessService(db *sql.DB) *CommissionAccessService {
	return &CommissionAccessService{db: db}
}

// CreateAccess creates a new commission access code
func (cas *CommissionAccessService) CreateAccess(ctx context.Context, department, studyProgram, description, createdBy string, year int, expiresAt int64, maxAccess int) (*CommissionAccess, error) {
	accessCode, err := generateAccessCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate access code: %w", err)
	}

	query := `
		INSERT INTO commission_members (access_code, department, study_program, year, description, 
			is_active, expires_at, created_by, max_access, access_count)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, 0)
	`

	result, err := cas.db.ExecContext(ctx, query, accessCode, department, studyProgram, year, description, expiresAt, createdBy, maxAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to create commission access: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted ID: %w", err)
	}

	return &CommissionAccess{
		ID:             int(id),
		AccessCode:     accessCode,
		Department:     department,
		StudyProgram:   studyProgram,
		Year:           year,
		Description:    description,
		IsActive:       true,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now().Unix(),
		CreatedBy:      createdBy,
		MaxAccess:      maxAccess,
		AccessCount:    0,
		LastAccessedAt: nil,
	}, nil
}

// ValidateAccess validates and records access attempt
func (cas *CommissionAccessService) ValidateAccess(ctx context.Context, accessCode string) (*CommissionAccess, error) {
	if accessCode == "" {
		return nil, fmt.Errorf("access code is required")
	}

	// Get access record
	query := `
		SELECT id, access_code, department, study_program, year, description, 
			is_active, expires_at, created_at, last_accessed_at, created_by, access_count, max_access
		FROM commission_members 
		WHERE access_code = ?
	`

	var access CommissionAccess
	var studyProgram, description sql.NullString
	var year sql.NullInt64
	var lastAccessed sql.NullInt64

	err := cas.db.QueryRowContext(ctx, query, accessCode).Scan(
		&access.ID, &access.AccessCode, &access.Department, &studyProgram, &year,
		&description, &access.IsActive, &access.ExpiresAt, &access.CreatedAt,
		&lastAccessed, &access.CreatedBy, &access.AccessCount, &access.MaxAccess,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid access code")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Handle nullable fields
	if studyProgram.Valid {
		access.StudyProgram = studyProgram.String
	}
	if year.Valid {
		access.Year = int(year.Int64)
	}
	if description.Valid {
		access.Description = description.String
	}
	if lastAccessed.Valid {
		timestamp := lastAccessed.Int64
		access.LastAccessedAt = &timestamp
	}

	// Validate access
	if !access.IsValid() {
		if !access.IsActive {
			return nil, fmt.Errorf("access code is deactivated")
		}
		if access.IsExpired() {
			return nil, fmt.Errorf("access code has expired")
		}
		if access.IsAccessLimitReached() {
			return nil, fmt.Errorf("access limit reached")
		}
	}

	// Update access count and last accessed time
	updateQuery := `
		UPDATE commission_members 
		SET access_count = access_count + 1, last_accessed_at = ?
		WHERE id = ?
	`
	_, err = cas.db.ExecContext(ctx, updateQuery, time.Now().Unix(), access.ID)
	if err != nil {
		log.Printf("Failed to update access count: %v", err)
		// Don't fail the request for this
	}

	return &access, nil
}

// GetAccess retrieves commission access by access code
func (cas *CommissionAccessService) GetAccess(ctx context.Context, accessCode string) (*CommissionAccess, error) {
	query := `
		SELECT id, access_code, department, study_program, year, description, 
			is_active, expires_at, created_at, last_accessed_at, created_by, access_count, max_access
		FROM commission_members 
		WHERE access_code = ?
	`

	var access CommissionAccess
	var studyProgram, description sql.NullString
	var year sql.NullInt64
	var lastAccessed sql.NullInt64

	err := cas.db.QueryRowContext(ctx, query, accessCode).Scan(
		&access.ID, &access.AccessCode, &access.Department, &studyProgram, &year,
		&description, &access.IsActive, &access.ExpiresAt, &access.CreatedAt,
		&lastAccessed, &access.CreatedBy, &access.AccessCount, &access.MaxAccess,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if studyProgram.Valid {
		access.StudyProgram = studyProgram.String
	}
	if year.Valid {
		access.Year = int(year.Int64)
	}
	if description.Valid {
		access.Description = description.String
	}
	if lastAccessed.Valid {
		timestamp := lastAccessed.Int64
		access.LastAccessedAt = &timestamp
	}

	return &access, nil
}

// ListAccesses retrieves all commission accesses
func (cas *CommissionAccessService) ListAccesses(ctx context.Context, createdBy string) ([]*CommissionAccess, error) {
	query := `
		SELECT id, access_code, department, study_program, year, description, 
			is_active, expires_at, created_at, last_accessed_at, created_by, access_count, max_access
		FROM commission_members 
		WHERE created_by = ? OR ? = ''
		ORDER BY created_at DESC
	`

	rows, err := cas.db.QueryContext(ctx, query, createdBy, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to query commission accesses: %w", err)
	}
	defer rows.Close()

	var accesses []*CommissionAccess
	for rows.Next() {
		var access CommissionAccess
		var studyProgram, description sql.NullString
		var year sql.NullInt64
		var lastAccessed sql.NullInt64

		err := rows.Scan(
			&access.ID, &access.AccessCode, &access.Department, &studyProgram, &year,
			&description, &access.IsActive, &access.ExpiresAt, &access.CreatedAt,
			&lastAccessed, &access.CreatedBy, &access.AccessCount, &access.MaxAccess,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan commission access: %w", err)
		}

		// Handle nullable fields
		if studyProgram.Valid {
			access.StudyProgram = studyProgram.String
		}
		if year.Valid {
			access.Year = int(year.Int64)
		}
		if description.Valid {
			access.Description = description.String
		}
		if lastAccessed.Valid {
			timestamp := lastAccessed.Int64
			access.LastAccessedAt = &timestamp
		}

		accesses = append(accesses, &access)
	}

	return accesses, nil
}

// DeactivateAccess deactivates a commission access
func (cas *CommissionAccessService) DeactivateAccess(ctx context.Context, accessCode string) error {
	query := `UPDATE commission_members SET is_active = 0 WHERE access_code = ?`
	result, err := cas.db.ExecContext(ctx, query, accessCode)
	if err != nil {
		return fmt.Errorf("failed to deactivate access: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("access code not found")
	}

	return nil
}

// CommissionAccessMiddleware for Chi router - extracts access code from URL
func (cas *CommissionAccessService) CommissionAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract access code from Chi URL parameter
		accessCode := chi.URLParam(r, "accessCode")

		// If not found in URL params, try to extract from path
		if accessCode == "" {
			path := strings.TrimPrefix(r.URL.Path, "/commission/")
			parts := strings.Split(path, "/")
			if len(parts) > 0 && parts[0] != "" {
				accessCode = parts[0]
			}
		}

		if accessCode == "" {
			cas.renderAccessError(w, "Access code is required", http.StatusBadRequest)
			return
		}

		// Validate access code
		access, err := cas.ValidateAccess(r.Context(), accessCode)
		if err != nil {
			cas.renderAccessError(w, fmt.Sprintf("Invalid or expired access code: %s", err.Error()), http.StatusForbidden)
			return
		}

		// Add access info to context
		ctx := context.WithValue(r.Context(), "commission_access", access)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// renderAccessError renders an error page for commission access issues
func (cas *CommissionAccessService) renderAccessError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	tmplStr := `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Access Error</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center">
    <div class="max-w-md mx-auto bg-white rounded-lg shadow-lg p-8 text-center">
        <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 mb-4">
            <svg class="h-6 w-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.99-.833-2.732 0L3.732 16c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
        </div>
        <h1 class="text-xl font-semibold text-gray-900 mb-2">Access Error</h1>
        <p class="text-gray-600 mb-6">{{.Message}}</p>
        <div class="text-sm text-gray-500">
            <p>If you believe this is an error, please contact your system administrator.</p>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("error").Parse(tmplStr)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Message string
	}{
		Message: message,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Failed to render error template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// generateAccessCode generates a secure random access code
func generateAccessCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetCommissionAccessFromContext retrieves commission access from context
func GetCommissionAccessFromContext(ctx context.Context) *CommissionAccess {
	if access, ok := ctx.Value("commission_access").(*CommissionAccess); ok {
		return access
	}
	return nil
}
