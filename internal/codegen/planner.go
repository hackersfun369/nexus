package codegen

// FileSpec describes a file to generate
type FileSpec struct {
	Path     string
	Template string
	Data     map[string]interface{}
}

// ProjectPlan is the full list of files to generate
type ProjectPlan struct {
	Intent    AppIntent
	PluginID  string
	OutputDir string
	Files     []FileSpec
}

// Planner decides what files to generate based on intent + plugin
type Planner struct{}

// NewPlanner creates a planner
func NewPlanner() *Planner { return &Planner{} }

// Plan returns a ProjectPlan for the given intent and plugin
func (p *Planner) Plan(intent AppIntent, outputDir string) ProjectPlan {
	plan := ProjectPlan{
		Intent:    intent,
		OutputDir: outputDir,
	}

	switch intent.Platform {
	case "backend":
		switch intent.Language {
		case "python":
			plan.PluginID = "python-fastapi"
			plan.Files = p.planPythonFastAPI(intent)
		case "go":
			plan.PluginID = "go-backend"
			plan.Files = p.planGoBackend(intent)
		default:
			plan.PluginID = "python-fastapi"
			plan.Files = p.planPythonFastAPI(intent)
		}
	case "android":
		plan.PluginID = "kotlin-android"
		plan.Files = p.planKotlinAndroid(intent)
	case "web":
		plan.PluginID = "react-web"
		plan.Files = p.planReactWeb(intent)
	case "cli":
		plan.PluginID = "go-backend"
		plan.Files = p.planGoCLI(intent)
	case "all":
		plan.PluginID = "flutter-mobile"
		plan.Files = p.planFlutter(intent)
	default:
		plan.PluginID = "react-web"
		plan.Files = p.planReactWeb(intent)
	}

	return plan
}

func base(intent AppIntent) map[string]interface{} {
	hasAuth := hasFeature(intent.Features, "auth")
	hasDB := hasFeature(intent.Features, "database")
	return map[string]interface{}{
		"AppName":     intent.AppName,
		"Description": intent.Description,
		"Language":    intent.Language,
		"Platform":    intent.Platform,
		"Framework":   intent.Framework,
		"Features":    intent.Features,
		"Entities":    intent.Entities,
		"HasAuth":     hasAuth,
		"HasDB":       hasDB,
	}
}

// ── PYTHON FASTAPI ────────────────────────────────────────────────────────────

func (p *Planner) planPythonFastAPI(intent AppIntent) []FileSpec {
	d := base(intent)
	files := []FileSpec{
		{Path: "main.py", Template: "python_fastapi_main", Data: d},
		{Path: "requirements.txt", Template: "python_requirements", Data: d},
		{Path: "README.md", Template: "readme", Data: d},
		{Path: "src/__init__.py", Template: "python_empty", Data: d},
		{Path: "src/api/__init__.py", Template: "python_empty", Data: d},
		{Path: "src/api/routes.py", Template: "python_routes", Data: d},
		{Path: "src/api/models.py", Template: "python_models", Data: d},
		{Path: "src/db/__init__.py", Template: "python_empty", Data: d},
		{Path: "src/db/database.py", Template: "python_database", Data: d},
		{Path: "tests/__init__.py", Template: "python_empty", Data: d},
		{Path: "tests/test_api.py", Template: "python_tests", Data: d},
		{Path: ".gitignore", Template: "gitignore_python", Data: d},
		{Path: "Dockerfile", Template: "dockerfile_python", Data: d},
	}
	if hasFeature(intent.Features, "auth") {
		files = append(files,
			FileSpec{Path: "src/auth/__init__.py", Template: "python_empty", Data: d},
			FileSpec{Path: "src/auth/jwt.py", Template: "python_auth_jwt", Data: d},
		)
	}
	return files
}

// ── GO BACKEND ────────────────────────────────────────────────────────────────

func (p *Planner) planGoBackend(intent AppIntent) []FileSpec {
	d := base(intent)
	return []FileSpec{
		{Path: "main.go", Template: "go_main", Data: d},
		{Path: "go.mod", Template: "go_mod", Data: d},
		{Path: "README.md", Template: "readme", Data: d},
		{Path: "internal/api/server.go", Template: "go_server", Data: d},
		{Path: "internal/api/routes.go", Template: "go_routes", Data: d},
		{Path: "internal/api/handlers.go", Template: "go_handlers", Data: d},
		{Path: "internal/db/db.go", Template: "go_db", Data: d},
		{Path: "Makefile", Template: "go_makefile", Data: d},
		{Path: ".gitignore", Template: "gitignore_go", Data: d},
		{Path: "Dockerfile", Template: "dockerfile_go", Data: d},
	}
}

