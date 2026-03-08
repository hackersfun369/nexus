package normalizer

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TypeScriptNormalizer converts TypeScript CST → Universal AST nodes
type TypeScriptNormalizer struct {
	source   []byte
	filePath string
}

// NewTypeScriptNormalizer creates a new TypeScript normalizer
func NewTypeScriptNormalizer(source []byte, filePath string) *TypeScriptNormalizer {
	return &TypeScriptNormalizer{
		source:   source,
		filePath: filePath,
	}
}

// Normalize converts the root CST node into a MODULE ASTNode
func (n *TypeScriptNormalizer) Normalize(root *sitter.Node) ASTNode {
	module := ASTNode{
		ID:            generateID(n.filePath),
		Kind:          KindModule,
		Language:      LangTypeScript,
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
		case "function_declaration":
			fn := n.normalizeFunction(child, 0)
			module.Children = append(module.Children, fn)

		case "class_declaration":
			cls := n.normalizeClass(child)
			module.Children = append(module.Children, cls)

		case "interface_declaration":
			iface := n.normalizeInterface(child)
			module.Children = append(module.Children, iface)

		case "import_statement":
			imp := n.normalizeImport(child)
			module.Children = append(module.Children, imp)

		case "export_statement":
			exported := n.normalizeExport(child)
			if exported != nil {
				module.Children = append(module.Children, *exported)
			}

		case "lexical_declaration", "variable_declaration":
			vars := n.normalizeVariableDeclaration(child)
			module.Children = append(module.Children, vars...)
		}
	}

	return module
}

// normalizeFunction converts function_declaration nodes
func (n *TypeScriptNormalizer) normalizeFunction(node *sitter.Node, nestingDepth int) ASTNode {
	fn := ASTNode{
		Kind:         KindFunction,
		Language:     LangTypeScript,
		NestingDepth: nestingDepth,
		Location:     n.locationFromNode(node),
		Visibility:   VisibilityPublic,
	}

	// Check if async — first child might be "async"
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "async" {
			fn.IsAsync = true
		}
	}

	// Extract name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		fn.Name = n.nodeText(nameNode)
		fn.QualifiedName = fn.Name
		fn.ID = generateID(n.filePath + "." + fn.Name)
	}

	// Extract type parameters (generics)
	typeParamsNode := node.ChildByFieldName("type_parameters")
	if typeParamsNode != nil {
		fn.Type = TypeDescriptor{Kind: "GENERIC", Name: fn.Name}
	}

	// Extract parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		fn.Parameters = n.normalizeParameters(paramsNode)
	}

	// Extract return type
	returnTypeNode := node.ChildByFieldName("return_type")
	if returnTypeNode != nil {
		fn.ReturnType = n.normalizeTypeAnnotation(returnTypeNode)
	} else {
		fn.ReturnType = UnknownType
	}

	// Extract body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		fn.DocComment = n.extractDocComment(bodyNode)
		fn.CyclomaticComplexity = n.computeComplexity(bodyNode)
		fn.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)

		// Nested functions
		for i := range bodyNode.ChildCount() {
			child := bodyNode.Child(i)
			if child == nil {
				continue
			}
			if child.Kind() == "function_declaration" ||
				child.Kind() == "lexical_declaration" {
				nested := n.normalizeFunction(child, nestingDepth+1)
				fn.Children = append(fn.Children, nested)
			}
		}
	}

	return fn
}

// normalizeClass converts class_declaration nodes
func (n *TypeScriptNormalizer) normalizeClass(node *sitter.Node) ASTNode {
	cls := ASTNode{
		Kind:       KindClass,
		Language:   LangTypeScript,
		Location:   n.locationFromNode(node),
		Visibility: VisibilityPublic,
	}

	// Extract name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		cls.Name = n.nodeText(nameNode)
		cls.QualifiedName = cls.Name
		cls.ID = generateID(n.filePath + "." + cls.Name)
	}

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
			case "method_definition":
				method := n.normalizeMethod(child)
				method.ParentID = cls.ID
				cls.Children = append(cls.Children, method)

			case "public_field_definition",
				"private_field_definition":
				field := n.normalizeClassField(child)
				field.ParentID = cls.ID
				cls.Children = append(cls.Children, field)
			}
		}
	}

	return cls
}

