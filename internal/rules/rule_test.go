package rules_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

var ctx = context.Background()

// ── TEST HELPERS ──────────────────────────────────────

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-rules-test-*")
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
		ID: "proj-001", Name: "test-app",
		RootPath: "/tmp", Platform: "web",
		PrimaryLanguage: "python",
	})
	return s
}

// mockRule is a test rule that returns a fixed set of findings
type mockRule struct {
	rules.BaseRule
	findings []rules.Finding
	err      error
}

func newMockRule(id string, sev rules.Severity, cat rules.Category, findings []rules.Finding) *mockRule {
	return &mockRule{
		BaseRule: rules.NewBaseRule(id, "Mock Rule "+id, cat, sev, "desc", "fix"),
		findings: findings,
	}
}

func (m *mockRule) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	return m.findings, m.err
}

func makeFinding(ruleID, nodeID, filePath string, sev rules.Severity, cat rules.Category) rules.Finding {
	return rules.Finding{
		RuleID:      ruleID,
		NodeID:      nodeID,
		FilePath:    filePath,
		Severity:    sev,
		Category:    cat,
		Title:       "Test Issue",
		Description: "Test description",
		Evidence:    "test evidence",
		Remediation: "test fix",
	}
}

// ── REGISTRY TESTS ────────────────────────────────────

func TestRegistry_Register_And_Get(t *testing.T) {
	r := rules.NewRegistry()
	rule := newMockRule("TEST-001", rules.SeverityHigh, rules.CategorySecurity, nil)
	r.Register(rule)

	got, ok := r.Get("TEST-001")
	if !ok {
		t.Fatal("Expected to find rule TEST-001")
	}
	if got.ID() != "TEST-001" {
		t.Errorf("Expected TEST-001, got %s", got.ID())
	}
	t.Logf("✅ Register and Get: %s", got.ID())
}

func TestRegistry_Count(t *testing.T) {
	r := rules.NewRegistry()
	r.Register(newMockRule("TEST-001", rules.SeverityHigh, rules.CategorySecurity, nil))
	r.Register(newMockRule("TEST-002", rules.SeverityMedium, rules.CategoryMaintainability, nil))
	r.Register(newMockRule("TEST-003", rules.SeverityLow, rules.CategoryStyle, nil))

	if r.Count() != 3 {
		t.Errorf("Expected 3 rules, got %d", r.Count())
	}
	t.Logf("✅ Count: %d", r.Count())
}

func TestRegistry_All_PreservesOrder(t *testing.T) {
	r := rules.NewRegistry()
	r.Register(newMockRule("TEST-001", rules.SeverityHigh, rules.CategorySecurity, nil))
	r.Register(newMockRule("TEST-002", rules.SeverityMedium, rules.CategoryMaintainability, nil))
	r.Register(newMockRule("TEST-003", rules.SeverityLow, rules.CategoryStyle, nil))

	all := r.All()
	if all[0].ID() != "TEST-001" || all[1].ID() != "TEST-002" || all[2].ID() != "TEST-003" {
		t.Error("Expected rules in registration order")
	}
	t.Logf("✅ All preserves order: %s, %s, %s", all[0].ID(), all[1].ID(), all[2].ID())
}

func TestRegistry_ByCategory(t *testing.T) {
	r := rules.NewRegistry()
	r.Register(newMockRule("SEC-001", rules.SeverityHigh, rules.CategorySecurity, nil))
	r.Register(newMockRule("SEC-002", rules.SeverityCritical, rules.CategorySecurity, nil))
	r.Register(newMockRule("MAINT-001", rules.SeverityMedium, rules.CategoryMaintainability, nil))

	secRules := r.ByCategory(rules.CategorySecurity)
	if len(secRules) != 2 {
		t.Errorf("Expected 2 security rules, got %d", len(secRules))
	}
	t.Logf("✅ ByCategory: %d security rules", len(secRules))
}

