package plugin_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/hackersfun369/nexus/internal/plugin"
)

var ctx = context.Background()

func newTestStore(t *testing.T) (*plugin.Store, *sql.DB) {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-plugin-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	db, err := sql.Open("sqlite3", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create tables
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS plugin (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			kind TEXT NOT NULL,
			platform TEXT,
			language TEXT,
			description TEXT,
			author TEXT,
			registry_url TEXT,
			download_url TEXT,
			sha256 TEXT,
			install_path TEXT,
			status TEXT NOT NULL DEFAULT 'available',
			manifest TEXT,
			installed_at DATETIME,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS plugin_dependency (
			plugin_id TEXT NOT NULL,
			depends_on TEXT NOT NULL,
			min_version TEXT,
			PRIMARY KEY (plugin_id, depends_on)
		);
		CREATE TABLE IF NOT EXISTS plugin_capability (
			plugin_id TEXT NOT NULL,
			capability TEXT NOT NULL,
			PRIMARY KEY (plugin_id, capability)
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	return plugin.NewStore(db), db
}

func samplePlugin() plugin.Plugin {
	return plugin.Plugin{
		ID:          "python-fastapi",
		Name:        "Python FastAPI",
		Version:     "1.0.0",
		Kind:        plugin.KindLanguage,
		Platform:    "backend",
		Language:    "python",
		Description: "Python FastAPI backend plugin",
		Author:      "nexus",
		DownloadURL: "https://example.com/python-fastapi-1.0.0.json",
		SHA256:      "abc123",
		Status:      plugin.StatusAvailable,
		UpdatedAt:   time.Now(),
	}
}

// ── STORE TESTS ───────────────────────────────────────

func TestStore_SaveAndGet(t *testing.T) {
	s, _ := newTestStore(t)
	p := samplePlugin()

	if err := s.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("Expected ID %s, got %s", p.ID, got.ID)
	}
	if got.Version != p.Version {
		t.Errorf("Expected version %s, got %s", p.Version, got.Version)
	}
	if got.Status != plugin.StatusAvailable {
		t.Errorf("Expected status available, got %s", got.Status)
	}
	t.Logf("✅ Store: Save and Get")
}

func TestStore_Get_NotFound(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent plugin")
	}
	t.Logf("✅ Store: Get not found returns error")
}

func TestStore_List_ByKind(t *testing.T) {
	s, _ := newTestStore(t)
	s.Save(ctx, samplePlugin())
	s.Save(ctx, plugin.Plugin{
		ID: "nexus-rules-security", Name: "Security Rules",
		Version: "1.0.0", Kind: plugin.KindRule,
		Status: plugin.StatusAvailable, UpdatedAt: time.Now(),
	})

	plugins, err := s.List(ctx, plugin.Filter{Kind: plugin.KindLanguage})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(plugins) != 1 {
		t.Errorf("Expected 1 language plugin, got %d", len(plugins))
	}
	t.Logf("✅ Store: List by kind")
}

func TestStore_List_ByStatus(t *testing.T) {
	s, _ := newTestStore(t)
	p := samplePlugin()
	s.Save(ctx, p)

	now := time.Now()
	p2 := samplePlugin()
	p2.ID = "kotlin-android"
	p2.Status = plugin.StatusInstalled
	p2.InstalledAt = &now
	s.Save(ctx, p2)

	installed, err := s.List(ctx, plugin.Filter{Status: plugin.StatusInstalled})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(installed) != 1 {
		t.Errorf("Expected 1 installed plugin, got %d", len(installed))
	}
	t.Logf("✅ Store: List by status")
}

func TestStore_SetStatus(t *testing.T) {
	s, _ := newTestStore(t)
	s.Save(ctx, samplePlugin())

	if err := s.SetStatus(ctx, "python-fastapi", plugin.StatusInstalled); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	p, _ := s.Get(ctx, "python-fastapi")
	if p.Status != plugin.StatusInstalled {
		t.Errorf("Expected installed, got %s", p.Status)
	}
	t.Logf("✅ Store: SetStatus")
}

func TestStore_Delete(t *testing.T) {
	s, _ := newTestStore(t)
	s.Save(ctx, samplePlugin())

	if err := s.Delete(ctx, "python-fastapi"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := s.Get(ctx, "python-fastapi")
	if err == nil {
		t.Error("Expected error after delete")
	}
	t.Logf("✅ Store: Delete")
}

func TestStore_Capabilities(t *testing.T) {
	s, _ := newTestStore(t)
	s.Save(ctx, samplePlugin())

	caps := []string{"http_server", "orm", "authentication"}
	if err := s.SaveCapabilities(ctx, "python-fastapi", caps); err != nil {
		t.Fatalf("SaveCapabilities: %v", err)
	}

	// Mark as installed so FindByCapability works
	s.SetStatus(ctx, "python-fastapi", plugin.StatusInstalled)

	found, err := s.FindByCapability(ctx, "authentication")
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("Expected 1 plugin with authentication, got %d", len(found))
	}
	t.Logf("✅ Store: Capabilities")
}

// ── MANIFEST TESTS ────────────────────────────────────

func TestManifestToPlugin(t *testing.T) {
	m := plugin.Manifest{
		ID:          "react-web",
		Name:        "React Web",
		Version:     "3.0.0",
		Kind:        plugin.KindFramework,
		Platform:    "web",
		Language:    "typescript",
		Description: "React web framework plugin",
		Author:      "nexus",
		DownloadURL: "https://example.com/react-web.json",
		SHA256:      "def456",
		Capabilities: []string{"ui_components", "routing", "state_management"},
	}

	p := plugin.ManifestToPlugin(m, "https://plugins.nexus.dev")
	if p.ID != m.ID {
		t.Errorf("Expected ID %s, got %s", m.ID, p.ID)
	}
	if p.Status != plugin.StatusAvailable {
		t.Errorf("Expected available, got %s", p.Status)
	}
	if p.Manifest == "" {
		t.Error("Expected manifest JSON to be set")
	}
	t.Logf("✅ ManifestToPlugin: converts correctly")
}

// ── REGISTRY MANIFEST ─────────────────────────────────

func TestRegistryManifest_Structure(t *testing.T) {
	manifest := plugin.RegistryManifest{
		SchemaVersion: "1",
		UpdatedAt:     time.Now(),
		Plugins: map[string]plugin.Manifest{
			"python-fastapi": {
				ID: "python-fastapi", Name: "Python FastAPI",
				Version: "1.0.0", Kind: plugin.KindLanguage,
			},
			"kotlin-android": {
				ID: "kotlin-android", Name: "Kotlin Android",
				Version: "1.2.0", Kind: plugin.KindLanguage,
			},
		},
	}

	if len(manifest.Plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(manifest.Plugins))
	}
	if manifest.SchemaVersion != "1" {
		t.Errorf("Expected schema version 1, got %s", manifest.SchemaVersion)
	}
	t.Logf("✅ RegistryManifest: structure correct")
}

func TestDefaultRegistryURL(t *testing.T) {
	if plugin.DefaultRegistryURL == "" {
		t.Error("Expected non-empty default registry URL")
	}
	t.Logf("✅ DefaultRegistryURL: %s", plugin.DefaultRegistryURL)
}
