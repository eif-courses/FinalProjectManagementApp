// i18n/translator.go
package i18n

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"strings"
	"sync"
	"text/template"
)

type Translator struct {
	translations map[string]map[string]interface{}
	mutex        sync.RWMutex
}

var globalTranslator *Translator
var once sync.Once

func GetTranslator() *Translator {
	once.Do(func() {
		globalTranslator = &Translator{
			translations: make(map[string]map[string]interface{}),
		}
		globalTranslator.LoadTranslations()
	})
	return globalTranslator
}

func (t *Translator) LoadTranslations() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	locales := []string{"en", "lt"}

	for _, locale := range locales {
		var config map[string]interface{}
		_, err := toml.DecodeFile(fmt.Sprintf("locales/%s.toml", locale), &config)
		if err != nil {
			return fmt.Errorf("failed to load %s translations: %w", locale, err)
		}
		t.translations[locale] = config
	}

	return nil
}

func (t *Translator) T(locale, key string, data ...interface{}) string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Get translation for locale
	localeMap, exists := t.translations[locale]
	if !exists {
		// Fallback to English or Lithuanian
		if locale != "en" {
			return t.T("en", key, data...)
		}
		return key // Return key if no translation found
	}

	// Navigate nested keys (e.g., "dashboard.title")
	value := t.getNestedValue(localeMap, key)
	if value == nil {
		// Try fallback locale
		if locale != "en" {
			return t.T("en", key, data...)
		}
		return key
	}

	str, ok := value.(string)
	if !ok {
		return key
	}

	// Handle template variables
	if len(data) > 0 {
		return t.processTemplate(str, data[0])
	}

	return str
}

func (t *Translator) getNestedValue(m map[string]interface{}, key string) interface{} {
	keys := strings.Split(key, ".")
	current := m

	for i, k := range keys {
		if i == len(keys)-1 {
			return current[k]
		}

		next, exists := current[k]
		if !exists {
			return nil
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil
		}

		current = nextMap
	}

	return nil
}

func (t *Translator) processTemplate(text string, data interface{}) string {
	tmpl, err := template.New("translation").Parse(text)
	if err != nil {
		return text
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return text
	}

	return buf.String()
}

// Helper function for templates
func T(locale, key string, data ...interface{}) string {
	return GetTranslator().T(locale, key, data...)
}
