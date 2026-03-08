package symbols

import (
	"fmt"
	"strings"
	"sync"
)

// SymbolKind represents what kind of symbol this is
type SymbolKind string

const (
	SymbolFunction  SymbolKind = "FUNCTION"
	SymbolClass     SymbolKind = "CLASS"
	SymbolInterface SymbolKind = "INTERFACE"
	SymbolVariable  SymbolKind = "VARIABLE"
	SymbolParameter SymbolKind = "PARAMETER"
	SymbolField     SymbolKind = "FIELD"
	SymbolConstant  SymbolKind = "CONSTANT"
	SymbolImport    SymbolKind = "IMPORT"
	SymbolModule    SymbolKind = "MODULE"
)

// Language represents a programming language
type Language string

const (
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
	LangJava       Language = "java"
)

// Symbol represents a named entity in source code
type Symbol struct {
	// Identity
	ID            string
	Name          string
	QualifiedName string
	Kind          SymbolKind
	Language      Language

	// Location
	FilePath  string
	StartLine uint
	StartCol  uint
	EndLine   uint
	EndCol    uint

	// Scope
	ScopeID   string // ID of enclosing scope (function/class/module)
	ScopeKind string // "function"|"class"|"module"|"global"

	// Type info
	TypeName string
	TypeKind string

	// Properties
	IsExported bool
	IsAsync    bool
	IsStatic   bool
	IsAbstract bool
	Visibility string

	// References — filled during resolution pass
	References []Reference
	DefinedIn  string // Module qualified name
}

// Reference represents a usage of a symbol
type Reference struct {
	FilePath  string
	StartLine uint
	StartCol  uint
	SymbolID  string // The symbol being referenced
}

// Scope represents a lexical scope in the code
type Scope struct {
	ID       string
	Kind     string // "global"|"module"|"class"|"function"
	Name     string
	ParentID string
	FilePath string
	Symbols  map[string]*Symbol // name → symbol
}

// SymbolTable tracks all symbols across all files in a project
type SymbolTable struct {
	mu sync.RWMutex

	// All symbols by ID
	symbols map[string]*Symbol

	// Symbols indexed by qualified name
	byQualifiedName map[string]*Symbol

	// Symbols indexed by file path
	byFile map[string][]*Symbol

	// Scopes by ID
	scopes map[string]*Scope

	// Global scope per file
	fileScopes map[string]*Scope

	// Unresolved references — filled in pass 2
	unresolved []UnresolvedRef
}

// UnresolvedRef tracks a reference we couldn't resolve yet
type UnresolvedRef struct {
	Name      string
	FilePath  string
	StartLine uint
	StartCol  uint
	ScopeID   string
}

// New creates a new empty SymbolTable
func New() *SymbolTable {
	return &SymbolTable{
		symbols:         make(map[string]*Symbol),
		byQualifiedName: make(map[string]*Symbol),
		byFile:          make(map[string][]*Symbol),
		scopes:          make(map[string]*Scope),
		fileScopes:      make(map[string]*Scope),
		unresolved:      []UnresolvedRef{},
	}
}

// ── SCOPE MANAGEMENT ──────────────────────────────────

// OpenScope creates a new scope
func (st *SymbolTable) OpenScope(id, kind, name, parentID, filePath string) *Scope {
	st.mu.Lock()
	defer st.mu.Unlock()

	scope := &Scope{
		ID:       id,
		Kind:     kind,
		Name:     name,
		ParentID: parentID,
		FilePath: filePath,
		Symbols:  make(map[string]*Symbol),
	}

	st.scopes[id] = scope

	if kind == "module" || kind == "global" {
		st.fileScopes[filePath] = scope
	}

	return scope
}

// GetScope returns a scope by ID
func (st *SymbolTable) GetScope(id string) (*Scope, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	scope, ok := st.scopes[id]
	return scope, ok
}

// GetFileScope returns the top-level scope for a file
func (st *SymbolTable) GetFileScope(filePath string) (*Scope, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	scope, ok := st.fileScopes[filePath]
	return scope, ok
}

// ── SYMBOL REGISTRATION ───────────────────────────────

// Define registers a new symbol in the table
func (st *SymbolTable) Define(sym *Symbol) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if sym.ID == "" {
		return fmt.Errorf("symbol ID cannot be empty")
	}
	if sym.Name == "" {
		return fmt.Errorf("symbol name cannot be empty")
	}

	// Store by ID
	st.symbols[sym.ID] = sym

	// Store by qualified name
	if sym.QualifiedName != "" {
		st.byQualifiedName[sym.QualifiedName] = sym
	}

	// Store by file
	st.byFile[sym.FilePath] = append(st.byFile[sym.FilePath], sym)

	// Register in scope
	if sym.ScopeID != "" {
		if scope, ok := st.scopes[sym.ScopeID]; ok {
			scope.Symbols[sym.Name] = sym
		}
	}

	return nil
}

