package treesitter_test

import (
	"context"
	"testing"

	"github.com/hackersfun369/nexus/internal/parser/treesitter"
)

var ctx = context.Background()

func TestParsePython_SimpleFunction(t *testing.T) {
	source := []byte(`
def greet(name):
    return "Hello, " + name
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.Python)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	if result.RootKind() != "module" {
		t.Errorf("Expected root kind 'module', got '%s'", result.RootKind())
	}
	t.Logf("✅ Python parsed: %d nodes", result.NodeCount())
}

func TestParsePython_Class(t *testing.T) {
	source := []byte(`
class Animal:
    def __init__(self, name: str):
        self.name = name
    def speak(self) -> str:
        return f"{self.name} speaks"
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.Python)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	t.Logf("✅ Python class parsed: %d nodes", result.NodeCount())
}

func TestParsePython_SyntaxError_Recovers(t *testing.T) {
	source := []byte(`
def valid():
    return 42

def broken(

def another_valid():
    pass
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.Python)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.Tree == nil {
		t.Error("Expected tree even with syntax errors")
	}
	if !result.HasError {
		t.Error("Expected HasError to be true for broken code")
	}
	t.Logf("✅ Python error recovery: %d errors found", len(result.Errors))
}

func TestParseTypeScript_SimpleFunction(t *testing.T) {
	source := []byte(`
function greet(name: string): string {
    return "Hello, " + name;
}
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.TypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	t.Logf("✅ TypeScript parsed: %d nodes", result.NodeCount())
}

func TestParseTypeScript_Interface(t *testing.T) {
	source := []byte(`
interface User {
    id: number;
    name: string;
    email?: string;
}

class UserService {
    private users: User[] = [];
    addUser(user: User): void {
        this.users.push(user);
    }
}
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.TypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	t.Logf("✅ TypeScript interface+class parsed: %d nodes", result.NodeCount())
}

func TestParseTypeScript_AsyncFunction(t *testing.T) {
	source := []byte(`
async function fetchUser(id: number): Promise<User> {
    const response = await fetch("/api/users/" + id);
    return response.json();
}
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.TypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	t.Logf("✅ TypeScript async parsed: %d nodes", result.NodeCount())
}

func TestParseJava_SimpleClass(t *testing.T) {
	source := []byte(`
public class HelloWorld {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.Java)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	t.Logf("✅ Java parsed: %d nodes", result.NodeCount())
}

func TestParseJava_Annotations(t *testing.T) {
	source := []byte(`
@RestController
@RequestMapping("/api/users")
public class UserController {

    @Autowired
    private UserService userService;

    @GetMapping("/{id}")
    public ResponseEntity<User> getUser(@PathVariable Long id) {
        return ResponseEntity.ok(userService.findById(id));
    }
}
`)
	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, source, treesitter.Java)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HasError {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}
	t.Logf("✅ Java annotations parsed: %d nodes", result.NodeCount())
}

func TestParse_UnsupportedLanguage_ReturnsError(t *testing.T) {
	p := treesitter.New()
	defer p.Close()

	_, err := p.Parse(ctx, []byte("code"), "ruby")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
	t.Logf("✅ Unsupported language rejected: %v", err)
}
