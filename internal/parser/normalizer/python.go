package normalizer

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// PythonNormalizer converts Python CST → Universal AST nodes
type PythonNormalizer struct {
	source   []byte
	filePath string
}

// NewPythonNormalizer creates a new Python normalizer
func NewPythonNormalizer(source []byte, filePath string) *PythonNormalizer {
	return &PythonNormalizer{
		source:   source,
		filePath: filePath,
	}
}

// Normalize converts the root CST node into a MODULE ASTNode
func (n *PythonNormalizer) Normalize(root *sitter.Node) ASTNode {
	module := ASTNode{
		ID:            generateID(n.filePath),
		Kind:          KindModule,
		Language:      LangPython,
		Name:          moduleNameFromPath(n.filePath),
		QualifiedName: qualifiedNameFromPath(n.filePath),
		Location: SourceLocation{
			File:      n.filePath,
			StartLine: 0,
			StartCol:  0,
			EndLine:   root.EndPosition().Row,
			EndCol:    root.EndPosition().Column,
		},
		LinesOfCode: int(root.EndPosition().Row),
	}

	// Walk top-level children
	for i := range root.ChildCount() {
		child := root.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "function_definition":
			fn := n.normalizeFunction(child, 0)
			module.Children = append(module.Children, fn)

		case "class_definition":
			cls := n.normalizeClass(child)
			module.Children = append(module.Children, cls)

		case "import_statement", "import_from_statement":
			imp := n.normalizeImport(child)
			module.Children = append(module.Children, imp)

		case "decorated_definition":
			decorated := n.normalizeDecorated(child)
			if decorated != nil {
				module.Children = append(module.Children, *decorated)
			}
		}
	}

	return module
}

// normalizeFunction converts a function_definition node
func (n *PythonNormalizer) normalizeFunction(node *sitter.Node, nestingDepth int) ASTNode {
	fn := ASTNode{
		Kind:         KindFunction,
		Language:     LangPython,
		NestingDepth: nestingDepth,
		Location:     n.locationFromNode(node),
	}

	// Extract function name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		fn.Name = n.nodeText(nameNode)
		fn.QualifiedName = fn.Name
		fn.ID = generateID(n.filePath + "." + fn.Name)
	}

	// Extract parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fn.Parameters = n.normalizeParameters(paramsNode)
		fn.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)
	}

	// Extract return type annotation
	returnNode := node.ChildByFieldName("return_type")
	if returnNode != nil {
		fn.ReturnType = n.normalizeTypeAnnotation(returnNode)
	} else {
		fn.ReturnType = UnknownType
	}

	// Check if async
	firstChild := node.Child(0)
	if firstChild != nil && firstChild.Kind() == "async" {
		fn.IsAsync = true
	}

	// Check if constructor
	if fn.Name == "__init__" {
		fn.IsConstructor = true
	}

	// Extract doc comment
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fn.DocComment = n.extractDocComment(bodyNode)
		fn.CyclomaticComplexity = n.computeComplexity(bodyNode)

		// Normalize body children (nested functions, classes)
		for i := range bodyNode.ChildCount() {
			child := bodyNode.Child(i)
			if child == nil {
				continue
			}
			if child.Kind() == "function_definition" {
				nested := n.normalizeFunction(child, nestingDepth+1)
				fn.Children = append(fn.Children, nested)
			}
		}
	}

	// Determine visibility
	fn.Visibility = n.pythonVisibility(fn.Name)

	return fn
}

