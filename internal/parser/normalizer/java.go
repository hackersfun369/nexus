package normalizer

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// JavaNormalizer converts Java CST → Universal AST nodes
type JavaNormalizer struct {
	source   []byte
	filePath string
}

// NewJavaNormalizer creates a new Java normalizer
func NewJavaNormalizer(source []byte, filePath string) *JavaNormalizer {
	return &JavaNormalizer{
		source:   source,
		filePath: filePath,
	}
}

// Normalize converts the root CST node into a MODULE ASTNode
func (n *JavaNormalizer) Normalize(root *sitter.Node) ASTNode {
	module := ASTNode{
		ID:            generateID(n.filePath),
		Kind:          KindModule,
		Language:      LangJava,
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

	for i := range root.ChildCount() {
		child := root.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "class_declaration":
			cls := n.normalizeClass(child)
			module.Children = append(module.Children, cls)

		case "interface_declaration":
			iface := n.normalizeInterface(child)
			module.Children = append(module.Children, iface)

		case "enum_declaration":
			enum := n.normalizeEnum(child)
			module.Children = append(module.Children, enum)

		case "import_declaration":
			imp := n.normalizeImport(child)
			module.Children = append(module.Children, imp)

		case "package_declaration":
			module.QualifiedName = n.extractPackageName(child)
		}
	}

	return module
}

// normalizeClass converts class_declaration nodes
func (n *JavaNormalizer) normalizeClass(node *sitter.Node) ASTNode {
	cls := ASTNode{
		Kind:     KindClass,
		Language: LangJava,
		Location: n.locationFromNode(node),
	}

	// Extract modifiers (public, abstract, final)
	cls.Visibility, cls.IsAbstract = n.extractModifiers(node)

	// Extract name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		cls.Name = n.nodeText(nameNode)
		cls.QualifiedName = cls.Name
		cls.ID = generateID(n.filePath + "." + cls.Name)
	}

	// Extract annotations
	cls.Annotations = n.extractAnnotations(node)

	// Extract body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		cls.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)

		for i := range bodyNode.ChildCount() {
			child := bodyNode.Child(i)
			if child == nil {
				continue
			}

			switch child.Kind() {
			case "method_declaration":
				method := n.normalizeMethod(child)
				method.ParentID = cls.ID
				cls.Children = append(cls.Children, method)

			case "constructor_declaration":
				ctor := n.normalizeConstructor(child)
				ctor.ParentID = cls.ID
				cls.Children = append(cls.Children, ctor)

			case "field_declaration":
				fields := n.normalizeField(child)
				for _, f := range fields {
					f.ParentID = cls.ID
					cls.Children = append(cls.Children, f)
				}

			case "class_declaration":
				// Inner class
				inner := n.normalizeClass(child)
				inner.ParentID = cls.ID
				cls.Children = append(cls.Children, inner)
			}
		}
	}

	return cls
}

// normalizeInterface converts interface_declaration nodes
func (n *JavaNormalizer) normalizeInterface(node *sitter.Node) ASTNode {
	iface := ASTNode{
		Kind:     KindInterface,
		Language: LangJava,
		Location: n.locationFromNode(node),
	}

	iface.Visibility, _ = n.extractModifiers(node)
	iface.Annotations = n.extractAnnotations(node)

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		iface.Name = n.nodeText(nameNode)
		iface.QualifiedName = iface.Name
		iface.ID = generateID(n.filePath + "." + iface.Name)
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		for i := range bodyNode.ChildCount() {
			child := bodyNode.Child(i)
			if child == nil {
				continue
			}

			switch child.Kind() {
			case "method_declaration":
				method := n.normalizeMethod(child)
				method.ParentID = iface.ID
				iface.Children = append(iface.Children, method)

			case "constant_declaration":
				fields := n.normalizeField(child)
				for _, f := range fields {
					f.ParentID = iface.ID
					iface.Children = append(iface.Children, f)
				}
			}
		}
	}

	return iface
}

