package normalizer_test

import (
	"testing"

	"github.com/hackersfun369/nexus/internal/parser/normalizer"
	"github.com/hackersfun369/nexus/internal/parser/treesitter"
)

// helper — parse TypeScript and normalize
func normalizeTypeScript(t *testing.T, source string) normalizer.ASTNode {
	t.Helper()

	p := treesitter.New()
	defer p.Close()

	result, err := p.Parse(ctx, []byte(source), treesitter.TypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	n := normalizer.NewTypeScriptNormalizer([]byte(source), "test.ts")
	return n.Normalize(result.Tree.RootNode())
}

// ── MODULE ────────────────────────────────────────────

func TestNormalizeTS_Module_Kind(t *testing.T) {
	module := normalizeTypeScript(t, `const x = 1;`)

	if module.Kind != normalizer.KindModule {
		t.Errorf("Expected KindModule, got %s", module.Kind)
	}
	if module.Language != normalizer.LangTypeScript {
		t.Errorf("Expected LangTypeScript, got %s", module.Language)
	}
	t.Logf("✅ TS Module kind: %s", module.Kind)
}

// ── FUNCTIONS ─────────────────────────────────────────

func TestNormalizeTS_SimpleFunction(t *testing.T) {
	module := normalizeTypeScript(t, `
function greet(name: string): string {
    return "Hello, " + name;
}
`)
	fns := filterByKind(module.Children, normalizer.KindFunction)
	if len(fns) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(fns))
	}

	fn := fns[0]
	if fn.Name != "greet" {
		t.Errorf("Expected 'greet', got '%s'", fn.Name)
	}
	if len(fn.Parameters) != 1 {
		t.Errorf("Expected 1 param, got %d", len(fn.Parameters))
	}
	if fn.Parameters[0].Type.Name != "string" {
		t.Errorf("Expected type 'string', got '%s'", fn.Parameters[0].Type.Name)
	}
	if fn.ReturnType.Name != "string" {
		t.Errorf("Expected return 'string', got '%s'", fn.ReturnType.Name)
	}
	t.Logf("✅ TS function: %s(%s): %s",
		fn.Name,
		fn.Parameters[0].Type.Name,
		fn.ReturnType.Name)
}

func TestNormalizeTS_AsyncFunction(t *testing.T) {
	module := normalizeTypeScript(t, `
async function fetchData(url: string): Promise<string> {
    return "";
}
`)
	fns := filterByKind(module.Children, normalizer.KindFunction)
	if len(fns) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(fns))
	}

	fn := fns[0]
	if !fn.IsAsync {
		t.Error("Expected IsAsync to be true")
	}
	if fn.ReturnType.Kind != "GENERIC" {
		t.Errorf("Expected GENERIC return type, got '%s'", fn.ReturnType.Kind)
	}
	t.Logf("✅ TS async function: %s returns %s<%s>",
		fn.Name, fn.ReturnType.Kind, fn.ReturnType.Name)
}

func TestNormalizeTS_Complexity_Simple(t *testing.T) {
	module := normalizeTypeScript(t, `
function simple(): number {
    return 42;
}
`)
	fn := filterByKind(module.Children, normalizer.KindFunction)[0]

	if fn.CyclomaticComplexity != 1 {
		t.Errorf("Expected complexity 1, got %d", fn.CyclomaticComplexity)
	}
	t.Logf("✅ TS simple complexity: %d", fn.CyclomaticComplexity)
}

func TestNormalizeTS_Complexity_WithIf(t *testing.T) {
	module := normalizeTypeScript(t, `
function check(x: number): string {
    if (x > 0) {
        return "positive";
    } else {
        return "negative";
    }
}
`)
	fn := filterByKind(module.Children, normalizer.KindFunction)[0]

	if fn.CyclomaticComplexity < 2 {
		t.Errorf("Expected complexity >= 2, got %d", fn.CyclomaticComplexity)
	}
	t.Logf("✅ TS if complexity: %d", fn.CyclomaticComplexity)
}

// ── CLASSES ───────────────────────────────────────────

func TestNormalizeTS_Class(t *testing.T) {
	module := normalizeTypeScript(t, `
class UserService {
    private users: string[] = [];

    addUser(name: string): void {
        this.users.push(name);
    }

    getUsers(): string[] {
        return this.users;
    }
}
`)
	classes := filterByKind(module.Children, normalizer.KindClass)
	if len(classes) != 1 {
		t.Fatalf("Expected 1 class, got %d", len(classes))
	}

	cls := classes[0]
	if cls.Name != "UserService" {
		t.Errorf("Expected 'UserService', got '%s'", cls.Name)
	}

	methods := filterByKind(cls.Children, normalizer.KindFunction)
	if len(methods) < 2 {
		t.Errorf("Expected at least 2 methods, got %d", len(methods))
	}
	t.Logf("✅ TS class: %s with %d methods", cls.Name, len(methods))
}

