package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"github.com/go-chi/chi/v5"
)

func StudentListHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get students from database
	students, total, err := getStudentsWithFilter(filter)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = templates.StudentList(user, students, locale, pagination).Render(r.Context(), w)
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
		Limit:     20,
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

	return filter
}

// Mock implementations - replace with real database queries
func getStudentsWithFilter(filter *database.StudentFilter) ([]database.StudentSummaryView, int, error) {
	// TODO: Implement actual database query
	students := getMockStudents()

	// Apply filters
	filteredStudents := applyFilters(students, filter)

	// Apply pagination
	start := (filter.Page - 1) * filter.Limit
	end := start + filter.Limit
	if end > len(filteredStudents) {
		end = len(filteredStudents)
	}

	if start >= len(filteredStudents) {
		return []database.StudentSummaryView{}, len(filteredStudents), nil
	}

	return filteredStudents[start:end], len(filteredStudents), nil
}

func applyFilters(students []database.StudentSummaryView, filter *database.StudentFilter) []database.StudentSummaryView {
	var filtered []database.StudentSummaryView

	for _, student := range students {
		// Apply search filter
		if filter.Search != nil && *filter.Search != "" {
			searchTerm := strings.ToLower(*filter.Search)
			studentText := strings.ToLower(student.StudentName + " " + student.StudentLastname + " " + student.FinalProjectTitle)
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

		// Apply year filter
		if filter.Year != nil && student.CurrentYear != *filter.Year {
			continue
		}

		filtered = append(filtered, student)
	}

	return filtered
}

func getStudentDocuments(studentID int) ([]database.Document, error) {
	// TODO: Implement actual database query
	// Mock documents for now
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
		{
			StudentRecord: database.StudentRecord{
				ID:                1,
				StudentGroup:      "PI22B",
				StudentName:       "Studentas",
				StudentLastname:   "Studentauskas",
				FinalProjectTitle: "Gyvūnų internetinė parduotuvė",
				StudentEmail:      "student@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "draft",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                2,
				StudentGroup:      "PIT22",
				StudentName:       "TestVardas",
				StudentLastname:   "StudentasPavarde",
				FinalProjectTitle: "Baigiamųjų darbų talpykla automatiniai testai",
				StudentEmail:      "test@example.com",
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
		{
			StudentRecord: database.StudentRecord{
				ID:                3,
				StudentGroup:      "PI22S",
				StudentName:       "Aleksandr",
				StudentLastname:   "Michalovsrij",
				FinalProjectTitle: "Automobilių interneto svetainė",
				StudentEmail:      "aleksandr@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "draft",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                4,
				StudentGroup:      "PI22S",
				StudentName:       "Raimondas",
				StudentLastname:   "Kalinovskis",
				FinalProjectTitle: "nepradėtas pildyti",
				StudentEmail:      "raimondas@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                5,
				StudentGroup:      "PI22S",
				StudentName:       "Vitalius",
				StudentLastname:   "Kunigiskis",
				FinalProjectTitle: "CRM sistema",
				StudentEmail:      "vitalius@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "draft",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                6,
				StudentGroup:      "PI22B",
				StudentName:       "Karolis",
				StudentLastname:   "Pakalnis",
				FinalProjectTitle: "nepradėtas pildyti",
				StudentEmail:      "karolis@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                7,
				StudentGroup:      "PIT22",
				StudentName:       "Satvijas",
				StudentLastname:   "Motiejūnas",
				FinalProjectTitle: "Interneto svetainės Baigiamųjų darbų talpykla automatiniai testai",
				StudentEmail:      "satvijas@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Programų sistemų inžinerija",
				ReviewerName:      "Ona Onaitienė",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "draft",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                8,
				StudentGroup:      "PI22S",
				StudentName:       "Tauras",
				StudentLastname:   "Petrauskas",
				FinalProjectTitle: "nepradėtas pildyti",
				StudentEmail:      "tauras@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                9,
				StudentGroup:      "PI22S",
				StudentName:       "Augustinas",
				StudentLastname:   "Jarmolavičius",
				FinalProjectTitle: "nepradėtas pildyti",
				StudentEmail:      "augustinas@example.com",
				SupervisorEmail:   "penworld@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "",
			HasSupervisorReport:    false,
			HasReviewerReport:      false,
			SupervisorReportSigned: false,
			ReviewerReportSigned:   false,
		},
		{
			StudentRecord: database.StudentRecord{
				ID:                10,
				StudentGroup:      "PI22B",
				StudentName:       "Arnoldas",
				StudentLastname:   "Gaigalas",
				FinalProjectTitle: "nepradėtas pildyti",
				StudentEmail:      "arnoldas@example.com",
				SupervisorEmail:   "m.gzegozevskis@eif.viko.lt",
				StudyProgram:      "Informatikos inžinerija",
				ReviewerName:      "Petras Petraitis",
				CurrentYear:       2024,
			},
			TopicApproved:          false,
			TopicStatus:            "",
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
