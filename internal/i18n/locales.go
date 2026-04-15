package i18n

import (
	"embed"
	"encoding/json"
)

//go:embed locales/*.json
var localeFS embed.FS

// loadMessages reads the translation map for the given locale.
// Returns an empty map if the file does not exist or is invalid.
func loadMessages(locale string) map[string]string {
	data, err := localeFS.ReadFile("locales/" + locale + ".json")
	if err != nil {
		return make(map[string]string)
	}
	var msgs map[string]string
	if err := json.Unmarshal(data, &msgs); err != nil {
		return make(map[string]string)
	}
	return msgs
}
