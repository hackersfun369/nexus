package normalizer_test

import (
	"context"
	"testing"

	"github.com/hackersfun369/nexus/internal/parser/normalizer"
	"github.com/hackersfun369/nexus/internal/parser/treesitter"
)

var ctx = context.Background()

// helper — parse Python source and normalize it
func normalizePython(t *testing.T, source string) normalizer.ASTNode {
	t.Helper()

	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, []byte(source), treesitter.Python)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	n := normalizer.NewPythonNormalizer([]byte(source), "test.py")
	return n.Normalize(result.Tree.RootNode())
}

// ── MODULE ────────────────────────────────────────────

func TestNormalizePython_Module_Kind(t *testing.T) {
	module := normalizePython(t, `x = 1`)

	if module.Kind != normalizer.KindModule {
		t.Errorf("Expected KindModule, got %s", module.Kind)
	}
	if module.Language != normalizer.LangPython {
		t.Errorf("Expected LangPython, got %s", module.Language)
	}
	t.Logf("✅ Module kind correct: %s", module.Kind)
}

// ── FUNCTIONS ─────────────────────────────────────────

func TestNormalizePython_SimpleFunction(t *testing.T) {
	module := normalizePython(t, `
def greet(name):
    return "Hello"
`)
	if len(module.Children) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(module.Children))
	}

	fn := module.Children[0]
	if fn.Kind != normalizer.KindFunction {
		t.Errorf("Expected KindFunction, got %s", fn.Kind)
	}
	if fn.Name != "greet" {
		t.Errorf("Expected name 'greet', got '%s'", fn.Name)
	}
	if len(fn.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].Name != "name" {
		t.Errorf("Expected param 'name', got '%s'", fn.Parameters[0].Name)
	}
	t.Logf("✅ Simple function: name=%s params=%d", fn.Name, len(fn.Parameters))
}

