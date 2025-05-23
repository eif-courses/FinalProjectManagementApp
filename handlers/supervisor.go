package handlers

import (
	_ "database/sql"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/i18n"
)

type SupervisorHandler struct {
	db        *sqlx.DB // Changed from *sql.DB to *sqlx.DB
	templates *template.Template
	localizer *i18n.Localizer
}

func NewSupervisorHandler(db *sqlx.DB, localizer *i18n.Localizer) *SupervisorHandler {
	// Parse templates with helper functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"seq": func(start, end int) []int {
			seq := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				seq = append(seq, i)
			}
			return seq
		},
	}

	tmpl := template.New("").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseGlob("templates/layouts/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("templates/supervisor/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("templates/components/*.html"))

	return &SupervisorHandler{
		db:        db,
		templates: tmpl,
		localizer: localizer,
	}
}

// Helper functions for parsing form values
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &i
}

func (h *SupervisorHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	lang := i18n.GetLangFromContext(r.Context())

	// Get students for this supervisor
	query := `
        SELECT sr.*, 
               CASE WHEN ptr.id IS NOT NULL THEN 1 ELSE 0 END as has_topic,
               ptr.status as topic_status,
               CASE WHEN sup.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
               sup.is_signed as supervisor_report_signed
        FROM student_records sr
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
        LEFT JOIN supervisor_reports sup ON sr.id = sup.student_record_id
        WHERE sr.supervisor_email = ?
        ORDER BY sr.student_group, sr.student_lastname
    `

	var students []struct {
		database.StudentRecord
		HasTopic               bool   `db:"has_topic"`
		TopicStatus            string `db:"topic_status"`
		HasSupervisorReport    bool   `db:"has_supervisor_report"`
		SupervisorReportSigned bool   `db:"supervisor_report_signed"`
	}

	err := h.db.Select(&students, query, user.Email)
	if err != nil {
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": h.localizer.T(lang, "supervisor.dashboard_title"),
		"User":  user,
		"Lang":  lang,
		"Data": map[string]interface{}{
			"Students": students,
			"Total":    len(students),
		},
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SupervisorHandler) TopicModalHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	lang := i18n.GetLangFromContext(r.Context())

	// Fetch topic with comments
	var topic database.ProjectTopicRegistration
	err := h.db.Get(&topic, "SELECT * FROM project_topic_registrations WHERE student_record_id = ?", studentID)
	if err != nil {
		http.Error(w, "Topic not found", http.StatusNotFound)
		return
	}

	// Fetch comments
	var comments []database.TopicRegistrationComment
	err = h.db.Select(&comments,
		"SELECT * FROM topic_registration_comments WHERE topic_registration_id = ? ORDER BY created_at DESC",
		topic.ID)

	data := map[string]interface{}{
		"Topic":    topic,
		"Comments": comments,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "topic_modal.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SupervisorHandler) CreateReportHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	user := auth.GetUserFromContext(r.Context())
	lang := i18n.GetLangFromContext(r.Context())

	if r.Method == "POST" {
		// Handle form submission
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Convert studentID string to int
		studentRecordID, err := strconv.Atoi(studentID)
		if err != nil {
			http.Error(w, "Invalid student ID", http.StatusBadRequest)
			return
		}

		// Create report
		report := database.SupervisorReport{
			StudentRecordID:     studentRecordID, // Now using int
			SupervisorName:      r.FormValue("supervisor_name"),
			SupervisorPosition:  r.FormValue("supervisor_position"),
			SupervisorWorkplace: r.FormValue("supervisor_workplace"),
			SupervisorComments:  r.FormValue("supervisor_comments"),
			OtherMatch:          parseFloat(r.FormValue("other_match")),
			OneMatch:            parseFloat(r.FormValue("one_match")),
			OwnMatch:            parseFloat(r.FormValue("own_match")),
			JoinMatch:           parseFloat(r.FormValue("join_match")),
			Grade:               parseIntPtr(r.FormValue("grade")), // Using parseIntPtr for *int
			IsPassOrFailed:      r.FormValue("is_pass_or_failed") == "1",
			FinalComments:       r.FormValue("final_comments"),
		}

		action := r.FormValue("action")
		if action == "sign" {
			report.IsSigned = true
		}

		// Save to database
		_, err = h.db.NamedExec(`
            INSERT INTO supervisor_reports 
            (student_record_id, supervisor_name, supervisor_position, supervisor_workplace,
             supervisor_comments, other_match, one_match, own_match, join_match,
             grade, is_pass_or_failed, final_comments, is_signed)
            VALUES 
            (:student_record_id, :supervisor_name, :supervisor_position, :supervisor_workplace,
             :supervisor_comments, :other_match, :one_match, :own_match, :join_match,
             :grade, :is_pass_or_failed, :final_comments, :is_signed)
        `, report)

		if err != nil {
			http.Error(w, "Failed to save report", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/supervisor/students", http.StatusFound)
		return
	}

	// Show form
	var student database.StudentRecord
	err := h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	grades := []struct {
		Value int
		Label string
	}{
		{10, h.localizer.T(lang, "grades.10")},
		{9, h.localizer.T(lang, "grades.9")},
		{8, h.localizer.T(lang, "grades.8")},
		{7, h.localizer.T(lang, "grades.7")},
		{6, h.localizer.T(lang, "grades.6")},
		{5, h.localizer.T(lang, "grades.5")},
	}

	data := map[string]interface{}{
		"Title":      h.localizer.T(lang, "reports.supervisor_report"),
		"User":       user,
		"Lang":       lang,
		"Student":    student,
		"Report":     database.SupervisorReport{}, // Empty for new report
		"Grades":     grades,
		"Department": student.Department,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "report_form.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
