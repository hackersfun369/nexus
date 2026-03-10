package plugin

import "time"

type Kind string

const (
	KindLanguage  Kind = "language"
	KindFramework Kind = "framework"
	KindPlatform  Kind = "platform"
	KindRule      Kind = "rule"
)

type Status string

const (
	StatusAvailable Status = "available"
	StatusInstalled Status = "installed"
	StatusDisabled  Status = "disabled"
)

type Plugin struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Kind        Kind       `json:"kind"`
	Platform    string     `json:"platform"`
	Language    string     `json:"language"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	RegistryURL string     `json:"registry_url"`
	DownloadURL string     `json:"download_url"`
	SHA256      string     `json:"sha256"`
	InstallPath string     `json:"install_path"`
	Status      Status     `json:"status"`
	Manifest    string     `json:"manifest"`
	InstalledAt *time.Time `json:"installed_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Manifest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Kind         Kind              `json:"kind"`
	Platform     string            `json:"platform"`
	Language     string            `json:"language"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	MinNexus     string            `json:"min_nexus"`
	Dependencies []Dependency      `json:"dependencies"`
	Capabilities []string          `json:"capabilities"`
	Templates    map[string]string `json:"templates"`
	Grammar      *GrammarRef       `json:"grammar,omitempty"`
	Rules        *RulesRef         `json:"rules,omitempty"`
	DownloadURL  string            `json:"download_url"`
	SHA256       string            `json:"sha256"`
}

type Dependency struct {
	ID         string `json:"id"`
	MinVersion string `json:"min_version"`
}

type GrammarRef struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type RulesRef struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type RegistryManifest struct {
	SchemaVersion string              `json:"schema_version"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Plugins       map[string]Manifest `json:"plugins"`
}

type Filter struct {
	Kind     Kind
	Platform string
	Language string
	Status   Status
}