func TestRegistry_BySeverity(t *testing.T) {
	r := rules.NewRegistry()
	r.Register(newMockRule("SEC-001", rules.SeverityHigh, rules.CategorySecurity, nil))
	r.Register(newMockRule("SEC-002", rules.SeverityCritical, rules.CategorySecurity, nil))
	r.Register(newMockRule("MAINT-001", rules.SeverityHigh, rules.CategoryMaintainability, nil))

	highRules := r.BySeverity(rules.SeverityHigh)
	if len(highRules) != 2 {
		t.Errorf("Expected 2 high rules, got %d", len(highRules))
	}
	t.Logf("✅ BySeverity: %d high rules", len(highRules))
}

func TestRegistry_Register_Idempotent(t *testing.T) {
	r := rules.NewRegistry()
	rule := newMockRule("TEST-001", rules.SeverityHigh, rules.CategorySecurity, nil)
	r.Register(rule)
	r.Register(rule)
	r.Register(rule)

	if r.Count() != 1 {
		t.Errorf("Expected 1 rule after 3 registrations, got %d", r.Count())
	}
	t.Logf("✅ Register idempotent: count=%d", r.Count())
}

// ── ENGINE TESTS ──────────────────────────────────────

func TestEngine_RunAll_NoRules(t *testing.T) {
	s := newTestStore(t)
	e := rules.NewEngine(rules.NewRegistry(), s)

	result, err := e.RunAll(ctx, "proj-001")
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if result.RulesRun != 0 {
		t.Errorf("Expected 0 rules run, got %d", result.RulesRun)
	}
	if result.IssuesFound != 0 {
		t.Errorf("Expected 0 issues, got %d", result.IssuesFound)
	}
	t.Logf("✅ RunAll no rules: %d rules, %d issues", result.RulesRun, result.IssuesFound)
}

func TestEngine_RunAll_WritesIssuesToStore(t *testing.T) {
	s := newTestStore(t)

	// Need a node in store for the issue to reference
	s.WriteModule(ctx, store.Module{
		ID: "mod-001", ProjectID: "proj-001",
		FilePath: "src/a.py", QualifiedName: "a",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", Checksum: "abc",
	})

	findings := []rules.Finding{
		{
			RuleID: "SEC-001", NodeID: "mod-001", FilePath: "src/a.py",
			Severity: rules.SeverityCritical, Category: rules.CategorySecurity,
			Title: "Issue 1", Description: "desc", Evidence: "ev", Remediation: "fix",
			StartLine: 10,
		},
		{
			RuleID: "SEC-001", NodeID: "mod-001", FilePath: "src/a.py",
			Severity: rules.SeverityHigh, Category: rules.CategorySecurity,
			Title: "Issue 2", Description: "desc", Evidence: "ev", Remediation: "fix",
			StartLine: 20,
		},
	}

	reg := rules.NewRegistry()
	reg.Register(newMockRule("SEC-001", rules.SeverityCritical, rules.CategorySecurity, findings))

	e := rules.NewEngine(reg, s)
	result, err := e.RunAll(ctx, "proj-001")
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if result.IssuesFound != 2 {
		t.Errorf("Expected 2 issues, got %d", result.IssuesFound)
	}

	issues, _ := s.QueryIssues(ctx, store.IssueFilter{ProjectID: "proj-001"})
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues in store, got %d", len(issues))
	}
	t.Logf("✅ RunAll writes issues: %d found, %d in store", result.IssuesFound, len(issues))
}

func TestEngine_RunAll_ContinuesOnRuleError(t *testing.T) {
	s := newTestStore(t)

	failRule := newMockRule("FAIL-001", rules.SeverityHigh, rules.CategorySecurity, nil)
	failRule.err = errors.New("rule exploded")

	s.WriteModule(ctx, store.Module{
		ID: "mod-001", ProjectID: "proj-001",
		FilePath: "src/a.py", QualifiedName: "a",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", Checksum: "abc",
	})

	findings := []rules.Finding{
		makeFinding("OK-001", "mod-001", "src/a.py", rules.SeverityLow, rules.CategoryStyle),
	}
	okRule := newMockRule("OK-001", rules.SeverityLow, rules.CategoryStyle, findings)

	reg := rules.NewRegistry()
	reg.Register(failRule)
	reg.Register(okRule)

	e := rules.NewEngine(reg, s)
	result, err := e.RunAll(ctx, "proj-001")
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if !result.HasErrors() {
		t.Error("Expected errors in result")
	}
	if result.IssuesFound != 1 {
		t.Errorf("Expected 1 issue from okRule, got %d", result.IssuesFound)
	}
	t.Logf("✅ RunAll continues on error: issues=%d errors=%d", result.IssuesFound, len(result.Errors))
}

