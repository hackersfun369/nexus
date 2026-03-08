package normalizer_test

import (
	"testing"

	"github.com/hackersfun369/nexus/internal/parser/normalizer"
	"github.com/hackersfun369/nexus/internal/parser/treesitter"
)

// helper — parse Java and normalize
func normalizeJava(t *testing.T, source string) normalizer.ASTNode {
	t.Helper()

	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, []byte(source), treesitter.Java)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	n := normalizer.NewJavaNormalizer([]byte(source), "Test.java")
	return n.Normalize(result.Tree.RootNode())
}

// ── MODULE ────────────────────────────────────────────

func TestNormalizeJava_Module_Kind(t *testing.T) {
	module := normalizeJava(t, `
public class Hello {}
`)
	if module.Kind != normalizer.KindModule {
		t.Errorf("Expected KindModule, got %s", module.Kind)
	}
	if module.Language != normalizer.LangJava {
		t.Errorf("Expected LangJava, got %s", module.Language)
	}
	t.Logf("✅ Java module kind: %s", module.Kind)
}

// ── CLASSES ───────────────────────────────────────────

func TestNormalizeJava_SimpleClass(t *testing.T) {
	module := normalizeJava(t, `
public class Animal {
    private String name;

    public Animal(String name) {
        this.name = name;
    }

    public String getName() {
        return this.name;
    }
}
`)
	classes := filterByKind(module.Children, normalizer.KindClass)
	if len(classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(classes))
	}

	cls := classes[0]
	if cls.Name != "Animal" {
		t.Errorf("Expected 'Animal', got '%s'", cls.Name)
	}
	if cls.Visibility != normalizer.VisibilityPublic {
		t.Errorf("Expected PUBLIC, got %s", cls.Visibility)
	}
	t.Logf("✅ Java class: %s visibility=%s children=%d",
		cls.Name, cls.Visibility, len(cls.Children))
}