// normalizeInterface converts interface_declaration nodes
func (n *TypeScriptNormalizer) normalizeInterface(node *sitter.Node) ASTNode {
	iface := ASTNode{
		Kind:       KindInterface,
		Language:   LangTypeScript,
		Location:   n.locationFromNode(node),
		Visibility: VisibilityPublic,
	}

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
			case "property_signature":
				field := n.normalizePropertySignature(child)
				field.ParentID = iface.ID
				iface.Children = append(iface.Children, field)

			case "method_signature":
				method := n.normalizeMethodSignature(child)
				method.ParentID = iface.ID
				iface.Children = append(iface.Children, method)
			}
		}
	}

	return iface
}

// normalizeMethod converts method_definition nodes inside classes
func (n *TypeScriptNormalizer) normalizeMethod(node *sitter.Node) ASTNode {
	method := ASTNode{
		Kind:     KindFunction,
		Language: LangTypeScript,
		Location: n.locationFromNode(node),
	}

	// Extract name
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		method.Name = n.nodeText(nameNode)
		method.QualifiedName = method.Name
		method.ID = generateID(n.filePath + "." + method.Name)
	}

	// Check constructor
	if method.Name == "constructor" {
		method.IsConstructor = true
	}

	// Check static/async modifiers
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "static":
			method.IsStatic = true
		case "async":
			method.IsAsync = true
		case "abstract":
			method.IsAbstract = true
		}
	}

	// Visibility from accessibility modifier
	method.Visibility = n.extractVisibilityModifier(node)

	// Parameters
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		method.Parameters = n.normalizeParameters(paramsNode)
	}

	// Return type
	returnTypeNode := node.ChildByFieldName("return_type")
	if returnTypeNode != nil {
		method.ReturnType = n.normalizeTypeAnnotation(returnTypeNode)
	} else {
		method.ReturnType = UnknownType
	}

	// Body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		method.CyclomaticComplexity = n.computeComplexity(bodyNode)
		method.LinesOfCode = int(node.EndPosition().Row - node.StartPosition().Row)
	}

	return method
}

// normalizeClassField converts field definitions
func (n *TypeScriptNormalizer) normalizeClassField(node *sitter.Node) ASTNode {
	field := ASTNode{
		Kind:     KindField,
		Language: LangTypeScript,
		Location: n.locationFromNode(node),
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		field.Name = n.nodeText(nameNode)
		field.ID = generateID(n.filePath + ".field." + field.Name)
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		field.Type = n.normalizeTypeAnnotation(typeNode)
	} else {
		field.Type = UnknownType
	}

	field.Visibility = n.extractVisibilityModifier(node)
	return field
}

// normalizePropertySignature converts interface property signatures
func (n *TypeScriptNormalizer) normalizePropertySignature(node *sitter.Node) ASTNode {
	field := ASTNode{
		Kind:       KindField,
		Language:   LangTypeScript,
		Location:   n.locationFromNode(node),
		Visibility: VisibilityPublic,
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		field.Name = n.nodeText(nameNode)
		field.ID = generateID(n.filePath + ".prop." + field.Name)
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		field.Type = n.normalizeTypeAnnotation(typeNode)
	} else {
		field.Type = UnknownType
	}

	// Check optional (?)
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child != nil && child.Kind() == "?" {
			field.Type.Nullable = true
		}
	}

	return field
}

