package normalizer

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

// NodeKind represents the type of an AST node
type NodeKind string

const (
	KindModule     NodeKind = "MODULE"
	KindFunction   NodeKind = "FUNCTION"
	KindClass      NodeKind = "CLASS"
	KindInterface  NodeKind = "INTERFACE"
	KindVariable   NodeKind = "VARIABLE"
	KindParameter  NodeKind = "PARAMETER"
	KindField      NodeKind = "FIELD"
	KindConstant   NodeKind = "CONSTANT"
	KindImport     NodeKind = "IMPORT"
	KindCall       NodeKind = "CALL"
	KindReturn     NodeKind = "RETURN"
	KindIf         NodeKind = "IF"
	KindLoop       NodeKind = "LOOP"
	KindAssignment NodeKind = "ASSIGNMENT"
	KindExpression NodeKind = "EXPRESSION"
)

// Visibility represents the access level of a symbol
type Visibility string

const (
	VisibilityPublic    Visibility = "PUBLIC"
	VisibilityPrivate   Visibility = "PRIVATE"
	VisibilityProtected Visibility = "PROTECTED"
	VisibilityInternal  Visibility = "INTERNAL"
)

// Language represents a programming language
type Language string

const (
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
	LangJava       Language = "java"
)

// Position represents a location in source code
type Position struct {
	Line   uint
	Column uint
}

// SourceLocation represents a range in source code
type SourceLocation struct {
	File      string
	StartLine uint
	StartCol  uint
	EndLine   uint
	EndCol    uint
}

// TypeDescriptor describes a type
type TypeDescriptor struct {
	Kind     string           // PRIMITIVE|NAMED|GENERIC|UNION|ARRAY|UNKNOWN
	Name     string           // "str", "int", "List", "MyClass"
	TypeArgs []TypeDescriptor // For generics: List[T] → [T]
	Nullable bool
	Inferred bool // true if inferred, not declared
}

// UnknownType is the default type when inference fails
var UnknownType = TypeDescriptor{
	Kind:     "UNKNOWN",
	Name:     "unknown",
	Inferred: true,
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string
	Type         TypeDescriptor
	DefaultValue string
	IsVariadic   bool
}

// ASTNode is the universal AST node used across all languages
type ASTNode struct {
	// Identity
	ID            string
	Kind          NodeKind
	Language      Language
	Name          string
	QualifiedName string

	// Location
	Location SourceLocation

	// Type info
	Type       TypeDescriptor
	Visibility Visibility

	// Documentation
	DocComment string

	// Function-specific
	Parameters    []Parameter
	ReturnType    TypeDescriptor
	IsAsync       bool
	IsStatic      bool
	IsAbstract    bool
	IsConstructor bool

	// Metrics (computed during normalization)
	CyclomaticComplexity int
	NestingDepth         int
	LinesOfCode          int

	// Annotations / decorators
	Annotations []string

	// Tree structure
	Children []ASTNode
	ParentID string

	// Raw source
	RawSource string
}

// DetectLanguage detects the language from a file extension
func DetectLanguage(filePath string) Language {
	if len(filePath) == 0 {
		return ""
	}
	switch {
	case hasSuffix(filePath, ".py"):
		return LangPython
	case hasSuffix(filePath, ".ts"), hasSuffix(filePath, ".tsx"):
		return LangTypeScript
	case hasSuffix(filePath, ".js"), hasSuffix(filePath, ".jsx"):
		return LangTypeScript
	case hasSuffix(filePath, ".java"):
		return LangJava
	default:
		return ""
	}
}

// NewNormalizer returns the correct normalizer for a language
func NewNormalizer(lang Language) Normalizer {
	switch lang {
	case LangPython:
		return &PythonNormalizer{}
	case LangTypeScript:
		return &TypeScriptNormalizer{}
	case LangJava:
		return &JavaNormalizer{}
	default:
		return nil
	}
}

// Normalizer is the interface all language normalizers implement
// Normalizer is the interface all language normalizers implement
type Normalizer interface {
	Normalize(node *tree_sitter.Node) ASTNode
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
