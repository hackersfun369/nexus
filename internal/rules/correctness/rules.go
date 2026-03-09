package correctness

import (
	"context"
	"fmt"
	"strings"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// ── NEXUS-CORR-001: Missing Return Type ──────────────

type MissingReturnType struct {
	rules.BaseRule
}

func NewMissingReturnType() *MissingReturnType {
	return &MissingReturnType{
		BaseRule: rules.NewBaseRule(
			"NEXUS-CORR-001",
			"Missing Return Type Annotation",
			rules.CategoryCorrectness,
			rules.SeverityMedium,
			"Public function is missing a return type annotation.",
			"Add explicit return type annotations to all public functions.",
		),
	}
}

func (r *MissingReturnType) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-CORR-001 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if fn.Visibility != "PUBLIC" {
			continue
		}
		if fn.IsConstructor {
			continue
		}
		// ReturnType is stored as JSON — empty object or "null" means missing
		if fn.ReturnType != "" && fn.ReturnType != "{}" && fn.ReturnType != "null" {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Missing return type on '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Public function '%s' has no return type annotation. "+
					"Explicit return types improve readability and catch type errors early.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function=%s visibility=PUBLIC return_type=missing", fn.Name),
			Remediation: fmt.Sprintf("Add a return type annotation to '%s'.", fn.Name),
			InferenceChain: []string{
				"function.visibility=PUBLIC",
				"function.return_type=missing",
				"is_constructor=false",
				"missing annotation → NEXUS-CORR-001",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-CORR-002: Unused Parameter ─────────────────

type UnusedParameter struct {
	rules.BaseRule
}

func NewUnusedParameter() *UnusedParameter {
	return &UnusedParameter{
		BaseRule: rules.NewBaseRule(
			"NEXUS-CORR-002",
			"Unused Parameter",
			rules.CategoryCorrectness,
			rules.SeverityLow,
			"Function has parameters that appear to be unused.",
			"Remove unused parameters or prefix with underscore to indicate intentional non-use.",
		),
	}
}

func (r *UnusedParameter) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-CORR-002 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if fn.FanOut <= 3 {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Unused parameter(s) in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' has %d unused parameter(s). "+
					"Unused parameters add noise and may indicate a logic error.",
				fn.Name, fn.FanOut,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function=%s unused_params=%d", fn.Name, fn.FanOut),
			Remediation: fmt.Sprintf("Remove unused parameters from '%s' or prefix with '_' to signal intentional non-use.", fn.Name),
			InferenceChain: []string{
				fmt.Sprintf("function.unused_params=%d", fn.FanOut),
				"unused_params > 0",
				"unused parameter → NEXUS-CORR-002",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-CORR-003: Empty Catch Block ────────────────

type EmptyCatchBlock struct {
	rules.BaseRule
}

func NewEmptyCatchBlock() *EmptyCatchBlock {
	return &EmptyCatchBlock{
		BaseRule: rules.NewBaseRule(
			"NEXUS-CORR-003",
			"Empty Catch Block",
			rules.CategoryCorrectness,
			rules.SeverityMedium,
			"Function contains an empty exception handler.",
			"Always handle or log exceptions — never silently swallow them.",
		),
	}
}

var emptyCatchPatterns = []string{
	"except_pass", "empty_catch", "bare_except",
	"swallow_error", "noop_handler",
}

func (r *EmptyCatchBlock) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-CORR-003 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, emptyCatchPatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Empty catch block in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests an empty exception handler. "+
					"Silently swallowing exceptions hides bugs and makes debugging impossible.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function=%s matches_empty_catch_pattern=true", fn.Name),
			Remediation: "Add logging or re-raise the exception. Never use bare 'except: pass'.",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches empty catch pattern",
				"empty catch block → NEXUS-CORR-003",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-CORR-004: Duplicate Code ───────────────────

type DuplicateCode struct {
	rules.BaseRule
	minLines int
}

func NewDuplicateCode(minLines int) *DuplicateCode {
	return &DuplicateCode{
		BaseRule: rules.NewBaseRule(
			"NEXUS-CORR-004",
			"Duplicate Code",
			rules.CategoryCorrectness,
			rules.SeverityMedium,
			"Two or more functions share identical checksums indicating copied code.",
			"Extract duplicated logic into a shared function.",
		),
		minLines: minLines,
	}
}

func (r *DuplicateCode) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-CORR-004 query: %w", err)
	}

	// Group functions by checksum
	checksums := make(map[string][]store.Function)
	for _, fn := range fns {
		if fn.LinesOfCode < r.minLines {
			continue
		}
		if fn.Checksum == "" {
			continue
		}
		checksums[fn.Checksum] = append(checksums[fn.Checksum], fn)
	}

	var findings []rules.Finding
	seen := make(map[string]bool)
	for checksum, group := range checksums {
		if len(group) < 2 {
			continue
		}
		if seen[checksum] {
			continue
		}
		seen[checksum] = true

		names := make([]string, 0, len(group))
		for _, fn := range group {
			names = append(names, fn.Name)
		}

		// Emit one finding per duplicate function
		for _, fn := range group {
			findings = append(findings, rules.Finding{
				RuleID:   r.ID(),
				NodeID:   fn.ID,
				FilePath: fn.Language,
				Severity: r.Severity(),
				Category: r.Category(),
				Title:    fmt.Sprintf("Duplicate code in '%s'", fn.Name),
				Description: fmt.Sprintf(
					"Function '%s' appears to be a duplicate of: %s. "+
						"Duplicated code increases maintenance burden and bug surface.",
					fn.Name, strings.Join(names, ", "),
				),
				StartLine:   fn.StartLine,
				Evidence:    fmt.Sprintf("checksum=%s duplicate_count=%d", checksum[:8], len(group)),
				Remediation: "Extract the shared logic into a single reusable function.",
				InferenceChain: []string{
					fmt.Sprintf("function.checksum=%s", checksum[:8]),
					fmt.Sprintf("duplicate_count=%d", len(group)),
					"identical checksums → NEXUS-CORR-004",
				},
			})
		}
	}
	return findings, nil
}

// ── NEXUS-CORR-005: High Cognitive Complexity ────────

type HighCognitiveComplexity struct {
	rules.BaseRule
	threshold int
}

func NewHighCognitiveComplexity(threshold int) *HighCognitiveComplexity {
	return &HighCognitiveComplexity{
		BaseRule: rules.NewBaseRule(
			"NEXUS-CORR-005",
			"High Cognitive Complexity",
			rules.CategoryCorrectness,
			rules.SeverityMedium,
			"Function has high cognitive complexity making it hard to understand.",
			"Simplify logic by extracting sub-functions and reducing branching.",
		),
		threshold: threshold,
	}
}

func (r *HighCognitiveComplexity) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-CORR-005 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if fn.CyclomaticComplexity <= r.threshold {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("High cognitive complexity in '%s' (%d)", fn.Name, fn.CyclomaticComplexity),
			Description: fmt.Sprintf(
				"Function '%s' has cognitive complexity of %d (threshold: %d). "+
					"High cognitive complexity makes code hard to read and reason about.",
				fn.Name, fn.CyclomaticComplexity, r.threshold,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("cognitive_complexity=%d threshold=%d", fn.CyclomaticComplexity, r.threshold),
			Remediation: fmt.Sprintf("Simplify '%s' by extracting helper functions and reducing nested conditions.", fn.Name),
			InferenceChain: []string{
				fmt.Sprintf("function.cognitive_complexity=%d", fn.CyclomaticComplexity),
				fmt.Sprintf("threshold=%d", r.threshold),
				"cognitive_complexity > threshold → NEXUS-CORR-005",
			},
		})
	}
	return findings, nil
}

// ── HELPERS ───────────────────────────────────────────

func matchesPatterns(name string, patterns []string) bool {
	lower := strings.ToLower(name)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// DefaultCorrectnessRules returns all correctness rules
func DefaultCorrectnessRules() []rules.Rule {
	return []rules.Rule{
		NewMissingReturnType(),
		NewUnusedParameter(),
		NewEmptyCatchBlock(),
		NewDuplicateCode(5),
		NewHighCognitiveComplexity(15),
	}
}