// normalizeEnum converts enum_declaration nodes
func (n *JavaNormalizer) normalizeEnum(node *sitter.Node) ASTNode {
	enum := ASTNode{
		Kind:     KindClass, // Enums are a special kind of class
		Language: LangJava,
		Location: n.locationFromNode(node),
	}

	enum.Visibility, _ = n.extractModifiers(node)

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		enum.Name = n.nodeText(nameNode)
		enum.QualifiedName = enum.Name
		enum.ID = generateID(n.filePath + "." + enum.Name)
	}

	enum.Annotations = n.extractAnnotations(node)
	return enum
}

// normalizeMethod converts method_declaration nodes
func (n *JavaNormalizer) normalizeMethod(node *sitter.Node) ASTNode {
	method := ASTNode{
		Kind:     KindFunction,
		Language: LangJava,
		Location: n.locationFromNode(node),
	}

	// Modifiers
	method.Visibility, method.IsAbstract = n.extractModifiers(node)
	method.IsStatic = n.hasModifier(node, "static")

	// Annotations
	method.Annotations = n.extractAnnotations(node)

	// Return type
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		method.ReturnType = n.normalizeType(typeNode)
	} else {
		method.ReturnType = UnknownType
	}

	// Name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		method.Name = n.nodeText(nameNode)
		method.QualifiedName = method.Name
		method.ID = generateID(n.filePath + "." + method.Name)
	}

	// Parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		method.Parameters = n.normalizeParameters(paramsNode)
	}

	// Body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		method.CyclomaticComplexity = n.computeComplexity(bodyNode)
		method.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)
	}

	return method
}

// normalizeConstructor converts constructor_declaration nodes
func (n *JavaNormalizer) normalizeConstructor(node *sitter.Node) ASTNode {
	ctor := ASTNode{
		Kind:          KindFunction,
		Language:      LangJava,
		Location:      n.locationFromNode(node),
		IsConstructor: true,
	}

	ctor.Visibility, _ = n.extractModifiers(node)
	ctor.Annotations = n.extractAnnotations(node)

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		ctor.Name = n.nodeText(nameNode)
		ctor.QualifiedName = ctor.Name
		ctor.ID = generateID(n.filePath + ".ctor." + ctor.Name)
	}

	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		ctor.Parameters = n.normalizeParameters(paramsNode)
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		ctor.CyclomaticComplexity = n.computeComplexity(bodyNode)
		ctor.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)
	}

	ctor.ReturnType = TypeDescriptor{Kind: "PRIMITIVE", Name: "void"}
	return ctor
}

// normalizeField converts field_declaration nodes
func (n *JavaNormalizer) normalizeField(node *sitter.Node) []ASTNode {
	fields := []ASTNode{}

	visibility, _ := n.extractModifiers(node)

	// Get type
	typeNode := node.ChildByFieldName("type")
	fieldType := UnknownType
	if typeNode != nil {
		fieldType = n.normalizeType(typeNode)
	}

	// Get declarators (can be multiple: int x, y, z)
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Kind() == "variable_declarator" {
			field := ASTNode{
				Kind:       KindField,
				Language:   LangJava,
				Location:   n.locationFromNode(child),
				Type:       fieldType,
				Visibility: visibility,
			}

			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				field.Name = n.nodeText(nameNode)
				field.ID = generateID(fmt.Sprintf("%s.field.%s", n.filePath, field.Name))
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// normalizeParameters extracts method parameters
func (n *JavaNormalizer) normalizeParameters(node *sitter.Node) []Parameter {
	params := []Parameter{}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "formal_parameter":
			param := n.normalizeFormalParameter(child)
			params = append(params, param)

		case "spread_parameter":
			param := n.normalizeSpreadParameter(child)
			params = append(params, param)
		}
	}

	return params
}

// normalizeFormalParameter handles Type name parameters
func (n *JavaNormalizer) normalizeFormalParameter(node *sitter.Node) Parameter {
	param := Parameter{Type: UnknownType}

	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		param.Type = n.normalizeType(typeNode)
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		param.Name = n.nodeText(nameNode)
	}

	return param
}

// normalizeSpreadParameter handles Type... name parameters
func (n *JavaNormalizer) normalizeSpreadParameter(node *sitter.Node) Parameter {
	param := Parameter{Type: UnknownType, IsVariadic: true}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "variable_declarator" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				param.Name = n.nodeText(nameNode)
			}
		}
	}

	return param
}

