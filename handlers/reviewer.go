package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/database"
	"FinalProjectManagementApp/i18n"
)

type ReviewerHandler struct {
	db        *sqlx.DB
	templates *template.Template
	localizer *i18n.Localizer
}

func NewReviewerHandler(db *sqlx.DB, localizer *i18n.Localizer) *ReviewerHandler {
	funcMap := template.FuncMap{
		"printf": fmt.Sprintf,
		"seq": func(start, end int) []int {
			seq := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				seq = append(seq, i)
			}
			return seq
		},
		// Add both conversion functions
		"float64": func(i int) float64 {
			return float64(i)
		},
		"int": func(f float64) int {
			return int(f)
		},
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

// DocumentsModalHandler shows documents for a student
func (h *ReviewerHandler) DocumentsModalHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	lang := i18n.GetLangFromContext(r.Context())

	// Fetch student info
	var student database.StudentRecord
	err := h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Fetch documents
	var documents []database.Document
	err = h.db.Select(&documents,
		"SELECT * FROM documents WHERE student_record_id = ? ORDER BY uploaded_date DESC",
		studentID)

	// Fetch video
	var video database.Video
	_ = h.db.Get(&video, "SELECT * FROM videos WHERE student_record_id = ?", studentID)

	data := map[string]interface{}{
		"Student": student,
		"Documents": map[string]interface{}{
			"Thesis":         getDocumentByType(documents, "thesis"),
			"SourceCode":     getDocumentByType(documents, "code"),
			"Recommendation": getDocumentByType(documents, "recommendation"),
			"Video":          &video,
		},
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "documents_modal.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ViewSupervisorReportHandler shows supervisor report
func (h *ReviewerHandler) ViewSupervisorReportHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	lang := i18n.GetLangFromContext(r.Context())
	user := auth.GetUserFromContext(r.Context())

	// Fetch student
	var student database.StudentRecord
	err := h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Fetch supervisor report
	var report database.SupervisorReport
	err = h.db.Get(&report,
		"SELECT * FROM supervisor_reports WHERE student_record_id = ?",
		studentID)
	if err != nil {
		http.Error(w, "Supervisor report not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Title":      h.localizer.T(lang, "reports.supervisor_report"),
		"User":       user,
		"Lang":       lang,
		"Student":    student,
		"Report":     report,
		"Department": student.Department,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	// Use supervisor's report view template
	if err := h.templates.ExecuteTemplate(w, "supervisor_report_view.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// CreateReportHandler shows create form for new report
func (h *ReviewerHandler) CreateReportHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	user := auth.GetUserFromContext(r.Context())
	lang := i18n.GetLangFromContext(r.Context())

	if r.Method == "POST" {
		h.SaveReportHandler(w, r)
		return
	}

	// Fetch student
	var student database.StudentRecord
	err := h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Try to get existing report
	var report database.ReviewerReport
	_ = h.db.Get(&report,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?",
		studentID)

	data := map[string]interface{}{
		"Title":      h.localizer.T(lang, "reports.reviewer_report"),
		"User":       user,
		"Lang":       lang,
		"Student":    student,
		"Report":     report,
		"Department": student.Department,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "report_form.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// SaveReportHandler saves reviewer report
func (h *ReviewerHandler) SaveReportHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	studentRecordID, err := strconv.Atoi(studentID)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Parse grade
	grade, err := strconv.ParseFloat(r.FormValue("grade"), 64)
	if err != nil {
		http.Error(w, "Invalid grade", http.StatusBadRequest)
		return
	}

	// Check if report exists
	var existingID int
	err = h.db.Get(&existingID,
		"SELECT id FROM reviewer_reports WHERE student_record_id = ?",
		studentRecordID)

	action := r.FormValue("action")
	isSigned := action == "sign"

	reportData := map[string]interface{}{
		"student_record_id":             studentRecordID,
		"reviewer_personal_details":     r.FormValue("reviewer_personal_details"),
		"grade":                         grade,
		"review_goals":                  r.FormValue("review_goals"),
		"review_theory":                 r.FormValue("review_theory"),
		"review_practical":              r.FormValue("review_practical"),
		"review_theory_practical_link":  r.FormValue("review_theory_practical_link"),
		"review_results":                r.FormValue("review_results"),
		"review_practical_significance": nullableString(r.FormValue("review_practical_significance")),
		"review_language":               r.FormValue("review_language"),
		"review_pros":                   r.FormValue("review_pros"),
		"review_cons":                   r.FormValue("review_cons"),
		"review_questions":              r.FormValue("review_questions"),
		"is_signed":                     isSigned,
	}

	if err == sql.ErrNoRows {
		// Create new report
		_, err = h.db.NamedExec(`
			INSERT INTO reviewer_reports 
			(student_record_id, reviewer_personal_details, grade, review_goals, review_theory,
			 review_practical, review_theory_practical_link, review_results, 
			 review_practical_significance, review_language, review_pros, review_cons,
			 review_questions, is_signed)
			VALUES 
			(:student_record_id, :reviewer_personal_details, :grade, :review_goals, :review_theory,
			 :review_practical, :review_theory_practical_link, :review_results,
			 :review_practical_significance, :review_language, :review_pros, :review_cons,
			 :review_questions, :is_signed)
		`, reportData)
	} else {
		// Update existing report
		_, err = h.db.NamedExec(`
			UPDATE reviewer_reports SET
				reviewer_personal_details = :reviewer_personal_details,
				grade = :grade,
				review_goals = :review_goals,
				review_theory = :review_theory,
				review_practical = :review_practical,
				review_theory_practical_link = :review_theory_practical_link,
				review_results = :review_results,
				review_practical_significance = :review_practical_significance,
				review_language = :review_language,
				review_pros = :review_pros,
				review_cons = :review_cons,
				review_questions = :review_questions,
				is_signed = :is_signed
			WHERE student_record_id = :student_record_id
		`, reportData)
	}

	if err != nil {
		http.Error(w, "Failed to save report", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/reviewer", http.StatusFound)
}

// EditReportHandler shows edit form for existing report
func (h *ReviewerHandler) EditReportHandler(w http.ResponseWriter, r *http.Request) {
	h.CreateReportHandler(w, r)
}

// ViewReportHandler shows read-only view of report
func (h *ReviewerHandler) ViewReportHandler(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentID")
	user := auth.GetUserFromContext(r.Context())
	lang := i18n.GetLangFromContext(r.Context())

	// Fetch student
	var student database.StudentRecord
	err := h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Fetch report
	var report database.ReviewerReport
	err = h.db.Get(&report,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?",
		studentID)
	if err != nil {
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Title":      h.localizer.T(lang, "reports.reviewer_report"),
		"User":       user,
		"Lang":       lang,
		"Student":    student,
		"Report":     report,
		"Department": student.Department,
		"T": func(key string, args ...interface{}) string {
			return h.localizer.T(lang, key, args...)
		},
	}

	if err := h.templates.ExecuteTemplate(w, "report_view.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Helper function for nullable string
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
