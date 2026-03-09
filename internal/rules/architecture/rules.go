package architecture

import (
	"context"
	"fmt"
	"strings"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// ── NEXUS-ARCH-001: Circular Dependency Risk ──────────

type CircularDependencyRisk struct {
	rules.BaseRule
}

func NewCircularDependencyRisk() *CircularDependencyRisk {
	return &CircularDependencyRisk{
		BaseRule: rules.NewBaseRule(
			"NEXUS-ARCH-001",
			"Circular Dependency Risk",
			rules.CategoryArchitecture,
			rules.SeverityHigh,
			"Module has high cycle risk indicating potential circular dependencies.",
			"Restructure modules to eliminate circular imports using dependency inversion.",
		),
	}
}

func (r *CircularDependencyRisk) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-ARCH-001 query: %w", err)
	}
	var findings []rules.Finding
	for _, mod := range modules {
		if mod.CycleRisk <= 0.5 {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   mod.ID,
			FilePath: mod.FilePath,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Circular dependency risk in '%s'", mod.FilePath),
			Description: fmt.Sprintf(
				"Module '%s' has cycle risk of %.2f. "+
					"High cycle risk indicates circular imports that create tight coupling.",
				mod.FilePath, mod.CycleRisk,
			),
			StartLine:   0,
			Evidence:    fmt.Sprintf("cycle_risk=%.2f threshold=0.50", mod.CycleRisk),
			Remediation: "Break the cycle by extracting shared types into a separate module or using dependency inversion.",
			InferenceChain: []string{
				fmt.Sprintf("module.cycle_risk=%.2f", mod.CycleRisk),
				"cycle_risk > 0.50",
				"circular dependency risk → NEXUS-ARCH-001",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-ARCH-002: Abstract Class Without Subclasses ─

type AbstractClassWithoutSubclasses struct {
	rules.BaseRule
}

func NewAbstractClassWithoutSubclasses() *AbstractClassWithoutSubclasses {
	return &AbstractClassWithoutSubclasses{
		BaseRule: rules.NewBaseRule(
			"NEXUS-ARCH-002",
			"Abstract Class Without Subclasses",
			rules.CategoryArchitecture,
			rules.SeverityMedium,
			"Abstract class has no concrete implementations in the codebase.",
			"Remove the abstract class or add concrete implementations.",
		),
	}
}

func (r *AbstractClassWithoutSubclasses) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-ARCH-002 query: %w", err)
	}

	// Collect all class names that appear as base classes via naming convention
	// Classes named with "Base", "Abstract", "Interface" prefix/suffix
	var findings []rules.Finding
	for _, cls := range classes {
		if !cls.IsAbstract {
			continue
		}
		// Check if any other class name suggests it extends this one
		hasSubclass := false
		lowerName := strings.ToLower(cls.Name)
		for _, other := range classes {
			if other.ID == cls.ID {
				continue
			}
			otherLower := strings.ToLower(other.Name)
			// Simple heuristic: subclass name contains base class name
			baseName := strings.TrimPrefix(lowerName, "abstract")
			baseName = strings.TrimPrefix(baseName, "base")
			baseName = strings.TrimSuffix(baseName, "base")
			baseName = strings.TrimSuffix(baseName, "abstract")
			baseName = strings.TrimSpace(baseName)
			if baseName != "" && strings.Contains(otherLower, baseName) {
				hasSubclass = true
				break
			}
		}
		if hasSubclass {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   cls.ID,
			FilePath: cls.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Abstract class '%s' has no subclasses", cls.Name),
			Description: fmt.Sprintf(
				"Abstract class '%s' has no detected concrete implementations. "+
					"Unused abstractions add complexity without benefit.",
				cls.Name,
			),
			StartLine:   cls.StartLine,
			Evidence:    fmt.Sprintf("class=%s is_abstract=true subclasses=0", cls.Name),
			Remediation: fmt.Sprintf("Either add concrete implementations of '%s' or convert it to a concrete class.", cls.Name),
			InferenceChain: []string{
				fmt.Sprintf("class.name=%s", cls.Name),
				"class.is_abstract=true",
				"no subclasses detected → NEXUS-ARCH-002",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-ARCH-003: Static Method Overuse ─────────────

type StaticMethodOveruse struct {
	rules.BaseRule
	threshold int
}

func NewStaticMethodOveruse(threshold int) *StaticMethodOveruse {
	return &StaticMethodOveruse{
		BaseRule: rules.NewBaseRule(
			"NEXUS-ARCH-003",
			"Static Method Overuse",
			rules.CategoryArchitecture,
			rules.SeverityLow,
			"Class has too many static methods suggesting procedural rather than OO design.",
			"Convert static methods to instance methods or extract into a service class.",
		),
		threshold: threshold,
	}
}

func (r *StaticMethodOveruse) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-ARCH-003 query: %w", err)
	}

	// Count static methods per module
	staticCount := make(map[string]int)
	moduleNames := make(map[string]string)
	for _, fn := range fns {
		if fn.IsStatic {
			staticCount[fn.ModuleID]++
			moduleNames[fn.ModuleID] = fn.Language
		}
	}

	var findings []rules.Finding
	for moduleID, count := range staticCount {
		if count <= r.threshold {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   moduleID,
			FilePath: moduleNames[moduleID],
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Static method overuse in module (%d static methods)", count),
			Description: fmt.Sprintf(
				"Module has %d static methods (threshold: %d). "+
					"Excessive static methods indicate procedural design that's hard to test and extend.",
				count, r.threshold,
			),
			StartLine:   0,
			Evidence:    fmt.Sprintf("static_method_count=%d threshold=%d", count, r.threshold),
			Remediation: "Convert static methods to instance methods. Use dependency injection for testability.",
			InferenceChain: []string{
				fmt.Sprintf("module.static_method_count=%d", count),
				fmt.Sprintf("threshold=%d", r.threshold),
				"static_count > threshold → NEXUS-ARCH-003",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-ARCH-004: Async Method in Sync Context ──────

type AsyncMethodInSyncContext struct {
	rules.BaseRule
}

func NewAsyncMethodInSyncContext() *AsyncMethodInSyncContext {
	return &AsyncMethodInSyncContext{
		BaseRule: rules.NewBaseRule(
			"NEXUS-ARCH-004",
			"Async Method in Sync Context",
			rules.CategoryArchitecture,
			rules.SeverityMedium,
			"Async method detected alongside non-async methods suggesting inconsistent patterns.",
			"Ensure async methods are called from async contexts to avoid deadlocks.",
		),
	}
}

func (r *AsyncMethodInSyncContext) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-ARCH-004 query: %w", err)
	}

	// Group by module — flag modules that mix async and sync public methods
	type moduleInfo struct {
		asyncCount int
		syncCount  int
		asyncFns   []store.Function
	}
	modules := make(map[string]*moduleInfo)
	for _, fn := range fns {
		if fn.Visibility != "PUBLIC" || fn.IsConstructor {
			continue
		}
		if _, ok := modules[fn.ModuleID]; !ok {
			modules[fn.ModuleID] = &moduleInfo{}
		}
		if fn.IsAsync {
			modules[fn.ModuleID].asyncCount++
			modules[fn.ModuleID].asyncFns = append(modules[fn.ModuleID].asyncFns, fn)
		} else {
			modules[fn.ModuleID].syncCount++
		}
	}

	var findings []rules.Finding
	for _, info := range modules {
		if info.asyncCount == 0 || info.syncCount == 0 {
			continue
		}
		for _, fn := range info.asyncFns {
			findings = append(findings, rules.Finding{
				RuleID:   r.ID(),
				NodeID:   fn.ID,
				FilePath: fn.Language,
				Severity: r.Severity(),
				Category: r.Category(),
				Title:    fmt.Sprintf("Async method '%s' mixed with sync methods", fn.Name),
				Description: fmt.Sprintf(
					"Module mixes async function '%s' with %d synchronous public methods. "+
						"Mixed async/sync patterns can cause subtle bugs and deadlocks.",
					fn.Name, info.syncCount,
				),
				StartLine:   fn.StartLine,
				Evidence:    fmt.Sprintf("async_count=%d sync_count=%d", info.asyncCount, info.syncCount),
				Remediation: "Make the entire module async or extract async methods into a dedicated async service.",
				InferenceChain: []string{
					fmt.Sprintf("function.name=%s is_async=true", fn.Name),
					fmt.Sprintf("module.sync_count=%d", info.syncCount),
					"mixed async/sync → NEXUS-ARCH-004",
				},
			})
		}
	}
	return findings, nil
}

// ── NEXUS-ARCH-005: Interface Segregation Violation ───

type InterfaceSegregationViolation struct {
	rules.BaseRule
	maxMethods int
}

func NewInterfaceSegregationViolation(maxMethods int) *InterfaceSegregationViolation {
	return &InterfaceSegregationViolation{
		BaseRule: rules.NewBaseRule(
			"NEXUS-ARCH-005",
			"Interface Segregation Violation",
			rules.CategoryArchitecture,
			rules.SeverityMedium,
			"Interface or abstract class has too many methods violating ISP.",
			"Split the interface into smaller, focused interfaces.",
		),
		maxMethods: maxMethods,
	}
}

func (r *InterfaceSegregationViolation) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-ARCH-005 query: %w", err)
	}
	var findings []rules.Finding
	for _, cls := range classes {
		if cls.Kind != "INTERFACE" && !cls.IsAbstract {
			continue
		}
		if cls.MethodCount <= r.maxMethods {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   cls.ID,
			FilePath: cls.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Interface '%s' has too many methods (%d)", cls.Name, cls.MethodCount),
			Description: fmt.Sprintf(
				"Interface '%s' defines %d methods (max: %d). "+
					"Large interfaces force implementors to provide methods they don't need.",
				cls.Name, cls.MethodCount, r.maxMethods,
			),
			StartLine:   cls.StartLine,
			Evidence:    fmt.Sprintf("method_count=%d max_methods=%d kind=%s", cls.MethodCount, r.maxMethods, cls.Kind),
			Remediation: fmt.Sprintf("Split '%s' into smaller focused interfaces following the Interface Segregation Principle.", cls.Name),
			InferenceChain: []string{
				fmt.Sprintf("class.kind=%s method_count=%d", cls.Kind, cls.MethodCount),
				fmt.Sprintf("max_methods=%d", r.maxMethods),
				"method_count > max → NEXUS-ARCH-005",
			},
		})
	}
	return findings, nil
}

// DefaultArchitectureRules returns all architecture rules
func DefaultArchitectureRules() []rules.Rule {
	return []rules.Rule{
		NewCircularDependencyRisk(),
		NewAbstractClassWithoutSubclasses(),
		NewStaticMethodOveruse(5),
		NewAsyncMethodInSyncContext(),
		NewInterfaceSegregationViolation(7),
	}
}
