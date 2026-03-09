package architecture_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules/architecture"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-arch-test-*")
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

func writeClass(t *testing.T, s store.GraphStore, cls store.Class) {
	t.Helper()
	cls.ProjectID = "proj-001"
	if cls.ModuleID == "" {
		cls.ModuleID = "mod-001"
	}
	if cls.Annotations == "" {
		cls.Annotations = "[]"
	}
	if cls.Kind == "" {
		cls.Kind = "CLASS"
	}
	s.WriteClass(ctx, cls)
}

// ── NEXUS-ARCH-001 ────────────────────────────────────

func TestCircularDependencyRisk_Detects(t *testing.T) {
	s := newTestStore(t)
	s.WriteModule(ctx, store.Module{
		ID: "mod-002", ProjectID: "proj-001",
		FilePath: "src/b.py", QualifiedName: "b",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", CycleRisk: 0.8,
		Checksum: "xyz",
	})

	rule := architecture.NewCircularDependencyRisk()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].NodeID != "mod-002" {
		t.Errorf("Expected mod-002, got %s", findings[0].NodeID)
	}
	t.Logf("✅ CircularDependencyRisk: detected %s", findings[0].NodeID)
}

func TestCircularDependencyRisk_NoFindings(t *testing.T) {
	s := newTestStore(t)
	// mod-001 has CycleRisk=0 (default)

	rule := architecture.NewCircularDependencyRisk()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ CircularDependencyRisk: no findings for clean module")
}

func TestCircularDependencyRisk_RuleID(t *testing.T) {
	rule := architecture.NewCircularDependencyRisk()
	if rule.ID() != "NEXUS-ARCH-001" {
		t.Errorf("Expected NEXUS-ARCH-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-ARCH-002 ────────────────────────────────────

func TestAbstractClassWithoutSubclasses_Detects(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, store.Class{
		ID: "cls-001", Name: "AbstractRepository",
		Language: "python", Visibility: "PUBLIC",
		IsAbstract: true, StartLine: 5, Checksum: "c1",
	})

	rule := architecture.NewAbstractClassWithoutSubclasses()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	t.Logf("✅ AbstractClassWithoutSubclasses: detected %s", findings[0].NodeID)
}

func TestAbstractClassWithoutSubclasses_IgnoresConcrete(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, store.Class{
		ID: "cls-001", Name: "UserService",
		Language: "python", Visibility: "PUBLIC",
		IsAbstract: false, StartLine: 5, Checksum: "c1",
	})

	rule := architecture.NewAbstractClassWithoutSubclasses()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for concrete class, got %d", len(findings))
	}
	t.Logf("✅ AbstractClassWithoutSubclasses: ignores concrete class")
}

func TestAbstractClassWithoutSubclasses_RuleID(t *testing.T) {
	rule := architecture.NewAbstractClassWithoutSubclasses()
	if rule.ID() != "NEXUS-ARCH-002" {
		t.Errorf("Expected NEXUS-ARCH-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-ARCH-003 ────────────────────────────────────

func TestStaticMethodOveruse_Detects(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 7; i++ {
		writeFunction(t, s, store.Function{
			ID:         fmt.Sprintf("fn-%03d", i),
			Name:       fmt.Sprintf("static_fn_%d", i),
			Language:   "python",
			Visibility: "PUBLIC",
			IsStatic:   true,
			StartLine:  i * 10,
			Checksum:   fmt.Sprintf("c%d", i),
		})
	}

	rule := architecture.NewStaticMethodOveruse(5)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	t.Logf("✅ StaticMethodOveruse: detected module with %d static methods", 7)
}

func TestStaticMethodOveruse_NoFindings(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "static_fn",
		Language: "python", Visibility: "PUBLIC",
		IsStatic: true, StartLine: 10, Checksum: "c1",
	})

	rule := architecture.NewStaticMethodOveruse(5)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings, got %d", len(findings))
	}
	t.Logf("✅ StaticMethodOveruse: no findings for low static count")
}