// ── GO CLI ────────────────────────────────────────────────────────────────────

func (p *Planner) planGoCLI(intent AppIntent) []FileSpec {
	d := base(intent)
	return []FileSpec{
		{Path: "main.go", Template: "go_cli_main", Data: d},
		{Path: "go.mod", Template: "go_mod", Data: d},
		{Path: "README.md", Template: "readme", Data: d},
		{Path: "cmd/root.go", Template: "go_cli_root", Data: d},
		{Path: "cmd/version.go", Template: "go_cli_version", Data: d},
		{Path: ".gitignore", Template: "gitignore_go", Data: d},
		{Path: "Makefile", Template: "go_makefile", Data: d},
	}
}

// ── REACT WEB ─────────────────────────────────────────────────────────────────

func (p *Planner) planReactWeb(intent AppIntent) []FileSpec {
	d := base(intent)
	return []FileSpec{
		{Path: "package.json", Template: "react_package_json", Data: d},
		{Path: "vite.config.ts", Template: "react_vite_config", Data: d},
		{Path: "index.html", Template: "react_index_html", Data: d},
		{Path: "README.md", Template: "readme", Data: d},
		{Path: "src/main.tsx", Template: "react_main", Data: d},
		{Path: "src/App.tsx", Template: "react_app", Data: d},
		{Path: "src/index.css", Template: "react_css", Data: d},
		{Path: "src/pages/HomePage.tsx", Template: "react_homepage", Data: d},
		{Path: "src/components/Layout.tsx", Template: "react_layout", Data: d},
		{Path: ".gitignore", Template: "gitignore_node", Data: d},
		{Path: "tsconfig.json", Template: "react_tsconfig", Data: d},
	}
}

// ── KOTLIN ANDROID ────────────────────────────────────────────────────────────

func (p *Planner) planKotlinAndroid(intent AppIntent) []FileSpec {
	d := base(intent)
	pkg := "com.nexus." + toLower(intent.AppName)
	d["Package"] = pkg
	d["PackagePath"] = "com/nexus/" + toLower(intent.AppName)
	return []FileSpec{
		{Path: "README.md", Template: "readme", Data: d},
		{Path: "settings.gradle.kts", Template: "kotlin_settings", Data: d},
		{Path: "build.gradle.kts", Template: "kotlin_root_build", Data: d},
		{Path: "app/build.gradle.kts", Template: "kotlin_app_build", Data: d},
		{Path: "app/src/main/AndroidManifest.xml", Template: "kotlin_manifest", Data: d},
		{Path: "app/src/main/java/" + d["PackagePath"].(string) + "/MainActivity.kt", Template: "kotlin_main_activity", Data: d},
		{Path: "app/src/main/java/" + d["PackagePath"].(string) + "/ui/HomeScreen.kt", Template: "kotlin_home_screen", Data: d},
		{Path: "app/src/main/java/" + d["PackagePath"].(string) + "/ui/theme/Theme.kt", Template: "kotlin_theme", Data: d},
		{Path: ".gitignore", Template: "gitignore_android", Data: d},
	}
}

// ── FLUTTER ───────────────────────────────────────────────────────────────────

func (p *Planner) planFlutter(intent AppIntent) []FileSpec {
	d := base(intent)
	pkg := "com.nexus." + toLower(intent.AppName)
	d["Package"] = pkg
	return []FileSpec{
		{Path: "README.md", Template: "readme", Data: d},
		{Path: "pubspec.yaml", Template: "flutter_pubspec", Data: d},
		{Path: "lib/main.dart", Template: "flutter_main", Data: d},
		{Path: "lib/app.dart", Template: "flutter_app", Data: d},
		{Path: "lib/screens/home_screen.dart", Template: "flutter_home", Data: d},
		{Path: "lib/widgets/app_bar.dart", Template: "flutter_appbar", Data: d},
		{Path: ".gitignore", Template: "gitignore_flutter", Data: d},
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

func toLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}
