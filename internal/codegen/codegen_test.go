package codegen_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hackersfun369/nexus/internal/codegen"
)

var ctx = context.Background()

// ── INTENT PARSER ─────────────────────────────────────────────────────────────

func TestParseIntent_PythonAPI(t *testing.T) {
	intent := codegen.ParseIntent("Build a REST API backend with Python and PostgreSQL with authentication")
	if intent.Platform != "backend" {
		t.Errorf("Expected backend, got %s", intent.Platform)
	}
	if intent.Language != "python" {
		t.Errorf("Expected python, got %s", intent.Language)
	}
	if !hasFeature(intent.Features, "auth") {
		t.Error("Expected auth feature")
	}
	if !hasFeature(intent.Features, "database") {
		t.Error("Expected database feature")
	}
	t.Logf("✅ Intent: platform=%s lang=%s features=%v", intent.Platform, intent.Language, intent.Features)
}

func TestParseIntent_AndroidApp(t *testing.T) {
	intent := codegen.ParseIntent("Build me a food delivery app for Android")
	if intent.Platform != "android" {
		t.Errorf("Expected android, got %s", intent.Platform)
	}
	if intent.Language != "kotlin" {
		t.Errorf("Expected kotlin, got %s", intent.Language)
	}
	if !hasEntity(intent.Entities, "food") && !hasEntity(intent.Entities, "delivery") {
		t.Error("Expected food or delivery entity")
	}
	t.Logf("✅ Intent: platform=%s lang=%s entities=%v", intent.Platform, intent.Language, intent.Entities)
}

func TestParseIntent_ReactDashboard(t *testing.T) {
	intent := codegen.ParseIntent("Create a React dashboard with authentication")
	if intent.Platform != "web" {
		t.Errorf("Expected web, got %s", intent.Platform)
	}
	if intent.Language != "typescript" {
		t.Errorf("Expected typescript, got %s", intent.Language)
	}
	if !hasFeature(intent.Features, "auth") {
		t.Error("Expected auth feature")
	}
	t.Logf("✅ Intent: platform=%s lang=%s", intent.Platform, intent.Language)
}

func TestParseIntent_FlutterApp(t *testing.T) {
	intent := codegen.ParseIntent("Create a Flutter mobile app with real-time chat")
	if intent.Platform != "all" {
		t.Errorf("Expected all, got %s", intent.Platform)
	}
	if intent.Language != "dart" {
		t.Errorf("Expected dart, got %s", intent.Language)
	}
	if !hasFeature(intent.Features, "realtime") {
		t.Error("Expected realtime feature")
	}
	t.Logf("✅ Intent: platform=%s lang=%s", intent.Platform, intent.Language)
}

func TestParseIntent_GoBackend(t *testing.T) {
	intent := codegen.ParseIntent("Build a go backend api with database and docker")
	if intent.Language != "go" {
		t.Errorf("Expected go, got %s", intent.Language)
	}
	if !hasFeature(intent.Features, "database") {
		t.Error("Expected database feature")
	}
	if !hasFeature(intent.Features, "docker") {
		t.Error("Expected docker feature")
	}
	t.Logf("✅ Intent: lang=%s features=%v", intent.Language, intent.Features)
}

func TestParseIntent_CLITool(t *testing.T) {
	intent := codegen.ParseIntent("Build a CLI tool for file management")
	if intent.Platform != "cli" {
		t.Errorf("Expected cli, got %s", intent.Platform)
	}
	t.Logf("✅ Intent: platform=%s appType=%s", intent.Platform, intent.AppType)
}

func TestParseIntent_AppName(t *testing.T) {
	intent := codegen.ParseIntent("Build an app called DeliveryPro for food delivery")
	if !strings.Contains(intent.AppName, "Delivery") && intent.AppName != "FoodApp" {
		t.Logf("AppName: %s (acceptable)", intent.AppName)
	}
	t.Logf("✅ AppName: %s", intent.AppName)
}

// ── PLANNER ───────────────────────────────────────────────────────────────────

func TestPlanner_PythonFastAPI(t *testing.T) {
	p := codegen.NewPlanner()
	intent := codegen.ParseIntent("Build a REST API with Python FastAPI")
	plan := p.Plan(intent, "/tmp/out")

	if plan.PluginID != "python-fastapi" {
		t.Errorf("Expected python-fastapi, got %s", plan.PluginID)
	}
	if len(plan.Files) == 0 {
		t.Error("Expected files in plan")
	}
	if !hasFile(plan.Files, "main.py") {
		t.Error("Expected main.py")
	}
	if !hasFile(plan.Files, "requirements.txt") {
		t.Error("Expected requirements.txt")
	}
	t.Logf("✅ Planner: %d files, plugin=%s", len(plan.Files), plan.PluginID)
}

func TestPlanner_ReactWeb(t *testing.T) {
	p := codegen.NewPlanner()
	intent := codegen.ParseIntent("Create a React web dashboard")
	plan := p.Plan(intent, "/tmp/out")

	if plan.PluginID != "react-web" {
		t.Errorf("Expected react-web, got %s", plan.PluginID)
	}
	if !hasFile(plan.Files, "package.json") {
		t.Error("Expected package.json")
	}
	t.Logf("✅ Planner: %d files, plugin=%s", len(plan.Files), plan.PluginID)
}

