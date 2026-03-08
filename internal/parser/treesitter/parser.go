package treesitter

import (
	"context"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Language represents a supported programming language
type Language string

const (
	Python     Language = "python"
	TypeScript Language = "typescript"
	Java       Language = "java"
)

// ParseResult holds the result of parsing a source file
type ParseResult struct {
	Tree     *sitter.Tree
	Language Language
	Source   []byte
	HasError bool
	Errors   []ParseError
}

// ParseError represents a syntax error found during parsing
type ParseError struct {
	StartLine uint
	StartCol  uint
	EndLine   uint
	EndCol    uint
	Message   string
}

// Parser wraps tree-sitter for all supported languages
type Parser struct {
	parser *sitter.Parser
}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{
		parser: sitter.NewParser(),
	}
}

// Parse parses source code in the given language
func (p *Parser) Parse(ctx context.Context, source []byte, lang Language) (*ParseResult, error) {
	tsLang, err := getLanguage(lang)
	if err != nil {
		return nil, err
	}

	if err := p.parser.SetLanguage(tsLang); err != nil {
		return nil, fmt.Errorf("failed to set language %s: %w", lang, err)
	}

	tree := p.parser.Parse(source, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse source code")
	}

	result := &ParseResult{
		Tree:     tree,
		Language: lang,
		Source:   source,
		HasError: tree.RootNode().HasError(),
	}

	if result.HasError {
		result.Errors = collectErrors(tree.RootNode(), source)
	}

	return result, nil
}

// Close releases parser resources
func (p *Parser) Close() {
	p.parser.Close()
}

// getLanguage returns the tree-sitter language for the given language
func getLanguage(lang Language) (*sitter.Language, error) {
	switch lang {
	case Python:
		return sitter.NewLanguage(python.Language()), nil
	case TypeScript:
		return sitter.NewLanguage(typescript.LanguageTypescript()), nil
	case Java:
		return sitter.NewLanguage(java.Language()), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// collectErrors walks the tree and collects ERROR nodes
func collectErrors(node *sitter.Node, source []byte) []ParseError {
	errors := []ParseError{}

	if node.Kind() == "ERROR" || node.IsMissing() {
		start := node.StartPosition()
		end := node.EndPosition()
		errors = append(errors, ParseError{
			StartLine: start.Row,
			StartCol:  start.Column,
			EndLine:   end.Row,
			EndCol:    end.Column,
			Message:   fmt.Sprintf("syntax error at line %d", start.Row+1),
		})
	}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child != nil {
			errors = append(errors, collectErrors(child, source)...)
		}
	}

	return errors
}

// NodeCount returns total number of nodes in the tree
func (r *ParseResult) NodeCount() uint {
	return r.Tree.RootNode().DescendantCount()
}

// RootKind returns the kind of the root node
func (r *ParseResult) RootKind() string {
	return r.Tree.RootNode().Kind()
}
