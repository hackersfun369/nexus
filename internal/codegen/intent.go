package codegen

import (
	"strings"
)

// AppIntent holds what the user wants to build
type AppIntent struct {
	AppName     string
	AppType     string   // api, mobile, web, cli, desktop
	Platform    string   // android, web, backend, cli, mac, windows
	Language    string   // python, kotlin, typescript, go, etc
	Framework   string   // fastapi, react, jetpack, etc
	Features    []string // auth, database, payments, realtime, etc
	Entities    []string // user, product, order, etc
	Description string
}

// ParseIntent extracts structured intent from a natural language prompt
func ParseIntent(prompt string) AppIntent {
	lower := strings.ToLower(prompt)
	intent := AppIntent{Description: prompt}

	// App type
	intent.AppType = detectAppType(lower)

	// Platform
	intent.Platform = detectPlatform(lower)

	// Language
	intent.Language = detectLanguage(lower)

	// Framework
	intent.Framework = detectFramework(lower, intent.Platform, intent.Language)

	// Features
	intent.Features = detectFeatures(lower)

	// Entities
	intent.Entities = detectEntities(lower)

	// App name
	intent.AppName = inferAppName(lower, intent.AppType)

	return intent
}

func detectAppType(s string) string {
	switch {
	case containsAny(s, "mobile app", "android app", "ios app", "flutter"):
		return "mobile"
	case containsAny(s, "rest api", "api backend", "backend api", "web api", "fastapi", "gin api"):
		return "api"
	case containsAny(s, "web app", "website", "dashboard", "react app", "frontend"):
		return "web"
	case containsAny(s, "cli tool", "command line", "terminal tool", "command-line"):
		return "cli"
	case containsAny(s, "desktop app", "electron", "windows app", "mac app"):
		return "desktop"
	case containsAny(s, "api", "backend", "server", "service"):
		return "api"
	default:
		return "web"
	}
}

func detectPlatform(s string) string {
	// Strong explicit hints take priority
	for _, hint := range []struct{ phrase, platform string }{
		{"for android", "android"}, {"for web", "web"}, {"for ios", "mac"},
		{"for windows", "windows"}, {"for cli", "cli"}, {"for backend", "backend"},
		{"for mobile", "all"}, {"android app", "android"}, {"web app", "web"},
		{"web dashboard", "web"}, {"web ui", "web"},
	} {
		if strings.Contains(s, hint.phrase) {
			return hint.platform
		}
	}
	// fallthrough to original logic
	return detectPlatformOriginal(s)
}

func detectPlatformOriginal(s string) string {
	switch {
	case containsAny(s, "android", "kotlin", "jetpack"):
		return "android"
	case containsAny(s, "ios", "swift", "swiftui"):
		return "mac"
	case containsAny(s, "flutter", "dart"):
		return "all"
	case containsAny(s, "windows", "electron", "winforms", "wpf"):
		return "windows"
	case containsAny(s, "cli", "command line", "terminal"):
		return "cli"
	case containsAny(s, "backend", "api", "server", "fastapi", "django", "flask", "gin", "express"):
		return "backend"
	case containsAny(s, "web", "react", "vue", "svelte", "next", "dashboard", "website"):
		return "web"
	default:
		return "web"
	}
}

func detectLanguage(s string) string {
	switch {
	case containsAny(s, "python", "fastapi", "django", "flask"):
		return "python"
	case containsAny(s, "kotlin", "android"):
		return "kotlin"
	case containsAny(s, "typescript", "react", "vue", "next", "angular"):
		return "typescript"
	case containsAny(s, "javascript", "node", "express"):
		return "javascript"
	case containsAny(s, " go ", "golang", "gin", " chi "):
		return "go"
	case containsAny(s, "rust", "cargo"):
		return "rust"
	case containsAny(s, "swift", "swiftui", "ios"):
		return "swift"
	case containsAny(s, "dart", "flutter"):
		return "dart"
	case containsAny(s, "java", "spring"):
		return "java"
	case containsAny(s, "c#", "csharp", ".net", "dotnet"):
		return "csharp"
	default:
		return ""
	}
}

