package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

type analysisSummary struct {
	ProjectID  string         `json:"project_id"`
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
	ByCategory map[string]int `json:"by_category"`
}

func (s *Server) handleRunAnalysis(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")

	if _, err := s.store.GetProject(r.Context(), projectID); err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	reg := s.registry()
	engine := rules.NewEngine(reg, s.store)

	result, err := engine.RunAll(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"project_id":   projectID,
		"issues_found": result.IssuesFound,
		"rules_run":    result.RulesRun,
		"status":       "complete",
	})
}

func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")

	filter := store.IssueFilter{ProjectID: projectID}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filter.Severity = severity
	}
	if category := r.URL.Query().Get("category"); category != "" {
		filter.Category = category
	}

	issues, err := s.store.QueryIssues(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"issues":     issues,
		"count":      len(issues),
	})
}

func (s *Server) handleAnalysisSummary(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")

	issues, err := s.store.QueryIssues(r.Context(), store.IssueFilter{ProjectID: projectID})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	bySeverity := make(map[string]int)
	byCategory := make(map[string]int)
	for _, issue := range issues {
		bySeverity[issue.Severity]++
		byCategory[issue.Category]++
	}

	respond(w, http.StatusOK, analysisSummary{
		ProjectID:  projectID,
		Total:      len(issues),
		BySeverity: bySeverity,
		ByCategory: byCategory,
	})
}
