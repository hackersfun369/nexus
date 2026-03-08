package extractor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hackersfun369/nexus/internal/parser/normalizer"
)

// Builder converts a normalized ASTNode tree into a GraphDelta
type Builder struct {
	projectID string

	// Existing node checksums from the graph
	// Used to detect what actually changed
	existingChecksums map[string]string

	// Existing edges from the graph
	// Used to detect new/removed edges
	existingEdges map[string]bool
}

// NewBuilder creates a new delta builder
func NewBuilder(projectID string) *Builder {
	return &Builder{
		projectID:         projectID,
		existingChecksums: make(map[string]string),
		existingEdges:     make(map[string]bool),
	}
}

// WithExistingChecksums provides existing node checksums
// so the builder can detect modifications vs additions
func (b *Builder) WithExistingChecksums(checksums map[string]string) *Builder {
	b.existingChecksums = checksums
	return b
}

// WithExistingEdges provides existing edge IDs
func (b *Builder) WithExistingEdges(edges map[string]bool) *Builder {
	b.existingEdges = edges
	return b
}

// Build converts a module ASTNode into a GraphDelta
func (b *Builder) Build(module normalizer.ASTNode) GraphDelta {
	delta := GraphDelta{
		ProjectID: b.projectID,
		FilePath:  module.Location.File,
		AppliedAt: time.Now(),
	}

	// Track which node IDs we saw in this parse
	// Any existing node NOT seen = deleted
	seenNodeIDs := make(map[string]bool)

	// Process module node itself
	moduleNode := b.buildModuleNode(module)
	b.applyNodeChange(&delta, moduleNode, seenNodeIDs)

	// Process all children recursively
	b.processChildren(&delta, module, moduleNode.ID, seenNodeIDs)

	// Mark nodes as deleted if they existed before but not seen now
	for existingID := range b.existingChecksums {
		if !seenNodeIDs[existingID] {
			delta.DeleteNode(existingID)
		}
	}

	return delta
}

// processChildren recursively processes child nodes
func (b *Builder) processChildren(
	delta *GraphDelta,
	parent normalizer.ASTNode,
	parentNodeID string,
	seen map[string]bool,
) {
	for _, child := range parent.Children {
		var childNode GraphNode

		switch child.Kind {
		case normalizer.KindFunction:
			childNode = b.buildFunctionNode(child)
		case normalizer.KindClass:
			childNode = b.buildClassNode(child)
		case normalizer.KindInterface:
			childNode = b.buildInterfaceNode(child)
		case normalizer.KindField:
			childNode = b.buildFieldNode(child)
		case normalizer.KindImport:
			childNode = b.buildImportNode(child)
		default:
			continue
		}

		b.applyNodeChange(delta, childNode, seen)

		// Add CONTAINS edge from parent → child
		containsEdge := GraphEdge{
			ID:         edgeID(EdgeContains, parentNodeID, childNode.ID),
			Kind:       EdgeContains,
			FromNodeID: parentNodeID,
			ToNodeID:   childNode.ID,
			ProjectID:  b.projectID,
		}
		b.applyEdgeChange(delta, containsEdge)

		// Recurse into children
		if len(child.Children) > 0 {
			b.processChildren(delta, child, childNode.ID, seen)
		}
	}
}

// applyNodeChange decides ADD vs MODIFY based on existing checksums
func (b *Builder) applyNodeChange(
	delta *GraphDelta,
	node GraphNode,
	seen map[string]bool,
) {
	// Compute checksum
	node.Checksum = computeChecksum(node)
	seen[node.ID] = true

	existingChecksum, exists := b.existingChecksums[node.ID]

	if !exists {
		// New node
		node.Version = 1
		node.CreatedAt = time.Now()
		node.UpdatedAt = time.Now()
		delta.AddNode(node)
	} else if existingChecksum != node.Checksum {
		// Modified node
		node.UpdatedAt = time.Now()
		delta.ModifyNode(node, detectChangedFields(node))
	}
	// Unchanged — don't include in delta
}

// applyEdgeChange decides ADD vs skip based on existing edges
func (b *Builder) applyEdgeChange(delta *GraphDelta, edge GraphEdge) {
	if !b.existingEdges[edge.ID] {
		delta.AddEdge(edge)
	}
}

