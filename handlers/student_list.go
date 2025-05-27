package handlers

import (
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"net/http"
	"strconv"
	"strings"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/go-chi/chi/v5"
)

type StudentListHandler struct {
	db *sqlx.DB
}

// NewSupervisorReportHandler creates a new handler instance
func NewStudentListHandler(db *sqlx.DB) *StudentListHandler {
	return &StudentListHandler{
		db: db,
	}
}

func (h *StudentListHandler) StudentTableDisplayHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check if user has permission to view student list
	if !canViewStudentList(user.Role) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse filter parameters
	filter := parseStudentFilter(r)

	// Create filter params for template
	filterParams := &database.TemplateFilterParams{
		Limit:        filter.Limit,
		Group:        getStringValue(filter.Group),
		StudyProgram: getStringValue(filter.StudyProgram),
		TopicStatus:  getStringValue(filter.TopicStatus),
		Search:       getStringValue(filter.Search),
	}

	// Get students from database
	students, total, err := h.getStudentsWithFilter(filter)
	if err != nil {
		http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
		return
	}

	// Create pagination
	pagination := database.NewPaginationInfo(filter.Page, filter.Limit, total)

	// Get locale
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}

	// Get search value for template
	searchValue := ""
	if filter.Search != nil {
		searchValue = *filter.Search
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Return only the table and pagination part for HTMX updates
		err = templates.StudentTableWithPagination(user, students, locale, pagination).Render(r.Context(), w)
	} else {
		// Return full page for regular requests
		err = templates.StudentList(user, students, locale, pagination, searchValue, filterParams).Render(r.Context(), w)
	}

	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func DocumentsAPIHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	studentIDStr := chi.URLParam(r, "id")
	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Fetch documents for student
	documents, err := getStudentDocuments(studentID)
	if err != nil {
		http.Error(w, "Failed to fetch documents", http.StatusInternalServerError)
		return
	}

	response := database.NewSuccessResponse(documents, "Documents fetched successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func canViewStudentList(role string) bool {
	allowedRoles := []string{"admin", "department_head", "supervisor", "reviewer"}
	for _, allowedRole := range allowedRoles {
		if role == allowedRole {
			return true
		}
	}
	return false
}

func parseStudentFilter(r *http.Request) *database.StudentFilter {
	filter := &database.StudentFilter{
		Page:      1,
		Limit:     10,
		SortBy:    "student_name",
		SortOrder: "asc",
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	// Parse other filters
	if dept := r.URL.Query().Get("department"); dept != "" {
		filter.Department = &dept
	}
	if prog := r.URL.Query().Get("study_program"); prog != "" {
		filter.StudyProgram = &prog
	}
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			filter.Year = &year
		}
	}
	if group := r.URL.Query().Get("group"); group != "" {
		filter.Group = &group
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}
	if topicStatus := r.URL.Query().Get("topic_status"); topicStatus != "" {
		filter.TopicStatus = &topicStatus
	}

	return filter
}

// Helper functions
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

//func getStudentsWithFilter(filter *database.StudentFilter) ([]database.StudentSummaryView, int, error) {
//	students := getMockStudents()
//	filteredStudents := applyFilters(students, filter)
//
//	// Apply pagination
//	start := (filter.Page - 1) * filter.Limit
//	end := start + filter.Limit
//	if end > len(filteredStudents) {
//		end = len(filteredStudents)
//	}
//
//	if start >= len(filteredStudents) {
//		return []database.StudentSummaryView{}, len(filteredStudents), nil
//	}
//
//	return filteredStudents[start:end], len(filteredStudents), nil
//}

