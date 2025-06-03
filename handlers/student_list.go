package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/components/templates"
	"FinalProjectManagementApp/database"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type StudentListHandler struct {
	db *sqlx.DB
}

// NewStudentListHandler creates a new handler instance
func NewStudentListHandler(db *sqlx.DB) *StudentListHandler {
	return &StudentListHandler{
		db: db,
	}
}

// ImportModalHandler shows the import modal
func (h *StudentListHandler) ImportModalHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	err := templates.ImportModal(user).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// PreviewHandler handles file preview
func (h *StudentListHandler) PreviewHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Add role validation to use the user variable
	if user.Role != auth.RoleAdmin && user.Role != auth.RoleDepartmentHead {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	records, err := h.processFile(file, header.Filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing file: %v", err), http.StatusBadRequest)
		return
	}

	// Show ALL records in preview - REMOVED THE LIMIT
	previewRecords := records
	// REMOVED: if len(records) > 5 {
	//     previewRecords = records[:5]
	// }

	// Render preview template with all records
	err = templates.ImportPreview(previewRecords, len(records)).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render preview", http.StatusInternalServerError)
	}
}
func (h *StudentListHandler) ProcessImportHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse options
	overwriteExisting := r.FormValue("overwrite_existing") == "true"
	validateEmails := r.FormValue("validate_emails") == "true"
	sendNotifications := r.FormValue("send_notifications") == "true"

	records, err := h.processFile(file, header.Filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing file: %v", err), http.StatusBadRequest)
		return
	}

	// Get department for department heads
	var allowedDepartment string
	if user.Role == auth.RoleDepartmentHead {
		var departmentHead database.DepartmentHead
		err := h.db.Get(&departmentHead, "SELECT * FROM department_heads WHERE email = ? AND is_active = 1", user.Email)
		if err != nil {
			http.Error(w, "Failed to get department head info", http.StatusInternalServerError)
			return
		}
		allowedDepartment = departmentHead.Department
		log.Printf("Department head %s importing for department: %s", user.Email, allowedDepartment)
	}

	// Use database.ImportOptions with department restriction
	importResult, err := h.importStudentRecords(records, database.ImportOptions{
		OverwriteExisting: overwriteExisting,
		ValidateEmails:    validateEmails,
		SendNotifications: sendNotifications,
		ImportedByEmail:   user.Email,
		AllowedDepartment: allowedDepartment, // Add this field to ImportOptions
		UserRole:          user.Role,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Import failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Create audit log
	h.createImportAuditLog(user.Email, len(records), importResult.SuccessCount, importResult.ErrorCount)

	err = templates.ImportResults(importResult, "lt").Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render results", http.StatusInternalServerError)
	}
}

//====================================================EXPORT HANDLER===============

// Helper functions for status determination
func getTopicRegistrationStatus(student database.StudentSummaryView) string {
	if student.TopicApproved {
		return "Patvirtinta katedros vedėjo"
	}

	status := getTopicStatus(student.TopicStatus)
	switch status {
	case "submitted":
		return "Pateikta peržiūrai"
	case "supervisor_approved":
		return "Vadovas patvirtino"
	case "revision_requested":
		return "Prašoma pataisymų"
	case "draft":
		return "Juodraštis"
	case "rejected":
		return "Atmesta"
	default:
		return "Nepradėtas pildyti"
	}
}

func getReviewerReportStatus(student database.StudentSummaryView) string {
	if student.ReviewerName == "" {
		return "Nepaskirtas"
	}

	if !student.HasReviewerReport {
		return "Neužpildyta"
	}

	if student.ReviewerReportSigned.Valid && student.ReviewerReportSigned.Bool {
		return "Pasirašyta"
	}

	return "Užpildyta"
}

func getSupervisorReportStatus(student database.StudentSummaryView) string {
	if !student.HasSupervisorReport {
		return "Neužpildyta"
	}

	if student.SupervisorReportSigned.Valid && student.SupervisorReportSigned.Bool {
		return "Pasirašyta"
	}

	return "Užpildyta"
}

// Add this method to fetch actual documents from database
func (h *StudentListHandler) getStudentDocuments(studentID int) ([]database.Document, error) {
	var documents []database.Document

	query := `
        SELECT id, student_record_id, document_type, original_filename, 
               storage_path, file_size, status, uploaded_at
        FROM documents 
        WHERE student_record_id = ?
    `

	err := h.db.Select(&documents, query, studentID)
	return documents, err
}

