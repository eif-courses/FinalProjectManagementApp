// supervisor_report_types.go
package handlers

import (
	"FinalProjectManagementApp/database" // Replace with your actual module path
	"time"
)

// SupervisorReportFormProps represents the props for the supervisor report form
type SupervisorReportFormProps struct {
	// Student and context data
	StudentRecord database.StudentRecord     `json:"student_record"`
	InitialReport *database.SupervisorReport `json:"initial_report,omitempty"`

	// Form configuration
	ButtonLabel string `json:"button_label"`
	ModalTitle  string `json:"modal_title"`
	FormVariant string `json:"form_variant"` // "lt" or "en"

	// Form state
	IsModalOpen bool `json:"is_modal_open"`
	IsSaving    bool `json:"is_saving"`

	// Validation errors
	ValidationErrors map[string]string `json:"validation_errors,omitempty"`

	// Current user info (supervisor)
	CurrentSupervisorName  string `json:"current_supervisor_name"`
	CurrentSupervisorEmail string `json:"current_supervisor_email"`
}

// SupervisorReportFormData represents the data being edited in the form
type SupervisorReportFormData struct {
	SupervisorComments  string    `json:"supervisor_comments" form:"supervisor_comments"`
	SupervisorWorkplace string    `json:"supervisor_workplace" form:"supervisor_workplace"`
	SupervisorPosition  string    `json:"supervisor_position" form:"supervisor_position"`
	OtherMatch          float64   `json:"other_match" form:"other_match"` // Maps to database.other_match
	OneMatch            float64   `json:"one_match" form:"one_match"`     // Maps to database.one_match
	OwnMatch            float64   `json:"own_match" form:"own_match"`     // Maps to database.own_match
	JoinMatch           float64   `json:"join_match" form:"join_match"`   // Maps to database.join_match
	IsPassOrFailed      bool      `json:"is_pass_or_failed" form:"is_pass_or_failed"`
	Grade               *int      `json:"grade" form:"grade"` // Optional grade 1-10
	FinalComments       string    `json:"final_comments" form:"final_comments"`
	SubmissionDate      time.Time `json:"submission_date"`
}

// ToSupervisorReportData converts form data to database.SupervisorReportData
func (f *SupervisorReportFormData) ToSupervisorReportData(studentRecordID int, supervisorName string) *database.SupervisorReportData {
	return &database.SupervisorReportData{
		StudentRecordID:     studentRecordID,
		SupervisorComments:  f.SupervisorComments,
		SupervisorName:      supervisorName,
		SupervisorPosition:  f.SupervisorPosition,
		SupervisorWorkplace: f.SupervisorWorkplace,
		IsPassOrFailed:      f.IsPassOrFailed,
		OtherMatch:          f.OtherMatch,
		OneMatch:            f.OneMatch,
		OwnMatch:            f.OwnMatch,
		JoinMatch:           f.JoinMatch,
		Grade:               f.Grade,
		FinalComments:       f.FinalComments,
	}
}

// ToSupervisorReport converts form data to database.SupervisorReport model
func (f *SupervisorReportFormData) ToSupervisorReport(studentRecordID int, supervisorName string) *database.SupervisorReport {
	return &database.SupervisorReport{
		StudentRecordID:     studentRecordID,
		SupervisorComments:  f.SupervisorComments,
		SupervisorName:      supervisorName,
		SupervisorPosition:  f.SupervisorPosition,
		SupervisorWorkplace: f.SupervisorWorkplace,
		IsPassOrFailed:      f.IsPassOrFailed,
		IsSigned:            false, // Will be signed separately
		OtherMatch:          f.OtherMatch,
		OneMatch:            f.OneMatch,
		OwnMatch:            f.OwnMatch,
		JoinMatch:           f.JoinMatch,
		Grade:               f.Grade,
		FinalComments:       f.FinalComments,
		CreatedDate:         time.Now(),
		UpdatedDate:         time.Now(),
	}
}

// NewSupervisorReportFormData creates form data from existing report or defaults
func NewSupervisorReportFormData(report *database.SupervisorReport) *SupervisorReportFormData {
	if report == nil {
		return &SupervisorReportFormData{
			IsPassOrFailed: true, // Default to pass
			SubmissionDate: time.Now(),
			OtherMatch:     0.0,
			OneMatch:       0.0,
			OwnMatch:       0.0,
			JoinMatch:      0.0,
		}
	}

	return &SupervisorReportFormData{
		SupervisorComments:  report.SupervisorComments,
		SupervisorWorkplace: report.SupervisorWorkplace,
		SupervisorPosition:  report.SupervisorPosition,
		OtherMatch:          report.OtherMatch,
		OneMatch:            report.OneMatch,
		OwnMatch:            report.OwnMatch,
		JoinMatch:           report.JoinMatch,
		IsPassOrFailed:      report.IsPassOrFailed,
		Grade:               report.Grade,
		FinalComments:       report.FinalComments,
		SubmissionDate:      time.Now(),
	}
}

// GetTotalSimilarity calculates total similarity percentage
func (f *SupervisorReportFormData) GetTotalSimilarity() float64 {
	return f.OtherMatch + f.OneMatch + f.OwnMatch + f.JoinMatch
}

// GetSimilarityStatus returns similarity assessment
func (f *SupervisorReportFormData) GetSimilarityStatus() string {
	total := f.GetTotalSimilarity()
	if total <= 15 {
		return "Low"
	} else if total <= 25 {
		return "Moderate"
	} else {
		return "High"
	}
}

// GetSimilarityColor returns CSS color class for similarity level
func (f *SupervisorReportFormData) GetSimilarityColor() string {
	total := f.GetTotalSimilarity()
	if total <= 15 {
		return "text-green-600"
	} else if total <= 25 {
		return "text-yellow-600"
	} else {
		return "text-red-600"
	}
}
