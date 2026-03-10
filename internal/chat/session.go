package chat

import (
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// Session holds the current chat state
type Session struct {
	ProjectID       string
	ProjectName     string
	ProjectRootPath string
	SeverityFilter  string
	CategoryFilter  string
	History         []HistoryEntry
	Store           store.GraphStore
	Config          rules.Config
}

// HistoryEntry records a single exchange
type HistoryEntry struct {
	Input    string
	Response string
}

// NewSession creates a fresh session
func NewSession(s store.GraphStore, cfg rules.Config) *Session {
	return &Session{
		Store:   s,
		Config:  cfg,
		History: []HistoryEntry{},
	}
}

// HasProject returns true if a project is selected
func (s *Session) HasProject() bool {
	return s.ProjectID != ""
}

// SetProject updates the active project
func (s *Session) SetProject(p store.Project) {
	s.ProjectID = p.ID
	s.ProjectName = p.Name
	s.ProjectRootPath = p.RootPath
}

// ClearProject removes the active project
func (s *Session) ClearProject() {
	s.ProjectID = ""
	s.ProjectName = ""
	s.ProjectRootPath = ""
}

// AddHistory appends a history entry
func (s *Session) AddHistory(input, response string) {
	s.History = append(s.History, HistoryEntry{
		Input:    input,
		Response: response,
	})
	// Keep last 50 entries
	if len(s.History) > 50 {
		s.History = s.History[len(s.History)-50:]
	}
}

// Prompt returns the current prompt string
func (s *Session) Prompt() string {
	if s.HasProject() {
		return "nexus [" + s.ProjectName + "] > "
	}
	return "nexus > "
}
