package symbols_test

import (
	"testing"

	"github.com/hackersfun369/nexus/internal/parser/symbols"
)

// ── HELPERS ───────────────────────────────────────────

func makeSymbol(id, name, qualName string, kind symbols.SymbolKind, file string) *symbols.Symbol {
	return &symbols.Symbol{
		ID:            id,
		Name:          name,
		QualifiedName: qualName,
		Kind:          kind,
		Language:      symbols.LangPython,
		FilePath:      file,
		ScopeID:       "scope-" + file,
	}
}

// ── DEFINE AND LOOKUP ─────────────────────────────────

func TestSymbolTable_Define_And_LookupByID(t *testing.T) {
	st := symbols.New()

	sym := makeSymbol("fn-001", "greet", "auth.service.greet",
		symbols.SymbolFunction, "src/auth/service.py")

	err := st.Define(sym)
	if err != nil {
		t.Fatalf("Define failed: %v", err)
	}

	got, ok := st.LookupByID("fn-001")
	if !ok {
		t.Fatal("Expected to find symbol by ID")
	}
	if got.Name != "greet" {
		t.Errorf("Expected 'greet', got '%s'", got.Name)
	}
	t.Logf("✅ Define and LookupByID: %s", got.Name)
}

func TestSymbolTable_LookupByQualifiedName(t *testing.T) {
	st := symbols.New()

	sym := makeSymbol("fn-001", "greet", "auth.service.greet",
		symbols.SymbolFunction, "src/auth/service.py")
	st.Define(sym)

	got, ok := st.LookupByQualifiedName("auth.service.greet")
	if !ok {
		t.Fatal("Expected to find symbol by qualified name")
	}
	if got.ID != "fn-001" {
		t.Errorf("Expected ID 'fn-001', got '%s'", got.ID)
	}
	t.Logf("✅ LookupByQualifiedName: %s", got.QualifiedName)
}

func TestSymbolTable_LookupByFile(t *testing.T) {
	st := symbols.New()

	st.Define(makeSymbol("fn-001", "greet", "service.greet",
		symbols.SymbolFunction, "src/service.py"))
	st.Define(makeSymbol("fn-002", "farewell", "service.farewell",
		symbols.SymbolFunction, "src/service.py"))
	st.Define(makeSymbol("fn-003", "other", "other.other",
		symbols.SymbolFunction, "src/other.py"))

	syms := st.LookupByFile("src/service.py")
	if len(syms) != 2 {
		t.Errorf("Expected 2 symbols in service.py, got %d", len(syms))
	}
	t.Logf("✅ LookupByFile: %d symbols in service.py", len(syms))
}

func TestSymbolTable_LookupByKind(t *testing.T) {
	st := symbols.New()

	st.Define(makeSymbol("fn-001", "greet", "s.greet",
		symbols.SymbolFunction, "a.py"))
	st.Define(makeSymbol("cls-001", "User", "s.User",
		symbols.SymbolClass, "a.py"))
	st.Define(makeSymbol("fn-002", "farewell", "s.farewell",
		symbols.SymbolFunction, "a.py"))

	fns := st.LookupByKind(symbols.SymbolFunction)
	if len(fns) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(fns))
	}

	classes := st.LookupByKind(symbols.SymbolClass)
	if len(classes) != 1 {
		t.Errorf("Expected 1 class, got %d", len(classes))
	}
	t.Logf("✅ LookupByKind: %d functions, %d classes", len(fns), len(classes))
}

func TestSymbolTable_Define_EmptyID_ReturnsError(t *testing.T) {
	st := symbols.New()

	sym := &symbols.Symbol{Name: "greet"} // No ID
	err := st.Define(sym)

	if err == nil {
		t.Error("Expected error for empty ID")
	}
	t.Logf("✅ Empty ID rejected: %v", err)
}

func TestSymbolTable_Define_EmptyName_ReturnsError(t *testing.T) {
	st := symbols.New()

	sym := &symbols.Symbol{ID: "fn-001"} // No Name
	err := st.Define(sym)

	if err == nil {
		t.Error("Expected error for empty Name")
	}
	t.Logf("✅ Empty name rejected: %v", err)
}

// ── SCOPES ────────────────────────────────────────────

func TestSymbolTable_Scope_OpenAndGet(t *testing.T) {
	st := symbols.New()

	scope := st.OpenScope("scope-001", "module", "service",
		"", "src/service.py")

	if scope.ID != "scope-001" {
		t.Errorf("Expected scope ID 'scope-001', got '%s'", scope.ID)
	}

	got, ok := st.GetScope("scope-001")
	if !ok {
		t.Fatal("Expected to find scope")
	}
	if got.Kind != "module" {
		t.Errorf("Expected kind 'module', got '%s'", got.Kind)
	}
	t.Logf("✅ Scope open and get: %s (%s)", got.ID, got.Kind)
}

