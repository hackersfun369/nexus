package complexity

import (
	"context"
	"fmt"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// ── NEXUS-COMP-001: High Cyclomatic Complexity ────────

type HighCyclomaticComplexity struct {
	rules.BaseRule
	threshold int
}

func NewHighCyclomaticComplexity(threshold int) *HighCyclomaticComplexity {
	return &HighCyclomaticComplexity{
		BaseRule: rules.NewBaseRule(
			"NEXUS-COMP-001",
			"High Cyclomatic Complexity",
			rules.CategoryMaintainability,
			rules.SeverityMedium,
			"Function has high cyclomatic complexity making it hard to test and maintain.",
			"Refactor the function by extracting sub-functions or simplifying conditional logic.",
		),
		threshold: threshold,
	}
}

func (r *HighCyclomaticComplexity) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{
		ProjectID:     projectID,
		MinComplexity: r.threshold,
	})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-COMP-001 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("High cyclomatic complexity in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' has cyclomatic complexity of %d (threshold: %d). "+
					"High complexity increases the risk of bugs and makes testing harder.",
				fn.Name, fn.CyclomaticComplexity, r.threshold,
			),
			StartLine: fn.StartLine,
			Evidence:  fmt.Sprintf("cyclomatic_complexity=%d", fn.CyclomaticComplexity),
			Remediation: fmt.Sprintf(
				"Refactor '%s' to reduce complexity below %d. "+
					"Extract helper functions, simplify conditions, or use early returns.",
				fn.Name, r.threshold,
			),
			InferenceChain: []string{
				fmt.Sprintf("function.cyclomatic_complexity=%d", fn.CyclomaticComplexity),
				fmt.Sprintf("threshold=%d", r.threshold),
				"complexity > threshold → NEXUS-COMP-001",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-COMP-002: Long Function ─────────────────────

type LongFunction struct {
	rules.BaseRule
	maxLines int
}

func NewLongFunction(maxLines int) *LongFunction {
	return &LongFunction{
		BaseRule: rules.NewBaseRule(
			"NEXUS-COMP-002",
			"Long Function",
			rules.CategoryMaintainability,
			rules.SeverityLow,
			"Function exceeds recommended line count.",
			"Break the function into smaller, focused functions.",
		),
		maxLines: maxLines,
	}
}

func (r *LongFunction) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-COMP-002 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if fn.LinesOfCode <= r.maxLines {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Long function '%s' (%d lines)", fn.Name, fn.LinesOfCode),
			Description: fmt.Sprintf(
				"Function '%s' has %d lines of code (max: %d). "+
					"Long functions are harder to understand and test.",
				fn.Name, fn.LinesOfCode, r.maxLines,
			),
			StartLine: fn.StartLine,
			Evidence:  fmt.Sprintf("lines_of_code=%d", fn.LinesOfCode),
			Remediation: fmt.Sprintf(
				"Split '%s' into smaller functions, each with a single responsibility.",
				fn.Name,
			),
			InferenceChain: []string{
				fmt.Sprintf("function.lines_of_code=%d", fn.LinesOfCode),
				fmt.Sprintf("max_lines=%d", r.maxLines),
				"lines > max → NEXUS-COMP-002",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-COMP-003: Too Many Parameters ───────────────

type TooManyParameters struct {
	rules.BaseRule
	maxParams int
}

func NewTooManyParameters(maxParams int) *TooManyParameters {
	return &TooManyParameters{
		BaseRule: rules.NewBaseRule(
			"NEXUS-COMP-003",
			"Too Many Parameters",
			rules.CategoryMaintainability,
			rules.SeverityLow,
			"Function has too many parameters indicating poor cohesion.",
			"Group related parameters into a struct or object.",
		),
		maxParams: maxParams,
	}
}

func (r *TooManyParameters) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-COMP-003 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if fn.ParameterCount <= r.maxParams {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Too many parameters in '%s' (%d)", fn.Name, fn.ParameterCount),
			Description: fmt.Sprintf(
				"Function '%s' has %d parameters (max: %d). "+
					"Functions with many parameters are hard to call correctly and test.",
				fn.Name, fn.ParameterCount, r.maxParams,
			),
			StartLine: fn.StartLine,
			Evidence:  fmt.Sprintf("parameter_count=%d", fn.ParameterCount),
			Remediation: fmt.Sprintf(
				"Reduce parameters in '%s' by grouping related ones into a config struct.",
				fn.Name,
			),
			InferenceChain: []string{
				fmt.Sprintf("function.parameter_count=%d", fn.ParameterCount),
				fmt.Sprintf("max_params=%d", r.maxParams),
				"params > max → NEXUS-COMP-003",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-COMP-004: Deep Nesting ──────────────────────

type DeepNesting struct {
	rules.BaseRule
	maxDepth int
}

func NewDeepNesting(maxDepth int) *DeepNesting {
	return &DeepNesting{
		BaseRule: rules.NewBaseRule(
			"NEXUS-COMP-004",
			"Deep Nesting",
			rules.CategoryMaintainability,
			rules.SeverityMedium,
			"Function has deeply nested code blocks reducing readability.",
			"Flatten nesting using early returns, guard clauses, or extracted functions.",
		),
		maxDepth: maxDepth,
	}
}

func (r *DeepNesting) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-COMP-004 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if fn.NestingDepth <= r.maxDepth {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Deep nesting in '%s' (depth %d)", fn.Name, fn.NestingDepth),
			Description: fmt.Sprintf(
				"Function '%s' has nesting depth of %d (max: %d). "+
					"Deeply nested code is hard to read and reason about.",
				fn.Name, fn.NestingDepth, r.maxDepth,
			),
			StartLine: fn.StartLine,
			Evidence:  fmt.Sprintf("nesting_depth=%d", fn.NestingDepth),
			Remediation: fmt.Sprintf(
				"Reduce nesting in '%s' using early returns, guard clauses, or extracted helper functions.",
				fn.Name,
			),
			InferenceChain: []string{
				fmt.Sprintf("function.nesting_depth=%d", fn.NestingDepth),
				fmt.Sprintf("max_depth=%d", r.maxDepth),
				"depth > max → NEXUS-COMP-004",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-COMP-005: God Class ─────────────────────────

type GodClass struct {
	rules.BaseRule
	maxMethods int
	maxLines   int
}

func NewGodClass(maxMethods, maxLines int) *GodClass {
	return &GodClass{
		BaseRule: rules.NewBaseRule(
			"NEXUS-COMP-005",
			"God Class",
			rules.CategoryArchitecture,
			rules.SeverityHigh,
			"Class has too many responsibilities (too many methods or lines of code).",
			"Apply the Single Responsibility Principle — split into focused classes.",
		),
		maxMethods: maxMethods,
		maxLines:   maxLines,
	}
}

func (r *GodClass) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-COMP-005 query: %w", err)
	}

	var findings []rules.Finding
	for _, cls := range classes {
		tooManyMethods := cls.MethodCount > r.maxMethods
		tooLong := cls.LinesOfCode > r.maxLines

		if !tooManyMethods && !tooLong {
			continue
		}

		evidence := ""
		reason := ""
		if tooManyMethods && tooLong {
			evidence = fmt.Sprintf("method_count=%d lines_of_code=%d", cls.MethodCount, cls.LinesOfCode)
			reason = fmt.Sprintf("has %d methods and %d lines", cls.MethodCount, cls.LinesOfCode)
		} else if tooManyMethods {
			evidence = fmt.Sprintf("method_count=%d", cls.MethodCount)
			reason = fmt.Sprintf("has %d methods (max: %d)", cls.MethodCount, r.maxMethods)
		} else {
			evidence = fmt.Sprintf("lines_of_code=%d", cls.LinesOfCode)
			reason = fmt.Sprintf("has %d lines (max: %d)", cls.LinesOfCode, r.maxLines)
		}

		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   cls.ID,
			FilePath: cls.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("God class '%s'", cls.Name),
			Description: fmt.Sprintf(
				"Class '%s' %s. God classes violate the Single Responsibility Principle "+
					"and become maintenance bottlenecks.",
				cls.Name, reason,
			),
			StartLine: cls.StartLine,
			Evidence:  evidence,
			Remediation: fmt.Sprintf(
				"Split '%s' into smaller classes each with a single responsibility.",
				cls.Name,
			),
			InferenceChain: []string{
				evidence,
				fmt.Sprintf("max_methods=%d max_lines=%d", r.maxMethods, r.maxLines),
				"exceeds threshold → NEXUS-COMP-005",
			},
		})
	}
	return findings, nil
}

// DefaultComplexityRules returns all complexity rules with sensible defaults
func DefaultComplexityRules() []rules.Rule {
	return []rules.Rule{
		NewHighCyclomaticComplexity(10),
		NewLongFunction(50),
		NewTooManyParameters(5),
		NewDeepNesting(4),
		NewGodClass(20, 300),
	}
}