// DefineAll registers multiple symbols at once
func (st *SymbolTable) DefineAll(symbols []*Symbol) []error {
	errs := []error{}
	for _, sym := range symbols {
		if err := st.Define(sym); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// ── SYMBOL LOOKUP ─────────────────────────────────────

// LookupByID returns a symbol by its ID
func (st *SymbolTable) LookupByID(id string) (*Symbol, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	sym, ok := st.symbols[id]
	return sym, ok
}

// LookupByQualifiedName returns a symbol by qualified name
func (st *SymbolTable) LookupByQualifiedName(qualName string) (*Symbol, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	sym, ok := st.byQualifiedName[qualName]
	return sym, ok
}

// LookupInScope resolves a name starting from a scope,
// walking up the scope chain until found
func (st *SymbolTable) LookupInScope(name, scopeID string) (*Symbol, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	currentID := scopeID
	for currentID != "" {
		scope, ok := st.scopes[currentID]
		if !ok {
			break
		}

		if sym, found := scope.Symbols[name]; found {
			return sym, true
		}

		currentID = scope.ParentID
	}

	return nil, false
}

// LookupByFile returns all symbols defined in a file
func (st *SymbolTable) LookupByFile(filePath string) []*Symbol {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.byFile[filePath]
}

// LookupByKind returns all symbols of a given kind
func (st *SymbolTable) LookupByKind(kind SymbolKind) []*Symbol {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := []*Symbol{}
	for _, sym := range st.symbols {
		if sym.Kind == kind {
			result = append(result, sym)
		}
	}
	return result
}

// ── REFERENCE TRACKING ────────────────────────────────

// AddReference records a reference to a symbol
func (st *SymbolTable) AddReference(symbolID string, ref Reference) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	sym, ok := st.symbols[symbolID]
	if !ok {
		return fmt.Errorf("symbol %s not found", symbolID)
	}

	sym.References = append(sym.References, ref)
	return nil
}

// AddUnresolved records a reference we couldn't resolve
func (st *SymbolTable) AddUnresolved(ref UnresolvedRef) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.unresolved = append(st.unresolved, ref)
}

// GetUnresolved returns all unresolved references
func (st *SymbolTable) GetUnresolved() []UnresolvedRef {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := make([]UnresolvedRef, len(st.unresolved))
	copy(result, st.unresolved)
	return result
}

// ── CROSS-FILE RESOLUTION ─────────────────────────────

// ResolveImport tries to resolve an import to a symbol
// e.g. "from auth.service import UserService" → finds UserService symbol
func (st *SymbolTable) ResolveImport(importPath, symbolName string) (*Symbol, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Try qualified name directly
	qualName := importPath + "." + symbolName
	if sym, ok := st.byQualifiedName[qualName]; ok {
		return sym, true
	}

	// Try just the symbol name across all files matching the import path
	for _, sym := range st.symbols {
		if sym.Name == symbolName &&
			strings.Contains(sym.FilePath, strings.ReplaceAll(importPath, ".", "/")) {
			return sym, true
		}
	}

	return nil, false
}

// ── STATISTICS ────────────────────────────────────────

// Stats returns statistics about the symbol table
func (st *SymbolTable) Stats() SymbolTableStats {
	st.mu.RLock()
	defer st.mu.RUnlock()

	stats := SymbolTableStats{
		TotalSymbols:   len(st.symbols),
		TotalScopes:    len(st.scopes),
		TotalFiles:     len(st.byFile),
		UnresolvedRefs: len(st.unresolved),
		ByKind:         make(map[SymbolKind]int),
	}

	for _, sym := range st.symbols {
		stats.ByKind[sym.Kind]++
		stats.TotalReferences += len(sym.References)
	}

	return stats
}

// SymbolTableStats holds statistics about the symbol table
type SymbolTableStats struct {
	TotalSymbols    int
	TotalScopes     int
	TotalFiles      int
	TotalReferences int
	UnresolvedRefs  int
	ByKind          map[SymbolKind]int
}

// ── MERGE ─────────────────────────────────────────────

// Merge merges another symbol table into this one
// Used when incrementally reparsing files
func (st *SymbolTable) Merge(other *SymbolTable) {
	st.mu.Lock()
	defer st.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for id, sym := range other.symbols {
		st.symbols[id] = sym
	}
	for qn, sym := range other.byQualifiedName {
		st.byQualifiedName[qn] = sym
	}
	for file, syms := range other.byFile {
		st.byFile[file] = syms
	}
	for id, scope := range other.scopes {
		st.scopes[id] = scope
	}
	for file, scope := range other.fileScopes {
		st.fileScopes[file] = scope
	}
}

// RemoveFile removes all symbols from a file
// Used when a file is deleted or reparsed
func (st *SymbolTable) RemoveFile(filePath string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	syms, ok := st.byFile[filePath]
	if !ok {
		return
	}

	for _, sym := range syms {
		delete(st.symbols, sym.ID)
		if sym.QualifiedName != "" {
			delete(st.byQualifiedName, sym.QualifiedName)
		}
	}

	delete(st.byFile, filePath)

	// Remove file scope
	if scope, ok := st.fileScopes[filePath]; ok {
		delete(st.scopes, scope.ID)
		delete(st.fileScopes, filePath)
	}
}

// Size returns total number of symbols
func (st *SymbolTable) Size() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.symbols)
}
