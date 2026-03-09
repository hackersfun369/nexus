package docrules

import (
	"context"
	"fmt"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

type MissingPublicFunctionDoc struct {
	rules.BaseRule
}

func NewMissingPublicFunctionDoc() *MissingPublicFunctionDoc {
	return &MissingPublicFunctionDoc{
		BaseRule: rules.NewBaseRule(
			"NEXUS-DOC-001",
			"Missing Public Function Documentation",
			rules.CategoryDocumentation,
			rules.SeverityLow,
			"Public function is missing a documentation comment.",
			"Add a doc comment describing the function's purpose, parameters, and return value.",
		),
	}
}

func (r *MissingPublicFunctionDoc) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-DOC-001 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if fn.Visibility != "PUBLIC" {
			continue
		}
		if fn.IsConstructor {
			continue
		}
		if fn.DocComment != "" {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:      r.ID(),
			NodeID:      fn.ID,
			FilePath:    fn.Language,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Title:       fmt.Sprintf("Missing doc comment on public function '%s'", fn.Name),
			Description: fmt.Sprintf("Public function '%s' has no documentation comment.", fn.Name),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("visibility=PUBLIC doc_comment=empty function=%s", fn.Name),
			Remediation: fmt.Sprintf("Add a doc comment above '%s'.", fn.Name),
			InferenceChain: []string{
				"function.visibility=PUBLIC",
				"function.doc_comment=empty",
				"is_constructor=false",
				"missing doc → NEXUS-DOC-001",
			},
		})
	}
	return findings, nil
}

type MissingPublicClassDoc struct {
	rules.BaseRule
}

func NewMissingPublicClassDoc() *MissingPublicClassDoc {
	return &MissingPublicClassDoc{
		BaseRule: rules.NewBaseRule(
			"NEXUS-DOC-002",
			"Missing Public Class Documentation",
			rules.CategoryDocumentation,
			rules.SeverityLow,
			"Public class or interface is missing a documentation comment.",
			"Add a doc comment describing the class purpose and usage.",
		),
	}
}

func (r *MissingPublicClassDoc) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-DOC-002 query: %w", err)
	}
	var findings []rules.Finding
	for _, cls := range classes {
		if cls.Visibility != "PUBLIC" {
			continue
		}
		if cls.DocComment != "" {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:      r.ID(),
			NodeID:      cls.ID,
			FilePath:    cls.Language,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Title:       fmt.Sprintf("Missing doc comment on public %s '%s'", cls.Kind, cls.Name),
			Description: fmt.Sprintf("Public %s '%s' has no documentation comment.", cls.Kind, cls.Name),
			StartLine:   cls.StartLine,
			Evidence:    fmt.Sprintf("visibility=PUBLIC doc_comment=empty %s=%s", cls.Kind, cls.Name),
			Remediation: fmt.Sprintf("Add a doc comment above '%s'.", cls.Name),
			InferenceChain: []string{
				fmt.Sprintf("class.visibility=PUBLIC kind=%s", cls.Kind),
				"class.doc_comment=empty",
				"missing doc → NEXUS-DOC-002",
			},
		})
	}
	return findings, nil
}

type MissingModuleDoc struct {
	rules.BaseRule
}

func NewMissingModuleDoc() *MissingModuleDoc {
	return &MissingModuleDoc{
		BaseRule: rules.NewBaseRule(
			"NEXUS-DOC-003",
			"Missing Module Documentation",
			rules.CategoryDocumentation,
			rules.SeverityInfo,
			"Module file has no top-level documentation comment.",
			"Add a module-level doc comment describing the file's purpose.",
		),
	}
}

func (r *MissingModuleDoc) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-DOC-003 query: %w", err)
	}
	var findings []rules.Finding
	for _, mod := range modules {
		if mod.LinesOfCode < 10 {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:      r.ID(),
			NodeID:      mod.ID,
			FilePath:    mod.FilePath,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Title:       fmt.Sprintf("Missing module doc in '%s'", mod.FilePath),
			Description: fmt.Sprintf("Module '%s' has no top-level documentation comment.", mod.FilePath),
			StartLine:   0,
			Evidence:    fmt.Sprintf("file=%s lines=%d doc_comment=empty", mod.FilePath, mod.LinesOfCode),
			Remediation: fmt.Sprintf("Add a top-level comment at the start of '%s'.", mod.FilePath),
			InferenceChain: []string{
				fmt.Sprintf("module.lines_of_code=%d", mod.LinesOfCode),
				"module.doc_comment=empty",
				"substantial module without doc → NEXUS-DOC-003",
			},
		})
	}
	return findings, nil
}

func DefaultDocumentationRules() []rules.Rule {
	return []rules.Rule{
		NewMissingPublicFunctionDoc(),
		NewMissingPublicClassDoc(),
		NewMissingModuleDoc(),
	}
}