// ExportHandler handles data export
func (h *StudentListHandler) ExportHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse filters from query parameters
	filters := &database.TemplateFilterParams{
		Page:         1,
		Limit:        10000, // Export all
		Group:        r.URL.Query().Get("group"),
		StudyProgram: r.URL.Query().Get("study_program"),
		TopicStatus:  r.URL.Query().Get("topic_status"),
		Search:       r.URL.Query().Get("search"),
	}

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			filters.Year = year
		}
	}

	// Get students based on user role using your existing methods
	var students []database.StudentSummaryView
	var err error

	switch user.Role {
	case auth.RoleDepartmentHead:
		students, _, err = h.getStudentsForDepartmentHead(user, filters)
	case auth.RoleAdmin:
		students, _, err = h.getAllStudents(filters)
	case auth.RoleSupervisor:
		students, _, err = h.getStudentsForSupervisor(user.Email, filters)
	default:
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch students: %v", err), http.StatusInternalServerError)
		return
	}

	// Create Excel file
	file := excelize.NewFile()
	defer file.Close()

	sheetName := "Students"
	file.NewSheet(sheetName)
	file.DeleteSheet("Sheet1")

	// Extended headers with status columns
	headers := []string{
		"StudentName",
		"StudentLastname",
		"StudentNumber",
		"StudentEmail",
		"StudentGroup",
		"FinalProjectTitle",
		"FinalProjectTitleEn",
		"SupervisorEmail",
		"StudyProgram",
		"Department",
		"ProgramCode",
		"CurrentYear",
		"ReviewerEmail",
		"ReviewerName",
		// Status columns
		"TopicRegistrationStatus", // Temos registravimo lapas status
		"HasVideo",                // Video document
		"HasThesisPDF",            // PDF document
		"HasRecommendation",       // Recommendation document
		"HasSourceCode",           // Source code/GitHub
		"ReviewerReportStatus",    // Recenzento status
		"SupervisorReportStatus",  // Vadovo status
	}

	// Style for headers
	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// Write headers
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheetName, cell, header)
		file.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write data with additional processing for status fields
	for i, student := range students {
		row := i + 2

		// Basic student data
		file.SetCellValue(sheetName, fmt.Sprintf("A%d", row), student.StudentName)
		file.SetCellValue(sheetName, fmt.Sprintf("B%d", row), student.StudentLastname)
		file.SetCellValue(sheetName, fmt.Sprintf("C%d", row), student.StudentNumber)
		file.SetCellValue(sheetName, fmt.Sprintf("D%d", row), student.StudentEmail)
		file.SetCellValue(sheetName, fmt.Sprintf("E%d", row), student.StudentGroup)
		file.SetCellValue(sheetName, fmt.Sprintf("F%d", row), student.FinalProjectTitle)
		file.SetCellValue(sheetName, fmt.Sprintf("G%d", row), student.FinalProjectTitleEn)
		file.SetCellValue(sheetName, fmt.Sprintf("H%d", row), student.SupervisorEmail)
		file.SetCellValue(sheetName, fmt.Sprintf("I%d", row), student.StudyProgram)
		file.SetCellValue(sheetName, fmt.Sprintf("J%d", row), student.Department)
		file.SetCellValue(sheetName, fmt.Sprintf("K%d", row), student.ProgramCode)
		file.SetCellValue(sheetName, fmt.Sprintf("L%d", row), student.CurrentYear)
		file.SetCellValue(sheetName, fmt.Sprintf("M%d", row), student.ReviewerEmail)
		file.SetCellValue(sheetName, fmt.Sprintf("N%d", row), student.ReviewerName)

		// Topic Registration Status
		topicStatus := getTopicRegistrationStatus(student)
		file.SetCellValue(sheetName, fmt.Sprintf("O%d", row), topicStatus)

		// Document statuses - fetch from database
		documents, _ := h.getStudentDocuments(student.ID)

		// Check for specific document types
		hasVideo := "Ne"
		hasThesisPDF := "Ne"
		hasRecommendation := "Ne"

		for _, doc := range documents {
			switch doc.DocumentType {
			case "video":
				hasVideo = "Taip"
			case "thesis", "thesis_pdf":
				hasThesisPDF = "Taip"
			case "recommendation":
				hasRecommendation = "Taip"
			}
		}

		file.SetCellValue(sheetName, fmt.Sprintf("P%d", row), hasVideo)
		file.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), hasThesisPDF)
		file.SetCellValue(sheetName, fmt.Sprintf("R%d", row), hasRecommendation)

		// Source code status
		sourceCodeStatus := "Ne"
		if student.HasSourceCode {
			sourceCodeStatus = "Taip"
		}
		file.SetCellValue(sheetName, fmt.Sprintf("S%d", row), sourceCodeStatus)

		// Reviewer Report Status
		reviewerStatus := getReviewerReportStatus(student)
		file.SetCellValue(sheetName, fmt.Sprintf("T%d", row), reviewerStatus)

		// Supervisor Report Status
		supervisorStatus := getSupervisorReportStatus(student)
		file.SetCellValue(sheetName, fmt.Sprintf("U%d", row), supervisorStatus)
	}

	// Auto-size columns
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		file.SetColWidth(sheetName, col, col, 15)
	}

	// Set response headers
	filename := fmt.Sprintf("students_export_%s.xlsx", time.Now().Format("2006-01-02_15-04"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// Write file to response
	err = file.Write(w)
	if err != nil {
		http.Error(w, "Failed to write Excel file", http.StatusInternalServerError)
	}
}

// SampleExcelHandler provides a sample Excel file
func (h *StudentListHandler) SampleExcelHandler(w http.ResponseWriter, r *http.Request) {
	file := excelize.NewFile()
	defer file.Close()

	sheetName := "Students"
	file.NewSheet(sheetName)
	file.DeleteSheet("Sheet1")

	// Headers - including FinalProjectTitleEn
	headers := []string{
		"StudentName",
		"StudentLastname",
		"StudentNumber",
		"StudentEmail",
		"StudentGroup",
		"FinalProjectTitle",
		"FinalProjectTitleEn", // Added this column
		"SupervisorEmail",
		"StudyProgram",
		"Department",
		"ProgramCode",
		"CurrentYear",
		"ReviewerEmail",
		"ReviewerName",
	}

	// Sample data - added English title column
	sampleData := [][]interface{}{
		{
			"Jonas",
			"Jonaitis",
			"s123456",
			"jonas.jonaitis@stud.viko.lt",
			"PI22A",
			"Pavyzdinis baigiamasis darbas",
			"Sample Final Project", // English title
			"supervisor@viko.lt",
			"Programų sistemos",
			"Programinės įrangos",
			"6531BX028",
			2025,
			"reviewer@viko.lt",
			"Dr. Recenzentas",
		},
		{
			"Petras",
			"Petraitis",
			"s123457",
			"petras.petraitis@stud.viko.lt",
			"PI22B",
			"Kitas pavyzdinis darbas",
			"Another Sample Project", // English title
			"supervisor2@viko.lt",
			"Programų sistemos",
			"Programinės įrangos",
			"6531BX028",
			2025,
			"reviewer2@viko.lt",
			"Prof. Kitas Recenzentas",
		},
	}

	// Style headers (optional but nice to have)
	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})

	// Write headers with style
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		file.SetCellValue(sheetName, cell, header)
		file.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write sample data
	for i, row := range sampleData {
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			file.SetCellValue(sheetName, cell, value)
		}
	}

	// Auto-size columns for better readability (optional)
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		file.SetColWidth(sheetName, col, col, 15)
	}

	// Add a note/instruction row (optional)
	instructionRow := 4
	cell, _ := excelize.CoordinatesToCellName(1, instructionRow)
	file.SetCellValue(sheetName, cell, "Pastaba: FinalProjectTitleEn stulpelis yra neprivalomas. Jei neturite angliško pavadinimo, palikite tuščią.")

	// Merge cells for the instruction
	endCell, _ := excelize.CoordinatesToCellName(len(headers), instructionRow)
	file.MergeCell(sheetName, cell, endCell)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\"student_import_sample.xlsx\"")

	err := file.Write(w)
	if err != nil {
		http.Error(w, "Failed to write sample file", http.StatusInternalServerError)
	}
}

// Helper methods for file processing and import
func (h *StudentListHandler) processFile(file io.Reader, filename string) ([]map[string]string, error) {
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") || strings.HasSuffix(strings.ToLower(filename), ".xls") {
		return h.processExcelFile(file)
	} else if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		return h.processCSVFile(file)
	}
	return nil, fmt.Errorf("unsupported file type")
}

// Add this temporary debug function
func (h *StudentListHandler) debugExcelHeaders(file io.Reader) {
	content, _ := io.ReadAll(file)
	f, _ := excelize.OpenReader(strings.NewReader(string(content)))
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) > 0 {
		rows, _ := f.GetRows(sheets[0])
		if len(rows) > 0 {
			log.Printf("=== EXCEL HEADERS ===")
			for i, header := range rows[0] {
				log.Printf("Column %d: '%s'", i+1, header)
			}
			log.Printf("===================")
		}
	}
}

func (h *StudentListHandler) debugRecord(record map[string]string, rowNum int) {
	log.Printf("=== Row %d Debug ===", rowNum)
	for key, value := range record {
		log.Printf("  %s: %s", key, value)
	}
	log.Printf("==================")
}

