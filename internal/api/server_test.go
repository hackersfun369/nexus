package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/api"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

var ctx = context.Background()

func newTestServer(t *testing.T) (*api.Server, store.GraphStore) {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-api-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	s, err := store.NewSQLiteStore(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	cfg := rules.DefaultConfig()
	srv := api.NewServer(s, cfg, ":0")
	return srv, s
}

func do(t *testing.T, srv *api.Server, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	return rr
}

// ── HEALTH ────────────────────────────────────────────

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodGet, "/health", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("Expected status=ok, got %v", resp["status"])
	}
	t.Logf("✅ GET /health → %d", rr.Code)
}

func TestReady(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodGet, "/ready", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	t.Logf("✅ GET /ready → %d", rr.Code)
}

// ── PROJECTS ──────────────────────────────────────────

func TestCreateProject(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodPost, "/api/v1/projects", map[string]string{
		"id":               "proj-001",
		"name":             "my-app",
		"root_path":        "/home/user/my-app",
		"platform":         "web",
		"primary_language": "python",
	})
	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	t.Logf("✅ POST /api/v1/projects → %d", rr.Code)
}

func TestCreateProject_MissingFields(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodPost, "/api/v1/projects", map[string]string{
		"name": "incomplete",
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rr.Code)
	}
	t.Logf("✅ POST /api/v1/projects (missing fields) → %d", rr.Code)
}

func TestGetProject(t *testing.T) {
	srv, s := newTestServer(t)
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	rr := do(t, srv, http.MethodGet, "/api/v1/projects/proj-001", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	t.Logf("✅ GET /api/v1/projects/proj-001 → %d", rr.Code)
}

func TestGetProject_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodGet, "/api/v1/projects/nonexistent", nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rr.Code)
	}
	t.Logf("✅ GET /api/v1/projects/nonexistent → %d", rr.Code)
}

func TestListProjects(t *testing.T) {
	srv, s := newTestServer(t)
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	rr := do(t, srv, http.MethodGet, "/api/v1/projects", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if int(resp["count"].(float64)) != 1 {
		t.Errorf("Expected count=1, got %v", resp["count"])
	}
	t.Logf("✅ GET /api/v1/projects → %d", rr.Code)
}

func TestDeleteProject(t *testing.T) {
	srv, s := newTestServer(t)
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	rr := do(t, srv, http.MethodDelete, "/api/v1/projects/proj-001", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	t.Logf("✅ DELETE /api/v1/projects/proj-001 → %d", rr.Code)
}

// ── ANALYSIS ──────────────────────────────────────────

func TestRunAnalysis(t *testing.T) {
	srv, s := newTestServer(t)
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	rr := do(t, srv, http.MethodPost, "/api/v1/projects/proj-001/analysis/run", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["status"] != "complete" {
		t.Errorf("Expected status=complete, got %v", resp["status"])
	}
	t.Logf("✅ POST /api/v1/projects/proj-001/analysis/run → %d", rr.Code)
}

func TestRunAnalysis_ProjectNotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodPost, "/api/v1/projects/nonexistent/analysis/run", nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", rr.Code)
	}
	t.Logf("✅ POST /api/v1/projects/nonexistent/analysis/run → %d", rr.Code)
}

func TestListIssues(t *testing.T) {
	srv, s := newTestServer(t)
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	rr := do(t, srv, http.MethodGet, "/api/v1/projects/proj-001/analysis/issues", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	t.Logf("✅ GET /api/v1/projects/proj-001/analysis/issues → %d", rr.Code)
}

func TestAnalysisSummary(t *testing.T) {
	srv, s := newTestServer(t)
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	rr := do(t, srv, http.MethodGet, "/api/v1/projects/proj-001/analysis/summary", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["project_id"] != "proj-001" {
		t.Errorf("Expected project_id=proj-001, got %v", resp["project_id"])
	}
	t.Logf("✅ GET /api/v1/projects/proj-001/analysis/summary → %d", rr.Code)
}

// ── MIDDLEWARE ────────────────────────────────────────

func TestCORS_Headers(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodGet, "/health", nil)
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header")
	}
	t.Logf("✅ CORS headers present")
}

func TestCORS_Preflight(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := do(t, srv, http.MethodOptions, "/api/v1/projects", nil)
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected 204 for OPTIONS, got %d", rr.Code)
	}
	t.Logf("✅ OPTIONS preflight → %d", rr.Code)
}
