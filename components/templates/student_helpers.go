// components/templates/student_helpers.go
package templates

import (
	"fmt"
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

func calculateProgress(data *StudentDashboardData) int {
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
