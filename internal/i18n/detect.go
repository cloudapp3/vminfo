package i18n

import (
	"os"
	"strings"
)

// Detect returns the locale string using this priority:
// 1. VMINFO_LANG env var
// 2. LANG / LC_ALL system locale
// 3. Default "en"
func Detect() string {
	// 1. App-specific env var
	if env := os.Getenv("VMINFO_LANG"); env != "" {
		return normalizeLocale(env)
	}
	// 2. System locale
	for _, key := range []string{"LC_ALL", "LANG"} {
		if val := os.Getenv(key); val != "" {
			code := normalizeLocale(val)
			if code != "" && code != "c" && code != "posix" {
				return code
			}
		}
	}
	return "en"
}

// normalizeLocale extracts the base language code from locale strings
// like "zh_CN.UTF-8" → "zh", "en_US" → "en".
func normalizeLocale(val string) string {
	val = strings.TrimSpace(strings.ToLower(val))
	if idx := strings.IndexAny(val, "_."); idx > 0 {
		val = val[:idx]
	}
	return val
}
