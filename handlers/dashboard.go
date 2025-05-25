package handlers

import (
	"net/http"

	"FinalProjectManagementApp/auth"
)

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard - Project Management</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f8f9fa; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .user-info { background: #e3f2fd; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .permissions { background: #f3e5f5; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
        .btn { background: #d13438; color: white; padding: 8px 16px; text-decoration: none; border-radius: 3px; }
        .role { font-weight: bold; color: #1976d2; }
        .permission { display: inline-block; background: #4caf50; color: white; padding: 4px 8px; margin: 2px; border-radius: 3px; font-size: 12px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Dashboard</h1>
        <a href="/auth/logout" class="btn">Logout</a>
    </div>
    
    <div class="user-info">
        <h2>Welcome, ` + user.Name + `!</h2>
        <p><strong>Email:</strong> ` + user.Email + `</p>
        <p><strong>Department:</strong> ` + user.Department + `</p>
        <p><strong>Job Title:</strong> ` + user.JobTitle + `</p>
        <p><strong>Role:</strong> <span class="role">` + user.Role + `</span></p>
        <p><strong>Login Time:</strong> ` + user.LoginTime.Format("2006-01-02 15:04:05") + `</p>
    </div>
    
    <div class="permissions">
        <h3>Your Permissions:</h3>`

	for _, perm := range user.Permissions {
		html += `<span class="permission">` + perm + `</span>`
	}

	html += `
    </div>
    
    <div>
        <h3>Available Actions:</h3>
        <p>Based on your role (` + user.Role + `), you can access different parts of the system.</p>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// Placeholder handlers for other routes
func StudentProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Profile - Coming Soon"))
}

func StudentTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Student Topic - Coming Soon"))
}

func SubmitTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Submit Topic - Coming Soon"))
}

func SupervisorDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Supervisor Dashboard - Coming Soon"))
}

func SupervisorStudentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Supervisor Students - Coming Soon"))
}

func SupervisorReportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Supervisor Reports - Coming Soon"))
}

func DepartmentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Department Dashboard - Coming Soon"))
}

func DepartmentStudentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Department Students - Coming Soon"))
}

func PendingTopicsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pending Topics - Coming Soon"))
}

func ApproveTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Approve Topic - Coming Soon"))
}

func RejectTopicHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Reject Topic - Coming Soon"))
}

func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Admin Dashboard - Coming Soon"))
}

func AdminUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Admin Users - Coming Soon"))
}

func AuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Audit Logs - Coming Soon"))
}
