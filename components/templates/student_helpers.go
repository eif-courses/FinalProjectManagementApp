// components/templates/student_helpers.go
package templates

import (
	"FinalProjectManagementApp/database"
	"fmt"
	"time"
)

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

func calculateProgress(data *database.StudentDashboardData) int {
	total := 7 // Total requirements
	completed := 0

	if data.TopicStatus == "approved" {
		completed++
	}
	if data.SourceCodeRepository != nil {
		completed++
	}
	if data.HasThesisPDF {
		completed++
	}
	if data.SupervisorReport != nil && data.SupervisorReport.IsSigned {
		completed++
	}
	if data.ReviewerReport != nil && data.ReviewerReport.IsSigned {
		completed++
	}
	if data.CompanyRecommendation != nil {
		completed++
	}
	if data.VideoPresentation != nil {
		completed++
	}

	return (completed * 100) / total
}

// Add helper functions for time formatting
func formatTime(t *time.Time) string {
	if t == nil {
		return "Not scheduled"
	}
	return t.Format("January 2, 2006 at 15:04")
}

func formatTimeShort(t *time.Time) string {
	if t == nil {
		return "Not set"
	}
	return t.Format("Jan 2, 2006")
}

// Helper function to format defense date string
func formatDefenseDate(dateStr string) string {
	if dateStr == "" {
		return "Not scheduled"
	}

	// Try to parse the date string and format it nicely
	if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
		return t.Format("Jan 2, 2006 at 15:04")
	}

	// If parsing fails, return the original string
	return dateStr
}