// normalizeMethodSignature converts interface method signatures
func (n *TypeScriptNormalizer) normalizeMethodSignature(node *sitter.Node) ASTNode {
	method := ASTNode{
		Kind:       KindFunction,
		Language:   LangTypeScript,
		Location:   n.locationFromNode(node),
		Visibility: VisibilityPublic,
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		method.Name = n.nodeText(nameNode)
		method.ID = generateID(n.filePath + ".method." + method.Name)
	}

	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		method.Parameters = n.normalizeParameters(paramsNode)
	}

	returnTypeNode := node.ChildByFieldName("return_type")
	if returnTypeNode != nil {
		method.ReturnType = n.normalizeTypeAnnotation(returnTypeNode)
	} else {
		method.ReturnType = UnknownType
	}

	return method
}

// normalizeImport converts import statements
func (n *TypeScriptNormalizer) normalizeImport(node *sitter.Node) ASTNode {
	return ASTNode{
		ID:        generateID(fmt.Sprintf("%s.import.%d", n.filePath, node.StartPosition().Row)),
		Kind:      KindImport,
		Language:  LangTypeScript,
		Location:  n.locationFromNode(node),
		RawSource: n.nodeText(node),
	}
}

// normalizeExport handles export statements
func (n *TypeScriptNormalizer) normalizeExport(node *sitter.Node) *ASTNode {
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "function_declaration":
			fn := n.normalizeFunction(child, 0)
			fn.Visibility = VisibilityPublic
			return &fn

		case "class_declaration":
			cls := n.normalizeClass(child)
			cls.Visibility = VisibilityPublic
			return &cls

		case "interface_declaration":
			iface := n.normalizeInterface(child)
			iface.Visibility = VisibilityPublic
			return &iface

		case "lexical_declaration", "variable_declaration":
			vars := n.normalizeVariableDeclaration(child)
			if len(vars) > 0 {
				return &vars[0]
			}
		}
	}
	return nil
}

// normalizeVariableDeclaration converts const/let/var declarations
func (n *TypeScriptNormalizer) normalizeVariableDeclaration(node *sitter.Node) []ASTNode {
	vars := []ASTNode{}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		if child.Kind() == "variable_declarator" {
			v := ASTNode{
				Kind:       KindVariable,
				Language:   LangTypeScript,
				Location:   n.locationFromNode(child),
				Visibility: VisibilityPublic,
			}

			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				v.Name = n.nodeText(nameNode)
				v.ID = generateID(n.filePath + ".var." + v.Name)
			}

			typeNode := child.ChildByFieldName("type")
			if typeNode != nil {
				v.Type = n.normalizeTypeAnnotation(typeNode)
			} else {
				v.Type = UnknownType
			}

			vars = append(vars, v)
		}
	}

	return vars
}

// normalizeParameters extracts function parameters
func (n *TypeScriptNormalizer) normalizeParameters(node *sitter.Node) []Parameter {
	params := []Parameter{}

	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "required_parameter":
			param := n.normalizeRequiredParameter(child)
			params = append(params, param)

		case "optional_parameter":
			param := n.normalizeOptionalParameter(child)
			params = append(params, param)

		case "rest_pattern":
			nameNode := child.Child(1)
			if nameNode != nil {
				params = append(params, Parameter{
					Name:       n.nodeText(nameNode),
					Type:       UnknownType,
					IsVariadic: true,
				})
			}
		}
	}

	return params
}

// normalizeRequiredParameter handles name: Type parameters
func (n *TypeScriptNormalizer) normalizeRequiredParameter(node *sitter.Node) Parameter {
	param := Parameter{Type: UnknownType}

	patternNode := node.ChildByFieldName("pattern")
	if patternNode != nil {
		param.Name = n.nodeText(patternNode)
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		param.Type = n.normalizeTypeAnnotation(typeNode)
	}

	return param
}

// normalizeOptionalParameter handles name?: Type parameters
func (n *TypeScriptNormalizer) normalizeOptionalParameter(node *sitter.Node) Parameter {
	param := n.normalizeRequiredParameter(node)
	param.Type.Nullable = true
	return param
}