func TestNormalizeTS_Constructor(t *testing.T) {
	module := normalizeTypeScript(t, `
class Animal {
    constructor(private name: string) {}
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	var constructor *normalizer.ASTNode
	for i := range methods {
		if methods[i].IsConstructor {
			constructor = &methods[i]
			break
		}
	}

	if constructor == nil {
		t.Error("Expected constructor to be detected")
	}
	t.Logf("✅ TS constructor detected: %s", constructor.Name)
}

func TestNormalizeTS_MethodVisibility(t *testing.T) {
	module := normalizeTypeScript(t, `
class MyClass {
    public publicMethod(): void {}
    private privateMethod(): void {}
    protected protectedMethod(): void {}
}
`)
	cls := filterByKind(module.Children, normalizer.KindClass)[0]
	methods := filterByKind(cls.Children, normalizer.KindFunction)

	visibilities := map[string]normalizer.Visibility{}
	for _, m := range methods {
		visibilities[m.Name] = m.Visibility
	}

	if visibilities["publicMethod"] != normalizer.VisibilityPublic {
		t.Errorf("Expected PUBLIC, got %s", visibilities["publicMethod"])
	}
	if visibilities["privateMethod"] != normalizer.VisibilityPrivate {
		t.Errorf("Expected PRIVATE, got %s", visibilities["privateMethod"])
	}
	if visibilities["protectedMethod"] != normalizer.VisibilityProtected {
		t.Errorf("Expected PROTECTED, got %s", visibilities["protectedMethod"])
	}
	t.Logf("✅ TS method visibility: public/private/protected all correct")
}

// ── INTERFACES ────────────────────────────────────────

func TestNormalizeTS_Interface(t *testing.T) {
	module := normalizeTypeScript(t, `
interface User {
    id: number;
    name: string;
    email?: string;
}
`)
	ifaces := filterByKind(module.Children, normalizer.KindInterface)
	if len(ifaces) != 1 {
		t.Fatalf("Expected 1 interface, got %d", len(ifaces))
	}

	iface := ifaces[0]
	if iface.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", iface.Name)
	}

	fields := filterByKind(iface.Children, normalizer.KindField)
	if len(fields) < 2 {
		t.Errorf("Expected at least 2 fields, got %d", len(fields))
	}
	t.Logf("✅ TS interface: %s with %d fields", iface.Name, len(fields))
}

func TestNormalizeTS_OptionalField_IsNullable(t *testing.T) {
	module := normalizeTypeScript(t, `
interface User {
    id: number;
    email?: string;
}
`)
	iface := filterByKind(module.Children, normalizer.KindInterface)[0]
	fields := filterByKind(iface.Children, normalizer.KindField)

	var emailField *normalizer.ASTNode
	for i := range fields {
		if fields[i].Name == "email" {
			emailField = &fields[i]
			break
		}
	}

	if emailField == nil {
		t.Fatal("email field not found")
	}
	if !emailField.Type.Nullable {
		t.Error("Expected email? to be nullable")
	}
	t.Logf("✅ TS optional field nullable: email? nullable=%v",
		emailField.Type.Nullable)
}

// ── TYPES ─────────────────────────────────────────────

func TestNormalizeTS_ArrayType(t *testing.T) {
	module := normalizeTypeScript(t, `
function getNames(): string[] {
    return [];
}
`)
	fn := filterByKind(module.Children, normalizer.KindFunction)[0]

	if fn.ReturnType.Kind != "ARRAY" {
		t.Errorf("Expected ARRAY type, got '%s'", fn.ReturnType.Kind)
	}
	t.Logf("✅ TS array type: %s", fn.ReturnType.Name)
}

func TestNormalizeTS_UnionType(t *testing.T) {
	module := normalizeTypeScript(t, `
function parse(input: string | number): string {
    return String(input);
}
`)
	fn := filterByKind(module.Children, normalizer.KindFunction)[0]

	if fn.Parameters[0].Type.Kind != "UNION" {
		t.Errorf("Expected UNION type, got '%s'", fn.Parameters[0].Type.Kind)
	}
	t.Logf("✅ TS union type: %s", fn.Parameters[0].Type.Name)
}

// ── IMPORTS ───────────────────────────────────────────

func TestNormalizeTS_Import(t *testing.T) {
	module := normalizeTypeScript(t, `
import { useState } from 'react';
import axios from 'axios';
`)
	imports := filterByKind(module.Children, normalizer.KindImport)

	if len(imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(imports))
	}
	t.Logf("✅ TS imports: %d found", len(imports))
}