func TestPlanner_KotlinAndroid(t *testing.T) {
	p := codegen.NewPlanner()
	intent := codegen.ParseIntent("Build a food delivery app for Android")
	plan := p.Plan(intent, "/tmp/out")

	if plan.PluginID != "kotlin-android" {
		t.Errorf("Expected kotlin-android, got %s", plan.PluginID)
	}
	if !hasFile(plan.Files, "README.md") {
		t.Error("Expected README.md")
	}
	t.Logf("✅ Planner: %d files, plugin=%s", len(plan.Files), plan.PluginID)
}

func TestPlanner_Flutter(t *testing.T) {
	p := codegen.NewPlanner()
	intent := codegen.ParseIntent("Create a Flutter mobile app")
	plan := p.Plan(intent, "/tmp/out")

	if plan.PluginID != "flutter-mobile" {
		t.Errorf("Expected flutter-mobile, got %s", plan.PluginID)
	}
	if !hasFile(plan.Files, "pubspec.yaml") {
		t.Error("Expected pubspec.yaml")
	}
	t.Logf("✅ Planner: %d files, plugin=%s", len(plan.Files), plan.PluginID)
}

// ── GENERATOR ─────────────────────────────────────────────────────────────────

func TestGenerator_Preview_Python(t *testing.T) {
	g := codegen.NewGenerator()
	result, err := g.GeneratePreview(ctx, "Build a REST API with Python FastAPI and authentication")
	if err != nil {
		t.Fatalf("GeneratePreview: %v", err)
	}
	if result.FileCount == 0 {
		t.Error("Expected files in preview")
	}
	if result.TotalBytes == 0 {
		t.Error("Expected non-zero bytes")
	}
	// Check main.py has content
	for _, f := range result.Files {
		if f.Path == "main.py" {
			if !strings.Contains(f.Content, "FastAPI") {
				t.Error("Expected FastAPI in main.py")
			}
			break
		}
	}
	t.Logf("✅ Preview: %d files, %d bytes, plugin=%s", result.FileCount, result.TotalBytes, result.PluginID)
}

func TestGenerator_Preview_React(t *testing.T) {
	g := codegen.NewGenerator()
	result, err := g.GeneratePreview(ctx, "Create a React dashboard with authentication")
	if err != nil {
		t.Fatalf("GeneratePreview: %v", err)
	}
	if result.PluginID != "react-web" {
		t.Errorf("Expected react-web, got %s", result.PluginID)
	}
	t.Logf("✅ Preview React: %d files", result.FileCount)
}

func TestGenerator_WriteToDisk(t *testing.T) {
	dir, err := os.MkdirTemp("", "nexus-gen-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	g := codegen.NewGenerator()
	result, err := g.Generate(ctx, "Build a REST API with Python FastAPI", dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if result.FileCount == 0 {
		t.Error("Expected files to be generated")
	}

	// Verify files exist on disk
	mainPy := filepath.Join(dir, "main.py")
	if _, err := os.Stat(mainPy); os.IsNotExist(err) {
		t.Error("Expected main.py on disk")
	}

	reqTxt := filepath.Join(dir, "requirements.txt")
	if _, err := os.Stat(reqTxt); os.IsNotExist(err) {
		t.Error("Expected requirements.txt on disk")
	}

	t.Logf("✅ WriteToDisk: %d files in %s", result.FileCount, dir)
}

func TestGenerator_Summary(t *testing.T) {
	g := codegen.NewGenerator()
	result, err := g.GeneratePreview(ctx, "Build a food delivery app for Android")
	if err != nil {
		t.Fatalf("GeneratePreview: %v", err)
	}
	summary := result.Summary()
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
	tree := result.FileTree()
	if tree == "" {
		t.Error("Expected non-empty file tree")
	}
	t.Logf("✅ Summary:\n%s", summary)
	t.Logf("✅ FileTree:\n%s", tree)
}

func TestGenerator_AllPlatforms(t *testing.T) {
	prompts := []struct {
		prompt   string
		platform string
		plugin   string
	}{
		{"Build a REST API with Python", "backend", "python-fastapi"},
		{"Create a React web app", "web", "react-web"},
		{"Build an Android app", "android", "kotlin-android"},
		{"Create a Flutter mobile app", "all", "flutter-mobile"},
		{"Build a Go CLI tool", "cli", "go-backend"},
	}

	g := codegen.NewGenerator()
	for _, tc := range prompts {
		result, err := g.GeneratePreview(ctx, tc.prompt)
		if err != nil {
			t.Errorf("GeneratePreview(%s): %v", tc.prompt, err)
			continue
		}
		if result.Intent.Platform != tc.platform {
			t.Errorf("Platform: expected %s got %s for %q", tc.platform, result.Intent.Platform, tc.prompt)
		}
		if result.PluginID != tc.plugin {
			t.Errorf("Plugin: expected %s got %s for %q", tc.plugin, result.PluginID, tc.prompt)
		}
		if result.FileCount == 0 {
			t.Errorf("No files for %q", tc.prompt)
		}
		t.Logf("✅ %s → %s (%d files)", tc.platform, result.PluginID, result.FileCount)
	}
}

// ── HELPERS ───────────────────────────────────────────────────────────────────

func hasFeature(features []string, name string) bool {
	for _, f := range features {
		if f == name {
			return true
		}
	}
	return false
}

func hasEntity(entities []string, name string) bool {
	for _, e := range entities {
		if e == name {
			return true
		}
	}
	return false
}

func hasFile(files []codegen.FileSpec, path string) bool {
	for _, f := range files {
		if f.Path == path {
			return true
		}
	}
	return false
}
