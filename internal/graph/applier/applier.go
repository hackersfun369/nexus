package applier

import (
	"context"
	"fmt"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/parser/extractor"
)

type Result struct {
	NodesAdded    int
	NodesModified int
	NodesDeleted  int
	EdgesAdded    int
	EdgesDeleted  int
	AppliedAt     time.Time
	Skipped       bool
}

type Applier struct {
	store store.GraphStore
}

func New(s store.GraphStore) *Applier {
	return &Applier{store: s}
}

func (a *Applier) Apply(ctx context.Context, delta extractor.GraphDelta) (Result, error) {
	if delta.IsEmpty() {
		return Result{Skipped: true, AppliedAt: time.Now()}, nil
	}
	result := Result{AppliedAt: time.Now()}

	for _, change := range delta.NodeChanges {
		if err := a.applyNodeChange(ctx, delta.ProjectID, delta.FilePath, change); err != nil {
			return Result{}, fmt.Errorf("apply node %s: %w", change.Node.ID, err)
		}
		switch change.Kind {
		case extractor.ChangeAdd:
			result.NodesAdded++
		case extractor.ChangeModify:
			result.NodesModified++
		case extractor.ChangeDelete:
			result.NodesDeleted++
		}
	}

	for _, change := range delta.EdgeChanges {
		if err := a.applyEdgeChange(ctx, delta.ProjectID, change); err != nil {
			return Result{}, fmt.Errorf("apply edge %s: %w", change.Edge.ID, err)
		}
		switch change.Kind {
		case extractor.ChangeAdd:
			result.EdgesAdded++
		case extractor.ChangeDelete:
			result.EdgesDeleted++
		}
	}

	return result, nil
}

func (a *Applier) applyNodeChange(ctx context.Context, projectID, filePath string, change extractor.NodeChange) error {
	switch change.Kind {
	case extractor.ChangeDelete:
		return a.deleteNode(ctx, change.Node)
	case extractor.ChangeAdd, extractor.ChangeModify:
		return a.writeNode(ctx, projectID, filePath, change.Node)
	default:
		return fmt.Errorf("unknown change kind: %s", change.Kind)
	}
}

func (a *Applier) writeNode(ctx context.Context, projectID, filePath string, node extractor.GraphNode) error {
	switch node.Kind {
	case extractor.NodeModule:
		return a.store.WriteModule(ctx, toStoreModule(projectID, node))
	case extractor.NodeFunction:
		moduleID := a.findModuleID(ctx, projectID, filePath)
		return a.store.WriteFunction(ctx, toStoreFunction(projectID, moduleID, node))
	case extractor.NodeClass, extractor.NodeInterface:
		moduleID := a.findModuleID(ctx, projectID, filePath)
		return a.store.WriteClass(ctx, toStoreClass(projectID, moduleID, node))
	case extractor.NodeImport:
		return nil
	default:
		return nil
	}
}

func (a *Applier) findModuleID(ctx context.Context, projectID, filePath string) string {
	m, err := a.store.GetModuleByPath(ctx, projectID, filePath)
	if err != nil {
		return ""
	}
	return m.ID
}

func (a *Applier) deleteNode(ctx context.Context, node extractor.GraphNode) error {
	switch node.Kind {
	case extractor.NodeModule:
		return a.store.DeleteModule(ctx, node.ID)
	case extractor.NodeFunction:
		return a.store.DeleteFunction(ctx, node.ID)
	case extractor.NodeClass, extractor.NodeInterface:
		return a.store.DeleteClass(ctx, node.ID)
	default:
		return nil
	}
}

func (a *Applier) applyEdgeChange(ctx context.Context, projectID string, change extractor.EdgeChange) error {
	switch change.Kind {
	case extractor.ChangeAdd:
		return a.store.WriteEdge(ctx, toStoreEdge(projectID, change.Edge))
	case extractor.ChangeDelete:
		return a.store.DeleteEdge(ctx, change.Edge.ID)
	default:
		return nil
	}
}

func toStoreModule(projectID string, n extractor.GraphNode) store.Module {
	return store.Module{
		ID:            n.ID,
		ProjectID:     projectID,
		FilePath:      n.FilePath,
		QualifiedName: n.QualifiedName,
		Language:      n.Language,
		LinesOfCode:   intProp(n.Properties, "linesOfCode"),
		ParseStatus:   "OK",
		ParseErrors:   "[]",
		Checksum:      n.Checksum,
	}
}

func toStoreFunction(projectID, moduleID string, n extractor.GraphNode) store.Function {
	return store.Function{
		ID:                   n.ID,
		ProjectID:            projectID,
		ModuleID:             moduleID,
		Name:                 n.Name,
		QualifiedName:        n.QualifiedName,
		Language:             n.Language,
		StartLine:            int(n.StartLine),
		StartCol:             int(n.StartCol),
		EndLine:              int(n.EndLine),
		EndCol:               int(n.EndCol),
		Visibility:           stringProp(n.Properties, "visibility"),
		Parameters:           "[]",
		ReturnType:           "{}",
		IsAsync:              boolProp(n.Properties, "isAsync"),
		IsStatic:             boolProp(n.Properties, "isStatic"),
		IsAbstract:           boolProp(n.Properties, "isAbstract"),
		IsConstructor:        boolProp(n.Properties, "isConstructor"),
		CyclomaticComplexity: intProp(n.Properties, "cyclomaticComplexity"),
		LinesOfCode:          intProp(n.Properties, "linesOfCode"),
		ParameterCount:       intProp(n.Properties, "parameterCount"),
		NestingDepth:         intProp(n.Properties, "nestingDepth"),
		Annotations:          "[]",
		Checksum:             n.Checksum,
	}
}

func toStoreClass(projectID, moduleID string, n extractor.GraphNode) store.Class {
	kind := "CLASS"
	if n.Kind == extractor.NodeInterface {
		kind = "INTERFACE"
	}
	return store.Class{
		ID:            n.ID,
		ProjectID:     projectID,
		ModuleID:      moduleID,
		Name:          n.Name,
		QualifiedName: n.QualifiedName,
		Language:      n.Language,
		Kind:          kind,
		StartLine:     int(n.StartLine),
		StartCol:      int(n.StartCol),
		EndLine:       int(n.EndLine),
		EndCol:        int(n.EndCol),
		Visibility:    stringProp(n.Properties, "visibility"),
		LinesOfCode:   intProp(n.Properties, "linesOfCode"),
		IsAbstract:    boolProp(n.Properties, "isAbstract"),
		Annotations:   "[]",
		Checksum:      n.Checksum,
	}
}

func toStoreEdge(projectID string, e extractor.GraphEdge) store.Edge {
	return store.Edge{
		ID:         e.ID,
		ProjectID:  projectID,
		Kind:       string(e.Kind),
		FromNodeID: e.FromNodeID,
		ToNodeID:   e.ToNodeID,
		Properties: "{}",
	}
}

func stringProp(props map[string]interface{}, key string) string {
	if v, ok := props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intProp(props map[string]interface{}, key string) int {
	if v, ok := props[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
	}
	return 0
}

func boolProp(props map[string]interface{}, key string) bool {
	if v, ok := props[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
