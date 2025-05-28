package templates

// Column visibility permissions
func canViewTopicRegistration(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

func canViewDocuments(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor" || role == "reviewer"
}

func canViewReviewer(role string) bool {
	// Only admins and department heads can see reviewer column
	return role == "admin" || role == "department_head"
}

func canViewSupervisor(role string) bool {
	// All these roles can see supervisor column
	return role == "admin" || role == "department_head" || role == "supervisor"
}

func canPerformActions(role string) bool {
	return role == "admin" || role == "department_head"
}

func canViewStudentDetails(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

// Registration permissions
func canViewRegistration(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

func canEditRegistration(role string, status string) bool {
	if role == "admin" || role == "department_head" {
		return true
	}
	if role == "supervisor" && (status == "draft" || status == "rejected") {
		return true
	}
	return false
}

func canCreateRegistration(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

func canApproveRegistration(role string) bool {
	return role == "admin" || role == "department_head"
}

// Document permissions
func canUploadDocuments(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

// Reviewer permissions
func canAssignReviewer(role string) bool {
	return role == "admin" || role == "department_head"
}

func canViewReview(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

func canEditReview(role string, reviewerName string, userEmail string) bool {
	if role == "admin" || role == "department_head" {
		return true
	}
	if role == "reviewer" && reviewerName != "" {
		// Check if this reviewer is assigned to this student
		return true // You'd need to implement actual reviewer assignment check
	}
	return false
}

func canSignReview(role string, reviewerName string, userEmail string) bool {
	if role == "admin" {
		return true
	}
	if role == "reviewer" && reviewerName != "" {
		// Check if this is the assigned reviewer
		return true // You'd need to implement actual reviewer check
	}
	return false
}

// Supervisor permissions
func canViewSupervisorReport(role string) bool {
	return role == "admin" || role == "department_head" || role == "supervisor"
}

func canSignSupervisorReport(role string, supervisorEmail string, userEmail string) bool {
	if role == "admin" {
		return true
	}
	if role == "supervisor" && supervisorEmail == userEmail {
		return true
	}
	return false
}

// Student management permissions
func canEditStudent(role string) bool {
	return role == "admin" || role == "department_head"
}

func canDeleteStudent(role string) bool {
	return role == "admin"
}

func canViewStudentHistory(role string) bool {
	return role == "admin" || role == "department_head"
}

// Export permissions
func canExportData(role string) bool {
	return role == "admin" || role == "department_head"
}

func canExportStudentData(role string) bool {
	return role == "admin" || role == "department_head"
}

// Filter permissions
func canFilterByStatus(role string) bool {
	return role == "admin" || role == "department_head"
}

func canEditSupervisorReport(userRole, supervisorEmail, userEmail string) bool {
	// Allow supervisors to edit their own reports, and admins to edit any
	return userRole == "admin" || userRole == "department_head" ||
		(userRole == "supervisor" && supervisorEmail == userEmail)
}

func canCreateSupervisorReport(userRole, supervisorEmail, userEmail string) bool {
	// Allow supervisors to create reports for their students, and admins to create any
	return userRole == "admin" || userRole == "department_head" ||
		(userRole == "supervisor" && supervisorEmail == userEmail)
}
