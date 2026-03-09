package loader_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hackersfun369/nexus/internal/rules"
	"github.com/hackersfun369/nexus/internal/rules/loader"
)

func TestDefaultConfig_Values(t *testing.T) {
	cfg := rules.DefaultConfig()
	if cfg.CyclomaticComplexityThreshold != 10 {
		t.Errorf("Expected 10, got %d", cfg.CyclomaticComplexityThreshold)
	}
	if cfg.MaxFunctionLines != 50 {
		t.Errorf("Expected 50, got %d", cfg.MaxFunctionLines)
	}
	if cfg.MinTestCoverage != 80.0 {
		t.Errorf("Expected 80.0, got %f", cfg.MinTestCoverage)
	}
	if len(cfg.DisabledRules) != 0 {
		t.Errorf("Expected 0 disabled rules, got %d", len(cfg.DisabledRules))
	}
	t.Logf("✅ DefaultConfig: all values correct")
}

func TestConfig_IsDisabled(t *testing.T) {
	cfg := rules.DefaultConfig()
	cfg.DisabledRules = []string{"NEXUS-SEC-001", "NEXUS-COMP-001"}
	if !cfg.IsDisabled("NEXUS-SEC-001") {
		t.Error("Expected NEXUS-SEC-001 to be disabled")
	}
	if cfg.IsDisabled("NEXUS-SEC-002") {
		t.Error("Expected NEXUS-SEC-002 to be enabled")
	}
	t.Logf("✅ Config.IsDisabled: works correctly")
}

func TestLoadConfig_NonExistentFile_ReturnsDefaults(t *testing.T) {
	cfg, err := rules.LoadConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.CyclomaticComplexityThreshold != 10 {
		t.Errorf("Expected default threshold 10, got %d", cfg.CyclomaticComplexityThreshold)
	}
	t.Logf("✅ LoadConfig: returns defaults for missing file")
}

func TestLoadConfig_ReadsFile(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-config-test-*")
	defer os.RemoveAll(dir)
	cfgPath := filepath.Join(dir, "nexus.json")
	data := `{"cyclomatic_complexity_threshold": 15, "max_function_lines": 100}`
	os.WriteFile(cfgPath, []byte(data), 0644)
	cfg, err := rules.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.CyclomaticComplexityThreshold != 15 {
		t.Errorf("Expected 15, got %d", cfg.CyclomaticComplexityThreshold)
	}
	if cfg.MaxFunctionLines != 100 {
		t.Errorf("Expected 100, got %d", cfg.MaxFunctionLines)
	}
	if cfg.MinTestCoverage != 80.0 {
		t.Errorf("Expected default 80.0, got %f", cfg.MinTestCoverage)
	}
	t.Logf("✅ LoadConfig: reads file correctly")
}

func TestSaveConfig_WritesFile(t *testing.T) {
	dir, _ := os.MkdirTemp("", "nexus-config-test-*")
	defer os.RemoveAll(dir)
	cfgPath := filepath.Join(dir, "nexus.json")
	cfg := rules.DefaultConfig()
	cfg.CyclomaticComplexityThreshold = 20
	if err := rules.SaveConfig(cfgPath, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	data, _ := os.ReadFile(cfgPath)
	var loaded map[string]interface{}
	json.Unmarshal(data, &loaded)
	if int(loaded["cyclomatic_complexity_threshold"].(float64)) != 20 {
		t.Errorf("Expected 20 in saved file")
	}
	t.Logf("✅ SaveConfig: writes file correctly")
}

func TestDefaultRegistry_LoadsAllRules(t *testing.T) {
	cfg := rules.DefaultConfig()
	reg := loader.DefaultRegistry(cfg)
	if reg.Count() != 33 {
		t.Errorf("Expected 33 rules, got %d", reg.Count())
	}
	t.Logf("✅ DefaultRegistry: %d rules loaded", reg.Count())
}

func TestDefaultRegistry_DisabledRulesExcluded(t *testing.T) {
	cfg := rules.DefaultConfig()
	cfg.DisabledRules = []string{"NEXUS-SEC-001", "NEXUS-SEC-002", "NEXUS-SEC-003"}
	reg := loader.DefaultRegistry(cfg)
	if reg.Count() != 30 {
		t.Errorf("Expected 30 rules (33-3), got %d", reg.Count())
	}
	if _, ok := reg.Get("NEXUS-SEC-001"); ok {
		t.Error("NEXUS-SEC-001 should be disabled")
	}
	if _, ok := reg.Get("NEXUS-COMP-001"); !ok {
		t.Error("NEXUS-COMP-001 should be enabled")
	}
	t.Logf("✅ DefaultRegistry: disabled rules excluded, count=%d", reg.Count())
}

func TestDefaultRegistry_AllCategoriesPresent(t *testing.T) {
	cfg := rules.DefaultConfig()
	reg := loader.DefaultRegistry(cfg)
	categories := []rules.Category{
		rules.CategoryMaintainability,
		rules.CategorySecurity,
		rules.CategoryCorrectness,
		rules.CategoryPerformance,
		rules.CategoryArchitecture,
		rules.CategoryDocumentation,
		rules.CategoryTestCoverage,
	}
	for _, cat := range categories {
		catRules := reg.ByCategory(cat)
		if len(catRules) == 0 {
			t.Errorf("Expected rules for category %s, got 0", cat)
		}
		t.Logf("  %s: %d rules", cat, len(catRules))
	}
	t.Logf("✅ DefaultRegistry: all categories present")
}

func TestDefaultRegistry_CustomThresholds(t *testing.T) {
	cfg := rules.DefaultConfig()
	cfg.CyclomaticComplexityThreshold = 20
	reg := loader.DefaultRegistry(cfg)
	rule, ok := reg.Get("NEXUS-COMP-001")
	if !ok {
		t.Fatal("NEXUS-COMP-001 not found")
	}
	if rule.ID() != "NEXUS-COMP-001" {
		t.Errorf("Expected NEXUS-COMP-001, got %s", rule.ID())
	}
	t.Logf("✅ DefaultRegistry: custom thresholds applied")
}