// normalizeClass converts a class_definition node
func (n *PythonNormalizer) normalizeClass(node *sitter.Node) ASTNode {
	cls := ASTNode{
		Kind:     KindClass,
		Language: LangPython,
		Location: n.locationFromNode(node),
	}

	// Extract class name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		cls.Name = n.nodeText(nameNode)
		cls.QualifiedName = cls.Name
		cls.ID = generateID(n.filePath + "." + cls.Name)
	}

	// Extract body — methods and fields
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		cls.DocComment = n.extractDocComment(bodyNode)
		cls.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)

		for i := range bodyNode.ChildCount() {
			child := bodyNode.Child(i)
			if child == nil {
				continue
			}

			switch child.Kind() {
			case "function_definition":
				method := n.normalizeFunction(child, 1)
				method.ParentID = cls.ID
				cls.Children = append(cls.Children, method)

			case "decorated_definition":
				decorated := n.normalizeDecorated(child)
				if decorated != nil {
					decorated.ParentID = cls.ID
					cls.Children = append(cls.Children, *decorated)
				}

			case "expression_statement":
				field := n.normalizeField(child)
				if field != nil {
					field.ParentID = cls.ID
					cls.Children = append(cls.Children, *field)
				}
			}
		}
	}

	cls.Visibility = VisibilityPublic
	return cls
}

// normalizeImport converts import statements
func (n *PythonNormalizer) normalizeImport(node *sitter.Node) ASTNode {
	return ASTNode{
		ID:        generateID(fmt.Sprintf("%s.import.%d", n.filePath, node.StartPosition().Row)),
		Kind:      KindImport,
		Language:  LangPython,
		Location:  n.locationFromNode(node),
		RawSource: n.nodeText(node),
	}
}

// normalizeDecorated handles @decorator + function/class
func (n *PythonNormalizer) normalizeDecorated(node *sitter.Node) *ASTNode {
	var decorators []string

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "decorator":
			decorators = append(decorators, n.nodeText(child))

		case "function_definition":
			fn := n.normalizeFunction(child, 0)
			fn.Annotations = decorators
			return &fn

		case "class_definition":
			cls := n.normalizeClass(child)
			cls.Annotations = decorators
			return &cls
		}
	}

	return nil
}

// normalizeField extracts class-level variable assignments
func (n *PythonNormalizer) normalizeField(node *sitter.Node) *ASTNode {
	text := n.nodeText(node)
	if !strings.Contains(text, "=") {
		return nil
	}

	return &ASTNode{
		ID:        generateID(fmt.Sprintf("%s.field.%d", n.filePath, node.StartPosition().Row)),
		Kind:      KindField,
		Language:  LangPython,
		Location:  n.locationFromNode(node),
		RawSource: text,
	}
}

// normalizeParameters extracts function parameters
func (n *PythonNormalizer) normalizeParameters(node *sitter.Node) []Parameter {
	params := []Parameter{}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "identifier":
			name := n.nodeText(child)
			if name != "self" && name != "cls" {
				params = append(params, Parameter{
					Name: name,
					Type: UnknownType,
				})
			}

		case "typed_parameter":
			param := n.normalizeTypedParameter(child)
			if param.Name != "self" && param.Name != "cls" {
				params = append(params, param)
			}

		case "default_parameter":
			param := n.normalizeDefaultParameter(child)
			if param.Name != "self" && param.Name != "cls" {
				params = append(params, param)
			}

		case "list_splat_pattern", "dictionary_splat_pattern":
			nameChild := child.Child(1)
			if nameChild != nil {
				params = append(params, Parameter{
					Name:       n.nodeText(nameChild),
					Type:       UnknownType,
					IsVariadic: true,
				})
			}
		}
	}

	return params
}

// normalizeTypedParameter handles name: Type parameters
func (n *PythonNormalizer) normalizeTypedParameter(node *sitter.Node) Parameter {
	param := Parameter{Type: UnknownType}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		param.Name = n.nodeText(nameNode)
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		param.Type = n.normalizeTypeAnnotation(typeNode)
	}

	return param
}

// normalizeDefaultParameter handles name=default parameters
func (n *PythonNormalizer) normalizeDefaultParameter(node *sitter.Node) Parameter {
	param := Parameter{Type: UnknownType}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		param.Name = n.nodeText(nameNode)
	}

	valueNode := node.ChildByFieldName("value")
	if valueNode != nil {
		param.DefaultValue = n.nodeText(valueNode)
	}

	return param
}

