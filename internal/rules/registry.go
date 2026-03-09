package rules

import (
	"encoding/json"
	"fmt"
	"os"
)

// ── CONFIG ────────────────────────────────────────────

// Config holds tunable thresholds for all rules
type Config struct {
	// Complexity
	CyclomaticComplexityThreshold int `json:"cyclomatic_complexity_threshold"`
	MaxFunctionLines              int `json:"max_function_lines"`
	MaxParameters                 int `json:"max_parameters"`
	MaxNestingDepth               int `json:"max_nesting_depth"`
	GodClassMaxMethods            int `json:"god_class_max_methods"`
	GodClassMaxLines              int `json:"god_class_max_lines"`

	// Performance
	MaxFanOut                 int `json:"max_fan_out"`
	MaxCouplingBetweenObjects int `json:"max_coupling_between_objects"`
	MaxLackOfCohesion         int `json:"max_lack_of_cohesion"`

	// Architecture
	MaxStaticMethods    int `json:"max_static_methods"`
	MaxInterfaceMethods int `json:"max_interface_methods"`

	// Correctness
	DuplicateCodeMinLines        int `json:"duplicate_code_min_lines"`
	CognitiveComplexityThreshold int `json:"cognitive_complexity_threshold"`

	// Test Coverage
	MinTestCoverage             float64 `json:"min_test_coverage"`
	UntestedComplexityThreshold int     `json:"untested_complexity_threshold"`
	HighComplexityThreshold     int     `json:"high_complexity_threshold"`
	HighComplexityCoverageMin   float64 `json:"high_complexity_coverage_min"`

	// Disabled rules
	DisabledRules []string `json:"disabled_rules"`
}

// DefaultConfig returns sensible defaults for all thresholds
func DefaultConfig() Config {
	return Config{
		CyclomaticComplexityThreshold: 10,
		MaxFunctionLines:              50,
		MaxParameters:                 5,
		MaxNestingDepth:               4,
		GodClassMaxMethods:            20,
		GodClassMaxLines:              300,
		MaxFanOut:                     10,
		MaxCouplingBetweenObjects:     10,
		MaxLackOfCohesion:             10,
		MaxStaticMethods:              5,
		MaxInterfaceMethods:           7,
		DuplicateCodeMinLines:         5,
		CognitiveComplexityThreshold:  15,
		MinTestCoverage:               80.0,
		UntestedComplexityThreshold:   5,
		HighComplexityThreshold:       8,
		HighComplexityCoverageMin:     60.0,
		DisabledRules:                 []string{},
	}
}

// LoadConfig reads a JSON config file, falling back to defaults for missing fields
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("LoadConfig: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("LoadConfig parse: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes a config to a JSON file
func SaveConfig(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveConfig marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("SaveConfig write: %w", err)
	}
	return nil
}

// IsDisabled returns true if a rule ID is in the disabled list
func (c Config) IsDisabled(ruleID string) bool {
	for _, id := range c.DisabledRules {
		if id == ruleID {
			return true
		}
	}
	return false
}