// normalizeType converts Java type nodes to TypeDescriptor
func (n *JavaNormalizer) normalizeType(node *sitter.Node) TypeDescriptor {
	text := strings.TrimSpace(n.nodeText(node))

	switch text {
	case "int", "long", "short", "byte",
		"float", "double", "boolean", "char":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: text}
	case "void":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "void"}
	case "String":
		return TypeDescriptor{Kind: "NAMED", Name: "String"}
	default:
		// Array types: String[], int[]
		if strings.HasSuffix(text, "[]") {
			inner := strings.TrimSuffix(text, "[]")
			return TypeDescriptor{
				Kind: "ARRAY",
				Name: text,
				TypeArgs: []TypeDescriptor{
					{Kind: "NAMED", Name: inner},
				},
			}
		}
		// Generic types: List<String>, Map<K,V>
		if strings.Contains(text, "<") {
			name := text[:strings.Index(text, "<")]
			return TypeDescriptor{Kind: "GENERIC", Name: name}
		}
		if text == "" {
			return UnknownType
		}
		return TypeDescriptor{Kind: "NAMED", Name: text}
	}
}

// normalizeImport converts import_declaration nodes
func (n *JavaNormalizer) normalizeImport(node *sitter.Node) ASTNode {
	return ASTNode{
		ID:        generateID(fmt.Sprintf("%s.import.%d", n.filePath, node.StartPosition().Row)),
		Kind:      KindImport,
		Language:  LangJava,
		Location:  n.locationFromNode(node),
		RawSource: n.nodeText(node),
	}
}

// computeComplexity computes cyclomatic complexity for Java
func (n *JavaNormalizer) computeComplexity(node *sitter.Node) int {
	complexity := 1

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "if_statement", "else_clause",
			"for_statement", "enhanced_for_statement",
			"while_statement", "do_statement",
			"catch_clause", "conditional_expression",
			"switch_label", "binary_expression",
			"ternary_expression":
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

// extractModifiers extracts visibility and abstract modifier
func (n *JavaNormalizer) extractModifiers(node *sitter.Node) (Visibility, bool) {
	visibility := VisibilityInternal // Java default (package-private)
	isAbstract := false

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Kind() == "modifiers" {
			text := n.nodeText(child)
			if strings.Contains(text, "public") {
				visibility = VisibilityPublic
			} else if strings.Contains(text, "private") {
				visibility = VisibilityPrivate
			} else if strings.Contains(text, "protected") {
				visibility = VisibilityProtected
			}
			isAbstract = strings.Contains(text, "abstract")
		}
	}

	return visibility, isAbstract
}

// hasModifier checks if a node has a specific modifier
func (n *JavaNormalizer) hasModifier(node *sitter.Node, modifier string) bool {
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "modifiers" {
			return strings.Contains(n.nodeText(child), modifier)
		}
	}
	return false
}

// extractAnnotations extracts @Annotation names from a node
func (n *JavaNormalizer) extractAnnotations(node *sitter.Node) []string {
	annotations := []string{}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "modifiers" {
			for j := range child.ChildCount() {
				mod := child.Child(j)
				if mod == nil {
					continue
				}
				if mod.Kind() == "marker_annotation" ||
					mod.Kind() == "annotation" {
					annotations = append(annotations, n.nodeText(mod))
				}
			}
		}
	}

	return annotations
}

// extractPackageName extracts the package name from package_declaration
func (n *JavaNormalizer) extractPackageName(node *sitter.Node) string {
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child != nil && child.Kind() == "scoped_identifier" {
			return n.nodeText(child)
		}
		if child != nil && child.Kind() == "identifier" {
			return n.nodeText(child)
		}
	}
	return ""
}

// locationFromNode creates a SourceLocation from a node
func (n *JavaNormalizer) locationFromNode(node *sitter.Node) SourceLocation {
	return SourceLocation{
		File:      n.filePath,
		StartLine: node.StartPosition().Row,
		StartCol:  node.StartPosition().Column,
		EndLine:   node.EndPosition().Row,
		EndCol:    node.EndPosition().Column,
	}
}

// nodeText extracts the source text of a node
func (n *JavaNormalizer) nodeText(node *sitter.Node) string {
	return string(n.source[node.StartByte():node.EndByte()])
}
