package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/store"
)

var ctx = context.Background()

func newTestStore(t *testing.T) store.GraphStore {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	s, err := store.NewSQLiteStore(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeProject(id, name string) store.Project {
	return store.Project{
		ID: id, Name: name,
		RootPath:        "/home/user/" + name,
		Platform:        "web-react",
		PrimaryLanguage: "typescript",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Version:         "0.1.0",
	}
}

func makeModule(id, projectID, filePath, language string) store.Module {
	return store.Module{
		ID: id, ProjectID: projectID,
		FilePath:      filePath,
		QualifiedName: filePath,
		Language:      language,
		LinesOfCode:   100,
		ParseStatus:   "OK",
		ParseErrors:   "[]",
		Checksum:      "abc123",
	}
}

func makeFunction(id, projectID, moduleID, name string) store.Function {
	return store.Function{
		ID: id, ProjectID: projectID, ModuleID: moduleID,
		Name: name, QualifiedName: name,
		Language:  "python",
		StartLine: 10, EndLine: 20,
		Visibility: "PUBLIC",
		Parameters: "[]", ReturnType: "{}",
		Annotations:          "[]",
		CyclomaticComplexity: 3,
		LinesOfCode:          10,
		Checksum:             "fn-checksum-" + id,
	}
}

func makeClass(id, projectID, moduleID, name string) store.Class {
	return store.Class{
		ID: id, ProjectID: projectID, ModuleID: moduleID,
		Name: name, QualifiedName: name,
		Language:  "java",
		Kind:      "CLASS",
		StartLine: 5, EndLine: 50,
		Visibility:  "PUBLIC",
		Annotations: "[]",
		Checksum:    "cls-checksum-" + id,
	}
}

func makeIssue(id, projectID, nodeID string) store.Issue {
	return store.Issue{
		ID: id, NodeID: nodeID, ProjectID: projectID,
		RuleID:         "NEXUS-SEC-001",
		Severity:       "CRITICAL",
		Category:       "SECURITY",
		Title:          "SQL Injection",
		Description:    "Unsanitized input in SQL query",
		FilePath:       "src/db.py",
		StartLine:      42,
		Evidence:       "query = sql + user_input",
		Remediation:    "Use parameterized queries",
		InferenceChain: "[]",
		Status:         store.IssueStatusOpen,
		DetectedAt:     time.Now(),
	}
}

func TestCreateAndGetProject(t *testing.T) {
	s := newTestStore(t)
	p := makeProject("proj-001", "my-app")
	if err := s.CreateProject(ctx, p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	got, err := s.GetProject(ctx, "proj-001")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "my-app" {
		t.Errorf("Expected my-app, got %s", got.Name)
	}
	t.Logf("✅ CreateAndGetProject: %s", got.Name)
}

func TestGetProject_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetProject(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected NotFoundError")
	}
	t.Logf("✅ GetProject NotFound: %v", err)
}

func TestUpdateProject(t *testing.T) {
	s := newTestStore(t)
	p := makeProject("proj-001", "my-app")
	s.CreateProject(ctx, p)
	p.Name = "updated-app"
	if err := s.UpdateProject(ctx, p); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	got, _ := s.GetProject(ctx, "proj-001")
	if got.Name != "updated-app" {
		t.Errorf("Expected updated-app, got %s", got.Name)
	}
	t.Logf("✅ UpdateProject: %s", got.Name)
}

func TestDeleteProject(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "my-app"))
	if err := s.DeleteProject(ctx, "proj-001"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	_, err := s.GetProject(ctx, "proj-001")
	if err == nil {
		t.Error("Expected project to be deleted")
	}
	t.Logf("✅ DeleteProject: project deleted")
}

func TestWriteAndGetModule(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	m := makeModule("mod-001", "proj-001", "src/service.py", "python")
	if err := s.WriteModule(ctx, m); err != nil {
		t.Fatalf("WriteModule: %v", err)
	}
	got, err := s.GetModule(ctx, "mod-001")
	if err != nil {
		t.Fatalf("GetModule: %v", err)
	}
	if got.FilePath != "src/service.py" {
		t.Errorf("Expected src/service.py, got %s", got.FilePath)
	}
	t.Logf("✅ WriteAndGetModule: %s", got.FilePath)
}

func TestGetModuleByPath(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/service.py", "python"))
	got, err := s.GetModuleByPath(ctx, "proj-001", "src/service.py")
	if err != nil {
		t.Fatalf("GetModuleByPath: %v", err)
	}
	if got.ID != "mod-001" {
		t.Errorf("Expected mod-001, got %s", got.ID)
	}
	t.Logf("✅ GetModuleByPath: %s", got.FilePath)
}

func TestQueryModules(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	s.WriteModule(ctx, makeModule("mod-002", "proj-001", "src/b.py", "python"))
	s.WriteModule(ctx, makeModule("mod-003", "proj-001", "src/c.ts", "typescript"))
	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryModules: %v", err)
	}
	if len(modules) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(modules))
	}
	t.Logf("✅ QueryModules: %d found", len(modules))
}

