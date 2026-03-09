package testcoverage

import (
	"context"
	"fmt"
	"strings"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// ── NEXUS-TEST-001: Low Test Coverage ────────────────

type LowTestCoverage struct {
	rules.BaseRule
	minCoverage float64
}

func NewLowTestCoverage(minCoverage float64) *LowTestCoverage {
	return &LowTestCoverage{
		BaseRule: rules.NewBaseRule(
			"NEXUS-TEST-001",
			"Low Test Coverage",
			rules.CategoryTestCoverage,
			rules.SeverityMedium,
			"Public function has low or no test coverage.",
			"Write unit tests to bring coverage above the minimum threshold.",
		),
		minCoverage: minCoverage,
	}
}

func (r *LowTestCoverage) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-TEST-001 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if fn.Visibility != "PUBLIC" {
			continue
		}
		if fn.IsConstructor {
			continue
		}
		// No coverage data means untested
		coverage := 0.0
		if fn.TestCoverage != nil {
			coverage = *fn.TestCoverage
		}
		if coverage >= r.minCoverage {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Low test coverage on '%s' (%.0f%%)", fn.Name, coverage),
			Description: fmt.Sprintf(
				"Public function '%s' has %.0f%% test coverage (minimum: %.0f%%). "+
					"Low coverage increases the risk of undetected regressions.",
				fn.Name, coverage, r.minCoverage,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("test_coverage=%.0f%% min_coverage=%.0f%%", coverage, r.minCoverage),
			Remediation: fmt.Sprintf("Write unit tests for '%s' covering normal, edge, and error cases.", fn.Name),
			InferenceChain: []string{
				fmt.Sprintf("function.test_coverage=%.0f%%", coverage),
				fmt.Sprintf("min_coverage=%.0f%%", r.minCoverage),
				"coverage < min → NEXUS-TEST-001",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-TEST-002: Untested Complex Function ─────────

type UntestedComplexFunction struct {
	rules.BaseRule
	minComplexity int
}

func NewUntestedComplexFunction(minComplexity int) *UntestedComplexFunction {
	return &UntestedComplexFunction{
		BaseRule: rules.NewBaseRule(
			"NEXUS-TEST-002",
			"Untested Complex Function",
			rules.CategoryTestCoverage,
			rules.SeverityHigh,
			"Complex function has no test coverage — highest risk combination.",
			"Prioritize writing tests for complex untested functions.",
		),
		minComplexity: minComplexity,
	}
}

func (r *UntestedComplexFunction) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-TEST-002 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if fn.CyclomaticComplexity < r.minComplexity {
			continue
		}
		coverage := 0.0
		if fn.TestCoverage != nil {
			coverage = *fn.TestCoverage
		}
		if coverage > 0 {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Untested complex function '%s' (complexity=%d)", fn.Name, fn.CyclomaticComplexity),
			Description: fmt.Sprintf(
				"Function '%s' has cyclomatic complexity of %d with 0%% test coverage. "+
					"Complex untested code is the highest risk combination for regressions.",
				fn.Name, fn.CyclomaticComplexity,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("cyclomatic_complexity=%d test_coverage=0%%", fn.CyclomaticComplexity),
			Remediation: fmt.Sprintf("Write tests for '%s' covering all %d code paths.", fn.Name, fn.CyclomaticComplexity),
			InferenceChain: []string{
				fmt.Sprintf("function.cyclomatic_complexity=%d", fn.CyclomaticComplexity),
				"function.test_coverage=0%",
				"complex + untested → NEXUS-TEST-002",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-TEST-003: Missing Test File ─────────────────

type MissingTestFile struct {
	rules.BaseRule
}

func NewMissingTestFile() *MissingTestFile {
	return &MissingTestFile{
		BaseRule: rules.NewBaseRule(
			"NEXUS-TEST-003",
			"Missing Test File",
			rules.CategoryTestCoverage,
			rules.SeverityMedium,
			"Source module has no corresponding test file.",
			"Create a test file for each source module.",
		),
	}
}

func (r *MissingTestFile) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-TEST-003 query: %w", err)
	}

	// Collect all file paths
	allPaths := make(map[string]bool)
	for _, mod := range modules {
		allPaths[mod.FilePath] = true
	}

	var findings []rules.Finding
	for _, mod := range modules {
		// Skip test files themselves
		if isTestFile(mod.FilePath) {
			continue
		}
		// Skip tiny files
		if mod.LinesOfCode < 10 {
			continue
		}
		// Check if a corresponding test file exists
		if hasTestFile(mod.FilePath, allPaths) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   mod.ID,
			FilePath: mod.FilePath,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Missing test file for '%s'", mod.FilePath),
			Description: fmt.Sprintf(
				"Module '%s' has no corresponding test file. "+
					"Every source module should have a dedicated test file.",
				mod.FilePath,
			),
			StartLine:   0,
			Evidence:    fmt.Sprintf("source_file=%s test_file=missing", mod.FilePath),
			Remediation: fmt.Sprintf("Create a test file for '%s'.", mod.FilePath),
			InferenceChain: []string{
				fmt.Sprintf("module.file_path=%s", mod.FilePath),
				"no corresponding test file found",
				"missing test file → NEXUS-TEST-003",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-TEST-004: Test Without Assertion ────────────

type TestWithoutAssertion struct {
	rules.BaseRule
}

func NewTestWithoutAssertion() *TestWithoutAssertion {
	return &TestWithoutAssertion{
		BaseRule: rules.NewBaseRule(
			"NEXUS-TEST-004",
			"Test Without Assertion",
			rules.CategoryTestCoverage,
			rules.SeverityLow,
			"Test function has no assertions — it cannot detect failures.",
			"Add assertions to verify expected behaviour.",
		),
	}
}

func (r *TestWithoutAssertion) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-TEST-004 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if !isTestFunction(fn.Name) {
			continue
		}
		// A test with 0 fan-out has no calls — likely no assertions
		if fn.FanOut > 0 {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Test '%s' has no assertions", fn.Name),
			Description: fmt.Sprintf(
				"Test function '%s' appears to have no assertions (fan-out=0). "+
					"Tests without assertions always pass and provide no safety guarantee.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function=%s is_test=true fan_out=0", fn.Name),
			Remediation: fmt.Sprintf("Add assertions to '%s' to verify expected behaviour.", fn.Name),
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"is_test_function=true",
				"fan_out=0 (no assertions detected)",
				"test without assertion → NEXUS-TEST-004",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-TEST-005: High Complexity No Coverage ───────

type HighComplexityNoCoverage struct {
	rules.BaseRule
	complexityThreshold int
	coverageThreshold   float64
}

func NewHighComplexityNoCoverage(complexityThreshold int, coverageThreshold float64) *HighComplexityNoCoverage {
	return &HighComplexityNoCoverage{
		BaseRule: rules.NewBaseRule(
			"NEXUS-TEST-005",
			"High Complexity Low Coverage",
			rules.CategoryTestCoverage,
			rules.SeverityHigh,
			"Function has both high complexity and low test coverage.",
			"Write comprehensive tests for complex functions first.",
		),
		complexityThreshold: complexityThreshold,
		coverageThreshold:   coverageThreshold,
	}
}

func (r *HighComplexityNoCoverage) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-TEST-005 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if fn.CyclomaticComplexity < r.complexityThreshold {
			continue
		}
		coverage := 0.0
		if fn.TestCoverage != nil {
			coverage = *fn.TestCoverage
		}
		if coverage >= r.coverageThreshold {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title: fmt.Sprintf("High complexity + low coverage in '%s' (complexity=%d, coverage=%.0f%%)",
				fn.Name, fn.CyclomaticComplexity, coverage),
			Description: fmt.Sprintf(
				"Function '%s' has complexity %d and only %.0f%% coverage (min: %.0f%%). "+
					"This combination maximises regression risk.",
				fn.Name, fn.CyclomaticComplexity, coverage, r.coverageThreshold,
			),
			StartLine: fn.StartLine,
			Evidence: fmt.Sprintf("cyclomatic_complexity=%d test_coverage=%.0f%% coverage_threshold=%.0f%%",
				fn.CyclomaticComplexity, coverage, r.coverageThreshold),
			Remediation: fmt.Sprintf("Write tests for all %d paths in '%s'.", fn.CyclomaticComplexity, fn.Name),
			InferenceChain: []string{
				fmt.Sprintf("function.cyclomatic_complexity=%d", fn.CyclomaticComplexity),
				fmt.Sprintf("function.test_coverage=%.0f%%", coverage),
				"high complexity + low coverage → NEXUS-TEST-005",
			},
		})
	}
	return findings, nil
}

// ── HELPERS ───────────────────────────────────────────

func isTestFile(path string) bool {
	return strings.Contains(path, "_test.") ||
		strings.Contains(path, "test_") ||
		strings.HasSuffix(path, "_test.py") ||
		strings.HasSuffix(path, "_test.ts") ||
		strings.HasSuffix(path, "_test.java") ||
		strings.Contains(path, "/test/") ||
		strings.Contains(path, "/tests/")
}

func hasTestFile(path string, allPaths map[string]bool) bool {
	// Python: foo.py → test_foo.py or foo_test.py
	// TypeScript: foo.ts → foo.test.ts or foo.spec.ts
	// Java: Foo.java → FooTest.java
	base := path
	ext := ""
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		base = path[:idx]
		ext = path[idx:]
	}
	candidates := []string{
		base + "_test" + ext,
		strings.Replace(base, "/", "/test_", 1) + ext,
		base + ".test" + ext,
		base + ".spec" + ext,
		base + "Test" + ext,
	}
	for _, c := range candidates {
		if allPaths[c] {
			return true
		}
	}
	return false
}

func isTestFunction(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "test_") ||
		strings.HasPrefix(lower, "test") ||
		strings.HasSuffix(lower, "_test") ||
		strings.HasPrefix(lower, "spec_") ||
		strings.HasPrefix(lower, "should_")
}

// DefaultTestCoverageRules returns all test coverage rules
func DefaultTestCoverageRules() []rules.Rule {
	return []rules.Rule{
		NewLowTestCoverage(80.0),
		NewUntestedComplexFunction(5),
		NewMissingTestFile(),
		NewTestWithoutAssertion(),
		NewHighComplexityNoCoverage(8, 60.0),
	}
}
