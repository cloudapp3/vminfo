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
	Version     = "dev"
	Commit      = "none"
	BuildTime   = "unknown"
	Channel     = "dev"
	Repository  = DefaultRepositoryURL
	Homepage    = DefaultHomepageURL
	License     = ""
	Description = DefaultDescription
)

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
