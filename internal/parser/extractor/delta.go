package extractor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// NodeKind mirrors normalizer.NodeKind for the graph layer
type NodeKind string

const (
	NodeModule    NodeKind = "MODULE"
	NodeFunction  NodeKind = "FUNCTION"
	NodeClass     NodeKind = "CLASS"
	NodeInterface NodeKind = "INTERFACE"
	NodeVariable  NodeKind = "VARIABLE"
	NodeField     NodeKind = "FIELD"
	NodeImport    NodeKind = "IMPORT"
)

// EdgeKind represents a relationship between nodes
type EdgeKind string

const (
	EdgeContains  EdgeKind = "CONTAINS"
	EdgeImports   EdgeKind = "IMPORTS"
	EdgeInherits  EdgeKind = "INHERITS"
	EdgeCalls     EdgeKind = "CALLS"
	EdgeUsesType  EdgeKind = "USES_TYPE"
	EdgeOverrides EdgeKind = "OVERRIDES"
)

// ChangeKind represents the type of change
type ChangeKind string

const (
	ChangeAdd    ChangeKind = "ADD"
	ChangeModify ChangeKind = "MODIFY"
	ChangeDelete ChangeKind = "DELETE"
)

// GraphNode represents a node to be written to the graph
type GraphNode struct {
	ID            string
	Kind          NodeKind
	ProjectID     string
	Name          string
	QualifiedName string
	FilePath      string
	StartLine     uint
	StartCol      uint
	EndLine       uint
	EndCol        uint
	Language      string
	Checksum      string
	Version       int
	IsDeleted     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// Kind-specific properties stored as JSON
	Properties map[string]interface{}
}

// GraphEdge represents an edge to be written to the graph
type GraphEdge struct {
	ID         string
	Kind       EdgeKind
	FromNodeID string
	ToNodeID   string
	ProjectID  string
	Properties map[string]interface{}
}

// NodeChange represents a single node change in the delta
type NodeChange struct {
	Kind ChangeKind
	Node GraphNode
	// For MODIFY: which fields changed
	ChangedFields []string
}

// EdgeChange represents a single edge change in the delta
type EdgeChange struct {
	Kind ChangeKind
	Edge GraphEdge
}

// GraphDelta is the complete set of changes to apply atomically
type GraphDelta struct {
	ProjectID string
	FilePath  string
	AppliedAt time.Time

	// Node changes
	NodeChanges []NodeChange

	// Edge changes
	EdgeChanges []EdgeChange

	// Summary
	NodesAdded    int
	NodesModified int
	NodesDeleted  int
	EdgesAdded    int
	EdgesDeleted  int
}

// IsEmpty returns true if the delta has no changes
func (d *GraphDelta) IsEmpty() bool {
	return len(d.NodeChanges) == 0 && len(d.EdgeChanges) == 0
}

// AddNode adds a node addition to the delta
func (d *GraphDelta) AddNode(node GraphNode) {
	d.NodeChanges = append(d.NodeChanges, NodeChange{
		Kind: ChangeAdd,
		Node: node,
	})
	d.NodesAdded++
}

// ModifyNode adds a node modification to the delta
func (d *GraphDelta) ModifyNode(node GraphNode, changedFields []string) {
	d.NodeChanges = append(d.NodeChanges, NodeChange{
		Kind:          ChangeModify,
		Node:          node,
		ChangedFields: changedFields,
	})
	d.NodesModified++
}

// DeleteNode adds a node deletion to the delta
func (d *GraphDelta) DeleteNode(nodeID string) {
	d.NodeChanges = append(d.NodeChanges, NodeChange{
		Kind: ChangeDelete,
		Node: GraphNode{ID: nodeID, IsDeleted: true},
	})
	d.NodesDeleted++
}

// AddEdge adds an edge addition to the delta
func (d *GraphDelta) AddEdge(edge GraphEdge) {
	d.EdgeChanges = append(d.EdgeChanges, EdgeChange{
		Kind: ChangeAdd,
		Edge: edge,
	})
	d.EdgesAdded++
}

// DeleteEdge adds an edge deletion to the delta
func (d *GraphDelta) DeleteEdge(edgeID string) {
	d.EdgeChanges = append(d.EdgeChanges, EdgeChange{
		Kind: ChangeDelete,
		Edge: GraphEdge{ID: edgeID},
	})
	d.EdgesDeleted++
}

// computeChecksum computes a SHA-256 checksum of a node's content
func computeChecksum(node GraphNode) string {
	data, _ := json.Marshal(map[string]interface{}{
		"id":         node.ID,
		"kind":       node.Kind,
		"name":       node.Name,
		"startLine":  node.StartLine,
		"endLine":    node.EndLine,
		"properties": node.Properties,
	})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// edgeID generates a deterministic edge ID
func edgeID(kind EdgeKind, fromID, toID string) string {
	raw := fmt.Sprintf("%s:%s:%s", kind, fromID, toID)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:16])
}