func (h *StudentListHandler) processExcelFile(file io.Reader) ([]map[string]string, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	f, err := excelize.OpenReader(strings.NewReader(string(content)))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file must contain at least a header row and one data row")
	}

	headers := rows[0]

	// Debug: print headers
	log.Printf("Excel headers found: %v", headers)

	var records []map[string]string

	for i := 1; i < len(rows); i++ {
		record := make(map[string]string)
		for j, header := range headers {
			if j < len(rows[i]) {
				// Trim spaces and normalize header
				normalizedHeader := strings.TrimSpace(header)
				value := ""
				if j < len(rows[i]) {
					value = strings.TrimSpace(rows[i][j])
				}
				record[normalizedHeader] = value
			}
		}

		// Debug first few records
		if i <= 3 {
			log.Printf("=== Row %d ===", i)
			for key, value := range record {
				log.Printf("  '%s': '%s'", key, value)
			}
			log.Printf("==================")
		}

		records = append(records, record)
	}

	return records, nil
}

func (h *StudentListHandler) processCSVFile(file io.Reader) ([]map[string]string, error) {
	reader := csv.NewReader(file)
	reader.Comma = '\t' // Tab-separated

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading headers: %v", err)
	}

	var records []map[string]string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading row: %v", err)
		}

		recordMap := make(map[string]string)
		for i, header := range headers {
			if i < len(record) {
				recordMap[header] = strings.TrimSpace(record[i])
			}
		}
		records = append(records, recordMap)
	}

	return records, nil
}

// Update this method signature
func (h *StudentListHandler) importStudentRecords(records []map[string]string, options database.ImportOptions) (*database.ImportResult, error) {
	result := &database.ImportResult{
		Errors:      []database.ImportError{},
		Duplicates:  []string{},
		NewStudents: []database.StudentRecord{},
	}

	tx, err := h.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	for i, record := range records {
		// Pass the full options to mapRecordToStudent
		student, err := h.mapRecordToStudent(record, options.ValidateEmails, options)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, database.ImportError{
				Row:     i + 2,
				Message: err.Error(),
				Data:    record,
			})
			continue
		}

		// Rest of the import logic remains the same...
		// Check for duplicates
		exists, existingID, err := h.studentExists(tx, student.StudentNumber, student.StudentEmail)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, database.ImportError{
				Row:     i + 2,
				Message: fmt.Sprintf("Error checking duplicate: %v", err),
				Data:    record,
			})
			continue
		}

		if exists {
			if options.OverwriteExisting {
				// Update existing student
				err = h.updateStudent(tx, existingID, student)
				if err != nil {
					result.ErrorCount++
					result.Errors = append(result.Errors, database.ImportError{
						Row:     i + 2,
						Message: fmt.Sprintf("Error updating student: %v", err),
						Data:    record,
					})
					continue
				}
				result.SuccessCount++
			} else {
				result.Duplicates = append(result.Duplicates, student.StudentNumber)
				continue
			}
		} else {
			// Insert new student
			err = h.insertStudent(tx, student)
			if err != nil {
				result.ErrorCount++
				result.Errors = append(result.Errors, database.ImportError{
					Row:     i + 2,
					Message: fmt.Sprintf("Database error: %v", err),
					Data:    record,
				})
				continue
			}
			result.SuccessCount++
			result.NewStudents = append(result.NewStudents, *student)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return result, nil
}

func (h *StudentListHandler) mapRecordToStudent(record map[string]string, validateEmails bool, options database.ImportOptions) (*database.StudentRecord, error) {
	// Helper function to get value with case-insensitive matching
	getValue := func(keys ...string) string {
		// First try exact matches
		for _, key := range keys {
			if val, ok := record[key]; ok && val != "" {
				return val
			}
		}

		// Then try case-insensitive matches
		for recordKey, recordValue := range record {
			for _, searchKey := range keys {
				if strings.EqualFold(strings.TrimSpace(recordKey), strings.TrimSpace(searchKey)) && recordValue != "" {
					return recordValue
				}
			}
		}

		return ""
	}

	// Debug: print what keys we have in the record
	log.Printf("Available keys in record: %v", getRecordKeys(record))

	student := &database.StudentRecord{
		// Try multiple possible header names with variations
		StudentName:         getValue("StudentName", "student_name", "Name", "Vardas", "Student Name"),
		StudentLastname:     getValue("StudentLastname", "student_lastname", "Lastname", "Pavardė", "Student Lastname", "StudentSurname"),
		StudentNumber:       getValue("StudentNumber", "student_number", "Number", "Numeris", "Student Number"),
		StudentEmail:        getValue("StudentEmail", "student_email", "Email", "El. paštas", "Student Email"),
		StudentGroup:        getValue("StudentGroup", "student_group", "Group", "Grupė", "Student Group"),
		FinalProjectTitle:   getValue("FinalProjectTitle", "final_project_title", "Title", "Tema", "Final Project Title"),
		FinalProjectTitleEn: getValue("FinalProjectTitleEn", "final_project_title_en", "TitleEn", "Title En", "FinalProjectTitle En"),
		SupervisorEmail:     getValue("SupervisorEmail", "supervisor_email", "Supervisor", "Vadovas", "Supervisor Email"),
		StudyProgram:        getValue("StudyProgram", "study_program", "Program", "Programa", "Study Program"),
		Department:          getValue("Department", "department", "Katedra", "Dept"),
		ProgramCode:         getValue("ProgramCode", "program_code", "Code", "Kodas", "Program Code"),
		ReviewerEmail:       getValue("ReviewerEmail", "reviewer_email", "Reviewer Email"),
		ReviewerName:        getValue("ReviewerName", "reviewer_name", "Reviewer", "Recenzentas", "Reviewer Name"),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Parse current year with multiple possible keys
	yearStr := getValue("CurrentYear", "current_year", "Year", "Metai", "Current Year")
	if yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			return nil, fmt.Errorf("invalid year: %s", yearStr)
		}
		student.CurrentYear = year
	}

	// Rest of your validation logic...
	// Log what we mapped
	log.Printf("Mapped student: %+v", student)

	// Validate required fields
	if student.StudentName == "" {
		return nil, fmt.Errorf("student name is required")
	}
	if student.StudentLastname == "" {
		return nil, fmt.Errorf("student lastname is required")
	}
	if student.StudentNumber == "" {
		return nil, fmt.Errorf("student number is required")
	}
	if student.StudentEmail == "" {
		return nil, fmt.Errorf("student email is required")
	}

	// Rest of your validation...
	return student, nil
}

// Helper function to get record keys for debugging
func getRecordKeys(record map[string]string) []string {
	keys := make([]string, 0, len(record))
	for k := range record {
		keys = append(keys, k)
	}
	return keys
}