func TestNormalizeJava_AbstractClass(t *testing.T) {
	module := normalizeJava(t, `
public abstract class Shape {
    public abstract double area();
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]

	if !cls.IsAbstract {
		t.Error("Expected IsAbstract to be true")
	}
	t.Logf("✅ Java abstract class: %s isAbstract=%v", cls.Name, cls.IsAbstract)
}

// ── METHODS ───────────────────────────────────────────

func TestNormalizeJava_Method_ReturnType(t *testing.T) {
	module := normalizeJava(t, `
public class Calculator {
    public int add(int x, int y) {
        return x + y;
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	var addMethod *normalizer.ASTNode
	for i := range methods {
		if methods[i].Name == "add" {
			addMethod = &methods[i]
			break
		}
	}

	if addMethod == nil {
		t.Fatal("add method not found")
	}
	if addMethod.ReturnType.Name != "int" {
		t.Errorf("Expected return type 'int', got '%s'", addMethod.ReturnType.Name)
	}
	if len(addMethod.Parameters) != 2 {
		t.Errorf("Expected 2 params, got %d", len(addMethod.Parameters))
	}
	t.Logf("✅ Java method: %s(%s, %s): %s",
		addMethod.Name,
		addMethod.Parameters[0].Type.Name,
		addMethod.Parameters[1].Type.Name,
		addMethod.ReturnType.Name)
}

func TestNormalizeJava_Method_Visibility(t *testing.T) {
	module := normalizeJava(t, `
public class MyClass {
    public void publicMethod() {}
    private void privateMethod() {}
    protected void protectedMethod() {}
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	vis := map[string]normalizer.Visibility{}
	for _, m := range methods {
		vis[m.Name] = m.Visibility
	}

	if vis["publicMethod"] != normalizer.VisibilityPublic {
		t.Errorf("Expected PUBLIC, got %s", vis["publicMethod"])
	}
	if vis["privateMethod"] != normalizer.VisibilityPrivate {
		t.Errorf("Expected PRIVATE, got %s", vis["privateMethod"])
	}
	if vis["protectedMethod"] != normalizer.VisibilityProtected {
		t.Errorf("Expected PROTECTED, got %s", vis["protectedMethod"])
	}
	t.Logf("✅ Java method visibility: public/private/protected correct")
}

func TestNormalizeJava_StaticMethod(t *testing.T) {
	module := normalizeJava(t, `
public class MathUtil {
    public static int square(int x) {
        return x * x;
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	if len(methods) == 0 {
		t.Fatal("No methods found")
	}
	if !methods[0].IsStatic {
		t.Error("Expected IsStatic to be true")
	}
	t.Logf("✅ Java static method: %s isStatic=%v", methods[0].Name, methods[0].IsStatic)
}

func TestNormalizeJava_Complexity_Simple(t *testing.T) {
	module := normalizeJava(t, `
public class Simple {
    public int getValue() {
        return 42;
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	if methods[0].CyclomaticComplexity != 1 {
		t.Errorf("Expected complexity 1, got %d", methods[0].CyclomaticComplexity)
	}
	t.Logf("✅ Java simple complexity: %d", methods[0].CyclomaticComplexity)
}

func TestNormalizeJava_Complexity_WithIf(t *testing.T) {
	module := normalizeJava(t, `
public class Checker {
    public String check(int x) {
        if (x > 0) {
            return "positive";
        } else {
            return "negative";
        }
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	if methods[0].CyclomaticComplexity < 2 {
		t.Errorf("Expected complexity >= 2, got %d", methods[0].CyclomaticComplexity)
	}
	t.Logf("✅ Java if complexity: %d", methods[0].CyclomaticComplexity)
}

// ── CONSTRUCTOR ───────────────────────────────────────

func TestNormalizeJava_Constructor(t *testing.T) {
	module := normalizeJava(t, `
public class Person {
    private String name;

    public Person(String name) {
        this.name = name;
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	var ctor *normalizer.ASTNode
	for i := range methods {
		if methods[i].IsConstructor {
			ctor = &methods[i]
			break
		}
	}

	if ctor == nil {
		t.Fatal("Constructor not found")
	}
	if len(ctor.Parameters) != 1 {
		t.Errorf("Expected 1 param, got %d", len(ctor.Parameters))
	}
	t.Logf("✅ Java constructor: %s params=%d", ctor.Name, len(ctor.Parameters))
}

// ── INTERFACE ─────────────────────────────────────────

func TestNormalizeJava_Interface(t *testing.T) {
	module := normalizeJava(t, `
public interface Repository {
    Object findById(Long id);
    void save(Object entity);
    void delete(Long id);
}
`)
	ifaces := filterByKind(module.Children, normalizer.KindInterface)
	if len(ifaces) != 1 {
		t.Fatalf("Expected 1 interface, got %d", len(ifaces))
	}

	iface := ifaces[0]
	if iface.Name != "Repository" {
		t.Errorf("Expected 'Repository', got '%s'", iface.Name)
	}

	methods := filterByKind(iface.Children, normalizer.KindFunction)
	if len(methods) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(methods))
	}
	t.Logf("✅ Java interface: %s with %d methods", iface.Name, len(methods))
}

// ── ANNOTATIONS ───────────────────────────────────────

func TestNormalizeJava_Annotations(t *testing.T) {
	module := normalizeJava(t, `
@RestController
@RequestMapping("/api")
public class ApiController {
    @GetMapping("/health")
    public String health() {
        return "ok";
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]

	if len(cls.Annotations) < 2 {
		t.Errorf("Expected >= 2 class annotations, got %d", len(cls.Annotations))
	}

	methods := filterByKind(cls.Children, normalizer.KindFunction)
	if len(methods[0].Annotations) < 1 {
		t.Errorf("Expected >= 1 method annotation, got %d",
			len(methods[0].Annotations))
	}

	t.Logf("✅ Java annotations: class=%v method=%v",
		cls.Annotations, methods[0].Annotations)
}

// ── FIELDS ────────────────────────────────────────────

func TestNormalizeJava_Fields(t *testing.T) {
	module := normalizeJava(t, `
public class Config {
    private String host;
    private int port;
    public static final String VERSION = "1.0";
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	fields := filterByKind(cls.Children, normalizer.KindField)

	if len(fields) < 2 {
		t.Errorf("Expected >= 2 fields, got %d", len(fields))
	}
	t.Logf("✅ Java fields: %d found", len(fields))
}

// ── GENERICS ──────────────────────────────────────────

func TestNormalizeJava_GenericReturnType(t *testing.T) {
	module := normalizeJava(t, `
public class UserService {
    public List<String> getNames() {
        return new ArrayList<>();
    }
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	if methods[0].ReturnType.Kind != "GENERIC" {
		t.Errorf("Expected GENERIC, got '%s'", methods[0].ReturnType.Kind)
	}
	t.Logf("✅ Java generic return: %s<%s>",
		methods[0].ReturnType.Kind, methods[0].ReturnType.Name)
}

// ── IMPORTS ───────────────────────────────────────────

func TestNormalizeJava_Imports(t *testing.T) {
	module := normalizeJava(t, `
import java.util.List;
import java.util.ArrayList;

public class MyClass {}
`)
	imports := filterByKind(module.Children, normalizer.KindImport)

	if len(imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(imports))
	}
	t.Logf("✅ Java imports: %d found", len(imports))
}
