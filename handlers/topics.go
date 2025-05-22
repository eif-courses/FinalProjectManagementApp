// handlers/topics.go
package handlers

import (
	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/i18n"
	"html/template"
	"net/http"
)

// SubmitTopicHandler handles topic submission form
func SubmitTopicHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		lang := i18n.GetLangFromContext(r.Context())

		if r.Method == "POST" {
			// Handle topic submission
			title := r.FormValue("title")
			titleEn := r.FormValue("title_en")
			problem := r.FormValue("problem")
			objective := r.FormValue("objective")
			tasks := r.FormValue("tasks")
			supervisor := r.FormValue("supervisor")

			// Validate required fields
			if title == "" || problem == "" || objective == "" {
				http.Error(w, localizer.T(lang, "missing_required_fields"), http.StatusBadRequest)
				return
			}

			// In real implementation, save to database
			// topicRegistration := &database.ProjectTopicRegistration{
			//     StudentRecordID: getStudentRecordID(user.Email),
			//     Title:          title,
			//     TitleEn:        titleEn,
			//     Problem:        problem,
			//     Objective:      objective,
			//     Tasks:          tasks,
			//     Supervisor:     supervisor,
			//     Status:         "submitted",
			// }
			// err := db.CreateTopicRegistration(topicRegistration)

			// Log or use the variables for now (remove when implementing database)
			_ = titleEn    // Will be used when database is implemented
			_ = tasks      // Will be used when database is implemented
			_ = supervisor // Will be used when database is implemented

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "success", "message": "` + localizer.T(lang, "topic_submitted") + `"}`))
			return
		}

		// Show topic submission form
		data := localizer.NewTemplateData(
			r.Context(),
			"submit_topic",
			user,
			map[string]interface{}{
				"Supervisors": getAvailableSupervisors(),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "submit-topic.html", data)
	}
}

// ViewTopicHandler shows topic details
func ViewTopicHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		topicID := r.URL.Query().Get("id")

		data := localizer.NewTemplateData(
			r.Context(),
			"view_topic",
			user,
			map[string]interface{}{
				"Topic":    getTopicByID(topicID),
				"Comments": getTopicComments(topicID),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "view-topic.html", data)
	}
}

// EditTopicHandler allows editing topic (students only, before approval)
func EditTopicHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		lang := i18n.GetLangFromContext(r.Context())
		topicID := r.URL.Query().Get("id")

		// Get topic and check ownership
		topic := getTopicByID(topicID)
		if !userOwnsTopic(user, topic) {
			http.Error(w, localizer.T(lang, "access_denied"), http.StatusForbidden)
			return
		}

		// Check if topic can be edited (not approved yet)
		if topic["Status"] == "approved" {
			http.Error(w, localizer.T(lang, "cannot_edit_approved_topic"), http.StatusForbidden)
			return
		}

		if r.Method == "POST" {
			// Handle topic update
			title := r.FormValue("title")
			titleEn := r.FormValue("title_en")
			problem := r.FormValue("problem")
			objective := r.FormValue("objective")
			tasks := r.FormValue("tasks")

			// In real implementation, update database
			// err := db.UpdateTopicRegistration(topicID, updatedData)

			// Use variables to avoid unused variable errors (remove when implementing database)
			_ = title
			_ = titleEn
			_ = problem
			_ = objective
			_ = tasks

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success", "message": "` + localizer.T(lang, "topic_updated") + `"}`))
			return
		}

		// Show edit form
		data := localizer.NewTemplateData(
			r.Context(),
			"edit_topic",
			user,
			map[string]interface{}{
				"Topic":       topic,
				"Supervisors": getAvailableSupervisors(),
			},
		)

		RenderTemplateWithI18n(w, tmpl, "edit-topic.html", data)
	}
}

// TopicCommentsHandler handles adding comments to topics
func TopicCommentsHandler(tmpl *template.Template, localizer *i18n.Localizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUserFromContext(r.Context())
		lang := i18n.GetLangFromContext(r.Context())

		if r.Method == "POST" {
			topicID := r.FormValue("topic_id")
			commentText := r.FormValue("comment")
			//fieldName := r.FormValue("field_name") // Optional: specific field comment

			if commentText == "" {
				http.Error(w, localizer.T(lang, "comment_required"), http.StatusBadRequest)
				return
			}

			// In real implementation, save comment to database
			// comment := &database.TopicRegistrationComment{
			//     TopicRegistrationID: topicID,
			//     FieldName:          fieldName,
			//     CommentText:        commentText,
			//     AuthorRole:         user.Role,
			//     AuthorName:         user.Name,
			// }
			// err := db.CreateTopicComment(comment)

			// Use variables to avoid unused variable errors (remove when implementing database)
			_ = user
			_ = topicID

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status": "success", "message": "` + localizer.T(lang, "comment_added") + `"}`))
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper functions for topic operations
func getAvailableSupervisors() []map[string]interface{} {
	// In real implementation, query database for supervisors
	return []map[string]interface{}{
		{
			"ID":             1,
			"Name":           "Dr. Jonas Petraitis",
			"Email":          "j.petraitis@viko.lt",
			"Department":     "Information Systems",
			"Specialization": "Software Engineering, Databases",
		},
		{
			"ID":             2,
			"Name":           "Prof. Rima Kazlauskienė",
			"Email":          "r.kazlauskiene@viko.lt",
			"Department":     "Information Systems",
			"Specialization": "Computer Networks, Security",
		},
		{
			"ID":             3,
			"Name":           "Dr. Aurimas Mockus",
			"Email":          "a.mockus@viko.lt",
			"Department":     "Information Systems",
			"Specialization": "AI, Machine Learning",
		},
	}
}

func getTopicByID(topicID string) map[string]interface{} {
	// In real implementation, query database
	return map[string]interface{}{
		"ID":             topicID,
		"Title":          "Thesis Management System",
		"TitleEn":        "Thesis Management System",
		"Problem":        "Šiuo metu trūksta centralizuotos sistemos baigiamųjų darbų valdymui universitete...",
		"Objective":      "Sukurti web aplikaciją baigiamųjų darbų valdymui...",
		"Tasks":          "1. Sistemos analizė\n2. Duomenų bazės projektavimas\n3. Sistemos realizacija",
		"Supervisor":     "Dr. Jonas Petraitis",
		"Status":         "submitted",
		"StudentName":    "Mantas Gzegožveskis",
		"StudentEmail":   "m.gzegozevskis@eif.viko.lt",
		"SubmittedAt":    "2024-01-15",
		"CurrentVersion": 1,
	}
}

func getTopicComments(topicID string) []map[string]interface{} {
	// In real implementation, query database
	return []map[string]interface{}{
		{
			"ID":          1,
			"CommentText": "Tema atrodo įdomi, tačiau reikėtų detalizuoti tikslus.",
			"AuthorName":  "Dr. Jonas Petraitis",
			"AuthorRole":  "supervisor",
			"FieldName":   "objective",
			"CreatedAt":   "2024-01-16",
			"IsRead":      0,
		},
		{
			"ID":          2,
			"CommentText": "Pataisyta pagal pateiktas pastabas.",
			"AuthorName":  "Mantas Gzegožveskis",
			"AuthorRole":  "student",
			"FieldName":   "objective",
			"CreatedAt":   "2024-01-17",
			"IsRead":      1,
		},
	}
}

func userOwnsTopic(user *auth.AuthenticatedUser, topic map[string]interface{}) bool {
	// In real implementation, check database
	studentEmail, ok := topic["StudentEmail"].(string)
	if !ok {
		return false
	}
	return user.Email == studentEmail || user.HasPermission("view_all_students")
}