// normalizeTypeAnnotation converts TypeScript type annotations
func (n *TypeScriptNormalizer) normalizeTypeAnnotation(node *sitter.Node) TypeDescriptor {
	text := strings.TrimSpace(n.nodeText(node))
	text = strings.TrimPrefix(text, ":")
	text = strings.TrimPrefix(text, "->")
	text = strings.TrimSpace(text)

	switch text {
	case "string":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "string"}
	case "number":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "number"}
	case "boolean":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "boolean"}
	case "void":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "void"}
	case "null":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "null", Nullable: true}
	case "undefined":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "undefined", Nullable: true}
	case "any":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "any"}
	case "never":
		return TypeDescriptor{Kind: "PRIMITIVE", Name: "never"}
	case "unknown":
		return TypeDescriptor{Kind: "UNKNOWN", Name: "unknown"}
	default:
		// Array types: string[] or Array<string>
		if strings.HasSuffix(text, "[]") {
			inner := strings.TrimSuffix(text, "[]")
			return TypeDescriptor{
				Kind: "ARRAY",
				Name: inner + "[]",
				TypeArgs: []TypeDescriptor{
					{Kind: "NAMED", Name: inner},
				},
			}
		}
		// Generic types: Promise<T>, Array<T>
		if strings.Contains(text, "<") {
			name := text[:strings.Index(text, "<")]
			return TypeDescriptor{Kind: "GENERIC", Name: name}
		}
		// Union types: string | number
		if strings.Contains(text, "|") {
			return TypeDescriptor{Kind: "UNION", Name: text}
		}
		if text == "" {
			return UnknownType
		}
		return TypeDescriptor{Kind: "NAMED", Name: text}
	}
}

// computeComplexity computes cyclomatic complexity
func (n *TypeScriptNormalizer) computeComplexity(node *sitter.Node) int {
	complexity := 1

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Kind() {
		case "if_statement", "else_clause",
			"for_statement", "for_in_statement",
			"while_statement", "do_statement",
			"catch_clause", "conditional_expression",
			"binary_expression", "switch_case":
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

// extractDocComment extracts JSDoc comments
func (n *TypeScriptNormalizer) extractDocComment(bodyNode *sitter.Node) string {
	for i := range bodyNode.ChildCount() {
		child := bodyNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "comment" {
			text := n.nodeText(child)
			if strings.HasPrefix(text, "/**") {
				text = strings.TrimPrefix(text, "/**")
				text = strings.TrimSuffix(text, "*/")
				return strings.TrimSpace(text)
			}
		}
		break
	}
	return ""
}

// extractVisibilityModifier extracts public/private/protected
// extractVisibilityModifier extracts public/private/protected
func (n *TypeScriptNormalizer) extractVisibilityModifier(node *sitter.Node) Visibility {
	for i := range node.ChildCount() {
		child := node.Child(i)
		if child == nil {
			continue
		}
		// Direct modifier keyword
		switch child.Kind() {
		case "private":
			return VisibilityPrivate
		case "protected":
			return VisibilityProtected
		case "public":
			return VisibilityPublic
		}
		// Wrapped inside accessibility_modifier node
		if child.Kind() == "accessibility_modifier" {
			text := strings.TrimSpace(n.nodeText(child))
			switch text {
			case "private":
				return VisibilityPrivate
			case "protected":
				return VisibilityProtected
			case "public":
				return VisibilityPublic
			}
		}
	}
	return VisibilityPublic
}

// locationFromNode creates a SourceLocation from a node
func (n *TypeScriptNormalizer) locationFromNode(node *sitter.Node) SourceLocation {
	return SourceLocation{
		File:      n.filePath,
		StartLine: node.StartPosition().Row,
		StartCol:  node.StartPosition().Column,
		EndLine:   node.EndPosition().Row,
		EndCol:    node.EndPosition().Column,
	}
}

// nodeText extracts the source text of a node
func (n *TypeScriptNormalizer) nodeText(node *sitter.Node) string {
	return string(n.source[node.StartByte():node.EndByte()])
}
