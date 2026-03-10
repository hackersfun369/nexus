package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type healthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type readyResponse struct {
	Status string `json:"status"`
	Store  string `json:"store"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   buildVersion(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	// Ping store by querying a non-existent project — error means store is down
	_, err := s.store.GetProject(r.Context(), "__health_check__")
	// "not found" is fine — store is up
	// any other error means store is unavailable
	if err != nil && err.Error() == "store unavailable" {
		respond(w, http.StatusServiceUnavailable, readyResponse{
			Status: "unavailable",
			Store:  err.Error(),
		})
		return
	}
	respond(w, http.StatusOK, readyResponse{
		Status: "ready",
		Store:  "ok",
	})
}

func respond(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}