// normalizeTypeAnnotation converts type annotation nodes
func (n *PythonNormalizer) normalizeTypeAnnotation(node *sitter.Node) TypeDescriptor {
	text := strings.TrimSpace(n.nodeText(node))

	// Remove leading colon or arrow if present
	text = strings.TrimPrefix(text, "->")
	text = strings.TrimPrefix(text, ":")
	text = strings.TrimSpace(text)

	switch text {
	case "str":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "str"}
	case "int":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "int"}
	case "float":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "float"}
	case "bool":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "bool"}
	case "None":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "None"}
	case "bytes":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "bytes"}
	default:
		// Generic types like List[str], Dict[str, int]
		if strings.Contains(text, "[") {
			name := text[:strings.Index(text, "[")]
			return TypeDescriptor{Kind: "GENERIC", Name: name, Inferred: false}
		}
		if text == "" {
			return UnknownType
		}
		return TypeDescriptor{Kind: "NAMED", Name: text, Inferred: false}
	}
}

// computeComplexity computes cyclomatic complexity of a function body
func (n *PythonNormalizer) computeComplexity(node *sitter.Node) int {
	complexity := 1 // Base complexity

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "if_statement", "elif_clause",
			"for_statement", "while_statement",
			"except_clause", "with_statement",
			"conditional_expression", "boolean_operator":
			complexity++
		}
		for i := range n.ChildCount() {
			child := n.Child(i)
			if child != nil {
				walk(child)
			}
		}
	}

	walk(node)
	return complexity
}

// extractDocComment extracts the first string expression as doc comment
func (n *PythonNormalizer) extractDocComment(bodyNode *sitter.Node) string {
	for i := range bodyNode.ChildCount() {
		child := bodyNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "expression_statement" {
			inner := child.Child(0)
			if inner != nil && (inner.Kind() == "string" || inner.Kind() == "concatenated_string") {
				text := n.nodeText(inner)
				text = strings.Trim(text, `"'`)
				text = strings.TrimPrefix(text, `"""`)
				text = strings.TrimSuffix(text, `"""`)
				return strings.TrimSpace(text)
			}
		}
		break
	}
	return ""
}

// pythonVisibility determines visibility from Python naming conventions
func (n *PythonNormalizer) pythonVisibility(name string) Visibility {
	if strings.HasPrefix(name, "__") && !strings.HasSuffix(name, "__") {
		return VisibilityPrivate
	}
	if strings.HasPrefix(name, "_") {
		return VisibilityInternal
	}
	return VisibilityPublic
}

// locationFromNode creates a SourceLocation from a tree-sitter node
func (n *PythonNormalizer) locationFromNode(node *sitter.Node) SourceLocation {
	return SourceLocation{
		File:      n.filePath,
		StartLine: node.StartPosition().Row,
		StartCol:  node.StartPosition().Column,
		EndLine:   node.EndPosition().Row,
		EndCol:    node.EndPosition().Column,
	}
}

// nodeText extracts the source text of a node
func (n *PythonNormalizer) nodeText(node *sitter.Node) string {
	return string(n.source[node.StartByte():node.EndByte()])
}

// ── HELPERS ───────────────────────────────────────────────

func generateID(seed string) string {
	// Simple deterministic ID from seed
	// In production this will be a content-addressed hash
	h := uint32(2166136261)
	for _, c := range seed {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}

func moduleNameFromPath(filePath string) string {
	parts := strings.Split(filePath, "/")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, ".py")
	name = strings.TrimSuffix(name, ".ts")
	name = strings.TrimSuffix(name, ".java")
	return name
}

func qualifiedNameFromPath(filePath string) string {
	result := strings.ReplaceAll(filePath, "/", ".")
	result = strings.TrimSuffix(result, ".py")
	result = strings.TrimSuffix(result, ".ts")
	result = strings.TrimSuffix(result, ".java")
	return result
}
