package i18n

import "fmt"

// Translator provides lightweight string translation.
// Keys are English text; missing keys fall back to the key itself.
type Translator struct {
	locale string
	msgs   map[string]string
}

// New creates a Translator for the given locale (e.g. "en", "zh").
// If the locale has no translation file, all lookups fall back to the key.
func New(locale string) *Translator {
	return &Translator{
		locale: locale,
		msgs:   loadMessages(locale),
	}
}

// T returns the translated string for key, or key itself as fallback.
func (t *Translator) T(key string) string {
	if v, ok := t.msgs[key]; ok {
		return v
	}
	return key
}

// Tf returns fmt.Sprintf(T(key), args...).
func (t *Translator) Tf(key string, args ...any) string {
	return fmt.Sprintf(t.T(key), args...)
}

// Locale returns the current locale code (e.g. "en", "zh").
func (t *Translator) Locale() string {
	return t.locale
}
