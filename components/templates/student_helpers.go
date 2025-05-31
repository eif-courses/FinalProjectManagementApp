// components/templates/student_helpers.go
package templates

import (
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

// Update the calculateProgress function to include topic registration
//func calculateProgress(data *database.StudentDashboardData) int {
//	progress := 0
//
//	// Topic registered and approved (25% total)
//	if data.TopicRegistration != nil {
//		progress += 15 // Draft or submitted
//		if data.TopicRegistration.Status == "approved" {
//			progress += 10 // Approved adds more
//		}
//	}
//
//	// Source code uploaded (25%)
//	if data.SourceCodeRepository != nil {
//		progress += 25
//	}
//
//	// Thesis PDF (25%)
//	if data.HasThesisPDF {
//		progress += 25
//	}
//
//	// Supervisor report (12.5%)
//	if data.SupervisorReport != nil && data.SupervisorReport.IsSigned {
//		progress += 12
//	}
//
//	// Reviewer report (12.5%)
//	if data.ReviewerReport != nil && data.ReviewerReport.IsSigned {
//		progress += 13
//	}
//
//	// OPTIONAL items (not included in progress):
//	// - Company recommendation
//	// - Video presentation
//
//	// Cap at 100%
//	if progress > 100 {
//		progress = 100
//	}
//
//	return progress
//}

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
//func formatDefenseDate(dateStr string) string {
//	if dateStr == "" {
//		return "Not scheduled"
//	}
//
//	// Try to parse the date string and format it nicely
//	if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
//		return t.Format("Jan 2, 2006 at 15:04")
//	}
//
//	// If parsing fails, return the original string
//	return dateStr
//}
