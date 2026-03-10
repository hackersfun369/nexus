package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hackersfun369/nexus/internal/graph/store"
)

type createProjectRequest struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	RootPath        string `json:"root_path"`
	Platform        string `json:"platform"`
	PrimaryLanguage string `json:"primary_language"`
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if projects == nil {
		projects = []store.Project{}
	}
	respond(w, http.StatusOK, map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
	})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.Name == "" || req.RootPath == "" {
		respondError(w, http.StatusBadRequest, "id, name, and root_path are required")
		return
	}
	project := store.Project{
		ID:              req.ID,
		Name:            req.Name,
		RootPath:        req.RootPath,
		Platform:        req.Platform,
		PrimaryLanguage: req.PrimaryLanguage,
	}
	if err := s.store.CreateProject(r.Context(), project); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, project)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}
	respond(w, http.StatusOK, project)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if err := s.store.DeleteProject(r.Context(), projectID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"deleted": projectID})
}