func TestSymbolTable_FileScope_RegisteredOnModuleOpen(t *testing.T) {
	st := symbols.New()

	st.OpenScope("scope-001", "module", "service",
		"", "src/service.py")

	scope, ok := st.GetFileScope("src/service.py")
	if !ok {
		t.Fatal("Expected file scope to be registered")
	}
	if scope.ID != "scope-001" {
		t.Errorf("Expected scope-001, got %s", scope.ID)
	}
	t.Logf("✅ File scope registered: %s", scope.ID)
}

func TestSymbolTable_LookupInScope_FindsLocal(t *testing.T) {
	st := symbols.New()

	// Create module scope
	st.OpenScope("scope-mod", "module", "service",
		"", "src/service.py")

	// Define symbol in that scope
	sym := &symbols.Symbol{
		ID:       "fn-001",
		Name:     "greet",
		Kind:     symbols.SymbolFunction,
		FilePath: "src/service.py",
		ScopeID:  "scope-mod",
	}
	st.Define(sym)

	// Look up in scope
	found, ok := st.LookupInScope("greet", "scope-mod")
	if !ok {
		t.Fatal("Expected to find 'greet' in scope")
	}
	if found.ID != "fn-001" {
		t.Errorf("Expected fn-001, got %s", found.ID)
	}
	t.Logf("✅ LookupInScope: found %s", found.Name)
}

func TestSymbolTable_LookupInScope_WalksUpChain(t *testing.T) {
	st := symbols.New()

	// Parent scope (module)
	st.OpenScope("scope-mod", "module", "service",
		"", "src/service.py")

	// Child scope (function)
	st.OpenScope("scope-fn", "function", "greet",
		"scope-mod", "src/service.py")

	// Define in parent
	sym := &symbols.Symbol{
		ID:       "cls-001",
		Name:     "User",
		Kind:     symbols.SymbolClass,
		FilePath: "src/service.py",
		ScopeID:  "scope-mod",
	}
	st.Define(sym)

	// Look up from child — should walk up to parent
	found, ok := st.LookupInScope("User", "scope-fn")
	if !ok {
		t.Fatal("Expected to find 'User' by walking up scope chain")
	}
	if found.ID != "cls-001" {
		t.Errorf("Expected cls-001, got %s", found.ID)
	}
	t.Logf("✅ Scope chain walk: found %s from child scope", found.Name)
}

func TestSymbolTable_LookupInScope_NotFound_ReturnsFalse(t *testing.T) {
	st := symbols.New()
	st.OpenScope("scope-mod", "module", "service", "", "src/service.py")

	_, ok := st.LookupInScope("nonexistent", "scope-mod")
	if ok {
		t.Error("Expected not found")
	}
	t.Logf("✅ LookupInScope not found: returns false correctly")
}

// ── REFERENCES ────────────────────────────────────────

func TestSymbolTable_AddReference(t *testing.T) {
	st := symbols.New()

	sym := makeSymbol("fn-001", "greet", "service.greet",
		symbols.SymbolFunction, "src/service.py")
	st.Define(sym)

	ref := symbols.Reference{
		FilePath:  "src/main.py",
		StartLine: 10,
		StartCol:  4,
		SymbolID:  "fn-001",
	}

	err := st.AddReference("fn-001", ref)
	if err != nil {
		t.Fatalf("AddReference failed: %v", err)
	}

	got, _ := st.LookupByID("fn-001")
	if len(got.References) != 1 {
		t.Errorf("Expected 1 reference, got %d", len(got.References))
	}
	if got.References[0].FilePath != "src/main.py" {
		t.Errorf("Expected 'src/main.py', got '%s'",
			got.References[0].FilePath)
	}
	t.Logf("✅ AddReference: %d references", len(got.References))
}

func TestSymbolTable_AddReference_UnknownSymbol_ReturnsError(t *testing.T) {
	st := symbols.New()

	err := st.AddReference("nonexistent", symbols.Reference{})
	if err == nil {
		t.Error("Expected error for unknown symbol")
	}
	t.Logf("✅ Unknown symbol reference rejected: %v", err)
}

func TestSymbolTable_AddUnresolved(t *testing.T) {
	st := symbols.New()

	st.AddUnresolved(symbols.UnresolvedRef{
		Name:      "SomeClass",
		FilePath:  "src/main.py",
		StartLine: 5,
	})

	unresolved := st.GetUnresolved()
	if len(unresolved) != 1 {
		t.Errorf("Expected 1 unresolved, got %d", len(unresolved))
	}
	if unresolved[0].Name != "SomeClass" {
		t.Errorf("Expected 'SomeClass', got '%s'", unresolved[0].Name)
	}
	t.Logf("✅ Unresolved tracking: %d refs", len(unresolved))
}