func TestQueryModules_FilterByLanguage(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	s.WriteModule(ctx, makeModule("mod-002", "proj-001", "src/b.ts", "typescript"))
	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001", Language: "python"})
	if err != nil {
		t.Fatalf("QueryModules: %v", err)
	}
	if len(modules) != 1 {
		t.Errorf("Expected 1 python module, got %d", len(modules))
	}
	t.Logf("✅ QueryModules FilterByLanguage: %d found", len(modules))
}

func TestWriteAndGetFunction(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	f := makeFunction("fn-001", "proj-001", "mod-001", "greet")
	if err := s.WriteFunction(ctx, f); err != nil {
		t.Fatalf("WriteFunction: %v", err)
	}
	got, err := s.GetFunction(ctx, "fn-001")
	if err != nil {
		t.Fatalf("GetFunction: %v", err)
	}
	if got.Name != "greet" {
		t.Errorf("Expected greet, got %s", got.Name)
	}
	if got.CyclomaticComplexity != 3 {
		t.Errorf("Expected complexity 3, got %d", got.CyclomaticComplexity)
	}
	t.Logf("✅ WriteAndGetFunction: %s complexity=%d", got.Name, got.CyclomaticComplexity)
}

func TestQueryFunctions_MinComplexity(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	f1 := makeFunction("fn-001", "proj-001", "mod-001", "simple")
	f1.CyclomaticComplexity = 1
	f2 := makeFunction("fn-002", "proj-001", "mod-001", "complex")
	f2.CyclomaticComplexity = 10
	s.WriteFunction(ctx, f1)
	s.WriteFunction(ctx, f2)
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: "proj-001", MinComplexity: 5})
	if err != nil {
		t.Fatalf("QueryFunctions: %v", err)
	}
	if len(fns) != 1 {
		t.Errorf("Expected 1 complex function, got %d", len(fns))
	}
	t.Logf("✅ QueryFunctions MinComplexity: %d found", len(fns))
}

func TestWriteAndGetClass(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.java", "java"))
	c := makeClass("cls-001", "proj-001", "mod-001", "UserService")
	if err := s.WriteClass(ctx, c); err != nil {
		t.Fatalf("WriteClass: %v", err)
	}
	got, err := s.GetClass(ctx, "cls-001")
	if err != nil {
		t.Fatalf("GetClass: %v", err)
	}
	if got.Name != "UserService" {
		t.Errorf("Expected UserService, got %s", got.Name)
	}
	t.Logf("✅ WriteAndGetClass: %s", got.Name)
}

func TestWriteAndGetIssue(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	i := makeIssue("issue-001", "proj-001", "mod-001")
	if err := s.WriteIssue(ctx, i); err != nil {
		t.Fatalf("WriteIssue: %v", err)
	}
	got, err := s.GetIssue(ctx, "issue-001")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if got.Severity != "CRITICAL" {
		t.Errorf("Expected CRITICAL, got %s", got.Severity)
	}
	t.Logf("✅ WriteAndGetIssue: %s %s", got.RuleID, got.Severity)
}

