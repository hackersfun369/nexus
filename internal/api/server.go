package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
	"github.com/hackersfun369/nexus/internal/rules/loader"
)

// Server holds the HTTP server and its dependencies
type Server struct {
	store  store.GraphStore
	router *chi.Mux
	http   *http.Server
	cfg    rules.Config
}

// NewServer creates a new API server
func NewServer(s store.GraphStore, cfg rules.Config, addr string) *Server {
	srv := &Server{
		store: s,
		cfg:   cfg,
	}
	srv.router = srv.buildRouter()
	srv.http = &http.Server{
		Addr:         addr,
		Handler:      srv.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return srv
}

// Start begins listening
func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// Router returns the underlying chi router (for testing)
func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(Logger())
	r.Use(Recoverer())
	r.Use(CORS())
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// Routes
	r.Get("/health", s.handleHealth)
	r.Get("/ready", s.handleReady)

	r.Route("/api/v1", func(r chi.Router) {
		// Projects
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", s.handleListProjects)
			r.Post("/", s.handleCreateProject)
			r.Route("/{projectID}", func(r chi.Router) {
				r.Get("/", s.handleGetProject)
				r.Delete("/", s.handleDeleteProject)
			})
		})

		// Analysis
		r.Route("/projects/{projectID}/analysis", func(r chi.Router) {
			r.Post("/run", s.handleRunAnalysis)
			r.Get("/issues", s.handleListIssues)
			r.Get("/summary", s.handleAnalysisSummary)
		})
		r.Post("/generate", s.handleGenerate)
		r.Get("/generate/zip", s.handleDownloadZip)
		r.Get("/generated", s.handleListGeneratedProjects)
		r.Get("/generated/{id}", s.handleGetGeneratedProject)
	})

	return r
}

// registry returns a fresh registry for a given config
func (s *Server) registry() *rules.Registry {
	return loader.DefaultRegistry(s.cfg)
}

// buildVersion info
func buildVersion() string {
	return fmt.Sprintf("nexus-api/%s", "dev")
}
