package security_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules/security"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-sec-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	s, err := store.NewSQLiteStore(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	s.WriteModule(ctx, store.Module{
		ID: "mod-001", ProjectID: "proj-001",
		FilePath: "src/a.py", QualifiedName: "a",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", Checksum: "abc",
	})
	return s
}

func writeFunction(t *testing.T, s store.GraphStore, id, name string) {
	t.Helper()
	s.WriteFunction(ctx, store.Function{
		ID: id, ProjectID: "proj-001", ModuleID: "mod-001",
		Name: name, QualifiedName: name,
		Language: "python", Visibility: "PUBLIC",
		Parameters: "[]", ReturnType: "{}", Annotations: "[]",
		StartLine: 10,
		Checksum:  "chk-" + id,
	})
}

// ── NEXUS-SEC-001 ─────────────────────────────────────

func TestHardcodedSecrets_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "get_password")
	writeFunction(t, s, "fn-002", "fetch_user")

	rule := security.NewHardcodedSecrets()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].NodeID != "fn-001" {
		t.Errorf("Expected fn-001, got %s", findings[0].NodeID)
	}
	if findings[0].CWE != "CWE-798" {
		t.Errorf("Expected CWE-798, got %s", findings[0].CWE)
	}
	t.Logf("✅ HardcodedSecrets: detected %s CWE=%s", findings[0].NodeID, findings[0].CWE)
}

func TestHardcodedSecrets_NoFindings(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "calculate_total")

	rule := security.NewHardcodedSecrets()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ HardcodedSecrets: no findings for safe function")
}

func TestHardcodedSecrets_RuleID(t *testing.T) {
	rule := security.NewHardcodedSecrets()
	if rule.ID() != "NEXUS-SEC-001" {
		t.Errorf("Expected NEXUS-SEC-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-SEC-002 ─────────────────────────────────────

func TestSQLInjectionRisk_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "execute_query")
	writeFunction(t, s, "fn-002", "get_users")

	rule := security.NewSQLInjectionRisk()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].CWE != "CWE-89" {
		t.Errorf("Expected CWE-89, got %s", findings[0].CWE)
	}
	t.Logf("✅ SQLInjectionRisk: detected %s CWE=%s", findings[0].NodeID, findings[0].CWE)
}

func TestSQLInjectionRisk_NoFindings(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "get_users")

	rule := security.NewSQLInjectionRisk()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ SQLInjectionRisk: no findings for safe function")
}

func TestSQLInjectionRisk_RuleID(t *testing.T) {
	rule := security.NewSQLInjectionRisk()
	if rule.ID() != "NEXUS-SEC-002" {
		t.Errorf("Expected NEXUS-SEC-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-SEC-003 ─────────────────────────────────────

func TestWeakCryptography_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "hash_md5")
	writeFunction(t, s, "fn-002", "hash_sha256")

	rule := security.NewWeakCryptography()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].CWE != "CWE-327" {
		t.Errorf("Expected CWE-327, got %s", findings[0].CWE)
	}
	t.Logf("✅ WeakCryptography: detected %s CWE=%s", findings[0].NodeID, findings[0].CWE)
}

func TestWeakCryptography_RuleID(t *testing.T) {
	rule := security.NewWeakCryptography()
	if rule.ID() != "NEXUS-SEC-003" {
		t.Errorf("Expected NEXUS-SEC-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-SEC-004 ─────────────────────────────────────

func TestOverlyBroadExceptionHandling_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "catch_all_errors")
	writeFunction(t, s, "fn-002", "handle_request")

	rule := security.NewOverlyBroadExceptionHandling()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].CWE != "CWE-390" {
		t.Errorf("Expected CWE-390, got %s", findings[0].CWE)
	}
	t.Logf("✅ OverlyBroadExceptionHandling: detected %s CWE=%s", findings[0].NodeID, findings[0].CWE)
}

func TestOverlyBroadExceptionHandling_RuleID(t *testing.T) {
	rule := security.NewOverlyBroadExceptionHandling()
	if rule.ID() != "NEXUS-SEC-004" {
		t.Errorf("Expected NEXUS-SEC-004, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-SEC-005 ─────────────────────────────────────

func TestInsecureDeserialization_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "load_pickle")
	writeFunction(t, s, "fn-002", "load_config")

	rule := security.NewInsecureDeserialization()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].CWE != "CWE-502" {
		t.Errorf("Expected CWE-502, got %s", findings[0].CWE)
	}
	t.Logf("✅ InsecureDeserialization: detected %s CWE=%s", findings[0].NodeID, findings[0].CWE)
}

func TestInsecureDeserialization_RuleID(t *testing.T) {
	rule := security.NewInsecureDeserialization()
	if rule.ID() != "NEXUS-SEC-005" {
		t.Errorf("Expected NEXUS-SEC-005, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── DEFAULT RULES ─────────────────────────────────────

func TestDefaultSecurityRules_Count(t *testing.T) {
	r := security.DefaultSecurityRules()
	if len(r) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(r))
	}
	ids := []string{
		"NEXUS-SEC-001", "NEXUS-SEC-002", "NEXUS-SEC-003",
		"NEXUS-SEC-004", "NEXUS-SEC-005",
	}
	for i, rule := range r {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultSecurityRules: %d rules", len(r))
}