func detectFramework(s, platform, language string) string {
	// Explicit framework mentions
	switch {
	case strings.Contains(s, "fastapi"):
		return "fastapi"
	case strings.Contains(s, "django"):
		return "django"
	case strings.Contains(s, "flask"):
		return "flask"
	case strings.Contains(s, "react"):
		return "react"
	case strings.Contains(s, "vue"):
		return "vue"
	case strings.Contains(s, "next"):
		return "nextjs"
	case strings.Contains(s, "gin"):
		return "gin"
	case strings.Contains(s, "jetpack compose") || strings.Contains(s, "compose"):
		return "jetpack-compose"
	case strings.Contains(s, "swiftui"):
		return "swiftui"
	case strings.Contains(s, "flutter"):
		return "flutter"
	case strings.Contains(s, "electron"):
		return "electron"
	}

	// Infer from platform + language
	switch platform {
	case "android":
		return "jetpack-compose"
	case "web":
		if language == "typescript" || language == "" {
			return "react"
		}
	case "backend":
		switch language {
		case "python":
			return "fastapi"
		case "go":
			return "chi"
		case "javascript":
			return "express"
		}
	case "cli":
		switch language {
		case "go":
			return "cobra"
		case "rust":
			return "clap"
		case "python":
			return "click"
		}
	case "mac":
		return "swiftui"
	}
	return ""
}

func detectFeatures(s string) []string {
	featureMap := map[string][]string{
		"auth":      {"auth", "authentication", "login", "signup", "register", "jwt", "oauth", "session"},
		"database":  {"database", "db", "postgres", "postgresql", "mysql", "sqlite", "mongodb", "storage"},
		"payments":  {"payment", "payments", "stripe", "billing", "subscription", "checkout"},
		"realtime":  {"realtime", "real-time", "websocket", "live", "socket", "push notification"},
		"search":    {"search", "elasticsearch", "full-text", "filter"},
		"upload":    {"upload", "file upload", "image upload", "storage", "s3", "media"},
		"email":     {"email", "smtp", "sendgrid", "notification", "mailer"},
		"cache":     {"cache", "caching", "redis", "memcache"},
		"api":       {"rest api", "graphql", "api", "endpoint"},
		"docker":    {"docker", "container", "kubernetes", "k8s", "deploy"},
		"testing":   {"test", "testing", "unit test", "e2e"},
		"maps":      {"map", "maps", "location", "gps", "geolocation"},
		"analytics": {"analytics", "tracking", "metrics", "dashboard"},
	}

	seen := map[string]bool{}
	var features []string
	for feature, keywords := range featureMap {
		for _, kw := range keywords {
			if strings.Contains(s, kw) && !seen[feature] {
				features = append(features, feature)
				seen[feature] = true
				break
			}
		}
	}
	return features
}

func detectEntities(s string) []string {
	entityKeywords := []string{
		"user", "product", "order", "item", "category", "post", "comment",
		"review", "invoice", "customer", "employee", "restaurant", "food",
		"delivery", "driver", "payment", "booking", "reservation", "event",
		"message", "notification", "report", "task", "project", "ticket",
	}
	seen := map[string]bool{}
	var entities []string
	for _, e := range entityKeywords {
		if strings.Contains(s, e) && !seen[e] {
			entities = append(entities, e)
			seen[e] = true
		}
	}
	return entities
}

func inferAppName(s, appType string) string {
	// Try to find "called X" or "named X" patterns
	for _, prefix := range []string{"called ", "named ", "app called ", "app named "} {
		if idx := strings.Index(s, prefix); idx != -1 {
			rest := s[idx+len(prefix):]
			words := strings.Fields(rest)
			if len(words) > 0 {
				return strings.Title(words[0])
			}
		}
	}

	// Infer from domain entities
	entities := detectEntities(s)
	if len(entities) > 0 {
		switch appType {
		case "api":
			return strings.Title(entities[0]) + "API"
		case "mobile":
			return strings.Title(entities[0]) + "App"
		default:
			return strings.Title(entities[0]) + "App"
		}
	}

	switch appType {
	case "api":
		return "MyAPI"
	case "mobile":
		return "MyApp"
	case "cli":
		return "mytool"
	default:
		return "MyApp"
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
