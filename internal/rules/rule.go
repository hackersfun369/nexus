package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/store"
)

// ── SEVERITY & CATEGORY ───────────────────────────────

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

type Category string

const (
	CategorySecurity        Category = "SECURITY"
	CategoryCorrectness     Category = "CORRECTNESS"
	CategoryPerformance     Category = "PERFORMANCE"
	CategoryMaintainability Category = "MAINTAINABILITY"
	CategoryArchitecture    Category = "ARCHITECTURE"
	CategoryStyle           Category = "STYLE"
	CategoryDocumentation   Category = "DOCUMENTATION"
	CategoryTestCoverage    Category = "TEST_COVERAGE"
)

// ── FINDING ───────────────────────────────────────────

type Finding struct {
	RuleID         string
	Severity       Severity
	Category       Category
	Title          string
	Description    string
	FilePath       string
	StartLine      int
	StartCol       int
	NodeID         string
	Evidence       string
	Remediation    string
	InferenceChain []string
	CWE            string
	OWASP          string
}

// ── RULE INTERFACE ────────────────────────────────────

type Rule interface {
	ID() string
	Name() string
	Severity() Severity
	Category() Category
	Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]Finding, error)
}

// ── BASE RULE ─────────────────────────────────────────

type BaseRule struct {
	id          string
	name        string
	category    Category
	severity    Severity
	description string
	remediation string
}

func NewBaseRule(id, name string, cat Category, sev Severity, description, remediation string) BaseRule {
	return BaseRule{
		id: id, name: name,
		category: cat, severity: sev,
		description: description,
		remediation: remediation,
	}
}

func (r BaseRule) ID() string          { return r.id }
func (r BaseRule) Name() string        { return r.name }
func (r BaseRule) Severity() Severity  { return r.severity }
func (r BaseRule) Category() Category  { return r.category }
func (r BaseRule) Description() string { return r.description }
func (r BaseRule) Remediation() string { return r.remediation }

// ── REGISTRY ──────────────────────────────────────────

type Registry struct {
	rules map[string]Rule
	order []string
}

func NewRegistry() *Registry {
	return &Registry{rules: make(map[string]Rule)}
}

func (r *Registry) Register(rule Rule) {
	if _, exists := r.rules[rule.ID()]; !exists {
		r.order = append(r.order, rule.ID())
	}
	r.rules[rule.ID()] = rule
}

func (r *Registry) Get(id string) (Rule, bool) {
	rule, ok := r.rules[id]
	return rule, ok
}

func (r *Registry) All() []Rule {
	rules := make([]Rule, 0, len(r.order))
	for _, id := range r.order {
		rules = append(rules, r.rules[id])
	}
	return rules
}

func (r *Registry) ByCategory(cat Category) []Rule {
	var result []Rule
	for _, id := range r.order {
		if r.rules[id].Category() == cat {
			result = append(result, r.rules[id])
		}
	}
	return result
}

func (r *Registry) BySeverity(sev Severity) []Rule {
	var result []Rule
	for _, id := range r.order {
		if r.rules[id].Severity() == sev {
			result = append(result, r.rules[id])
		}
	}
	return result
}

func (r *Registry) Count() int { return len(r.rules) }

// ── ENGINE ────────────────────────────────────────────

type Engine struct {
	registry *Registry
	store    store.GraphStore
}

func NewEngine(registry *Registry, s store.GraphStore) *Engine {
	return &Engine{registry: registry, store: s}
}

func (e *Engine) RunAll(ctx context.Context, projectID string) (EngineResult, error) {
	result := EngineResult{
		ProjectID: projectID,
		StartedAt: time.Now(),
	}

	for _, rule := range e.registry.All() {
		findings, err := rule.Analyze(ctx, projectID, e.store)
		if err != nil {
			result.Errors = append(result.Errors, RuleError{RuleID: rule.ID(), Err: err})
			continue
		}
		for _, f := range findings {
			issue := findingToIssue(projectID, f)
			if err := e.store.WriteIssue(ctx, issue); err != nil {
				result.Errors = append(result.Errors, RuleError{RuleID: rule.ID(), Err: err})
				continue
			}
			result.IssuesFound++
		}
		result.RulesRun++
	}

	result.CompletedAt = time.Now()
	return result, nil
}

func (e *Engine) RunRule(ctx context.Context, projectID, ruleID string) ([]Finding, error) {
	rule, ok := e.registry.Get(ruleID)
	if !ok {
		return nil, &RuleNotFoundError{ID: ruleID}
	}
	return rule.Analyze(ctx, projectID, e.store)
}

// ── RESULT TYPES ──────────────────────────────────────

type EngineResult struct {
	ProjectID   string
	RulesRun    int
	IssuesFound int
	Errors      []RuleError
	StartedAt   time.Time
	CompletedAt time.Time
}

func (r EngineResult) Duration() time.Duration { return r.CompletedAt.Sub(r.StartedAt) }
func (r EngineResult) HasErrors() bool         { return len(r.Errors) > 0 }

type RuleError struct {
	RuleID string
	Err    error
}

func (e RuleError) Error() string { return e.RuleID + ": " + e.Err.Error() }

type RuleNotFoundError struct{ ID string }

func (e *RuleNotFoundError) Error() string { return "rule not found: " + e.ID }

// ── HELPERS ───────────────────────────────────────────

func findingToIssue(projectID string, f Finding) store.Issue {
	chain := "[]"
	if len(f.InferenceChain) > 0 {
		chain = `["` + joinStrings(f.InferenceChain, `","`) + `"]`
	}
	prefix := projectID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	return store.Issue{
		ID: fmt.Sprintf("%s-%s-%08x", prefix, f.RuleID, fnv32(fmt.Sprintf("%s%s%s%s%d", projectID, f.RuleID, f.NodeID, f.FilePath, f.StartLine))), NodeID: f.NodeID,
		ProjectID:      projectID,
		RuleID:         f.RuleID,
		Severity:       string(f.Severity),
		Category:       string(f.Category),
		Title:          f.Title,
		Description:    f.Description,
		FilePath:       f.FilePath,
		StartLine:      f.StartLine,
		StartCol:       f.StartCol,
		Evidence:       f.Evidence,
		Remediation:    f.Remediation,
		InferenceChain: chain,
		CWE:            f.CWE,
		OWASP:          f.OWASP,
		Status:         store.IssueStatusOpen,
	}
}

func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