func (h *StudentListHandler) getStudentsWithFilter(filter *database.StudentFilter) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}
	where := []string{"1=1"}

	// Filters
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + strings.ToLower(*filter.Search) + "%"
		where = append(where, `(LOWER(student_name) LIKE ? OR LOWER(student_lastname) LIKE ? OR LOWER(final_project_title) LIKE ? OR LOWER(student_email) LIKE ?)`)
		args = append(args, search, search, search, search)
	}
	if filter.Group != nil {
		where = append(where, "student_group = ?")
		args = append(args, *filter.Group)
	}
	if filter.StudyProgram != nil {
		where = append(where, "study_program = ?")
		args = append(args, *filter.StudyProgram)
	}
	if filter.TopicStatus != nil {
		if *filter.TopicStatus == "not_started" {
			where = append(where, "(topic_status IS NULL OR topic_status = '')")
		} else {
			where = append(where, "topic_status = ?")
			args = append(args, *filter.TopicStatus)
		}
	}
	if filter.Year != nil {
		where = append(where, "current_year = ?")
		args = append(args, *filter.Year)
	}

	// Sorting & Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "student_name"
	}
	sortOrder := strings.ToUpper(filter.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}
	offset := (filter.Page - 1) * filter.Limit

	query := `
		SELECT *
		FROM student_summary_view
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY ` + sortBy + ` ` + sortOrder + `
		LIMIT ? OFFSET ?`

	args = append(args, filter.Limit, offset)

	err := h.db.Select(&students, query, args...)
	if err != nil {
		return nil, 0, err
	}

	// Count query (without limit/offset)
	countQuery := `SELECT COUNT(*) FROM student_summary_view WHERE ` + strings.Join(where, " AND ")
	var total int
	err = h.db.Get(&total, countQuery, args[:len(args)-2]...) // exclude limit/offset
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func applyFilters(students []database.StudentSummaryView, filter *database.StudentFilter) []database.StudentSummaryView {
	var filtered []database.StudentSummaryView

	for _, student := range students {
		// Apply search filter
		if filter.Search != nil && *filter.Search != "" {
			searchTerm := strings.ToLower(*filter.Search)
			studentText := strings.ToLower(student.StudentName + " " + student.StudentLastname + " " + student.FinalProjectTitle + " " + student.StudentEmail)
			if !strings.Contains(studentText, searchTerm) {
				continue
			}
		}

		// Apply group filter
		if filter.Group != nil && *filter.Group != "" && student.StudentGroup != *filter.Group {
			continue
		}

		// Apply study program filter
		if filter.StudyProgram != nil && *filter.StudyProgram != "" && student.StudyProgram != *filter.StudyProgram {
			continue
		}

		// Apply topic status filter
		if filter.TopicStatus != nil && *filter.TopicStatus != "" {
			switch *filter.TopicStatus {
			case "not_started":
				if student.TopicStatus != "" {
					continue
				}
			case "draft":
				if student.TopicStatus != "draft" {
					continue
				}
			case "submitted":
				if student.TopicStatus != "submitted" {
					continue
				}
			case "approved":
				if !student.TopicApproved {
					continue
				}
			case "rejected":
				if student.TopicStatus != "rejected" {
					continue
				}
			default:
				if student.TopicStatus != *filter.TopicStatus {
					continue
				}
			}
		}

		// Apply year filter
		if filter.Year != nil && student.CurrentYear != *filter.Year {
			continue
		}

		filtered = append(filtered, student)
	}

	return filtered
}

func getStudentDocuments(studentID int) ([]database.Document, error) {
	documents := []database.Document{
		{
			ID:               1,
			DocumentType:     "thesis",
			OriginalFilename: database.NullableString("Baigiamasis_darbas.pdf"),
			FileSize:         nullableInt64(1024000),
		},
		{
			ID:               2,
			DocumentType:     "video",
			OriginalFilename: database.NullableString("Gynyba_video.mp4"),
			FileSize:         nullableInt64(50240000),
		},
	}
	return documents, nil
}

