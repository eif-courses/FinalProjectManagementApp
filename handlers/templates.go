// handlers/templates.go
package handlers

import (
	"FinalProjectManagementApp/i18n"
	"html/template"
	"log"
	"net/http"
)

type User struct {
	Name       string
	Email      string
	Role       string
	Department string
}

func NewTemplateHandlerWithI18n(localizer *i18n.Localizer) *template.Template {
	// Create template without complex functions - we'll pass translations via data
	tmpl := template.New("")

	// Parse templates
	tmpl, err := tmpl.ParseGlob("templates/**/*.html")
	if err != nil {
		log.Fatal("Error parsing templates:", err)
	}

	// Debug: print template names
	log.Println("Loaded templates with i18n support:")
	for _, t := range tmpl.Templates() {
		log.Println(" -", t.Name())
	}

	return tmpl
}

func RenderTemplateWithI18n(w http.ResponseWriter, tmpl *template.Template, name string, data i18n.LocalizedTemplateData) {
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
