package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/jmoiron/sqlx"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/i18n"
)

type ReviewerHandler struct {
	db        *sqlx.DB // Changed from *sql.DB to *sqlx.DB
	templates *template.Template
	localizer *i18n.Localizer
}

func NewReviewerHandler(db *sqlx.DB, localizer *i18n.Localizer) *ReviewerHandler {
	funcMap := template.FuncMap{
		"printf": fmt.Sprintf,
	}

	tmpl := template.New("").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseGlob("templates/layouts/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("templates/reviewer/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("templates/components/*.html"))

	return &ReviewerHandler{
		db:        db,
		templates: tmpl,
		localizer: localizer,
	}
}

func (h *ReviewerHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	lang := i18n.GetLangFromContext(r.Context())

	// Get statistics
	var stats struct {
		TotalAssigned    int     `db:"total_assigned"`
		PendingReview    int     `db:"pending_review"`
		CompletedReviews int     `db:"completed_reviews"`
		AverageGrade     float64 `db:"average_grade"`
	}

	statsQuery := `
        SELECT 
            COUNT(DISTINCT sr.id) as total_assigned,
            COUNT(DISTINCT CASE WHEN rr.id IS NULL AND sup.is_signed = 1 THEN sr.id END) as pending_review,
            COUNT(DISTINCT CASE WHEN rr.is_signed = 1 THEN sr.id END) as completed_reviews,
            COALESCE(AVG(rr.grade), 0) as average_grade
        FROM student_records sr
        LEFT JOIN supervisor_reports sup ON sr.id = sup.student_record_id
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        WHERE sr.reviewer_email = ?
    `

	err := h.db.Get(&stats, statsQuery, user.Email)
	if err != nil {
		// Log error but continue with zero stats
		stats = struct {
			TotalAssigned    int     `db:"total_assigned"`
			PendingReview    int     `db:"pending_review"`
			CompletedReviews int     `db:"completed_reviews"`
			AverageGrade     float64 `db:"average_grade"`
		}{}
	}

	// Get students
	query := `
        SELECT sr.*,
               CASE WHEN d.id IS NOT NULL THEN 1 ELSE 0 END as has_thesis_document,
               CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_presentation_video,
               CASE WHEN sup.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
               sup.is_signed as supervisor_report_signed,
               CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
               rr.is_signed as reviewer_report_signed
        FROM student_records sr
        LEFT JOIN documents d ON sr.id = d.student_record_id AND d.document_type = 'thesis'
        LEFT JOIN videos v ON sr.id = v.student_record_id
        LEFT JOIN supervisor_reports sup ON sr.id = sup.student_record_id
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        WHERE sr.reviewer_email = ?
        ORDER BY sr.student_group, sr.student_lastname
    `

	var students []struct {
		database.StudentRecord
		HasThesisDocument      bool `db:"has_thesis_document"`
		HasPresentationVideo   bool `db:"has_presentation_video"`
		HasSupervisorReport    bool `db:"has_supervisor_report"`
		SupervisorReportSigned bool `db:"supervisor_report_signed"`
		HasReviewerReport      bool `db:"has_reviewer_report"`
		ReviewerReportSigned   bool `db:"reviewer_report_signed"`
	}

	err = h.db.Select(&students, query, user.Email)
	if err != nil {
		// Log error but continue with empty slice
		students = []struct {
			database.StudentRecord
			HasThesisDocument      bool `db:"has_thesis_document"`
			HasPresentationVideo   bool `db:"has_presentation_video"`
			HasSupervisorReport    bool `db:"has_supervisor_report"`
			SupervisorReportSigned bool `db:"supervisor_report_signed"`
			HasReviewerReport      bool `db:"has_reviewer_report"`
			ReviewerReportSigned   bool `db:"reviewer_report_signed"`
		}{}
	}

	data := map[string]interface{}{
		"Title": h.localizer.T(lang, "reviewer.dashboard_title"),
		"User":  user,
		"Lang":  lang,
		"Stats": stats,
		"Data": map[string]interface{}{
			"Students":   students,
			"TotalPages": 1, // Add pagination logic
		},
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
