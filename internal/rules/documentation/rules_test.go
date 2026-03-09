package docrules_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/store"
	documentation "github.com/hackersfun369/nexus/internal/rules/documentation"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-doc-test-*")
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

func writeFunction(t *testing.T, s store.GraphStore, id, name, visibility, docComment string, isConstructor bool) {
	t.Helper()
	s.WriteFunction(ctx, store.Function{
		ID: id, ProjectID: "proj-001", ModuleID: "mod-001",
		Name: name, QualifiedName: name,
		Language: "python", Visibility: visibility,
		Parameters: "[]", ReturnType: "{}", Annotations: "[]",
		DocComment:    docComment,
		IsConstructor: isConstructor,
		StartLine:     10,
		Checksum:      "chk-" + id,
	})
}

func writeClass(t *testing.T, s store.GraphStore, id, name, visibility, docComment string) {
	t.Helper()
	s.WriteClass(ctx, store.Class{
		ID: id, ProjectID: "proj-001", ModuleID: "mod-001",
		Name: name, QualifiedName: name,
		Language: "python", Kind: "CLASS",
		Visibility:  visibility,
		DocComment:  docComment,
		Annotations: "[]",
		StartLine:   5,
		Checksum:    "chk-" + id,
	})
}

func TestMissingPublicFunctionDoc_Detects(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "undocumented", "PUBLIC", "", false)
	writeFunction(t, s, "fn-002", "documented", "PUBLIC", "Does something useful.", false)

	rule := documentation.NewMissingPublicFunctionDoc()
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
	t.Logf("✅ MissingPublicFunctionDoc: detected %s", findings[0].NodeID)
}

func TestMissingPublicFunctionDoc_IgnoresPrivate(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "_private", "PRIVATE", "", false)

	rule := documentation.NewMissingPublicFunctionDoc()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for private function, got %d", len(findings))
	}
	t.Logf("✅ MissingPublicFunctionDoc: ignores private")
}

func TestMissingPublicFunctionDoc_IgnoresConstructors(t *testing.T) {
	s := newTestStore(t)
	writeFunction(t, s, "fn-001", "__init__", "PUBLIC", "", true)

	rule := documentation.NewMissingPublicFunctionDoc()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for constructor, got %d", len(findings))
	}
	t.Logf("✅ MissingPublicFunctionDoc: ignores constructors")
}

func TestMissingPublicFunctionDoc_RuleID(t *testing.T) {
	rule := documentation.NewMissingPublicFunctionDoc()
	if rule.ID() != "NEXUS-DOC-001" {
		t.Errorf("Expected NEXUS-DOC-001, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

func TestMissingPublicClassDoc_Detects(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, "cls-001", "Undocumented", "PUBLIC", "")
	writeClass(t, s, "cls-002", "Documented", "PUBLIC", "A well documented class.")

	rule := documentation.NewMissingPublicClassDoc()
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
	t.Logf("✅ MissingPublicClassDoc: detected %s", findings[0].NodeID)
}

func TestMissingPublicClassDoc_IgnoresPrivate(t *testing.T) {
	s := newTestStore(t)
	writeClass(t, s, "cls-001", "_Private", "PRIVATE", "")

	rule := documentation.NewMissingPublicClassDoc()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("Expected 0 findings for private class, got %d", len(findings))
	}
	t.Logf("✅ MissingPublicClassDoc: ignores private")
}

func TestMissingPublicClassDoc_RuleID(t *testing.T) {
	rule := documentation.NewMissingPublicClassDoc()
	if rule.ID() != "NEXUS-DOC-002" {
		t.Errorf("Expected NEXUS-DOC-002, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

func TestMissingModuleDoc_Detects(t *testing.T) {
	s := newTestStore(t)

	rule := documentation.NewMissingModuleDoc()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(findings))
	}
	t.Logf("✅ MissingModuleDoc: detected %s", findings[0].FilePath)
}

func TestMissingModuleDoc_IgnoresSmallFiles(t *testing.T) {
	s := newTestStore(t)
	s.WriteModule(ctx, store.Module{
		ID: "mod-002", ProjectID: "proj-001",
		FilePath: "src/tiny.py", QualifiedName: "tiny",
		Language: "python", ParseStatus: "OK",
		ParseErrors: "[]", LinesOfCode: 5,
		Checksum: "xyz",
	})

	rule := documentation.NewMissingModuleDoc()
	findings, err := rule.Analyze(ctx, "proj-001", s)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	for _, f := range findings {
		if f.FilePath == "src/tiny.py" {
			t.Error("Should not flag tiny module")
		}
	}
	t.Logf("✅ MissingModuleDoc: ignores small files, found %d", len(findings))
}

func TestMissingModuleDoc_RuleID(t *testing.T) {
	rule := documentation.NewMissingModuleDoc()
	if rule.ID() != "NEXUS-DOC-003" {
		t.Errorf("Expected NEXUS-DOC-003, got %s", rule.ID())
	}
	t.Logf("✅ Rule ID: %s", rule.ID())
}

func TestDefaultDocumentationRules_Count(t *testing.T) {
	r := documentation.DefaultDocumentationRules()
	if len(r) != 3 {
		t.Errorf("Expected 3 default rules, got %d", len(r))
	}
	ids := []string{"NEXUS-DOC-001", "NEXUS-DOC-002", "NEXUS-DOC-003"}
	for i, rule := range r {
		if rule.ID() != ids[i] {
			t.Errorf("Expected %s, got %s", ids[i], rule.ID())
		}
	}
	t.Logf("✅ DefaultDocumentationRules: %d rules", len(r))
}
