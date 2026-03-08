package extractor_test

import (
	"testing"

	"github.com/hackersfun369/nexus/internal/parser/extractor"
	"github.com/hackersfun369/nexus/internal/parser/normalizer"
)

// ── HELPERS ───────────────────────────────────────────

func buildDelta(t *testing.T, module normalizer.ASTNode) extractor.GraphDelta {
	t.Helper()
	b := extractor.NewBuilder("proj-001")
	return b.Build(module)
}

func makeModule(filePath string, children ...normalizer.ASTNode) normalizer.ASTNode {
	return normalizer.ASTNode{
		ID:       "mod-001",
		Kind:     normalizer.KindModule,
		Language: normalizer.LangPython,
		Name:     "service",
		Location: normalizer.SourceLocation{
			File:    filePath,
			EndLine: 100,
		},
		Children: children,
	}
}

func makeFunction(id, name string, complexity int) normalizer.ASTNode {
	return normalizer.ASTNode{
		ID:       id,
		Kind:     normalizer.KindFunction,
		Language: normalizer.LangPython,
		Name:     name,
		Location: normalizer.SourceLocation{
			File:      "service.py",
			StartLine: 10,
			EndLine:   20,
		},
		Visibility:           normalizer.VisibilityPublic,
		CyclomaticComplexity: complexity,
		ReturnType:           normalizer.UnknownType,
	}
}

func makeClass(id, name string) normalizer.ASTNode {
	return normalizer.ASTNode{
		ID:       id,
		Kind:     normalizer.KindClass,
		Language: normalizer.LangPython,
		Name:     name,
		Location: normalizer.SourceLocation{
			File:      "service.py",
			StartLine: 5,
			EndLine:   50,
		},
		Visibility: normalizer.VisibilityPublic,
	}
}

// ── BASIC DELTA TESTS ─────────────────────────────────

func TestBuild_EmptyModule_ProducesModuleNode(t *testing.T) {
	module := makeModule("service.py")
	delta := buildDelta(t, module)

	if delta.IsEmpty() {
		t.Fatal("Expected non-empty delta for module")
	}

	if delta.NodesAdded != 1 {
		t.Errorf("Expected 1 added node (module), got %d", delta.NodesAdded)
	}

	node := delta.NodeChanges[0].Node
	if node.Kind != extractor.NodeModule {
		t.Errorf("Expected NodeModule, got %s", node.Kind)
	}
	if node.FilePath != "service.py" {
		t.Errorf("Expected 'service.py', got '%s'", node.FilePath)
	}
	t.Logf("✅ Empty module: %d nodes added", delta.NodesAdded)
}

func TestBuild_WithFunction_ProducesFunctionNode(t *testing.T) {
	fn := makeFunction("fn-001", "greet", 1)
	module := makeModule("service.py", fn)
	delta := buildDelta(t, module)

	// Should have: 1 module + 1 function = 2 nodes
	if delta.NodesAdded != 2 {
		t.Errorf("Expected 2 added nodes, got %d", delta.NodesAdded)
	}

	// Find function node
	var fnNode *extractor.GraphNode
	for _, change := range delta.NodeChanges {
		if change.Node.Kind == extractor.NodeFunction {
			fnNode = &change.Node
			break
		}
	}

	if fnNode == nil {
		t.Fatal("Expected function node in delta")
	}
	if fnNode.Name != "greet" {
		t.Errorf("Expected 'greet', got '%s'", fnNode.Name)
	}
	t.Logf("✅ Function node: name=%s kind=%s", fnNode.Name, fnNode.Kind)
}

func TestBuild_WithClass_ProducesClassNode(t *testing.T) {
	cls := makeClass("cls-001", "UserService")
	module := makeModule("service.py", cls)
	delta := buildDelta(t, module)

	var clsNode *extractor.GraphNode
	for _, change := range delta.NodeChanges {
		if change.Node.Kind == extractor.NodeClass {
			clsNode = &change.Node
			break
		}
	}

	if clsNode == nil {
		t.Fatal("Expected class node in delta")
	}
	if clsNode.Name != "UserService" {
		t.Errorf("Expected 'UserService', got '%s'", clsNode.Name)
	}
	t.Logf("✅ Class node: name=%s kind=%s", clsNode.Name, clsNode.Kind)
}

func TestBuild_WithClassAndMethods_ProducesAllNodes(t *testing.T) {
	method1 := makeFunction("fn-001", "getData", 2)
	method2 := makeFunction("fn-002", "setData", 1)
	cls := makeClass("cls-001", "Service")
	cls.Children = []normalizer.ASTNode{method1, method2}

	module := makeModule("service.py", cls)
	delta := buildDelta(t, module)

	// module + class + 2 methods = 4 nodes
	if delta.NodesAdded != 4 {
		t.Errorf("Expected 4 nodes, got %d", delta.NodesAdded)
	}
	t.Logf("✅ Class with methods: %d nodes total", delta.NodesAdded)
}

