package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/schema"
)

func tempDB(t *testing.T) (*schema.Migrator, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-schema-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(dir, "nexus.db")
	db, err := schema.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	migrator := schema.NewMigrator(db)
	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}
	return migrator, cleanup
}

func TestMigrate_RunsSuccessfully(t *testing.T) {
	m, cleanup := tempDB(t)
	defer cleanup()
	if err := m.Migrate(); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}
	t.Logf("✅ Migrate: ran successfully")
}

func TestMigrate_Idempotent(t *testing.T) {
	m, cleanup := tempDB(t)
	defer cleanup()
	if err := m.Migrate(); err != nil {
		t.Fatalf("First migrate failed: %v", err)
	}
	if err := m.Migrate(); err != nil {
		t.Fatalf("Second migrate failed: %v", err)
	}
	t.Logf("✅ Migrate idempotent: ran twice without error")
}

func TestMigrate_Version_Updated(t *testing.T) {
	m, cleanup := tempDB(t)
	defer cleanup()
	m.Migrate()
	version, err := m.Version()
	if err != nil {
		t.Fatalf("Version() failed: %v", err)
	}
	if version == "none" {
		t.Error("Expected version to be set after migration")
	}
	t.Logf("✅ Schema version: %s", version)
}

func TestMigrate_Tables_Exist(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-schema-test-*")
	defer os.RemoveAll(dir)
	db, _ := schema.OpenDB(filepath.Join(dir, "nexus.db"))
	defer db.Close()
	m := schema.NewMigrator(db)
	m.Migrate()
	tables := []string{
		"schema_versions", "project", "nodes", "modules",
		"functions", "classes", "issues", "edges",
		"node_history", "builds",
	}
	for _, table := range tables {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`,
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("Table '%s' does not exist: %v", table, err)
		}
	}
	t.Logf("✅ All %d tables exist", len(tables))
}

func TestMigrate_Views_Exist(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-schema-test-*")
	defer os.RemoveAll(dir)
	db, _ := schema.OpenDB(filepath.Join(dir, "nexus.db"))
	defer db.Close()
	m := schema.NewMigrator(db)
	m.Migrate()
	views := []string{"v_urgent_issues", "v_project_health"}
	for _, view := range views {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='view' AND name=?`,
			view,
		).Scan(&name)
		if err != nil {
			t.Errorf("View '%s' does not exist: %v", view, err)
		}
	}
	t.Logf("✅ All %d views exist", len(views))
}

func TestMigrate_Indexes_Exist(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-schema-test-*")
	defer os.RemoveAll(dir)
	db, _ := schema.OpenDB(filepath.Join(dir, "nexus.db"))
	defer db.Close()
	m := schema.NewMigrator(db)
	m.Migrate()
	indexes := []string{
		"idx_nodes_project", "idx_nodes_type",
		"idx_functions_module", "idx_issues_severity",
		"idx_edges_from", "idx_edges_unique",
	}
	for _, idx := range indexes {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='index' AND name=?`,
			idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("Index '%s' does not exist: %v", idx, err)
		}
	}
	t.Logf("✅ All %d indexes exist", len(indexes))
}

func TestMigrate_ForeignKeys_Enabled(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-schema-test-*")
	defer os.RemoveAll(dir)
	db, _ := schema.OpenDB(filepath.Join(dir, "nexus.db"))
	defer db.Close()
	m := schema.NewMigrator(db)
	m.Migrate()
	var fkEnabled int
	db.QueryRow(`PRAGMA foreign_keys`).Scan(&fkEnabled)
	if fkEnabled != 1 {
		t.Error("Expected foreign keys to be enabled")
	}
	t.Logf("✅ Foreign keys enabled: %d", fkEnabled)
}

func TestMigrate_InsertAndQuery_Project(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-schema-test-*")
	defer os.RemoveAll(dir)
	db, _ := schema.OpenDB(filepath.Join(dir, "nexus.db"))
	defer db.Close()
	m := schema.NewMigrator(db)
	m.Migrate()
	_, err := db.Exec(
		`INSERT INTO project (id, name, root_path, platform, primary_language) VALUES (?, ?, ?, ?, ?)`,
		"proj-001", "my-app", "/home/user/my-app", "web-react", "typescript",
	)
	if err != nil {
		t.Fatalf("Insert project failed: %v", err)
	}
	var name string
	err = db.QueryRow(`SELECT name FROM project WHERE id = ?`, "proj-001").Scan(&name)
	if err != nil {
		t.Fatalf("Query project failed: %v", err)
	}
	if name != "my-app" {
		t.Errorf("Expected 'my-app', got '%s'", name)
	}
	t.Logf("✅ Insert and query project: %s", name)
}

func TestMigrate_ForeignKey_Enforced(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-schema-test-*")
	defer os.RemoveAll(dir)
	db, _ := schema.OpenDB(filepath.Join(dir, "nexus.db"))
	defer db.Close()
	m := schema.NewMigrator(db)
	m.Migrate()
	_, err := db.Exec(
		`INSERT INTO nodes (id, node_type, project_id, checksum) VALUES (?, ?, ?, ?)`,
		"node-001", "MODULE", "nonexistent-project", "abc123",
	)
	if err == nil {
		t.Error("Expected foreign key violation error")
	}
	t.Logf("✅ Foreign key enforced: insert rejected correctly")
}