func TestNormalizePython_TypedParameters(t *testing.T) {
	module := normalizePython(t, `
def add(x: int, y: int) -> int:
    return x + y
`)
	fn := module.Children[0]

	if len(fn.Parameters) != 2 {
		t.Fatalf("Expected 2 params, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].Type.Name != "int" {
		t.Errorf("Expected type 'int', got '%s'", fn.Parameters[0].Type.Name)
	}
	if fn.ReturnType.Name != "int" {
		t.Errorf("Expected return type 'int', got '%s'", fn.ReturnType.Name)
	}
	t.Logf("✅ Typed params: x:%s y:%s -> %s",
		fn.Parameters[0].Type.Name,
		fn.Parameters[1].Type.Name,
		fn.ReturnType.Name)
}

func TestNormalizePython_AsyncFunction(t *testing.T) {
	module := normalizePython(t, `
async def fetch_data(url: str) -> str:
    return ""
`)
	fn := module.Children[0]

	if !fn.IsAsync {
		t.Error("Expected IsAsync to be true")
	}
	t.Logf("✅ Async function detected: %s", fn.Name)
}

func TestNormalizePython_SelfParamExcluded(t *testing.T) {
	module := normalizePython(t, `
class MyClass:
    def method(self, value: int):
        pass
`)
	cls := module.Children[0]
	method := cls.Children[0]

	// self should be excluded from parameters
	for _, p := range method.Parameters {
		if p.Name == "self" {
			t.Error("'self' should be excluded from parameters")
		}
	}
	if len(method.Parameters) != 1 {
		t.Errorf("Expected 1 param (excluding self), got %d", len(method.Parameters))
	}
	t.Logf("✅ Self excluded: %d params", len(method.Parameters))
}

func TestNormalizePython_CyclomaticComplexity_Simple(t *testing.T) {
	module := normalizePython(t, `
def simple():
    return 42
`)
	fn := module.Children[0]

	if fn.CyclomaticComplexity != 1 {
		t.Errorf("Expected complexity 1, got %d", fn.CyclomaticComplexity)
	}
	t.Logf("✅ Simple complexity: %d", fn.CyclomaticComplexity)
}

func TestNormalizePython_CyclomaticComplexity_WithIf(t *testing.T) {
	module := normalizePython(t, `
def check(x: int) -> str:
    if x > 0:
        return "positive"
    else:
        return "negative"
`)
	fn := module.Children[0]

	if fn.CyclomaticComplexity < 2 {
		t.Errorf("Expected complexity >= 2, got %d", fn.CyclomaticComplexity)
	}
	t.Logf("✅ If complexity: %d", fn.CyclomaticComplexity)
}

func TestNormalizePython_Visibility_Public(t *testing.T) {
	module := normalizePython(t, `
def public_func():
    pass
`)
	fn := module.Children[0]

	if fn.Visibility != normalizer.VisibilityPublic {
		t.Errorf("Expected PUBLIC, got %s", fn.Visibility)
	}
	t.Logf("✅ Public visibility: %s", fn.Visibility)
}

func TestNormalizePython_Visibility_Private(t *testing.T) {
	module := normalizePython(t, `
def __private_func():
    pass
`)
	fn := module.Children[0]

	if fn.Visibility != normalizer.VisibilityPrivate {
		t.Errorf("Expected PRIVATE, got %s", fn.Visibility)
	}
	t.Logf("✅ Private visibility: %s", fn.Visibility)
}

func TestNormalizePython_Visibility_Internal(t *testing.T) {
	module := normalizePython(t, `
def _internal_func():
    pass
`)
	fn := module.Children[0]

	if fn.Visibility != normalizer.VisibilityInternal {
		t.Errorf("Expected INTERNAL, got %s", fn.Visibility)
	}
	t.Logf("✅ Internal visibility: %s", fn.Visibility)
}

// ── CLASSES ───────────────────────────────────────────

func TestNormalizePython_Class(t *testing.T) {
	module := normalizePython(t, `
class Animal:
    def __init__(self, name: str):
        self.name = name

    def speak(self) -> str:
        return "..."
`)
	if len(module.Children) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(module.Children))
	}

	cls := module.Children[0]
	if cls.Kind != normalizer.KindClass {
		t.Errorf("Expected KindClass, got %s", cls.Kind)
	}
	if cls.Name != "Animal" {
		t.Errorf("Expected name 'Animal', got '%s'", cls.Name)
	}
	t.Logf("✅ Class: %s with %d methods", cls.Name, len(cls.Children))
}

func TestNormalizePython_Constructor_Detected(t *testing.T) {
	module := normalizePython(t, `
class MyClass:
    def __init__(self):
        pass
`)
	cls := module.Children[0]
	init := cls.Children[0]

	if !init.IsConstructor {
		t.Error("Expected __init__ to be marked as constructor")
	}
	t.Logf("✅ Constructor detected: %s", init.Name)
}

func TestNormalizePython_Decorator(t *testing.T) {
	module := normalizePython(t, `
@staticmethod
def my_func():
    pass
`)
	fn := module.Children[0]

	if len(fn.Annotations) == 0 {
		t.Error("Expected decorator in annotations")
	}
	t.Logf("✅ Decorator: %v", fn.Annotations)
}

// ── IMPORTS ───────────────────────────────────────────

func TestNormalizePython_Import(t *testing.T) {
	module := normalizePython(t, `
import os
import sys
`)
	imports := filterByKind(module.Children, normalizer.KindImport)

	if len(imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(imports))
	}
	t.Logf("✅ Imports: %d found", len(imports))
}

// ── HELPERS ───────────────────────────────────────────

func filterByKind(nodes []normalizer.ASTNode, kind normalizer.NodeKind) []normalizer.ASTNode {
	result := []normalizer.ASTNode{}
	for _, n := range nodes {
		if n.Kind == kind {
			result = append(result, n)
		}
	}
	return result
}