// ── EDGES ─────────────────────────────────────────────

func TestBuild_ContainsEdges_Created(t *testing.T) {
	fn := makeFunction("fn-001", "greet", 1)
	module := makeModule("service.py", fn)
	delta := buildDelta(t, module)

	// Should have CONTAINS edge: module → function
	if delta.EdgesAdded == 0 {
		t.Fatal("Expected CONTAINS edges")
	}

	var containsEdge *extractor.GraphEdge
	for _, change := range delta.EdgeChanges {
		if change.Edge.Kind == extractor.EdgeContains {
			containsEdge = &change.Edge
			break
		}
	}

	if containsEdge == nil {
		t.Fatal("Expected CONTAINS edge")
	}
	if containsEdge.FromNodeID == "" {
		t.Error("CONTAINS edge missing FromNodeID")
	}
	if containsEdge.ToNodeID == "" {
		t.Error("CONTAINS edge missing ToNodeID")
	}
	t.Logf("✅ CONTAINS edge: %s → %s",
		containsEdge.FromNodeID, containsEdge.ToNodeID)
}

func TestBuild_NestedChildren_AllEdgesCreated(t *testing.T) {
	method1 := makeFunction("fn-001", "getData", 1)
	method2 := makeFunction("fn-002", "setData", 1)
	cls := makeClass("cls-001", "Service")
	cls.Children = []normalizer.ASTNode{method1, method2}

	module := makeModule("service.py", cls)
	delta := buildDelta(t, module)

	// Edges: module→class, class→method1, class→method2 = 3
	if delta.EdgesAdded < 3 {
		t.Errorf("Expected >= 3 edges, got %d", delta.EdgesAdded)
	}
	t.Logf("✅ Nested edges: %d total", delta.EdgesAdded)
}

// ── CHECKSUM AND CHANGE DETECTION ─────────────────────

func TestBuild_SameContent_NoChangeInDelta(t *testing.T) {
	fn := makeFunction("fn-001", "greet", 1)
	module := makeModule("service.py", fn)

	// First build — get checksums
	b1 := extractor.NewBuilder("proj-001")
	delta1 := b1.Build(module)

	// Collect checksums from first build
	checksums := map[string]string{}
	for _, change := range delta1.NodeChanges {
		checksums[change.Node.ID] = change.Node.Checksum
	}

	// Second build with same content + existing checksums
	b2 := extractor.NewBuilder("proj-001").
		WithExistingChecksums(checksums)
	delta2 := b2.Build(module)

	// Nothing changed — no additions or modifications
	if delta2.NodesAdded > 0 {
		t.Errorf("Expected 0 additions, got %d", delta2.NodesAdded)
	}
	if delta2.NodesModified > 0 {
		t.Errorf("Expected 0 modifications, got %d", delta2.NodesModified)
	}
	t.Logf("✅ Unchanged content: delta is empty (no changes)")
}

func TestBuild_ChangedFunction_ProducesModify(t *testing.T) {
	fn := makeFunction("fn-001", "greet", 1)
	module := makeModule("service.py", fn)

	// First build
	b1 := extractor.NewBuilder("proj-001")
	delta1 := b1.Build(module)

	checksums := map[string]string{}
	for _, change := range delta1.NodeChanges {
		checksums[change.Node.ID] = change.Node.Checksum
	}

	// Modify function complexity
	fnModified := makeFunction("fn-001", "greet", 5) // complexity changed
	moduleModified := makeModule("service.py", fnModified)

	b2 := extractor.NewBuilder("proj-001").
		WithExistingChecksums(checksums)
	delta2 := b2.Build(moduleModified)

	if delta2.NodesModified == 0 {
		t.Error("Expected at least 1 modification")
	}
	t.Logf("✅ Changed function: %d modifications detected", delta2.NodesModified)
}

func TestBuild_DeletedFunction_ProducesDelete(t *testing.T) {
	fn := makeFunction("fn-001", "greet", 1)
	module := makeModule("service.py", fn)

	// First build — has function
	b1 := extractor.NewBuilder("proj-001")
	delta1 := b1.Build(module)

	checksums := map[string]string{}
	for _, change := range delta1.NodeChanges {
		checksums[change.Node.ID] = change.Node.Checksum
	}

	// Second build — function removed
	moduleWithoutFn := makeModule("service.py") // no function
	b2 := extractor.NewBuilder("proj-001").
		WithExistingChecksums(checksums)
	delta2 := b2.Build(moduleWithoutFn)

	if delta2.NodesDeleted == 0 {
		t.Error("Expected at least 1 deletion")
	}
	t.Logf("✅ Deleted function: %d deletions detected", delta2.NodesDeleted)
}

