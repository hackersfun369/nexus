package loader

import (
	"github.com/hackersfun369/nexus/internal/rules"
	"github.com/hackersfun369/nexus/internal/rules/architecture"
	"github.com/hackersfun369/nexus/internal/rules/complexity"
	"github.com/hackersfun369/nexus/internal/rules/correctness"
	docrules "github.com/hackersfun369/nexus/internal/rules/documentation"
	"github.com/hackersfun369/nexus/internal/rules/performance"
	"github.com/hackersfun369/nexus/internal/rules/security"
	"github.com/hackersfun369/nexus/internal/rules/testcoverage"
)

// DefaultRegistry builds a Registry with all rules using the given config
func DefaultRegistry(cfg rules.Config) *rules.Registry {
	reg := rules.NewRegistry()

	allRules := []rules.Rule{}

	// Complexity
	allRules = append(allRules,
		complexity.NewHighCyclomaticComplexity(cfg.CyclomaticComplexityThreshold),
		complexity.NewLongFunction(cfg.MaxFunctionLines),
		complexity.NewTooManyParameters(cfg.MaxParameters),
		complexity.NewDeepNesting(cfg.MaxNestingDepth),
		complexity.NewGodClass(cfg.GodClassMaxMethods, cfg.GodClassMaxLines),
	)

	// Documentation
	allRules = append(allRules,
		docrules.NewMissingPublicFunctionDoc(),
		docrules.NewMissingPublicClassDoc(),
		docrules.NewMissingModuleDoc(),
	)

	// Security
	allRules = append(allRules,
		security.NewHardcodedSecrets(),
		security.NewSQLInjectionRisk(),
		security.NewWeakCryptography(),
		security.NewOverlyBroadExceptionHandling(),
		security.NewInsecureDeserialization(),
	)

	// Correctness
	allRules = append(allRules,
		correctness.NewMissingReturnType(),
		correctness.NewUnusedParameter(),
		correctness.NewEmptyCatchBlock(),
		correctness.NewDuplicateCode(cfg.DuplicateCodeMinLines),
		correctness.NewHighCognitiveComplexity(cfg.CognitiveComplexityThreshold),
	)

	// Performance
	allRules = append(allRules,
		performance.NewNPlusOneQueryRisk(),
		performance.NewHighFanOut(cfg.MaxFanOut),
		performance.NewHighCouplingBetweenObjects(cfg.MaxCouplingBetweenObjects),
		performance.NewLowCohesion(cfg.MaxLackOfCohesion),
		performance.NewBlockingIOInHotPath(),
	)

	// Architecture
	allRules = append(allRules,
		architecture.NewCircularDependencyRisk(),
		architecture.NewAbstractClassWithoutSubclasses(),
		architecture.NewStaticMethodOveruse(cfg.MaxStaticMethods),
		architecture.NewAsyncMethodInSyncContext(),
		architecture.NewInterfaceSegregationViolation(cfg.MaxInterfaceMethods),
	)

	// Test Coverage
	allRules = append(allRules,
		testcoverage.NewLowTestCoverage(cfg.MinTestCoverage),
		testcoverage.NewUntestedComplexFunction(cfg.UntestedComplexityThreshold),
		testcoverage.NewMissingTestFile(),
		testcoverage.NewTestWithoutAssertion(),
		testcoverage.NewHighComplexityNoCoverage(cfg.HighComplexityThreshold, cfg.HighComplexityCoverageMin),
	)

	// Register only enabled rules
	for _, rule := range allRules {
		if !cfg.IsDisabled(rule.ID()) {
			reg.Register(rule)
		}
	}

	return reg
}