func getMockStudents() []database.StudentSummaryView {
	return []database.StudentSummaryView{
		// 1. Topic not started (Nepradėta)
		{
			StudentRecord: database.StudentRecord{
				ID:                1,
				StudentGroup:      "PI22B",
				StudentName:       "Studentas",
				StudentLastname:   "Nepradėjęs",
				FinalProjectTitle: "",
				StudentEmail:      "student1@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		// 2. Topic in draft (Juodraštis)
		{
			StudentRecord: database.StudentRecord{
				ID:                2,
				StudentGroup:      "PI22B",
				StudentName:       "Studentas",
				StudentLastname:   "Juodraštis",
				FinalProjectTitle: "Gyvūnų internetinė parduotuvė",
				StudentEmail:      "student2@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "draft",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		// 3. Topic submitted (Pateikta)
		{
			StudentRecord: database.StudentRecord{
				ID:                3,
				StudentGroup:      "PIT22",
				StudentName:       "Studentas",
				StudentLastname:   "Pateikta",
				FinalProjectTitle: "Automatinių testų sistema",
				StudentEmail:      "student3@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Programų sistemų inžinerija",
				ReviewerName:      "",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "submitted",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		// 4. Topic approved, no reviewer assigned
		{
			StudentRecord: database.StudentRecord{
				ID:                4,
				StudentGroup:      "PI22S",
				StudentName:       "Studentas",
				StudentLastname:   "Patvirtinta",
				FinalProjectTitle: "Automobilių interneto svetainė",
				StudentEmail:      "student4@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "",
				CurrentYear:       2024,
			},
			TopicApproved:          true,
			TopicStatus:            "approved",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		// 5. Topic approved, reviewer assigned, no reports
		{
			StudentRecord: database.StudentRecord{
				ID:                5,
				StudentGroup:      "PI22S",
				StudentName:       "Studentas",
				StudentLastname:   "Su_Recenzentu",
				FinalProjectTitle: "CRM sistema",
				StudentEmail:      "student5@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          true,
			TopicStatus:            "approved",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		// 6. Supervisor report filled, not signed
		{
			StudentRecord: database.StudentRecord{
				ID:                6,
				StudentGroup:      "PIT22",
				StudentName:       "Studentas",
				StudentLastname:   "Vadovo_Ataskaita",
				FinalProjectTitle: "Baigiamųjų darbų sistema",
				StudentEmail:      "student6@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Programų sistemų inžinerija",
				ReviewerName:      "Bronius Bronislovas",
				CurrentYear:       2024,
			},
			TopicApproved:          true,
			TopicStatus:            "approved",
			HasSupervisorReport:    true,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		// 7. Supervisor report signed
		{
			StudentRecord: database.StudentRecord{
				ID:                7,
				StudentGroup:      "PI22B",
				StudentName:       "Studentas",
				StudentLastname:   "Vadovas_Pasirašė",
				FinalProjectTitle: "Internetinė parduotuvė",
				StudentEmail:      "student7@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          true,
			TopicStatus:            "approved",
			HasSupervisorReport:    true,
			HasReviewerReport:      false,
			SupervisorReportSigned: true,
			ReviewerReportSigned:   false,
		},
		// 8. Reviewer report filled, not signed
		{
			StudentRecord: database.StudentRecord{
				ID:                8,
				StudentGroup:      "PIT22",
				StudentName:       "Studentas",
				StudentLastname:   "Recenzento_Ataskaita",
				FinalProjectTitle: "Testų automatizavimas",
				StudentEmail:      "student8@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Programų sistemų inžinerija",
				ReviewerName:      "Ona Onaitienė",
				CurrentYear:       2024,
			},
			TopicApproved:          true,
			TopicStatus:            "approved",
			HasSupervisorReport:    true,
			HasReviewerReport:      true,
			SupervisorReportSigned: true,
			ReviewerReportSigned:   false,
		},
		// 9. Everything completed and signed
		{
			StudentRecord: database.StudentRecord{
				ID:                9,
				StudentGroup:      "PI22S",
				StudentName:       "Studentas",
				StudentLastname:   "Viskas_Baigta",
				FinalProjectTitle: "Pilnai užbaigtas projektas",
				StudentEmail:      "student9@stud.viko.lt",
				SupervisorEmail:   "penworld@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          true,
			TopicStatus:            "approved",
			HasSupervisorReport:    true,
			HasReviewerReport:      true,
			SupervisorReportSigned: true,
			ReviewerReportSigned:   true,
		},
		// 10. Rejected topic
		{
			StudentRecord: database.StudentRecord{
				ID:                10,
				StudentGroup:      "PI22B",
				StudentName:       "Studentas",
				StudentLastname:   "Atmesta_Tema",
				FinalProjectTitle: "Atmesta tema",
				StudentEmail:      "student10@stud.viko.lt",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "rejected",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
	}
}

// Helper function to create nullable int64
func nullableInt64(value int64) *int64 {
	return &value
}