// ── PROPERTIES ────────────────────────────────────────

func TestBuild_FunctionProperties_Populated(t *testing.T) {
	fn := normalizer.ASTNode{
		ID:                   "fn-001",
		Kind:                 normalizer.KindFunction,
		Language:             normalizer.LangPython,
		Name:                 "process",
		Location:             normalizer.SourceLocation{File: "service.py", StartLine: 5, EndLine: 20},
		Visibility:           normalizer.VisibilityPublic,
		IsAsync:              true,
		IsStatic:             false,
		CyclomaticComplexity: 3,
		LinesOfCode:          15,
		ReturnType: normalizer.TypeDescriptor{
			Kind: "PRIMITIVE",
			Name: "str",
		},
		Parameters: []normalizer.Parameter{
			{Name: "data", Type: normalizer.TypeDescriptor{Kind: "PRIMITIVE", Name: "str"}},
		},
	}

	module := makeModule("service.py", fn)
	delta := buildDelta(t, module)

	var fnNode *extractor.GraphNode
	for _, change := range delta.NodeChanges {
		if change.Node.Kind == extractor.NodeFunction {
			fnNode = &change.Node
			break
		}
	}

	if fnNode == nil {
		t.Fatal("Function node not found")
	}

	props := fnNode.Properties
	if props["isAsync"] != true {
		t.Errorf("Expected isAsync=true, got %v", props["isAsync"])
	}
	if props["cyclomaticComplexity"] != 3 {
		t.Errorf("Expected complexity=3, got %v", props["cyclomaticComplexity"])
	}
	if props["parameterCount"] != 1 {
		t.Errorf("Expected parameterCount=1, got %v", props["parameterCount"])
	}
	if props["returnTypeName"] != "str" {
		t.Errorf("Expected returnTypeName=str, got %v", props["returnTypeName"])
	}
	t.Logf("✅ Function properties: async=%v complexity=%v params=%v return=%v",
		props["isAsync"],
		props["cyclomaticComplexity"],
		props["parameterCount"],
		props["returnTypeName"])
}

// ── DELTA SUMMARY ─────────────────────────────────────

func TestBuild_DeltaSummary_Accurate(t *testing.T) {
	fn1 := makeFunction("fn-001", "greet", 1)
	fn2 := makeFunction("fn-002", "farewell", 2)
	cls := makeClass("cls-001", "Service")
	cls.Children = []normalizer.ASTNode{fn1, fn2}

	module := makeModule("service.py", cls)
	delta := buildDelta(t, module)

	// module + class + 2 functions = 4
	if delta.NodesAdded != 4 {
		t.Errorf("Expected NodesAdded=4, got %d", delta.NodesAdded)
	}
	if delta.NodesModified != 0 {
		t.Errorf("Expected NodesModified=0, got %d", delta.NodesModified)
	}
	if delta.NodesDeleted != 0 {
		t.Errorf("Expected NodesDeleted=0, got %d", delta.NodesDeleted)
	}
	// module→class, class→fn1, class→fn2 = 3
	if delta.EdgesAdded < 3 {
		t.Errorf("Expected EdgesAdded>=3, got %d", delta.EdgesAdded)
	}
	if delta.FilePath != "service.py" {
		t.Errorf("Expected FilePath='service.py', got '%s'", delta.FilePath)
	}

	t.Logf("✅ Delta summary: +%d nodes, ~%d modified, -%d deleted, +%d edges",
		delta.NodesAdded, delta.NodesModified,
		delta.NodesDeleted, delta.EdgesAdded)
}

func TestBuild_IsEmpty_TrueWhenNoChanges(t *testing.T) {
	fn := makeFunction("fn-001", "greet", 1)
	module := makeModule("service.py", fn)

	b1 := extractor.NewBuilder("proj-001")
	delta1 := b1.Build(module)

	checksums := map[string]string{}
	edges := map[string]bool{}
	for _, change := range delta1.NodeChanges {
		checksums[change.Node.ID] = change.Node.Checksum
	}
	for _, change := range delta1.EdgeChanges {
		edges[change.Edge.ID] = true
	}

	// Rebuild with same content and same existing state
	b2 := extractor.NewBuilder("proj-001").
		WithExistingChecksums(checksums).
		WithExistingEdges(edges)
	delta2 := b2.Build(module)

	if !delta2.IsEmpty() {
		t.Errorf("Expected empty delta, got +%d ~%d -%d",
			delta2.NodesAdded, delta2.NodesModified, delta2.NodesDeleted)
	}
	t.Logf("✅ IsEmpty: true when nothing changed")
}