func TestEngine_RunRule_SingleRule(t *testing.T) {
	s := newTestStore(t)

	s.WriteModule(ctx, store.Module{
		ID: "mod-001", ProjectID: "proj-001",
		FilePath: "src/a.py", QualifiedName: "a",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", Checksum: "abc",
	})

	findings := []rules.Finding{
		makeFinding("SEC-001", "mod-001", "src/a.py", rules.SeverityCritical, rules.CategorySecurity),
	}

	reg := rules.NewRegistry()
	reg.Register(newMockRule("SEC-001", rules.SeverityCritical, rules.CategorySecurity, findings))

	e := rules.NewEngine(reg, s)
	got, err := e.RunRule(ctx, "proj-001", "SEC-001")
	if err != nil {
		t.Fatalf("RunRule: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(got))
	}
	t.Logf("✅ RunRule: %d findings", len(got))
}

func TestEngine_RunRule_NotFound(t *testing.T) {
	s := newTestStore(t)
	e := rules.NewEngine(rules.NewRegistry(), s)

	_, err := e.RunRule(ctx, "proj-001", "NONEXISTENT")
	if err == nil {
		t.Error("Expected RuleNotFoundError")
	}
	t.Logf("✅ RunRule not found: %v", err)
}

func TestEngine_Result_Duration(t *testing.T) {
	s := newTestStore(t)
	e := rules.NewEngine(rules.NewRegistry(), s)

	result, _ := e.RunAll(ctx, "proj-001")

	if result.Duration() < 0 {
		t.Error("Expected non-negative duration")
	}
	if result.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}
	t.Logf("✅ Result duration: %v", result.Duration())
}

// ── BASE RULE TESTS ───────────────────────────────────

func TestBaseRule_Fields(t *testing.T) {
	b := rules.NewBaseRule(
		"TEST-001", "Test Rule",
		rules.CategorySecurity, rules.SeverityHigh,
		"A test rule", "Fix it",
	)

	if b.ID() != "TEST-001" {
		t.Errorf("Expected TEST-001, got %s", b.ID())
	}
	if b.Name() != "Test Rule" {
		t.Errorf("Expected Test Rule, got %s", b.Name())
	}
	if b.Severity() != rules.SeverityHigh {
		t.Errorf("Expected HIGH, got %s", b.Severity())
	}
	if b.Category() != rules.CategorySecurity {
		t.Errorf("Expected SECURITY, got %s", b.Category())
	}
	t.Logf("✅ BaseRule fields: id=%s sev=%s cat=%s", b.ID(), b.Severity(), b.Category())
}

func TestFinding_ToIssue_DetectedAt(t *testing.T) {
	s := newTestStore(t)

	s.WriteModule(ctx, store.Module{
		ID: "mod-001", ProjectID: "proj-001",
		FilePath: "src/a.py", QualifiedName: "a",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", Checksum: "abc",
	})

	findings := []rules.Finding{
		makeFinding("SEC-001", "mod-001", "src/a.py", rules.SeverityCritical, rules.CategorySecurity),
	}
	reg := rules.NewRegistry()
	reg.Register(newMockRule("SEC-001", rules.SeverityCritical, rules.CategorySecurity, findings))

	e := rules.NewEngine(reg, s)
	e.RunAll(ctx, "proj-001")

	issues, _ := s.QueryIssues(ctx, store.IssueFilter{ProjectID: "proj-001"})
	if len(issues) == 0 {
		t.Fatal("No issues found")
	}
	if issues[0].DetectedAt.IsZero() {
		t.Error("Expected DetectedAt to be set")
	}
	if time.Since(issues[0].DetectedAt) > time.Minute {
		t.Error("Expected DetectedAt to be recent")
	}
	t.Logf("✅ Issue DetectedAt: %v", issues[0].DetectedAt)
}
