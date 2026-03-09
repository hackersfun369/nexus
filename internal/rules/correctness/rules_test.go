package correctness_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules/correctness"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-corr-test-*")
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

func writeFunction(t *testing.T, s store.GraphStore, fn store.Function) {
	t.Helper()
	fn.ProjectID = "proj-001"
	fn.ModuleID = "mod-001"
	if fn.Parameters == "" {
		fn.Parameters = "[]"
	}
	if fn.Annotations == "" {
		fn.Annotations = "[]"
	}
	s.WriteFunction(ctx, fn)
}

// ── NEXUS-CORR-001 ────────────────────────────────────

func TestMissingReturnType_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "no_return_type",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "has_return_type",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: `{"type":"str"}`, StartLine: 20, Checksum: "c2",
	})

	rule := correctness.NewMissingReturnType()
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
	t.Logf("✅ MissingReturnType: detected %s", findings[0].NodeID)
}

func TestMissingReturnType_IgnoresPrivate(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "_private",
		Language: "python", Visibility: "PRIVATE",
		ReturnType: "{}", StartLine: 10, Checksum: "c1",
	})

	rule := correctness.NewMissingReturnType()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ MissingReturnType: ignores private")
}

func TestMissingReturnType_IgnoresConstructors(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "__init__",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", IsConstructor: true,
		StartLine: 10, Checksum: "c1",
	})

	rule := correctness.NewMissingReturnType()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ MissingReturnType: ignores constructors")
}

func TestMissingReturnType_RuleID(t *testing.T) {
	rule := correctness.NewMissingReturnType()
	if rule.ID() != "NEXUS-CORR-001" {
		t.Errorf("Expected NEXUS-CORR-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-CORR-002 ────────────────────────────────────

func TestUnusedParameter_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "has_unused",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", FanOut: 5,
		StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "all_used",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", FanOut: 0,
		StartLine: 20, Checksum: "c2",
	})

	rule := correctness.NewUnusedParameter()
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
	t.Logf("✅ UnusedParameter: detected %s", findings[0].NodeID)
}

func TestUnusedParameter_RuleID(t *testing.T) {
	rule := correctness.NewUnusedParameter()
	if rule.ID() != "NEXUS-CORR-002" {
		t.Errorf("Expected NEXUS-CORR-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-CORR-003 ────────────────────────────────────

func TestEmptyCatchBlock_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "bare_except_handler",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "proper_handler",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", StartLine: 20, Checksum: "c2",
	})

	rule := correctness.NewEmptyCatchBlock()
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
	t.Logf("✅ EmptyCatchBlock: detected %s", findings[0].NodeID)
}

func TestEmptyCatchBlock_RuleID(t *testing.T) {
	rule := correctness.NewEmptyCatchBlock()
	if rule.ID() != "NEXUS-CORR-003" {
		t.Errorf("Expected NEXUS-CORR-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-CORR-004 ────────────────────────────────────

func TestDuplicateCode_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "process_a",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", LinesOfCode: 10,
		StartLine: 10, Checksum: "same-checksum-xyz",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "process_b",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", LinesOfCode: 10,
		StartLine: 30, Checksum: "same-checksum-xyz",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-003", Name: "unique_fn",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", LinesOfCode: 8,
		StartLine: 50, Checksum: "unique-checksum-abc",
	})

	rule := correctness.NewDuplicateCode(5)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 2 {
		t.Errorf("Expected 2 findings (one per duplicate), got %d", len(findings))
	}
	t.Logf("✅ DuplicateCode: detected %d findings", len(findings))
}

func TestDuplicateCode_IgnoresSmallFunctions(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "tiny_a",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", LinesOfCode: 2,
		StartLine: 10, Checksum: "same-checksum",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "tiny_b",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", LinesOfCode: 2,
		StartLine: 20, Checksum: "same-checksum",
	})

	rule := correctness.NewDuplicateCode(5)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for small functions, got %d", len(findings))
	}
	t.Logf("✅ DuplicateCode: ignores small functions")
}

func TestDuplicateCode_RuleID(t *testing.T) {
	rule := correctness.NewDuplicateCode(5)
	if rule.ID() != "NEXUS-CORR-004" {
		t.Errorf("Expected NEXUS-CORR-004, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-CORR-005 ────────────────────────────────────

func TestHighCognitiveComplexity_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "simple",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", CyclomaticComplexity: 5,
		StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "complex",
		Language: "python", Visibility: "PUBLIC",
		ReturnType: "{}", CyclomaticComplexity: 20,
		StartLine: 30, Checksum: "c2",
	})

	rule := correctness.NewHighCognitiveComplexity(15)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].NodeID != "fn-002" {
		t.Errorf("Expected fn-002, got %s", findings[0].NodeID)
	}
	t.Logf("✅ HighCyclomaticComplexity: detected %s", findings[0].NodeID)
}

func TestHighCognitiveComplexity_RuleID(t *testing.T) {
	rule := correctness.NewHighCognitiveComplexity(15)
	if rule.ID() != "NEXUS-CORR-005" {
		t.Errorf("Expected NEXUS-CORR-005, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── DEFAULT RULES ─────────────────────────────────────

func TestDefaultCorrectnessRules_Count(t *testing.T) {
	r := correctness.DefaultCorrectnessRules()
	if len(r) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(r))
	}
	ids := []string{
		"NEXUS-CORR-001", "NEXUS-CORR-002", "NEXUS-CORR-003",
		"NEXUS-CORR-004", "NEXUS-CORR-005",
	}
	for i, rule := range r {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultCorrectnessRules: %d rules", len(r))
}
