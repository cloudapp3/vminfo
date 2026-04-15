package vminfo

import "strings"

const (
	AppName              = "vminfo"
	DefaultDescription   = "Host runtime information toolkit"
	DefaultRepositoryURL = "https://github.com/VPSMarket/vminfo"
	DefaultHomepageURL   = DefaultRepositoryURL
	DefaultSchemaVersion = "v1"
)

var (
	// Version is the application version injected at build time.
	Version = "dev"
	// Commit is the source revision injected at build time.
	Commit = "none"
	// BuildTime is the build timestamp injected at build time.
	BuildTime = "unknown"
	// Channel is the release channel injected at build time.
	Channel     = "dev"
	Repository  = DefaultRepositoryURL
	Homepage    = DefaultHomepageURL
	License     = "MIT"
	Description = DefaultDescription
)

// AppMetadata describes build and repository metadata for the vminfo CLI.
type AppMetadata struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Commit        string `json:"commit,omitempty"`
	BuildTime     string `json:"build_time,omitempty"`
	Channel       string `json:"channel,omitempty"`
	Repository    string `json:"repository,omitempty"`
	Homepage      string `json:"homepage,omitempty"`
	License       string `json:"license,omitempty"`
	Description   string `json:"description,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

// Metadata returns normalized application metadata for CLI and embedding use.
func Metadata() AppMetadata {
	return AppMetadata{
		Name:          AppName,
		Version:       normalizedField(Version, "dev"),
		Commit:        optionalField(Commit, "", "none", "unknown"),
		BuildTime:     optionalField(BuildTime, "", "unknown"),
		Channel:       optionalField(Channel, "", "unknown"),
		Repository:    optionalField(Repository, ""),
		Homepage:      optionalField(Homepage, ""),
		License:       optionalField(License, ""),
		Description:   optionalField(Description, ""),
		SchemaVersion: DefaultSchemaVersion,
	}
}

func normalizedField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func optionalField(value string, emptyMarkers ...string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, marker := range emptyMarkers {
		if strings.EqualFold(value, strings.TrimSpace(marker)) {
			return ""
		}
	}
	return value
}