// ── CROSS-FILE RESOLUTION ─────────────────────────────

func TestSymbolTable_ResolveImport(t *testing.T) {
	st := symbols.New()

	sym := &symbols.Symbol{
		ID:            "cls-001",
		Name:          "UserService",
		QualifiedName: "auth.service.UserService",
		Kind:          symbols.SymbolClass,
		FilePath:      "src/auth/service.py",
		ScopeID:       "scope-auth",
	}
	st.Define(sym)

	found, ok := st.ResolveImport("auth.service", "UserService")
	if !ok {
		t.Fatal("Expected to resolve import")
	}
	if found.ID != "cls-001" {
		t.Errorf("Expected cls-001, got %s", found.ID)
	}
	t.Logf("✅ ResolveImport: found %s", found.QualifiedName)
}

func TestSymbolTable_ResolveImport_NotFound(t *testing.T) {
	st := symbols.New()

	_, ok := st.ResolveImport("nonexistent.module", "SomeClass")
	if ok {
		t.Error("Expected not found for unknown import")
	}
	t.Logf("✅ ResolveImport not found: returns false correctly")
}

// ── MERGE AND REMOVE ──────────────────────────────────

func TestSymbolTable_RemoveFile(t *testing.T) {
	st := symbols.New()

	st.Define(makeSymbol("fn-001", "greet", "s.greet",
		symbols.SymbolFunction, "src/service.py"))
	st.Define(makeSymbol("fn-002", "farewell", "s.farewell",
		symbols.SymbolFunction, "src/service.py"))
	st.Define(makeSymbol("fn-003", "other", "o.other",
		symbols.SymbolFunction, "src/other.py"))

	if st.Size() != 3 {
		t.Fatalf("Expected 3 symbols, got %d", st.Size())
	}

	st.RemoveFile("src/service.py")

	if st.Size() != 1 {
		t.Errorf("Expected 1 symbol after removal, got %d", st.Size())
	}

	_, ok := st.LookupByID("fn-001")
	if ok {
		t.Error("fn-001 should be removed")
	}
	_, ok = st.LookupByID("fn-003")
	if !ok {
		t.Error("fn-003 should still exist")
	}
	t.Logf("✅ RemoveFile: %d symbols remain", st.Size())
}

func TestSymbolTable_Merge(t *testing.T) {
	st1 := symbols.New()
	st2 := symbols.New()

	st1.Define(makeSymbol("fn-001", "greet", "s.greet",
		symbols.SymbolFunction, "src/a.py"))
	st2.Define(makeSymbol("fn-002", "farewell", "s.farewell",
		symbols.SymbolFunction, "src/b.py"))

	st1.Merge(st2)

	if st1.Size() != 2 {
		t.Errorf("Expected 2 symbols after merge, got %d", st1.Size())
	}

	_, ok := st1.LookupByID("fn-002")
	if !ok {
		t.Error("fn-002 should exist after merge")
	}
	t.Logf("✅ Merge: %d symbols after merge", st1.Size())
}

// ── STATS ─────────────────────────────────────────────

func TestSymbolTable_Stats(t *testing.T) {
	st := symbols.New()

	st.OpenScope("scope-a", "module", "a", "", "src/a.py")
	st.Define(makeSymbol("fn-001", "greet", "a.greet",
		symbols.SymbolFunction, "src/a.py"))
	st.Define(makeSymbol("cls-001", "User", "a.User",
		symbols.SymbolClass, "src/a.py"))
	st.AddUnresolved(symbols.UnresolvedRef{Name: "Unknown"})

	stats := st.Stats()

	if stats.TotalSymbols != 2 {
		t.Errorf("Expected 2 symbols, got %d", stats.TotalSymbols)
	}
	if stats.TotalScopes != 1 {
		t.Errorf("Expected 1 scope, got %d", stats.TotalScopes)
	}
	if stats.UnresolvedRefs != 1 {
		t.Errorf("Expected 1 unresolved, got %d", stats.UnresolvedRefs)
	}
	if stats.ByKind[symbols.SymbolFunction] != 1 {
		t.Errorf("Expected 1 function, got %d",
			stats.ByKind[symbols.SymbolFunction])
	}
	t.Logf("✅ Stats: symbols=%d scopes=%d unresolved=%d",
		stats.TotalSymbols, stats.TotalScopes, stats.UnresolvedRefs)
}