func TestQueryIssues_BySeverity(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	i1 := makeIssue("issue-001", "proj-001", "mod-001")
	i1.Severity = "CRITICAL"
	i2 := makeIssue("issue-002", "proj-001", "mod-001")
	i2.Severity = "LOW"
	s.WriteIssue(ctx, i1)
	s.WriteIssue(ctx, i2)
	issues, err := s.QueryIssues(ctx, store.IssueFilter{ProjectID: "proj-001", Severity: "CRITICAL"})
	if err != nil {
		t.Fatalf("QueryIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("Expected 1 critical issue, got %d", len(issues))
	}
	t.Logf("✅ QueryIssues BySeverity: %d found", len(issues))
}

func TestUpdateIssueStatus(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	s.WriteIssue(ctx, makeIssue("issue-001", "proj-001", "mod-001"))
	if err := s.UpdateIssueStatus(ctx, "issue-001", store.IssueStatusResolved); err != nil {
		t.Fatalf("UpdateIssueStatus: %v", err)
	}
	got, _ := s.GetIssue(ctx, "issue-001")
	if got.Status != store.IssueStatusResolved {
		t.Errorf("Expected RESOLVED, got %s", got.Status)
	}
	t.Logf("✅ UpdateIssueStatus: %s", got.Status)
}

func TestWriteAndGetEdge(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	s.WriteModule(ctx, makeModule("mod-002", "proj-001", "src/b.py", "python"))
	e := store.Edge{
		ID: "edge-001", ProjectID: "proj-001",
		Kind:       "IMPORTS",
		FromNodeID: "mod-001", ToNodeID: "mod-002",
		Properties: "{}",
	}
	if err := s.WriteEdge(ctx, e); err != nil {
		t.Fatalf("WriteEdge: %v", err)
	}
	got, err := s.GetEdge(ctx, "edge-001")
	if err != nil {
		t.Fatalf("GetEdge: %v", err)
	}
	if got.Kind != "IMPORTS" {
		t.Errorf("Expected IMPORTS, got %s", got.Kind)
	}
	t.Logf("✅ WriteAndGetEdge: %s %s->%s", got.Kind, got.FromNodeID, got.ToNodeID)
}

func TestQueryEdges_ByFromNode(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	s.WriteModule(ctx, makeModule("mod-002", "proj-001", "src/b.py", "python"))
	s.WriteModule(ctx, makeModule("mod-003", "proj-001", "src/c.py", "python"))
	s.WriteEdge(ctx, store.Edge{ID: "e1", ProjectID: "proj-001", Kind: "IMPORTS", FromNodeID: "mod-001", ToNodeID: "mod-002", Properties: "{}"})
	s.WriteEdge(ctx, store.Edge{ID: "e2", ProjectID: "proj-001", Kind: "IMPORTS", FromNodeID: "mod-001", ToNodeID: "mod-003", Properties: "{}"})
	edges, err := s.QueryEdges(ctx, store.EdgeFilter{ProjectID: "proj-001", FromNodeID: "mod-001"})
	if err != nil {
		t.Fatalf("QueryEdges: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("Expected 2 edges, got %d", len(edges))
	}
	t.Logf("✅ QueryEdges ByFromNode: %d found", len(edges))
}

func TestDeleteEdge(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	s.WriteModule(ctx, makeModule("mod-002", "proj-001", "src/b.py", "python"))
	s.WriteEdge(ctx, store.Edge{ID: "edge-001", ProjectID: "proj-001", Kind: "IMPORTS", FromNodeID: "mod-001", ToNodeID: "mod-002", Properties: "{}"})
	if err := s.DeleteEdge(ctx, "edge-001"); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}
	_, err := s.GetEdge(ctx, "edge-001")
	if err == nil {
		t.Error("Expected edge to be deleted")
	}
	t.Logf("✅ DeleteEdge: edge deleted")
}

func TestGetProjectHealth(t *testing.T) {
	s := newTestStore(t)
	s.CreateProject(ctx, makeProject("proj-001", "app"))
	s.WriteModule(ctx, makeModule("mod-001", "proj-001", "src/a.py", "python"))
	fn := makeFunction("fn-001", "proj-001", "mod-001", "complex")
	fn.CyclomaticComplexity = 20
	s.WriteFunction(ctx, fn)
	i := makeIssue("issue-001", "proj-001", "mod-001")
	i.Severity = "CRITICAL"
	s.WriteIssue(ctx, i)
	health, err := s.GetProjectHealth(ctx, "proj-001")
	if err != nil {
		t.Fatalf("GetProjectHealth: %v", err)
	}
	if health.CriticalIssues != 1 {
		t.Errorf("Expected 1 critical issue, got %d", health.CriticalIssues)
	}
	if health.TotalFunctions != 1 {
		t.Errorf("Expected 1 function, got %d", health.TotalFunctions)
	}
	if health.ComplexFunctions != 1 {
		t.Errorf("Expected 1 complex function, got %d", health.ComplexFunctions)
	}
	t.Logf("✅ GetProjectHealth: critical=%d complex=%d functions=%d",
		health.CriticalIssues, health.ComplexFunctions, health.TotalFunctions)
}