// ── NODE BUILDERS ─────────────────────────────────────

func (b *Builder) buildModuleNode(n normalizer.ASTNode) GraphNode {
	node := b.baseNode(n)
	node.Kind = NodeModule
	node.Properties = map[string]interface{}{
		"linesOfCode": n.LinesOfCode,
		"docComment":  n.DocComment,
	}
	return node
}

func (b *Builder) buildFunctionNode(n normalizer.ASTNode) GraphNode {
	node := b.baseNode(n)
	node.Kind = NodeFunction

	params := []map[string]interface{}{}
	for _, p := range n.Parameters {
		params = append(params, map[string]interface{}{
			"name":       p.Name,
			"typeName":   p.Type.Name,
			"typeKind":   p.Type.Kind,
			"isVariadic": p.IsVariadic,
		})
	}

	node.Properties = map[string]interface{}{
		"isAsync":              n.IsAsync,
		"isStatic":             n.IsStatic,
		"isAbstract":           n.IsAbstract,
		"isConstructor":        n.IsConstructor,
		"visibility":           string(n.Visibility),
		"returnTypeName":       n.ReturnType.Name,
		"returnTypeKind":       n.ReturnType.Kind,
		"cyclomaticComplexity": n.CyclomaticComplexity,
		"nestingDepth":         n.NestingDepth,
		"linesOfCode":          n.LinesOfCode,
		"parameterCount":       len(n.Parameters),
		"parameters":           params,
		"annotations":          n.Annotations,
		"docComment":           n.DocComment,
	}
	return node
}

func (b *Builder) buildClassNode(n normalizer.ASTNode) GraphNode {
	node := b.baseNode(n)
	node.Kind = NodeClass
	node.Properties = map[string]interface{}{
		"visibility":  string(n.Visibility),
		"isAbstract":  n.IsAbstract,
		"linesOfCode": n.LinesOfCode,
		"annotations": n.Annotations,
		"docComment":  n.DocComment,
	}
	return node
}

func (b *Builder) buildInterfaceNode(n normalizer.ASTNode) GraphNode {
	node := b.baseNode(n)
	node.Kind = NodeInterface
	node.Properties = map[string]interface{}{
		"visibility":  string(n.Visibility),
		"annotations": n.Annotations,
		"docComment":  n.DocComment,
	}
	return node
}

func (b *Builder) buildFieldNode(n normalizer.ASTNode) GraphNode {
	node := b.baseNode(n)
	node.Kind = NodeField
	node.Properties = map[string]interface{}{
		"typeName":   n.Type.Name,
		"typeKind":   n.Type.Kind,
		"visibility": string(n.Visibility),
	}
	return node
}

func (b *Builder) buildImportNode(n normalizer.ASTNode) GraphNode {
	node := b.baseNode(n)
	node.Kind = NodeImport
	node.Properties = map[string]interface{}{
		"rawSource": n.RawSource,
	}
	return node
}

// baseNode creates a GraphNode from common ASTNode fields
func (b *Builder) baseNode(n normalizer.ASTNode) GraphNode {
	id := n.ID
	if id == "" {
		id = generateFallbackID(b.projectID, n)
	}

	return GraphNode{
		ID:            id,
		ProjectID:     b.projectID,
		Name:          n.Name,
		QualifiedName: n.QualifiedName,
		Language:      string(n.Language),
		FilePath:      n.Location.File,
		StartLine:     n.Location.StartLine,
		StartCol:      n.Location.StartCol,
		EndLine:       n.Location.EndLine,
		EndCol:        n.Location.EndCol,
	}
}

// detectChangedFields returns which properties likely changed
// In production this would do field-by-field comparison
func detectChangedFields(node GraphNode) []string {
	return []string{"properties", "checksum", "updatedAt"}
}

// generateFallbackID generates an ID when ASTNode has none
func generateFallbackID(projectID string, n normalizer.ASTNode) string {
	seed := fmt.Sprintf("%s:%s:%s:%d",
		projectID,
		n.Location.File,
		n.Name,
		n.Location.StartLine,
	)
	hash := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(hash[:8])
}
