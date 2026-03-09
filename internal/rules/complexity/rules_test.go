package complexity_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules/complexity"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-complexity-test-*")
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

func writeFunction(t *testing.T, s store.GraphStore, id, name string, complexity, lines, params, nesting int) {
	t.Helper()
	s.WriteFunction(ctx, store.Function{
		ID: id, ProjectID: "proj-001", ModuleID: "mod-001",
		Name: name, QualifiedName: name,
		Language: "python", Visibility: "PUBLIC",
		Parameters: "[]", ReturnType: "{}", Annotations: "[]",
		CyclomaticComplexity: complexity,
		LinesOfCode:          lines,
		ParameterCount:       params,
		NestingDepth:         nesting,
		StartLine:            10,
		Checksum:             "chk-" + id,
	})
}

func writeClass(t *testing.T, s store.GraphStore, id, name string, methods, lines int) {
	t.Helper()
	s.WriteClass(ctx, store.Class{
		ID: id, ProjectID: "proj-001", ModuleID: "mod-001",
		Name: name, QualifiedName: name,
		Language: "python", Kind: "CLASS",
		Visibility:  "PUBLIC",
		MethodCount: methods,
		LinesOfCode: lines,
		Annotations: "[]",
		StartLine:   5,
		Checksum:    "chk-" + id,
	})
}

// ── NEXUS-COMP-001 ────────────────────────────────────

func TestHighCyclomaticComplexity_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "simple", 3, 10, 1, 1)
	writeFunction(t, s, "fn-002", "complex", 15, 40, 3, 3)

	rule := complexity.NewHighCyclomaticComplexity(10)
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

func TestHighCyclomaticComplexity_NoFindings(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "simple", 3, 10, 1, 1)

	rule := complexity.NewHighCyclomaticComplexity(10)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ HighCyclomaticComplexity: no findings for simple function")
}

func TestHighCyclomaticComplexity_RuleID(t *testing.T) {
	rule := complexity.NewHighCyclomaticComplexity(10)
	if rule.ID() != "NEXUS-COMP-001" {
		t.Errorf("Expected NEXUS-COMP-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-COMP-002 ────────────────────────────────────

func TestLongFunction_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "short", 2, 10, 1, 1)
	writeFunction(t, s, "fn-002", "long", 5, 80, 2, 2)

	rule := complexity.NewLongFunction(50)
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
	t.Logf("✅ LongFunction: detected %s", findings[0].NodeID)
}

func TestLongFunction_RuleID(t *testing.T) {
	rule := complexity.NewLongFunction(50)
	if rule.ID() != "NEXUS-COMP-002" {
		t.Errorf("Expected NEXUS-COMP-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-COMP-003 ────────────────────────────────────

func TestTooManyParameters_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "few_params", 2, 10, 2, 1)
	writeFunction(t, s, "fn-002", "many_params", 3, 20, 8, 2)

	rule := complexity.NewTooManyParameters(5)
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
	t.Logf("✅ TooManyParameters: detected %s", findings[0].NodeID)
}

func TestTooManyParameters_RuleID(t *testing.T) {
	rule := complexity.NewTooManyParameters(5)
	if rule.ID() != "NEXUS-COMP-003" {
		t.Errorf("Expected NEXUS-COMP-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-COMP-004 ────────────────────────────────────

func TestDeepNesting_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "shallow", 2, 10, 1, 2)
	writeFunction(t, s, "fn-002", "deep", 4, 30, 2, 6)

	rule := complexity.NewDeepNesting(4)
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
	t.Logf("✅ DeepNesting: detected %s", findings[0].NodeID)
}

func TestDeepNesting_RuleID(t *testing.T) {
	rule := complexity.NewDeepNesting(4)
	if rule.ID() != "NEXUS-COMP-004" {
		t.Errorf("Expected NEXUS-COMP-004, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-COMP-005 ────────────────────────────────────

func TestGodClass_Detects_TooManyMethods(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, "cls-001", "Small", 5, 50)
	writeClass(t, s, "cls-002", "God", 25, 100)

	rule := complexity.NewGodClass(20, 300)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].NodeID != "cls-002" {
		t.Errorf("Expected cls-002, got %s", findings[0].NodeID)
	}
	t.Logf("✅ GodClass TooManyMethods: detected %s", findings[0].NodeID)
}

func TestGodClass_Detects_TooManyLines(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, "cls-001", "Small", 5, 50)
	writeClass(t, s, "cls-002", "Huge", 10, 500)

	rule := complexity.NewGodClass(20, 300)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	t.Logf("✅ GodClass TooManyLines: detected %s", findings[0].NodeID)
}

func TestGodClass_NoFindings(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, "cls-001", "Small", 5, 50)

	rule := complexity.NewGodClass(20, 300)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ GodClass: no findings for small class")
}

func TestGodClass_RuleID(t *testing.T) {
	rule := complexity.NewGodClass(20, 300)
	if rule.ID() != "NEXUS-COMP-005" {
		t.Errorf("Expected NEXUS-COMP-005, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── DEFAULT RULES ─────────────────────────────────────

func TestDefaultComplexityRules_Count(t *testing.T) {
	rules := complexity.DefaultComplexityRules()
	if len(rules) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(rules))
	}
	ids := []string{"NEXUS-COMP-001", "NEXUS-COMP-002", "NEXUS-COMP-003", "NEXUS-COMP-004", "NEXUS-COMP-005"}
	for i, rule := range rules {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultComplexityRules: %d rules", len(rules))
}
