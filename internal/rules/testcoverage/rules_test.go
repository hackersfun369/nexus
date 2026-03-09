package testcoverage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules/testcoverage"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-testcov-test-*")
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
		ParseErrors: "[]", LinesOfCode: 50,
		Checksum: "abc",
	})
	return s
}

func ptr(f float64) *float64 { return &f }

func writeFunction(t *testing.T, s store.GraphStore, fn store.Function) {
	t.Helper()
	fn.ProjectID = "proj-001"
	if fn.ModuleID == "" {
		fn.ModuleID = "mod-001"
	}
	if fn.Parameters == "" {
		fn.Parameters = "[]"
	}
	if fn.Annotations == "" {
		fn.Annotations = "[]"
	}
	if fn.ReturnType == "" {
		fn.ReturnType = "{}"
	}
	s.WriteFunction(ctx, fn)
}

// ── NEXUS-TEST-001 ────────────────────────────────────

func TestLowTestCoverage_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "low_coverage",
		Language: "python", Visibility: "PUBLIC",
		TestCoverage: ptr(40.0),
		StartLine:    10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "high_coverage",
		Language: "python", Visibility: "PUBLIC",
		TestCoverage: ptr(90.0),
		StartLine:    20, Checksum: "c2",
	})

	rule := testcoverage.NewLowTestCoverage(80.0)
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
	t.Logf("✅ LowTestCoverage: detected %s", findings[0].NodeID)
}

func TestLowTestCoverage_NilCoverageIsZero(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "no_coverage_data",
		Language: "python", Visibility: "PUBLIC",
		TestCoverage: nil,
		StartLine:    10, Checksum: "c1",
	})

	rule := testcoverage.NewLowTestCoverage(80.0)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding for nil coverage, got %d", len(findings))
	}
	t.Logf("✅ LowTestCoverage: nil coverage treated as 0%%")
}

func TestLowTestCoverage_IgnoresPrivate(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "_private",
		Language: "python", Visibility: "PRIVATE",
		TestCoverage: ptr(0.0),
		StartLine:    10, Checksum: "c1",
	})

	rule := testcoverage.NewLowTestCoverage(80.0)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for private function, got %d", len(findings))
	}
	t.Logf("✅ LowTestCoverage: ignores private")
}

func TestLowTestCoverage_RuleID(t *testing.T) {
	rule := testcoverage.NewLowTestCoverage(80.0)
	if rule.ID() != "NEXUS-TEST-001" {
		t.Errorf("Expected NEXUS-TEST-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-TEST-002 ────────────────────────────────────

func TestUntestedComplexFunction_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "complex_untested",
		Language: "python", Visibility: "PUBLIC",
		CyclomaticComplexity: 8, TestCoverage: nil,
		StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "complex_tested",
		Language: "python", Visibility: "PUBLIC",
		CyclomaticComplexity: 8, TestCoverage: ptr(85.0),
		StartLine: 20, Checksum: "c2",
	})

	rule := testcoverage.NewUntestedComplexFunction(5)
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
	t.Logf("✅ UntestedComplexFunction: detected %s", findings[0].NodeID)
}

func TestUntestedComplexFunction_RuleID(t *testing.T) {
	rule := testcoverage.NewUntestedComplexFunction(5)
	if rule.ID() != "NEXUS-TEST-002" {
		t.Errorf("Expected NEXUS-TEST-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-TEST-003 ────────────────────────────────────

func TestMissingTestFile_Detects(t *testing.T) {
	s := newTestStore(t)
	// mod-001 is src/a.py with 50 lines — no test file

	rule := testcoverage.NewMissingTestFile()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	t.Logf("✅ MissingTestFile: detected %s", findings[0].FilePath)
}

func TestMissingTestFile_NoFindingWhenTestExists(t *testing.T) {
	s := newTestStore(t)
	s.WriteModule(ctx, store.Module{
		ID: "mod-002", ProjectID: "proj-001",
		FilePath: "src/a_test.py", QualifiedName: "a_test",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", LinesOfCode: 20,
		Checksum: "xyz",
	})

	rule := testcoverage.NewMissingTestFile()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	for _, f := range findings {
		if f.FilePath == "src/a.py" {
			t.Error("Should not flag src/a.py when test file exists")
		}
	}
	t.Logf("✅ MissingTestFile: no finding when test file exists")
}

func TestMissingTestFile_RuleID(t *testing.T) {
	rule := testcoverage.NewMissingTestFile()
	if rule.ID() != "NEXUS-TEST-003" {
		t.Errorf("Expected NEXUS-TEST-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-TEST-004 ────────────────────────────────────

func TestTestWithoutAssertion_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "test_something",
		Language: "python", Visibility: "PUBLIC",
		FanOut: 0, StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "test_with_assert",
		Language: "python", Visibility: "PUBLIC",
		FanOut: 3, StartLine: 20, Checksum: "c2",
	})

	rule := testcoverage.NewTestWithoutAssertion()
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
	t.Logf("✅ TestWithoutAssertion: detected %s", findings[0].NodeID)
}

func TestTestWithoutAssertion_IgnoresNonTests(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "process_data",
		Language: "python", Visibility: "PUBLIC",
		FanOut: 0, StartLine: 10, Checksum: "c1",
	})

	rule := testcoverage.NewTestWithoutAssertion()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for non-test function, got %d", len(findings))
	}
	t.Logf("✅ TestWithoutAssertion: ignores non-test functions")
}

func TestTestWithoutAssertion_RuleID(t *testing.T) {
	rule := testcoverage.NewTestWithoutAssertion()
	if rule.ID() != "NEXUS-TEST-004" {
		t.Errorf("Expected NEXUS-TEST-004, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-TEST-005 ────────────────────────────────────

func TestHighComplexityNoCoverage_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "complex_low_cov",
		Language: "python", Visibility: "PUBLIC",
		CyclomaticComplexity: 10, TestCoverage: ptr(30.0),
		StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "complex_high_cov",
		Language: "python", Visibility: "PUBLIC",
		CyclomaticComplexity: 10, TestCoverage: ptr(90.0),
		StartLine: 20, Checksum: "c2",
	})

	rule := testcoverage.NewHighComplexityNoCoverage(8, 60.0)
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
	t.Logf("✅ HighComplexityNoCoverage: detected %s", findings[0].NodeID)
}

func TestHighComplexityNoCoverage_RuleID(t *testing.T) {
	rule := testcoverage.NewHighComplexityNoCoverage(8, 60.0)
	if rule.ID() != "NEXUS-TEST-005" {
		t.Errorf("Expected NEXUS-TEST-005, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── DEFAULT RULES ─────────────────────────────────────

func TestDefaultTestCoverageRules_Count(t *testing.T) {
	r := testcoverage.DefaultTestCoverageRules()
	if len(r) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(r))
	}
	ids := []string{
		"NEXUS-TEST-001", "NEXUS-TEST-002", "NEXUS-TEST-003",
		"NEXUS-TEST-004", "NEXUS-TEST-005",
	}
	for i, rule := range r {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultTestCoverageRules: %d rules", len(r))
}
