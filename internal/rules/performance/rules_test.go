package performance_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules/performance"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-perf-test-*")
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
	if fn.ReturnType == "" {
		fn.ReturnType = "{}"
	}
	s.WriteFunction(ctx, fn)
}

func writeClass(t *testing.T, s store.GraphStore, cls store.Class) {
	t.Helper()
	cls.ProjectID = "proj-001"
	cls.ModuleID = "mod-001"
	if cls.Annotations == "" {
		cls.Annotations = "[]"
	}
	if cls.Kind == "" {
		cls.Kind = "CLASS"
	}
	s.WriteClass(ctx, cls)
}

// ── NEXUS-PERF-001 ────────────────────────────────────

func TestNPlusOneQueryRisk_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "get_for_each_user",
		Language: "python", Visibility: "PUBLIC",
		StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "get_all_users",
		Language: "python", Visibility: "PUBLIC",
		StartLine: 20, Checksum: "c2",
	})

	rule := performance.NewNPlusOneQueryRisk()
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
	t.Logf("✅ NPlusOneQueryRisk: detected %s", findings[0].NodeID)
}

func TestNPlusOneQueryRisk_NoFindings(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "get_all_users",
		Language: "python", Visibility: "PUBLIC",
		StartLine: 10, Checksum: "c1",
	})

	rule := performance.NewNPlusOneQueryRisk()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ NPlusOneQueryRisk: no findings for safe function")
}

func TestNPlusOneQueryRisk_RuleID(t *testing.T) {
	rule := performance.NewNPlusOneQueryRisk()
	if rule.ID() != "NEXUS-PERF-001" {
		t.Errorf("Expected NEXUS-PERF-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-PERF-002 ────────────────────────────────────

func TestHighFanOut_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "low_fanout",
		Language: "python", Visibility: "PUBLIC",
		FanOut: 3, StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "high_fanout",
		Language: "python", Visibility: "PUBLIC",
		FanOut: 15, StartLine: 20, Checksum: "c2",
	})

	rule := performance.NewHighFanOut(10)
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
	t.Logf("✅ HighFanOut: detected %s", findings[0].NodeID)
}

func TestHighFanOut_RuleID(t *testing.T) {
	rule := performance.NewHighFanOut(10)
	if rule.ID() != "NEXUS-PERF-002" {
		t.Errorf("Expected NEXUS-PERF-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-PERF-003 ────────────────────────────────────

func TestHighCouplingBetweenObjects_Detects(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, store.Class{
		ID: "cls-001", Name: "LowCoupling",
		Language: "python", Visibility: "PUBLIC",
		CouplingBetweenObjects: 3,
		StartLine:              5, Checksum: "c1",
	})
	writeClass(t, s, store.Class{
		ID: "cls-002", Name: "HighCoupling",
		Language: "python", Visibility: "PUBLIC",
		CouplingBetweenObjects: 15,
		StartLine:              50, Checksum: "c2",
	})

	rule := performance.NewHighCouplingBetweenObjects(10)
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
	t.Logf("✅ HighCouplingBetweenObjects: detected %s", findings[0].NodeID)
}

func TestHighCouplingBetweenObjects_RuleID(t *testing.T) {
	rule := performance.NewHighCouplingBetweenObjects(10)
	if rule.ID() != "NEXUS-PERF-003" {
		t.Errorf("Expected NEXUS-PERF-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-PERF-004 ────────────────────────────────────

func TestLowCohesion_Detects(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, store.Class{
		ID: "cls-001", Name: "Cohesive",
		Language: "python", Visibility: "PUBLIC",
		LackOfCohesion: 3,
		StartLine:      5, Checksum: "c1",
	})
	writeClass(t, s, store.Class{
		ID: "cls-002", Name: "Scattered",
		Language: "python", Visibility: "PUBLIC",
		LackOfCohesion: 15,
		StartLine:      50, Checksum: "c2",
	})

	rule := performance.NewLowCohesion(10)
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
	t.Logf("✅ LowCohesion: detected %s", findings[0].NodeID)
}

func TestLowCohesion_RuleID(t *testing.T) {
	rule := performance.NewLowCohesion(10)
	if rule.ID() != "NEXUS-PERF-004" {
		t.Errorf("Expected NEXUS-PERF-004, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-PERF-005 ────────────────────────────────────

func TestBlockingIOInHotPath_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "sync_read_file",
		Language: "python", Visibility: "PUBLIC",
		StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "read_file_async",
		Language: "python", Visibility: "PUBLIC",
		StartLine: 20, Checksum: "c2",
	})

	rule := performance.NewBlockingIOInHotPath()
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
	t.Logf("✅ BlockingIOInHotPath: detected %s", findings[0].NodeID)
}

func TestBlockingIOInHotPath_RuleID(t *testing.T) {
	rule := performance.NewBlockingIOInHotPath()
	if rule.ID() != "NEXUS-PERF-005" {
		t.Errorf("Expected NEXUS-PERF-005, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── DEFAULT RULES ─────────────────────────────────────

func TestDefaultPerformanceRules_Count(t *testing.T) {
	r := performance.DefaultPerformanceRules()
	if len(r) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(r))
	}
	ids := []string{
		"NEXUS-PERF-001", "NEXUS-PERF-002", "NEXUS-PERF-003",
		"NEXUS-PERF-004", "NEXUS-PERF-005",
	}
	for i, rule := range r {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultPerformanceRules: %d rules", len(r))
}