func TestStaticMethodOveruse_RuleID(t *testing.T) {
	rule := architecture.NewStaticMethodOveruse(5)
	if rule.ID() != "NEXUS-ARCH-003" {
		t.Errorf("Expected NEXUS-ARCH-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-ARCH-004 ────────────────────────────────────

func TestAsyncMethodInSyncContext_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "async_handler",
		Language: "python", Visibility: "PUBLIC",
		IsAsync: true, StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "sync_handler",
		Language: "python", Visibility: "PUBLIC",
		IsAsync: false, StartLine: 20, Checksum: "c2",
	})

	rule := architecture.NewAsyncMethodInSyncContext()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	t.Logf("✅ AsyncMethodInSyncContext: detected %s", findings[0].NodeID)
}

func TestAsyncMethodInSyncContext_NoFindings_AllAsync(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, store.Function{
		ID: "fn-001", Name: "async_a",
		Language: "python", Visibility: "PUBLIC",
		IsAsync: true, StartLine: 10, Checksum: "c1",
	})
	writeFunction(t, s, store.Function{
		ID: "fn-002", Name: "async_b",
		Language: "python", Visibility: "PUBLIC",
		IsAsync: true, StartLine: 20, Checksum: "c2",
	})

	rule := architecture.NewAsyncMethodInSyncContext()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for all-async module, got %d", len(findings))
	}
	t.Logf("✅ AsyncMethodInSyncContext: no findings for all-async module")
}

func TestAsyncMethodInSyncContext_RuleID(t *testing.T) {
	rule := architecture.NewAsyncMethodInSyncContext()
	if rule.ID() != "NEXUS-ARCH-004" {
		t.Errorf("Expected NEXUS-ARCH-004, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── NEXUS-ARCH-005 ────────────────────────────────────

func TestInterfaceSegregationViolation_Detects(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, store.Class{
		ID: "cls-001", Name: "BigInterface",
		Language: "python", Kind: "INTERFACE",
		Visibility: "PUBLIC", MethodCount: 10,
		StartLine: 5, Checksum: "c1",
	})
	writeClass(t, s, store.Class{
		ID: "cls-002", Name: "SmallInterface",
		Language: "python", Kind: "INTERFACE",
		Visibility: "PUBLIC", MethodCount: 3,
		StartLine: 50, Checksum: "c2",
	})

	rule := architecture.NewInterfaceSegregationViolation(7)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	if findings[0].NodeID != "cls-001" {
		t.Errorf("Expected cls-001, got %s", findings[0].NodeID)
	}
	t.Logf("✅ InterfaceSegregationViolation: detected %s", findings[0].NodeID)
}

func TestInterfaceSegregationViolation_IgnoresConcreteClass(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, store.Class{
		ID: "cls-001", Name: "BigClass",
		Language: "python", Kind: "CLASS",
		Visibility: "PUBLIC", MethodCount: 10,
		IsAbstract: false,
		StartLine:  5, Checksum: "c1",
	})

	rule := architecture.NewInterfaceSegregationViolation(7)
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for concrete class, got %d", len(findings))
	}
	t.Logf("✅ InterfaceSegregationViolation: ignores concrete class")
}

func TestInterfaceSegregationViolation_RuleID(t *testing.T) {
	rule := architecture.NewInterfaceSegregationViolation(7)
	if rule.ID() != "NEXUS-ARCH-005" {
		t.Errorf("Expected NEXUS-ARCH-005, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

// ── DEFAULT RULES ─────────────────────────────────────

func TestDefaultArchitectureRules_Count(t *testing.T) {
	r := architecture.DefaultArchitectureRules()
	if len(r) != 5 {
		t.Errorf("Expected 5 default rules, got %d", len(r))
	}
	ids := []string{
		"NEXUS-ARCH-001", "NEXUS-ARCH-002", "NEXUS-ARCH-003",
		"NEXUS-ARCH-004", "NEXUS-ARCH-005",
	}
	for i, rule := range r {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultArchitectureRules: %d rules", len(r))
}