func (h *StudentListHandler) studentExists(tx *sqlx.Tx, studentNumber, email string) (bool, int, error) {
	var id int
	err := tx.Get(&id, "SELECT id FROM student_records WHERE student_number = ? OR student_email = ?", studentNumber, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, id, nil
}

func (h *StudentListHandler) insertStudent(tx *sqlx.Tx, student *database.StudentRecord) error {
	query := `
        INSERT INTO student_records (
            student_group, student_name, student_lastname, student_number,
            student_email, final_project_title, final_project_title_en,
            supervisor_email, study_program, department, program_code,
            current_year, reviewer_email, reviewer_name, created_at, updated_at
        ) VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
        )`

	result, err := tx.Exec(query,
		student.StudentGroup, student.StudentName, student.StudentLastname,
		student.StudentNumber, student.StudentEmail, student.FinalProjectTitle,
		student.FinalProjectTitleEn, student.SupervisorEmail, student.StudyProgram,
		student.Department, student.ProgramCode, student.CurrentYear,
		student.ReviewerEmail, student.ReviewerName, student.CreatedAt, student.UpdatedAt,
	)

	if err != nil {
		log.Printf("Insert error: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Inserted %d rows", rowsAffected)

	return nil
}

func (h *StudentListHandler) updateStudent(tx *sqlx.Tx, id int, student *database.StudentRecord) error {
	query := `
        UPDATE student_records SET
            student_group = ?, student_name = ?, student_lastname = ?,
            student_number = ?, student_email = ?, final_project_title = ?,
            final_project_title_en = ?, supervisor_email = ?, study_program = ?,
            department = ?, program_code = ?, current_year = ?,
            reviewer_email = ?, reviewer_name = ?, updated_at = ?
        WHERE id = ?`

	student.UpdatedAt = time.Now()
	_, err := tx.Exec(query,
		student.StudentGroup, student.StudentName, student.StudentLastname,
		student.StudentNumber, student.StudentEmail, student.FinalProjectTitle,
		student.FinalProjectTitleEn, student.SupervisorEmail, student.StudyProgram,
		student.Department, student.ProgramCode, student.CurrentYear,
		student.ReviewerEmail, student.ReviewerName, student.UpdatedAt, id,
	)
	return err
}

func (h *StudentListHandler) createImportAuditLog(userEmail string, totalRecords, successCount, errorCount int) {
	auditLog := database.AuditLog{
		UserEmail:    userEmail,
		UserRole:     "department_head",
		Action:       "import_students",
		ResourceType: "student_records",
		Details:      &[]string{fmt.Sprintf("Imported %d/%d students successfully", successCount, totalRecords)}[0],
		CreatedAt:    time.Now(),
		Success:      errorCount == 0,
	}

	// Create audit log
	database.CreateAuditLog(auditLog)
}

// Add missing getLocale function
func getLocale(r *http.Request) string {
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "lt"
	}
	return locale
}

// Update your main handler method
func (h *StudentListHandler) StudentTableDisplayHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !canViewStudentList(user.Role) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse filter parameters from URL
	filterParams := parseTemplateFilterParams(r)

	// Get filter options for current user
	filterOptions, err := h.getFilterOptions(user.Role, user.Email)
	if err != nil {
		log.Printf("Error getting filter options: %v", err)
		// Continue with empty options if there's an error
		filterOptions = &database.FilterOptions{}
	}

	// Apply role-based filtering
	var students []database.StudentSummaryView
	var total int

	switch user.Role {
	case auth.RoleSupervisor:
		students, total, err = h.getStudentsForSupervisor(user.Email, filterParams)
	case auth.RoleDepartmentHead:
		students, total, err = h.getStudentsForDepartmentHead(user, filterParams)
	case auth.RoleAdmin:
		students, total, err = h.getAllStudents(filterParams)
	case auth.RoleReviewer:
		students, total, err = h.getStudentsForReviewer(user.Email, filterParams)
	default:
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err != nil {
		log.Printf("Error fetching students: %v", err)
		http.Error(w, "Failed to fetch students", http.StatusInternalServerError)
		return
	}

	// Create pagination
	pagination := database.NewPaginationInfo(filterParams.Page, filterParams.Limit, total)

	// Get locale and search value
	locale := getLocale(r)
	searchValue := r.URL.Query().Get("search")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		err = templates.StudentTableWithPagination(user, students, locale, pagination).Render(r.Context(), w)
	} else {
		// Pass filter options to template
		err = templates.StudentList(user, students, locale, pagination, searchValue, filterParams, filterOptions).Render(r.Context(), w)
	}

	if err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

// Parse template filter params
func parseTemplateFilterParams(r *http.Request) *database.TemplateFilterParams {
	params := &database.TemplateFilterParams{
		Page:  1,
		Limit: 10,
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}

	// Parse year
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil && year > 0 {
			params.Year = year
		}
	}

	// Parse other filters
	params.Group = r.URL.Query().Get("group")
	params.StudyProgram = r.URL.Query().Get("study_program")
	params.TopicStatus = r.URL.Query().Get("topic_status")
	params.Search = r.URL.Query().Get("search")

	return params
}

