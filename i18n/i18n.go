// i18n/i18n.go - Complete i18n package (single file)
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

type Localizer struct {
	translations map[string]map[string]string
	fallback     string
}

type contextKey string

const (
	LangContextKey contextKey = "language"
	DefaultLang               = "lt"
	FallbackLang              = "en"
)

// NewLocalizer creates a new localizer instance
func NewLocalizer() *Localizer {
	return &Localizer{
		translations: make(map[string]map[string]string),
		fallback:     FallbackLang,
	}
}

// LoadTranslations loads translation files from directory with support for nested JSON
func (l *Localizer) LoadTranslations(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to read translation files: %w", err)
	}

	for _, file := range files {
		// Extract language code from filename (e.g., "lt.json" -> "lt")
		lang := strings.TrimSuffix(filepath.Base(file), ".json")

		data, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Parse as nested JSON (your current format)
		var nestedTranslations map[string]interface{}
		if err := json.Unmarshal(data, &nestedTranslations); err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}

		// Flatten the nested structure
		flatTranslations := make(map[string]string)
		l.flattenTranslations("", nestedTranslations, flatTranslations)

		l.translations[lang] = flatTranslations
		fmt.Printf("✅ Loaded %d translations for language: %s\n", len(flatTranslations), lang)
	}

	return nil
}

// flattenTranslations converts nested JSON to flat key-value pairs
func (l *Localizer) flattenTranslations(prefix string, nested map[string]interface{}, flat map[string]string) {
	for key, value := range nested {
		// Skip comment fields
		if key == "_comment" {
			continue
		}

		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case string:
			flat[fullKey] = v
		case map[string]interface{}:
			l.flattenTranslations(fullKey, v, flat)
		default:
			// Skip non-string, non-object values
			continue
		}
	}
}

// T translates a key for the given language
func (l *Localizer) T(lang, key string, args ...interface{}) string {
	// Try requested language
	if translations, exists := l.translations[lang]; exists {
		if value, found := translations[key]; found {
			if len(args) > 0 {
				return fmt.Sprintf(value, args...)
			}
			return value
		}
	}

	// Fallback to default language
	if lang != l.fallback {
		if translations, exists := l.translations[l.fallback]; exists {
			if value, found := translations[key]; found {
				if len(args) > 0 {
					return fmt.Sprintf(value, args...)
				}
				return value
			}
		}
	}

	// Return key if no translation found
	return key
}

// GetSupportedLanguages returns list of supported languages
func (l *Localizer) GetSupportedLanguages() []string {
	var langs []string
	for lang := range l.translations {
		langs = append(langs, lang)
	}
	return langs
}

// detectLanguage detects user's preferred language
func (l *Localizer) detectLanguage(r *http.Request) string {
	// 1. Check URL parameter
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if l.isSupported(lang) {
			return lang
		}
	}

	// 2. Check cookie
	if cookie, err := r.Cookie("language"); err == nil {
		if l.isSupported(cookie.Value) {
			return cookie.Value
		}
	}

	// 3. Check Accept-Language header
	if lang := l.parseAcceptLanguage(r.Header.Get("Accept-Language")); lang != "" {
		if l.isSupported(lang) {
			return lang
		}
	}

	// 4. Return default
	return DefaultLang
}

// parseAcceptLanguage parses Accept-Language header
func (l *Localizer) parseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}

	// Simple parsing - take first language
	parts := strings.Split(header, ",")
	if len(parts) > 0 {
		lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])
		// Handle cases like "en-US" -> "en"
		if strings.Contains(lang, "-") {
			lang = strings.Split(lang, "-")[0]
		}
		return lang
	}

	return ""
}

// isSupported checks if language is supported
func (l *Localizer) isSupported(lang string) bool {
	_, exists := l.translations[lang]
	return exists
}

// GetLangFromContext gets language from request context
func GetLangFromContext(ctx context.Context) string {
	if lang, ok := ctx.Value(LangContextKey).(string); ok {
		return lang
	}
	return DefaultLang
}

// LanguageMiddleware for language detection and context setting
func (l *Localizer) LanguageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := l.detectLanguage(r)

		// Set language header for client-side detection
		w.Header().Set("Content-Language", lang)

		// Add language to context
		ctx := context.WithValue(r.Context(), LangContextKey, lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LanguageSwitchHandler handles language switching
func (l *Localizer) LanguageSwitchHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if lang == "" || !l.isSupported(lang) {
		http.Error(w, "Invalid language", http.StatusBadRequest)
		return
	}

	// Set language cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "language",
		Value:    lang,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: false,      // Allow JS access for dynamic switching
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect back to referer or home
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}

	http.Redirect(w, r, referer, http.StatusFound)
}

// Extended TemplateData with language support
type LocalizedTemplateData struct {
	Title string
	User  interface{}
	Data  interface{}
	Lang  string
	T     func(string, ...interface{}) string
}

// NewTemplateData creates localized template data
func (l *Localizer) NewTemplateData(ctx context.Context, title string, user interface{}, data interface{}) LocalizedTemplateData {
	lang := GetLangFromContext(ctx)
	return LocalizedTemplateData{
		Title: l.T(lang, title),
		User:  user,
		Data:  data,
		Lang:  lang,
		T: func(key string, args ...interface{}) string {
			return l.T(lang, key, args...)
		},
	}
}

// RequestLocalizationMiddleware combines language detection with content type setting
func (l *Localizer) RequestLocalizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set UTF-8 charset for HTML responses
		if strings.HasSuffix(r.URL.Path, ".html") || strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		// Apply language middleware
		l.LanguageMiddleware(next).ServeHTTP(w, r)
	})
}
