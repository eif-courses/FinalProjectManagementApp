// handlers/students.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/i18n"
	"github.com/jmoiron/sqlx"
	"html/template"
	"net/http"
	"strings"
)

type Student struct {
	ID                     int    `db:"id"`
	Group                  string `db:"student_group"`
	StudentName            string `db:"student_name"`
	ProjectTitle           string `db:"final_project_title"`
	TopicApproved          bool   `db:"topic_approved"`
	HasVideo               bool   `db:"has_video"`
	SupervisorReportExists bool   `db:"supervisor_report_exists"`
}

// Create dummy students data for testing
func getDummyStudents() []Student {
	return []Student{
		{
			ID:                     1,
			Group:                  "PI22B",
			StudentName:            "Studentas Studentauskas",
			ProjectTitle:           "Gyvinų internetinė parduotuvė",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: false,
		},
		{
			ID:                     2,
			Group:                  "PIT22",
			StudentName:            "TestVardas StudentasPavarde",
			ProjectTitle:           "Baigiamųjų darbų talykla",
			TopicApproved:          true,
			HasVideo:               true,
			SupervisorReportExists: true,
		},
		{
			ID:                     3,
			Group:                  "PI22B",
			StudentName:            "Aleksandr Michalovskij",
			ProjectTitle:           "Automobilių interneto svetainė",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: true,
		},
		{
			ID:                     4,
			Group:                  "PI22S",
			StudentName:            "Raimondas Kalinovskis",
			ProjectTitle:           "Temos pavadinimas nenurodytas",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: true,
		},
		{
			ID:                     5,
			Group:                  "PI22S",
			StudentName:            "Vitalius Kunigiškis",
			ProjectTitle:           "CRM sistema",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: false,
		},
		{
			ID:                     6,
			Group:                  "PI22B",
			StudentName:            "Karolis Pakalnis",
			ProjectTitle:           "Temos pavadinimas nenurodytas",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: false,
		},
		{
			ID:                     7,
			Group:                  "PIT22",
			StudentName:            "Satvijas Motiejūnas",
			ProjectTitle:           "Interneto svetainės baigiamųjų darbų talykla automatiniai testai",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: false,
		},
		{
			ID:                     8,
			Group:                  "PI22S",
			StudentName:            "Tauras Petrauskas",
			ProjectTitle:           "Temos pavadinimas nenurodytas",
			TopicApproved:          false,
			HasVideo:               false,
			SupervisorReportExists: false,
		},
		{
			ID:                     9,
			Group:                  "PI22B",
			StudentName:            "Mantas Gzegožveskis",
			ProjectTitle:           "Thesis management system",
			TopicApproved:          true,
			HasVideo:               true,
			SupervisorReportExists: true,
		},
		{
			ID:                     10,
			Group:                  "PIT22",
			StudentName:            "Jonas Jonaitis",
			ProjectTitle:           "E-commerce platformos kūrimas",
			TopicApproved:          true,
			HasVideo:               false,
			SupervisorReportExists: false,
		},
	}
}

// Filter dummy data by query parameters
func getFilteredDummyStudents(r *http.Request) []Student {
	students := getDummyStudents()

	// Get filter parameters
	group := r.URL.Query().Get("group")
	search := r.URL.Query().Get("search")

	// Apply group filter
	if group != "" {
		filtered := []Student{}
		for _, student := range students {
			if student.Group == group {
				filtered = append(filtered, student)
			}
		}
		students = filtered
	}

	// Apply search filter
	if search != "" {
		filtered := []Student{}
		for _, student := range students {
			if strings.Contains(strings.ToLower(student.StudentName), strings.ToLower(search)) ||
				strings.Contains(strings.ToLower(student.ProjectTitle), strings.ToLower(search)) {
				filtered = append(filtered, student)
			}
		}
		students = filtered
	}

	return students
}

// StudentsTableHandlerWithI18n updated with role-based access
func StudentsTableHandlerWithI18n(tmpl *template.Template, db *sqlx.DB, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.GetLangFromContext(r.Context())
		user := auth.GetUserFromContext(r.Context())

		// Check permissions
		if !user.CanAccessStudents() {
			http.Error(w, localizer.T(lang, "access_denied"), http.StatusForbidden)
			return
		}

		// Get students based on user role
		var students []Student
		if user.IsDepartmentHead() || user.HasPermission("view_all_students") {
			// Department heads see all students in their department
			students = getFilteredDummyStudents(r)
		} else if user.IsSupervisor() {
			// Supervisors see only their assigned students
			students = getStudentsBySupervisor(user.Email)
		} else {
			// Default: no students visible
			students = []Student{}
		}

		data := localizer.NewTemplateData(
			r.Context(),
			"student_management",
			user,
			map[string]interface{}{
				"Students": students,
				"Years":    []string{"2024", "2023", "2022"},
				"Groups":   []string{"PI22B", "PI22S", "PIT22"},
				"Programs": getLocalizedPrograms(lang, localizer),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "students-table.html", data)
	}
}

// StudentProfileHandler handles student profile page
func StudentProfileHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())

		data := localizer.NewTemplateData(
			r.Context(),
			"my_profile",
			user,
			map[string]interface{}{
				"StudentData": getStudentData(user.Email),
				"Topics":      getStudentTopics(user.Email),
				"Documents":   getStudentDocuments(user.Email),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "student-profile.html", data)
	}
}

// Helper functions
func getStudentsBySupervisor(supervisorEmail string) []Student {
	// In real implementation, query database
	allStudents := getDummyStudents()
	var supervisorStudents []Student

	for _, student := range allStudents {
		// For demo, assign some students to supervisor
		if supervisorEmail == "j.petraitis@viko.lt" && (student.Group == "PI22B" || student.Group == "PIT22") {
			supervisorStudents = append(supervisorStudents, student)
		}
	}

	return supervisorStudents
}

func getLocalizedPrograms(lang string, localizer *i18n.Localizer) []string {
	programs := []string{
		localizer.T(lang, "program_software_systems"),
		localizer.T(lang, "program_information_systems"),
		localizer.T(lang, "program_computer_networks"),
	}
	return programs
}

func getStudentData(email string) map[string]interface{} {
	// In real implementation, query database
	return map[string]interface{}{
		"StudentNumber": "220001",
		"Group":         "PI22B",
		"Program":       "Informacinės sistemos",
		"Year":          2024,
	}
}

func getStudentTopics(email string) []map[string]interface{} {
	// In real implementation, query database
	return []map[string]interface{}{
		{
			"ID":         1,
			"Title":      "Thesis Management System",
			"Status":     "In Progress",
			"Supervisor": "Dr. Jonas Petraitis",
		},
	}
}

func getStudentDocuments(email string) []map[string]interface{} {
	// In real implementation, query database
	return []map[string]interface{}{
		{
			"ID":         1,
			"Name":       "Project Plan.pdf",
			"Type":       "plan",
			"UploadedAt": "2024-01-15",
		},
		{
			"ID":         2,
			"Name":       "Literature Review.pdf",
			"Type":       "literature",
			"UploadedAt": "2024-02-01",
		},
	}
}
