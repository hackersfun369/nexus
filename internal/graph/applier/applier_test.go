package applier_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/graph/applier"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/parser/extractor"
	"github.com/hackersfun369/nexus/internal/parser/normalizer"
)

var ctx = context.Background()

func newTestApplier(t *testing.T) (*applier.Applier, store.GraphStore) {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-applier-test-*")
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
		RootPath: "/tmp/test", Platform: "web",
		PrimaryLanguage: "python",
	})
	return applier.New(s), s
}

func buildDelta(projectID, filePath string, children ...normalizer.ASTNode) extractor.GraphDelta {
	module := normalizer.ASTNode{
		ID:       "mod-" + filePath,
		Kind:     normalizer.KindModule,
		Language: normalizer.LangPython,
		Name:     filePath,
		Location: normalizer.SourceLocation{File: filePath, EndLine: 100},
		Children: children,
	}
	return extractor.NewBuilder(projectID).Build(module)
}

func makeASTFunction(id, name string, complexity int) normalizer.ASTNode {
	return normalizer.ASTNode{
		ID:                   id,
		Kind:                 normalizer.KindFunction,
		Language:             normalizer.LangPython,
		Name:                 name,
		Location:             normalizer.SourceLocation{File: "src/service.py", StartLine: 10, EndLine: 20},
		Visibility:           normalizer.VisibilityPublic,
		CyclomaticComplexity: complexity,
		ReturnType:           normalizer.UnknownType,
	}
}

func makeASTClass(id, name string) normalizer.ASTNode {
	return normalizer.ASTNode{
		ID:         id,
		Kind:       normalizer.KindClass,
		Language:   normalizer.LangPython,
		Name:       name,
		Location:   normalizer.SourceLocation{File: "src/service.py", StartLine: 5, EndLine: 50},
		Visibility: normalizer.VisibilityPublic,
	}
}

func TestApply_EmptyDelta_Skipped(t *testing.T) {
	a, _ := newTestApplier(t)
	delta := extractor.GraphDelta{ProjectID: "proj-001", FilePath: "src/service.py"}
	result, err := a.Apply(ctx, delta)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.Skipped {
		t.Error("Expected Skipped=true for empty delta")
	}
	t.Logf("✅ EmptyDelta skipped")
}

func TestApply_Module_WrittenToStore(t *testing.T) {
	a, s := newTestApplier(t)
	delta := buildDelta("proj-001", "src/service.py")
	result, err := a.Apply(ctx, delta)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.NodesAdded == 0 {
		t.Error("Expected at least 1 node added")
	}
	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryModules: %v", err)
	}
	if len(modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(modules))
	}
	if modules[0].FilePath != "src/service.py" {
		t.Errorf("Expected src/service.py, got %s", modules[0].FilePath)
	}
	t.Logf("✅ Module written: %s", modules[0].FilePath)
}

func TestApply_Function_WrittenToStore(t *testing.T) {
	a, s := newTestApplier(t)
	fn := makeASTFunction("fn-001", "greet", 3)
	delta := buildDelta("proj-001", "src/service.py", fn)
	_, err := a.Apply(ctx, delta)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryFunctions: %v", err)
	}
	if len(fns) != 1 {
		t.Errorf("Expected 1 function, got %d", len(fns))
	}
	if fns[0].Name != "greet" {
		t.Errorf("Expected greet, got %s", fns[0].Name)
	}
	t.Logf("✅ Function written: %s complexity=%d", fns[0].Name, fns[0].CyclomaticComplexity)
}

func TestApply_Class_WrittenToStore(t *testing.T) {
	a, s := newTestApplier(t)
	cls := makeASTClass("cls-001", "UserService")
	delta := buildDelta("proj-001", "src/service.py", cls)
	_, err := a.Apply(ctx, delta)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryClasses: %v", err)
	}
	if len(classes) != 1 {
		t.Errorf("Expected 1 class, got %d", len(classes))
	}
	if classes[0].Name != "UserService" {
		t.Errorf("Expected UserService, got %s", classes[0].Name)
	}
	t.Logf("✅ Class written: %s", classes[0].Name)
}

func TestApply_Edges_WrittenToStore(t *testing.T) {
	a, s := newTestApplier(t)
	fn := makeASTFunction("fn-001", "greet", 1)
	delta := buildDelta("proj-001", "src/service.py", fn)
	result, err := a.Apply(ctx, delta)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.EdgesAdded == 0 {
		t.Error("Expected edges to be added")
	}
	var moduleID string
	for _, change := range delta.NodeChanges {
		if change.Node.Kind == extractor.NodeModule {
			moduleID = change.Node.ID
			break
		}
	}
	edges, err := s.QueryEdges(ctx, store.EdgeFilter{ProjectID: "proj-001", FromNodeID: moduleID})
	if err != nil {
		t.Fatalf("QueryEdges: %v", err)
	}
	if len(edges) == 0 {
		t.Error("Expected CONTAINS edges in store")
	}
	t.Logf("✅ Edges written: %d CONTAINS edges", len(edges))
}

func TestApply_Result_Summary_Accurate(t *testing.T) {
	a, _ := newTestApplier(t)
	fn1 := makeASTFunction("fn-001", "greet", 1)
	fn2 := makeASTFunction("fn-002", "farewell", 2)
	delta := buildDelta("proj-001", "src/service.py", fn1, fn2)
	result, err := a.Apply(ctx, delta)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.NodesAdded < 3 {
		t.Errorf("Expected >= 3 nodes added, got %d", result.NodesAdded)
	}
	if result.NodesModified != 0 {
		t.Errorf("Expected 0 modified, got %d", result.NodesModified)
	}
	if result.EdgesAdded == 0 {
		t.Error("Expected edges added")
	}
	t.Logf("✅ Result summary: +%d nodes, +%d edges", result.NodesAdded, result.EdgesAdded)
}

func TestApply_Idempotent_SameContent(t *testing.T) {
	a, s := newTestApplier(t)
	fn := makeASTFunction("fn-001", "greet", 1)
	delta := buildDelta("proj-001", "src/service.py", fn)
	if _, err := a.Apply(ctx, delta); err != nil {
		t.Fatalf("First apply: %v", err)
	}
	if _, err := a.Apply(ctx, delta); err != nil {
		t.Fatalf("Second apply: %v", err)
	}
	modules, _ := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	fns, _ := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: "proj-001"})
	if len(modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(modules))
	}
	if len(fns) != 1 {
		t.Errorf("Expected 1 function, got %d", len(fns))
	}
	t.Logf("✅ Idempotent: %d modules, %d functions after 2 applies", len(modules), len(fns))
}

func TestApply_MultipleFiles(t *testing.T) {
	a, s := newTestApplier(t)
	delta1 := buildDelta("proj-001", "src/a.py", makeASTFunction("fn-001", "funcA", 1))
	delta2 := buildDelta("proj-001", "src/b.py", makeASTFunction("fn-002", "funcB", 2))
	a.Apply(ctx, delta1)
	a.Apply(ctx, delta2)
	modules, _ := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	fns, _ := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: "proj-001"})
	if len(modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(modules))
	}
	if len(fns) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(fns))
	}
	t.Logf("✅ MultipleFiles: %d modules, %d functions", len(modules), len(fns))
}