// Role-based query methods using your StudentSummaryView model
func (h *StudentListHandler) getStudentsForSupervisor(supervisorEmail string, filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}

	baseQuery := `
        SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
        FROM student_records sr
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
        LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
        LEFT JOIN documents d ON sr.id = d.student_record_id AND d.document_type = 'thesis_source_code'
        WHERE sr.supervisor_email = ?` // Add this WHERE clause

	args = append(args, supervisorEmail)

	// Apply filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err := h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (h *StudentListHandler) getStudentsForDepartmentHead(user *auth.AuthenticatedUser, filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	// Get the department head's information
	var departmentHead database.DepartmentHead
	err := h.db.Get(&departmentHead, "SELECT * FROM department_heads WHERE email = ? AND is_active = 1", user.Email)
	if err != nil {
		log.Printf("Error getting department head info for %s: %v", user.Email, err)
		return nil, 0, fmt.Errorf("failed to get department head info: %v", err)
	}

	log.Printf("Department head %s belongs to department: %s", user.Email, departmentHead.Department)

	// Rest of the function remains the same...
	var students []database.StudentSummaryView
	var args []interface{}

	baseQuery := `
        SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
        FROM student_records sr
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
        LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
        LEFT JOIN documents d ON sr.id = d.student_record_id AND d.document_type = 'thesis_source_code'
        WHERE sr.department = ?`

	args = append(args, departmentHead.Department)

	// Apply additional filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err = h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	log.Printf("Found %d students in department %s", total, departmentHead.Department)

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (h *StudentListHandler) getStudentsForReviewer(reviewerEmail string, filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}

	baseQuery := `
        SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
        FROM student_records sr
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
        LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
        LEFT JOIN documents d ON sr.id = d.student_record_id AND d.document_type = 'thesis_source_code'
        WHERE sr.reviewer_email = ?` // Add this WHERE clause

	args = append(args, reviewerEmail)

	// Apply filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err := h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (h *StudentListHandler) getAllStudents(filters *database.TemplateFilterParams) ([]database.StudentSummaryView, int, error) {
	var students []database.StudentSummaryView
	var args []interface{}

	baseQuery := `
        SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video
        FROM student_records sr
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id
        LEFT JOIN supervisor_reports spr ON sr.id = spr.student_record_id
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        LEFT JOIN videos v ON sr.id = v.student_record_id AND v.status = 'ready'
        LEFT JOIN documents d ON sr.id = d.student_record_id AND d.document_type = 'thesis_source_code'
        WHERE 1=1`

	// Apply filters
	whereClause, filterArgs := buildWhereClause(filters)
	if whereClause != "" {
		baseQuery += " AND " + whereClause
		args = append(args, filterArgs...)
	}

	// Add ordering
	baseQuery += " ORDER BY sr.student_lastname, sr.student_name"

	// Updated count query
	countQuery := strings.Replace(baseQuery,
		`SELECT 
            sr.id, sr.student_group, sr.student_name, sr.student_lastname,
            sr.student_email, sr.final_project_title, sr.final_project_title_en,
            sr.supervisor_email, sr.reviewer_name, sr.reviewer_email,
            sr.study_program, sr.department, sr.current_year, sr.program_code,
            sr.student_number, sr.is_favorite, sr.is_public_defense, 
            sr.defense_date, sr.defense_location, sr.created_at, sr.updated_at,
            CASE 
                WHEN d.id IS NOT NULL AND d.document_type = 'thesis_source_code' 
                THEN 1 
                ELSE 0 
            END as has_source_code,
            COALESCE(ptr.status, '') as topic_status, 
            CASE WHEN ptr.approved_at IS NOT NULL THEN 1 ELSE 0 END as topic_approved,
            ptr.approved_by, ptr.approved_at,
            CASE WHEN spr.id IS NOT NULL THEN 1 ELSE 0 END as has_supervisor_report,
            spr.is_signed as supervisor_report_signed,
            CASE WHEN rr.id IS NOT NULL THEN 1 ELSE 0 END as has_reviewer_report,
            rr.is_signed as reviewer_report_signed,
            rr.grade as reviewer_grade,
            CASE WHEN v.id IS NOT NULL THEN 1 ELSE 0 END as has_video`,
		"SELECT COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " ORDER BY sr.student_lastname, sr.student_name", "", 1)

	var total int
	err := h.db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	// Execute main query
	err = h.db.Select(&students, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

// Helper function to build WHERE clause for filters
func buildWhereClause(filters *database.TemplateFilterParams) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filters.Group != "" {
		conditions = append(conditions, "sr.student_group = ?")
		args = append(args, filters.Group)
	}

	if filters.StudyProgram != "" {
		conditions = append(conditions, "sr.study_program = ?")
		args = append(args, filters.StudyProgram)
	}

	if filters.Year > 0 {
		conditions = append(conditions, "sr.current_year = ?")
		args = append(args, filters.Year)
	}

	if filters.TopicStatus != "" {
		if filters.TopicStatus == "not_started" {
			conditions = append(conditions, "(ptr.status IS NULL OR ptr.status = '')")
		} else {
			conditions = append(conditions, "ptr.status = ?")
			args = append(args, filters.TopicStatus)
		}
	}

	if filters.Search != "" {
		conditions = append(conditions, `(
            sr.student_name LIKE ? OR 
            sr.student_lastname LIKE ? OR 
            sr.student_email LIKE ? OR 
            sr.final_project_title LIKE ? OR 
            sr.final_project_title_en LIKE ?
        )`)
		searchPattern := "%" + filters.Search + "%"
		for i := 0; i < 5; i++ {
			args = append(args, searchPattern)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " AND "), args
}

// Keep your existing helper functions
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

func getStudentDocuments(studentID int) ([]database.Document, error) {
	documents := []database.Document{
		{
			ID:               1,
			DocumentType:     "thesis",
			OriginalFilename: database.NullableString("Baigiamasis_darbas.pdf"),
			FileSize:         database.NullableInt64(1024000),
		},
		{
			ID:               2,
			DocumentType:     "video",
			OriginalFilename: database.NullableString("Gynyba_video.mp4"),
			FileSize:         database.NullableInt64(50240000),
		},
	}
	return documents, nil
}

func getStringFromNullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func getTopicStatus(status sql.NullString) string {
	if status.Valid {
		return status.String
	}
	return ""
}

// Update getAvailableGroups to be department-specific for department heads
func (h *StudentListHandler) getAvailableGroups(userRole, userEmail string) ([]string, error) {
	var groups []string
	var query string
	var args []interface{}

	switch userRole {
	case auth.RoleSupervisor:
		query = `SELECT DISTINCT student_group FROM student_records WHERE supervisor_email = ? ORDER BY student_group`
		args = []interface{}{userEmail}
	case auth.RoleReviewer:
		query = `SELECT DISTINCT student_group FROM student_records WHERE reviewer_email = ? ORDER BY student_group`
		args = []interface{}{userEmail}
	case auth.RoleDepartmentHead:
		// Get department head's department first
		var department string
		err := h.db.Get(&department, "SELECT department FROM department_heads WHERE email = ? AND is_active = 1", userEmail)
		if err != nil {
			return groups, err
		}
		query = `SELECT DISTINCT student_group FROM student_records WHERE department = ? ORDER BY student_group`
		args = []interface{}{department}
	default:
		query = `SELECT DISTINCT student_group FROM student_records ORDER BY student_group`
		args = []interface{}{}
	}

	err := h.db.Select(&groups, query, args...)
	return groups, err
}

// Update getAvailableStudyPrograms to be department-specific for department heads
func (h *StudentListHandler) getAvailableStudyPrograms(userRole, userEmail string) ([]string, error) {
	var programs []string
	var query string
	var args []interface{}

	switch userRole {
	case auth.RoleSupervisor:
		query = `SELECT DISTINCT study_program FROM student_records WHERE supervisor_email = ? ORDER BY study_program`
		args = []interface{}{userEmail}
	case auth.RoleReviewer:
		query = `SELECT DISTINCT study_program FROM student_records WHERE reviewer_email = ? ORDER BY study_program`
		args = []interface{}{userEmail}
	case auth.RoleDepartmentHead:
		// Get department head's department first
		var department string
		err := h.db.Get(&department, "SELECT department FROM department_heads WHERE email = ? AND is_active = 1", userEmail)
		if err != nil {
			return programs, err
		}
		query = `SELECT DISTINCT study_program FROM student_records WHERE department = ? ORDER BY study_program`
		args = []interface{}{department}
	default:
		query = `SELECT DISTINCT study_program FROM student_records ORDER BY study_program`
		args = []interface{}{}
	}

	err := h.db.Select(&programs, query, args...)
	return programs, err
}

// Update getAvailableYears to be department-specific for department heads
func (h *StudentListHandler) getAvailableYears(userRole, userEmail string) ([]int, error) {
	var years []int
	var query string
	var args []interface{}

	switch userRole {
	case auth.RoleSupervisor:
		query = `SELECT DISTINCT current_year FROM student_records WHERE supervisor_email = ? ORDER BY current_year DESC`
		args = []interface{}{userEmail}
	case auth.RoleReviewer:
		query = `SELECT DISTINCT current_year FROM student_records WHERE reviewer_email = ? ORDER BY current_year DESC`
		args = []interface{}{userEmail}
	case auth.RoleDepartmentHead:
		// Get department head's department first
		var department string
		err := h.db.Get(&department, "SELECT department FROM department_heads WHERE email = ? AND is_active = 1", userEmail)
		if err != nil {
			return years, err
		}
		query = `SELECT DISTINCT current_year FROM student_records WHERE department = ? ORDER BY current_year DESC`
		args = []interface{}{department}
	default:
		query = `SELECT DISTINCT current_year FROM student_records ORDER BY current_year DESC`
		args = []interface{}{}
	}

	err := h.db.Select(&years, query, args...)
	return years, err
}

// Get all filter options for current user
func (h *StudentListHandler) getFilterOptions(userRole, userEmail string) (*database.FilterOptions, error) {
	groups, err := h.getAvailableGroups(userRole, userEmail)
	if err != nil {
		return nil, err
	}

	programs, err := h.getAvailableStudyPrograms(userRole, userEmail)
	if err != nil {
		return nil, err
	}

	years, err := h.getAvailableYears(userRole, userEmail)
	if err != nil {
		return nil, err
	}

	return &database.FilterOptions{
		Groups:        groups,
		StudyPrograms: programs,
		Years:         years,
	}, nil
}

//=====================================
// REVIEWER HANDLER FOR STUDENT LIST
// ==================================

func (h *StudentListHandler) ReviewerStudentsList(w http.ResponseWriter, r *http.Request) {
	// For now, just redirect to login if no token
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// TODO: Implement token validation and create reviewer session
	// For now, just show an error
	http.Error(w, "Reviewer access not yet implemented", http.StatusNotImplemented)
}
func (h *StudentListHandler) validateReviewerToken(token string) (string, error) {
	// Implement your token validation logic here
	// This could be JWT, database lookup, etc.
	// Return the reviewer email if valid
	return "", fmt.Errorf("not implemented")
}

// ReviewerReportSubmitHandler handles the submission of reviewer reports
func (h *StudentListHandler) ReviewerReportSubmitHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
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

	// Get student record
	var student database.StudentRecord
	err = h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if user can submit review
	if user.Role != auth.RoleReviewer || student.ReviewerEmail != user.Email {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse form data
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	isDraft := r.FormValue("is_draft") == "true"

	if isDraft {
		// Redirect to draft handler
		h.ReviewerReportSaveDraftHandler(w, r)
		return
	}

	// Extract grade
	grade, err := strconv.ParseFloat(r.FormValue("grade"), 64)
	if err != nil || grade < 1 || grade > 10 {
		http.Error(w, "Invalid grade", http.StatusBadRequest)
		return
	}

	// Check if report already exists
	var existingReport database.ReviewerReport
	err = h.db.Get(&existingReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?", studentID)

	tx, err := h.db.Beginx()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if err == sql.ErrNoRows {
		// Create new report
		_, err = tx.Exec(`
            INSERT INTO reviewer_reports (
                student_record_id,
                reviewer_personal_details,
                grade,
                review_goals,
                review_theory,
                review_practical,
                review_theory_practical_link,
                review_results,
                review_practical_significance,
                review_language,
                review_pros,
                review_cons,
                review_questions,
                is_signed,
                created_date
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
			studentID,
			r.FormValue("reviewer_personal_details"),
			grade,
			r.FormValue("review_goals"),
			r.FormValue("review_theory"),
			r.FormValue("review_practical"),
			r.FormValue("review_theory_practical_link"),
			r.FormValue("review_results"),
			r.FormValue("review_practical_significance"),
			r.FormValue("review_language"),
			r.FormValue("review_pros"),
			r.FormValue("review_cons"),
			r.FormValue("review_questions"),
			true, // Auto-sign on submission
		)
	} else if err == nil && !existingReport.IsSigned {
		// Update existing unsigned report
		_, err = tx.Exec(`
            UPDATE reviewer_reports SET
                reviewer_personal_details = ?,
                grade = ?,
                review_goals = ?,
                review_theory = ?,
                review_practical = ?,
                review_theory_practical_link = ?,
                review_results = ?,
                review_practical_significance = ?,
                review_language = ?,
                review_pros = ?,
                review_cons = ?,
                review_questions = ?,
                is_signed = ?,
                updated_date = NOW()
            WHERE id = ?`,
			r.FormValue("reviewer_personal_details"),
			grade,
			r.FormValue("review_goals"),
			r.FormValue("review_theory"),
			r.FormValue("review_practical"),
			r.FormValue("review_theory_practical_link"),
			r.FormValue("review_results"),
			r.FormValue("review_practical_significance"),
			r.FormValue("review_language"),
			r.FormValue("review_pros"),
			r.FormValue("review_cons"),
			r.FormValue("review_questions"),
			true,
			existingReport.ID,
		)
	} else {
		tx.Rollback()
		http.Error(w, "Report already signed", http.StatusBadRequest)
		return
	}

	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to save report", http.StatusInternalServerError)
		return
	}

	// Create audit log
	auditLog := database.AuditLog{
		UserEmail:    user.Email,
		UserRole:     user.Role,
		Action:       "submit_reviewer_report",
		ResourceType: "reviewer_reports",
		ResourceID:   &studentIDStr,
		Details:      &[]string{fmt.Sprintf("Submitted review for student %s", student.GetFullName())}[0],
		Success:      true,
	}
	database.CreateAuditLog(auditLog)

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to save report", http.StatusInternalServerError)
		return
	}

	// Return success message
	w.Header().Set("HX-Trigger", "reviewerReportSaved")
	w.Write([]byte(`
        <div class="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded">
            <div class="flex items-center">
                <svg class="h-5 w-5 text-green-400 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
                <span>Recenzija sėkmingai išsaugota ir pasirašyta!</span>
            </div>
        </div>
    `))
}
func getStringValue(ns *string) string {
	if ns != nil {
		return *ns
	}
	return ""
}

// Add this handler for saving drafts
func (h *StudentListHandler) ReviewerReportSaveDraftHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
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

	// Get student record
	var student database.StudentRecord
	err = h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if user can save draft
	if user.Role != auth.RoleReviewer || student.ReviewerEmail != user.Email {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Parse form data
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Extract grade (allow empty for draft)
	var grade float64
	if gradeStr := r.FormValue("grade"); gradeStr != "" {
		grade, _ = strconv.ParseFloat(gradeStr, 64)
	}

	// Check if report already exists
	var existingReport database.ReviewerReport
	err = h.db.Get(&existingReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?", studentID)

	tx, err := h.db.Beginx()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if err == sql.ErrNoRows {
		// Create new draft report
		_, err = tx.Exec(`
            INSERT INTO reviewer_reports (
                student_record_id,
                reviewer_personal_details,
                grade,
                review_goals,
                review_theory,
                review_practical,
                review_theory_practical_link,
                review_results,
                review_practical_significance,
                review_language,
                review_pros,
                review_cons,
                review_questions,
                is_signed,
                created_date
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
			studentID,
			r.FormValue("reviewer_personal_details"),
			grade,
			r.FormValue("review_goals"),
			r.FormValue("review_theory"),
			r.FormValue("review_practical"),
			r.FormValue("review_theory_practical_link"),
			r.FormValue("review_results"),
			r.FormValue("review_practical_significance"),
			r.FormValue("review_language"),
			r.FormValue("review_pros"),
			r.FormValue("review_cons"),
			r.FormValue("review_questions"),
			false, // Not signed for draft
		)
	} else if err == nil && !existingReport.IsSigned {
		// Update existing unsigned report
		_, err = tx.Exec(`
            UPDATE reviewer_reports SET
                reviewer_personal_details = ?,
                grade = ?,
                review_goals = ?,
                review_theory = ?,
                review_practical = ?,
                review_theory_practical_link = ?,
                review_results = ?,
                review_practical_significance = ?,
                review_language = ?,
                review_pros = ?,
                review_cons = ?,
                review_questions = ?,
                updated_date = NOW()
            WHERE id = ?`,
			r.FormValue("reviewer_personal_details"),
			grade,
			r.FormValue("review_goals"),
			r.FormValue("review_theory"),
			r.FormValue("review_practical"),
			r.FormValue("review_theory_practical_link"),
			r.FormValue("review_results"),
			r.FormValue("review_practical_significance"),
			r.FormValue("review_language"),
			r.FormValue("review_pros"),
			r.FormValue("review_cons"),
			r.FormValue("review_questions"),
			existingReport.ID,
		)
	} else {
		tx.Rollback()
		http.Error(w, "Report already signed", http.StatusBadRequest)
		return
	}

	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to save draft", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to save draft", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Write([]byte(`<div class="text-xs text-green-600">Draft saved</div>`))
}

// ========================================================
// REVIEWER TOKEN-BASED ACCESS METHODS
// ========================================================

// ShowReviewerStudentsWithToken - Show students for reviewer using access token
func (h *StudentListHandler) ShowReviewerStudentsWithToken(w http.ResponseWriter, r *http.Request) {
	accessToken := chi.URLParam(r, "accessToken")

	// Validate reviewer access
	reviewerToken, err := h.validateReviewerAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Update access count
	h.updateReviewerAccessCount(reviewerToken.ID)

	// Get students assigned to this reviewer
	students, err := h.getReviewerStudents(reviewerToken.ReviewerEmail)
	if err != nil {
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	// Get pagination info
	pagination := &database.PaginationInfo{
		Page:       1,
		Limit:      50,
		Total:      len(students),
		TotalPages: 1,
		HasPrev:    false,
		HasNext:    false,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = templates.ReviewerStudentList(accessToken, students, reviewerToken.ReviewerName, pagination).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// ReviewerReportModalHandlerWithToken - Show reviewer report form using token
func (h *StudentListHandler) ReviewerReportModalHandlerWithToken(w http.ResponseWriter, r *http.Request) {
	accessToken := chi.URLParam(r, "accessToken")
	studentIDStr := chi.URLParam(r, "studentId")

	// Validate reviewer access
	reviewerToken, err := h.validateReviewerAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Update access count
	h.updateReviewerAccessCount(reviewerToken.ID)

	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Get student record
	var student database.StudentRecord
	err = h.db.Get(&student, "SELECT * FROM student_records WHERE id = ? AND reviewer_email = ?",
		studentID, reviewerToken.ReviewerEmail)
	if err != nil {
		http.Error(w, "Student not found or access denied", http.StatusNotFound)
		return
	}

	// Check if report exists
	var existingReport database.ReviewerReport
	err = h.db.Get(&existingReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?", studentID)

	formData := &database.ReviewerReportFormData{}
	isReadOnly := false

	if err == nil {
		// Report exists
		formData = &database.ReviewerReportFormData{
			ReviewerPersonalDetails:     existingReport.ReviewerPersonalDetails,
			Grade:                       float64(existingReport.Grade),
			ReviewGoals:                 existingReport.ReviewGoals,
			ReviewTheory:                existingReport.ReviewTheory,
			ReviewPractical:             existingReport.ReviewPractical,
			ReviewTheoryPracticalLink:   existingReport.ReviewTheoryPracticalLink,
			ReviewResults:               existingReport.ReviewResults,
			ReviewPracticalSignificance: getStringValue(existingReport.ReviewPracticalSignificance),
			ReviewLanguage:              existingReport.ReviewLanguage,
			ReviewPros:                  existingReport.ReviewPros,
			ReviewCons:                  existingReport.ReviewCons,
			ReviewQuestions:             existingReport.ReviewQuestions,
		}

		// If report is signed, force read-only mode
		if existingReport.IsSigned {
			isReadOnly = true
		}
	}

	// Determine form variant
	formVariant := "lt"
	if r.URL.Query().Get("lang") == "en" {
		formVariant = "en"
	}

	// Pass access token to form props
	props := database.ReviewerReportFormProps{
		StudentRecord: &student,
		IsReadOnly:    isReadOnly,
		FormVariant:   formVariant,
		ReviewerName:  student.ReviewerName,
		AccessToken:   accessToken, // Add this line
	}

	err = templates.CompactReviewerForm(props, formData).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

// ReviewerReportSubmitHandlerWithToken - Handle report submission using token
func (h *StudentListHandler) ReviewerReportSubmitHandlerWithToken(w http.ResponseWriter, r *http.Request) {
	accessToken := chi.URLParam(r, "accessToken")
	studentIDStr := chi.URLParam(r, "studentId")

	// Validate reviewer access
	reviewerToken, err := h.validateReviewerAccessToken(accessToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	studentID, err := strconv.Atoi(studentIDStr)
	if err != nil {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}

	// Verify student is assigned to this reviewer
	var student database.StudentRecord
	err = h.db.Get(&student, "SELECT * FROM student_records WHERE id = ? AND reviewer_email = ?",
		studentID, reviewerToken.ReviewerEmail)
	if err != nil {
		http.Error(w, "Student not found or access denied", http.StatusNotFound)
		return
	}

	// Parse form data
	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	isDraft := r.FormValue("is_draft") == "true"

	// Extract grade
	grade, err := strconv.ParseFloat(r.FormValue("grade"), 64)
	if err != nil && !isDraft {
		http.Error(w, "Invalid grade", http.StatusBadRequest)
		return
	}

	// Check if report already exists
	var existingReport database.ReviewerReport
	err = h.db.Get(&existingReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?", studentID)

	tx, err := h.db.Beginx()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if err == sql.ErrNoRows {
		// Create new report
		_, err = tx.Exec(`
            INSERT INTO reviewer_reports (
                student_record_id,
                reviewer_personal_details,
                grade,
                review_goals,
                review_theory,
                review_practical,
                review_theory_practical_link,
                review_results,
                review_practical_significance,
                review_language,
                review_pros,
                review_cons,
                review_questions,
                is_signed,
                created_date
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())`,
			studentID,
			r.FormValue("reviewer_personal_details"),
			grade,
			r.FormValue("review_goals"),
			r.FormValue("review_theory"),
			r.FormValue("review_practical"),
			r.FormValue("review_theory_practical_link"),
			r.FormValue("review_results"),
			r.FormValue("review_practical_significance"),
			r.FormValue("review_language"),
			r.FormValue("review_pros"),
			r.FormValue("review_cons"),
			r.FormValue("review_questions"),
			!isDraft, // Sign if not draft
		)
	} else if err == nil && !existingReport.IsSigned {
		// Update existing unsigned report
		_, err = tx.Exec(`
            UPDATE reviewer_reports SET
                reviewer_personal_details = ?,
                grade = ?,
                review_goals = ?,
                review_theory = ?,
                review_practical = ?,
                review_theory_practical_link = ?,
                review_results = ?,
                review_practical_significance = ?,
                review_language = ?,
                review_pros = ?,
                review_cons = ?,
                review_questions = ?,
                is_signed = ?,
                updated_date = NOW()
            WHERE id = ?`,
			r.FormValue("reviewer_personal_details"),
			grade,
			r.FormValue("review_goals"),
			r.FormValue("review_theory"),
			r.FormValue("review_practical"),
			r.FormValue("review_theory_practical_link"),
			r.FormValue("review_results"),
			r.FormValue("review_practical_significance"),
			r.FormValue("review_language"),
			r.FormValue("review_pros"),
			r.FormValue("review_cons"),
			r.FormValue("review_questions"),
			!isDraft,
			existingReport.ID,
		)
	} else {
		tx.Rollback()
		http.Error(w, "Report already signed", http.StatusBadRequest)
		return
	}

	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to save report", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Failed to save report", http.StatusInternalServerError)
		return
	}

	if isDraft {
		w.Write([]byte(`<div class="text-xs text-green-600">Draft saved</div>`))
	} else {
		// Return success message for final submission
		w.Header().Set("HX-Trigger", "reviewerReportSaved")
		w.Write([]byte(`
            <div class="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded">
                <div class="flex items-center">
                    <svg class="h-5 w-5 text-green-400 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                    </svg>
                    <span>Recenzija sėkmingai išsaugota ir pasirašyta!</span>
                </div>
            </div>
        `))
	}
}

// Database helper methods for reviewer access tokens
func (h *StudentListHandler) validateReviewerAccessToken(accessToken string) (*database.ReviewerAccessToken, error) {
	var token database.ReviewerAccessToken
	query := `SELECT * FROM reviewer_access_tokens WHERE access_token = ? AND is_active = true`

	err := h.db.Get(&token, query, accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token")
	}

	if !token.CanAccess() {
		return nil, fmt.Errorf("access token expired or limit reached")
	}

	return &token, nil
}

func (h *StudentListHandler) updateReviewerAccessCount(tokenID int) {
	query := `UPDATE reviewer_access_tokens SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?`
	h.db.Exec(query, time.Now().Unix(), tokenID)
}

func (h *StudentListHandler) getReviewerStudents(reviewerEmail string) ([]database.StudentSummaryView, error) {
	query := `
        SELECT 
            sr.id,
            sr.student_group,
            sr.student_name,
            sr.student_lastname,
            sr.student_email,
            sr.final_project_title,
            sr.supervisor_email,
            sr.reviewer_email,
            sr.reviewer_name,
            COALESCE(ptr.status, '') as topic_status,
            CASE WHEN ptr.status = 'approved' THEN 1 ELSE 0 END as topic_approved,
            EXISTS(SELECT 1 FROM documents d WHERE d.student_record_id = sr.id AND d.document_type LIKE '%source%') as has_source_code,
            COALESCE(rr.id IS NOT NULL, false) as has_reviewer_report,
            COALESCE(rr.is_signed, false) as reviewer_report_signed,
            rr.grade as reviewer_grade,
            rr.review_questions as reviewer_questions
        FROM student_records sr
        LEFT JOIN project_topic_registrations ptr ON sr.id = ptr.student_record_id AND ptr.status = 'approved'
        LEFT JOIN reviewer_reports rr ON sr.id = rr.student_record_id
        WHERE sr.reviewer_email = ?
        ORDER BY sr.student_name, sr.student_lastname
    `

	var students []database.StudentSummaryView
	err := h.db.Select(&students, query, reviewerEmail)
	return students, err
}

// Update the existing ReviewerReportModalHandler to pass empty access token
func (h *StudentListHandler) ReviewerReportModalHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserContextKey).(*auth.AuthenticatedUser)
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

	// Get student record
	var student database.StudentRecord
	err = h.db.Get(&student, "SELECT * FROM student_records WHERE id = ?", studentID)
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if user can view/edit this review
	mode := r.URL.Query().Get("mode")
	isReadOnly := mode == "view"

	// For reviewers, check if they are assigned to this student
	if user.Role == auth.RoleReviewer && student.ReviewerEmail != user.Email {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if report exists
	var existingReport database.ReviewerReport
	err = h.db.Get(&existingReport,
		"SELECT * FROM reviewer_reports WHERE student_record_id = ?", studentID)

	formData := &database.ReviewerReportFormData{}
	if err == nil {
		// Report exists
		formData = &database.ReviewerReportFormData{
			ReviewerPersonalDetails:     existingReport.ReviewerPersonalDetails,
			Grade:                       float64(existingReport.Grade),
			ReviewGoals:                 existingReport.ReviewGoals,
			ReviewTheory:                existingReport.ReviewTheory,
			ReviewPractical:             existingReport.ReviewPractical,
			ReviewTheoryPracticalLink:   existingReport.ReviewTheoryPracticalLink,
			ReviewResults:               existingReport.ReviewResults,
			ReviewPracticalSignificance: getStringValue(existingReport.ReviewPracticalSignificance),
			ReviewLanguage:              existingReport.ReviewLanguage,
			ReviewPros:                  existingReport.ReviewPros,
			ReviewCons:                  existingReport.ReviewCons,
			ReviewQuestions:             existingReport.ReviewQuestions,
		}

		// If report is signed, force read-only mode
		if existingReport.IsSigned {
			isReadOnly = true
		}
	} else if err == sql.ErrNoRows {
		// No report exists - check if user can create one
		if user.Role != auth.RoleReviewer || student.ReviewerEmail != user.Email {
			isReadOnly = true
		}
	} else {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Determine form variant
	formVariant := "lt"
	if r.URL.Query().Get("lang") == "en" {
		formVariant = "en"
	}

	props := database.ReviewerReportFormProps{
		StudentRecord: &student,
		IsReadOnly:    isReadOnly,
		FormVariant:   formVariant,
		ReviewerName:  student.ReviewerName,
		AccessToken:   "", // Empty for authenticated users
	}

	err = templates.CompactReviewerForm(props, formData).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}
