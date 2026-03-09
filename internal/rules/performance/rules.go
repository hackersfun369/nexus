package performance

import (
	"context"
	"fmt"
	"strings"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// ── NEXUS-PERF-001: N+1 Query Risk ───────────────────

type NPlusOneQueryRisk struct {
	rules.BaseRule
}

func NewNPlusOneQueryRisk() *NPlusOneQueryRisk {
	return &NPlusOneQueryRisk{
		BaseRule: rules.NewBaseRule(
			"NEXUS-PERF-001",
			"N+1 Query Risk",
			rules.CategoryPerformance,
			rules.SeverityHigh,
			"Function name suggests database queries inside a loop.",
			"Use batch queries, eager loading, or joins instead of per-item queries.",
		),
	}
}

var nPlusOnePatterns = []string{
	"get_for_each", "fetch_per_item", "query_in_loop",
	"load_each", "find_per", "fetch_each",
	"get_each", "query_each", "load_per",
}

func (r *NPlusOneQueryRisk) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-PERF-001 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, nPlusOnePatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("N+1 query risk in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests database queries inside a loop. "+
					"N+1 queries cause severe performance degradation at scale.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function=%s matches_n_plus_one_pattern=true", fn.Name),
			Remediation: "Use batch queries, eager loading (SELECT IN), or JOIN to fetch all data in one query.",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches n+1 query pattern",
				"n+1 query risk → NEXUS-PERF-001",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-PERF-002: High Fan-Out ─────────────────────

type HighFanOut struct {
	rules.BaseRule
	threshold int
}

func NewHighFanOut(threshold int) *HighFanOut {
	return &HighFanOut{
		BaseRule: rules.NewBaseRule(
			"NEXUS-PERF-002",
			"High Fan-Out",
			rules.CategoryPerformance,
			rules.SeverityMedium,
			"Function calls too many other functions indicating poor cohesion.",
			"Reduce dependencies by extracting cohesive modules.",
		),
		threshold: threshold,
	}
}

func (r *HighFanOut) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-PERF-002 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if fn.FanOut <= r.threshold {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("High fan-out in '%s' (%d)", fn.Name, fn.FanOut),
			Description: fmt.Sprintf(
				"Function '%s' has fan-out of %d (threshold: %d). "+
					"High fan-out increases coupling and makes testing harder.",
				fn.Name, fn.FanOut, r.threshold,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("fan_out=%d threshold=%d", fn.FanOut, r.threshold),
			Remediation: fmt.Sprintf("Reduce dependencies in '%s' by grouping related calls into cohesive helper functions.", fn.Name),
			InferenceChain: []string{
				fmt.Sprintf("function.fan_out=%d", fn.FanOut),
				fmt.Sprintf("threshold=%d", r.threshold),
				"fan_out > threshold → NEXUS-PERF-002",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-PERF-003: Large Class Coupling ─────────────

type HighCouplingBetweenObjects struct {
	rules.BaseRule
	threshold int
}

func NewHighCouplingBetweenObjects(threshold int) *HighCouplingBetweenObjects {
	return &HighCouplingBetweenObjects{
		BaseRule: rules.NewBaseRule(
			"NEXUS-PERF-003",
			"High Coupling Between Objects",
			rules.CategoryPerformance,
			rules.SeverityMedium,
			"Class is coupled to too many other classes.",
			"Apply dependency inversion and reduce direct class dependencies.",
		),
		threshold: threshold,
	}
}

func (r *HighCouplingBetweenObjects) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-PERF-003 query: %w", err)
	}
	var findings []rules.Finding
	for _, cls := range classes {
		if cls.CouplingBetweenObjects <= r.threshold {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   cls.ID,
			FilePath: cls.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("High coupling in '%s' (CBO=%d)", cls.Name, cls.CouplingBetweenObjects),
			Description: fmt.Sprintf(
				"Class '%s' is coupled to %d other classes (threshold: %d). "+
					"High coupling makes changes expensive and testing difficult.",
				cls.Name, cls.CouplingBetweenObjects, r.threshold,
			),
			StartLine:   cls.StartLine,
			Evidence:    fmt.Sprintf("coupling_between_objects=%d threshold=%d", cls.CouplingBetweenObjects, r.threshold),
			Remediation: "Apply dependency inversion principle. Depend on abstractions not concretions.",
			InferenceChain: []string{
				fmt.Sprintf("class.coupling_between_objects=%d", cls.CouplingBetweenObjects),
				fmt.Sprintf("threshold=%d", r.threshold),
				"cbo > threshold → NEXUS-PERF-003",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-PERF-004: Low Cohesion ─────────────────────

type LowCohesion struct {
	rules.BaseRule
	threshold int
}

func NewLowCohesion(threshold int) *LowCohesion {
	return &LowCohesion{
		BaseRule: rules.NewBaseRule(
			"NEXUS-PERF-004",
			"Low Cohesion",
			rules.CategoryPerformance,
			rules.SeverityMedium,
			"Class has low cohesion — methods don't share instance variables.",
			"Split the class into smaller, more focused classes.",
		),
		threshold: threshold,
	}
}

func (r *LowCohesion) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-PERF-004 query: %w", err)
	}
	var findings []rules.Finding
	for _, cls := range classes {
		if cls.LackOfCohesion <= float64(r.threshold) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   cls.ID,
			FilePath: cls.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Low cohesion in '%s' (LCOM=%.0f)", cls.Name, cls.LackOfCohesion),
			Description: fmt.Sprintf(
				"Class '%s' has lack-of-cohesion metric of %.0f (threshold: %d). "+
					"Low cohesion means the class has multiple unrelated responsibilities.",
				cls.Name, cls.LackOfCohesion, r.threshold,
			),
			StartLine:   cls.StartLine,
			Evidence:    fmt.Sprintf("lack_of_cohesion=%.0f threshold=%d", cls.LackOfCohesion, r.threshold),
			Remediation: fmt.Sprintf("Split '%s' into smaller classes each focused on one responsibility.", cls.Name),
			InferenceChain: []string{
				fmt.Sprintf("class.lack_of_cohesion=%.0f", cls.LackOfCohesion),
				fmt.Sprintf("threshold=%d", r.threshold),
				"lcom > threshold → NEXUS-PERF-004",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-PERF-005: Blocking IO in Hot Path ──────────

type BlockingIOInHotPath struct {
	rules.BaseRule
}

func NewBlockingIOInHotPath() *BlockingIOInHotPath {
	return &BlockingIOInHotPath{
		BaseRule: rules.NewBaseRule(
			"NEXUS-PERF-005",
			"Blocking IO in Hot Path",
			rules.CategoryPerformance,
			rules.SeverityHigh,
			"Synchronous IO operation detected in a potentially hot code path.",
			"Use async IO or move blocking operations off the critical path.",
		),
	}
}

var blockingIOPatterns = []string{
	"sync_read", "sync_write", "blocking_call",
	"sync_fetch", "sync_request", "blocking_io",
	"sync_load", "sync_save", "blocking_read",
}

func (r *BlockingIOInHotPath) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-PERF-005 query: %w", err)
	}
	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, blockingIOPatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Blocking IO in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests synchronous/blocking IO. "+
					"Blocking IO in hot paths stalls execution and degrades throughput.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function=%s matches_blocking_io_pattern=true", fn.Name),
			Remediation: "Use async/await or non-blocking IO. Move heavy IO off the main execution path.",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches blocking io pattern",
				"blocking io risk → NEXUS-PERF-005",
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

// DefaultPerformanceRules returns all performance rules
func DefaultPerformanceRules() []rules.Rule {
	return []rules.Rule{
		NewNPlusOneQueryRisk(),
		NewHighFanOut(10),
		NewHighCouplingBetweenObjects(10),
		NewLowCohesion(10),
		NewBlockingIOInHotPath(),
	}
}
