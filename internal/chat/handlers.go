package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
	"github.com/hackersfun369/nexus/internal/rules/loader"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Handle processes a parsed command and returns a response string
func Handle(ctx context.Context, cmd ParsedCommand, sess *Session) (string, bool) {
	switch cmd.Intent {
	case IntentQuit:
		return colorGreen + "Goodbye! 👋" + colorReset, true

	case IntentHelp:
		return handleHelp(), false

	case IntentVersion:
		return colorCyan + "nexus dev — intelligent development system" + colorReset, false

	case IntentClearScreen:
		return "\033[2J\033[H", false

	case IntentSetProject:
		return handleSetProject(ctx, cmd, sess), false

	case IntentShowProject:
		return handleShowProject(sess), false

	case IntentAnalyze:
		return handleAnalyze(ctx, cmd, sess), false

	case IntentShowIssues:
		return handleShowIssues(ctx, cmd, sess), false

	case IntentShowSummary:
		return handleShowSummary(ctx, sess), false

	case IntentShowRules:
		return handleShowRules(sess), false

	case IntentExplain:
		return handleExplain(cmd), false

	case IntentFilter:
		return handleFilter(cmd, sess), false

	case IntentUnknown:
		return colorYellow + "I didn't understand that. Type " +
			colorBold + "help" + colorReset +
			colorYellow + " to see available commands." + colorReset, false
	}
	return "", false
}

func handleHelp() string {
	var b strings.Builder
	b.WriteString(colorBold + colorCyan + "\n  NEXUS — Available Commands\n" + colorReset)
	b.WriteString(colorDim + "  ─────────────────────────────────────────\n" + colorReset)

	cmds := [][2]string{
		{"analyze", "Run all rules on the current project"},
		{"issues", "List all findings (add: severity=high, category=SECURITY)"},
		{"summary", "Show issue counts by severity and category"},
		{"rules", "List all loaded rules"},
		{"use project <id>", "Switch to a project by ID"},
		{"show project", "Show the current project"},
		{"explain <rule-id>", "Explain what a rule checks (e.g. explain NEXUS-SEC-001)"},
		{"filter severity=<level>", "Filter issues by severity (HIGH/MEDIUM/LOW)"},
		{"version", "Show nexus version"},
		{"clear", "Clear the screen"},
		{"help", "Show this message"},
		{"quit", "Exit nexus"},
	}

	for _, c := range cmds {
		b.WriteString(fmt.Sprintf("  %s%-25s%s %s\n",
			colorGreen, c[0], colorReset, colorDim+c[1]+colorReset))
	}
	b.WriteString("")
	return b.String()
}

func handleSetProject(ctx context.Context, cmd ParsedCommand, sess *Session) string {
	target := cmd.Args["target"]
	if target == "" {
		// List available projects
		projects, err := sess.Store.ListProjects(ctx)
		if err != nil {
			return colorRed + "Error listing projects: " + err.Error() + colorReset
		}
		if len(projects) == 0 {
			return colorYellow + "No projects found. Create one via the API first." + colorReset
		}
		var b strings.Builder
		b.WriteString(colorBold + "\n  Available Projects:\n" + colorReset)
		for _, p := range projects {
			b.WriteString(fmt.Sprintf("  %s%-20s%s %s%s%s\n",
				colorCyan, p.ID, colorReset, colorDim, p.RootPath, colorReset))
		}
		return b.String()
	}

	// Try to find by ID or name
	project, err := sess.Store.GetProject(ctx, target)
	if err != nil {
		// Try listing and matching by name
		projects, lerr := sess.Store.ListProjects(ctx)
		if lerr != nil {
			return colorRed + "Project not found: " + target + colorReset
		}
		found := false
		for _, p := range projects {
			if strings.EqualFold(p.Name, target) {
				project = p
				found = true
				break
			}
		}
		if !found {
			return colorRed + "Project not found: " + target + colorReset
		}
	}

	sess.SetProject(project)
	return fmt.Sprintf("%s✓ Switched to project %s%s%s (%s)%s",
		colorGreen, colorBold, project.Name, colorReset+colorGreen, project.RootPath, colorReset)
}

func handleShowProject(sess *Session) string {
	if !sess.HasProject() {
		return colorYellow + "No project selected. Type " +
			colorBold + "use project <id>" + colorReset +
			colorYellow + " to select one." + colorReset
	}
	return fmt.Sprintf("%s%s  Project:%s %s\n%s  Path:   %s %s%s",
		colorBold, colorCyan, colorReset, sess.ProjectName,
		colorDim, colorReset, sess.ProjectRootPath, colorReset)
}

func handleAnalyze(ctx context.Context, cmd ParsedCommand, sess *Session) string {
	if !sess.HasProject() {
		return colorYellow + "No project selected. Use: " + colorBold + "use project <id>" + colorReset
	}

	fmt.Printf("%s  Analyzing project %s...%s\n", colorDim, sess.ProjectName, colorReset)

	reg := loader.DefaultRegistry(sess.Config)
	engine := rules.NewEngine(reg, sess.Store)

	result, err := engine.RunAll(ctx, sess.ProjectID)
	if err != nil {
		return colorRed + "Analysis failed: " + err.Error() + colorReset
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%s%s  Analysis Complete%s\n", colorBold, colorGreen, colorReset))
	b.WriteString(fmt.Sprintf("%s  ─────────────────────────────────%s\n", colorDim, colorReset))
	b.WriteString(fmt.Sprintf("  Rules run:    %s%d%s\n", colorCyan, result.RulesRun, colorReset))
	b.WriteString(fmt.Sprintf("  Issues found: %s%d%s\n", issueColor(result.IssuesFound), result.IssuesFound, colorReset))
	b.WriteString(fmt.Sprintf("  Duration:     %s%s%s\n", colorDim, result.Duration().Round(1000000), colorReset))

	if result.HasErrors() {
		b.WriteString(fmt.Sprintf("\n%s  %d rule(s) had errors%s\n", colorYellow, len(result.Errors), colorReset))
	}

	if result.IssuesFound > 0 {
		b.WriteString(fmt.Sprintf("\n%s  Type 'issues' to see findings or 'summary' for an overview.%s\n",
			colorDim, colorReset))
	}
	return b.String()
}

func handleShowIssues(ctx context.Context, cmd ParsedCommand, sess *Session) string {
	if !sess.HasProject() {
		return colorYellow + "No project selected." + colorReset
	}

	filter := store.IssueFilter{ProjectID: sess.ProjectID}

	// Apply session filters
	if sess.SeverityFilter != "" {
		filter.Severity = sess.SeverityFilter
	}
	if sess.CategoryFilter != "" {
		filter.Category = sess.CategoryFilter
	}

	// Apply inline filters from command args
	if target := cmd.Args["target"]; target != "" {
		lower := strings.ToLower(target)
		for _, sev := range []string{"high", "medium", "low", "critical"} {
			if strings.Contains(lower, sev) {
				filter.Severity = strings.ToUpper(sev)
			}
		}
	}

	issues, err := sess.Store.QueryIssues(ctx, filter)
	if err != nil {
		return colorRed + "Error fetching issues: " + err.Error() + colorReset
	}

	if len(issues) == 0 {
		return colorGreen + "  ✓ No issues found" + colorReset
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%s%s  Issues (%d)%s\n", colorBold, colorCyan, len(issues), colorReset))
	b.WriteString(fmt.Sprintf("%s  ─────────────────────────────────────────────%s\n", colorDim, colorReset))

	for i, issue := range issues {
		if i >= 20 {
			b.WriteString(fmt.Sprintf("%s  ... and %d more. Use filter to narrow results.%s\n",
				colorDim, len(issues)-20, colorReset))
			break
		}
		sev := severityBadge(issue.Severity)
		b.WriteString(fmt.Sprintf("  %s %s%s%s\n", sev, colorBold, issue.Title, colorReset))
		b.WriteString(fmt.Sprintf("     %s%s%s\n", colorDim, issue.FilePath, colorReset))
		if issue.StartLine > 0 {
			b.WriteString(fmt.Sprintf("     %sLine %d%s\n", colorDim, issue.StartLine, colorReset))
		}
		b.WriteString(fmt.Sprintf("     %s[%s]%s\n\n", colorDim, issue.RuleID, colorReset))
	}
	return b.String()
}

func handleShowSummary(ctx context.Context, sess *Session) string {
	if !sess.HasProject() {
		return colorYellow + "No project selected." + colorReset
	}

	issues, err := sess.Store.QueryIssues(ctx, store.IssueFilter{ProjectID: sess.ProjectID})
	if err != nil {
		return colorRed + "Error: " + err.Error() + colorReset
	}

	bySeverity := make(map[string]int)
	byCategory := make(map[string]int)
	for _, issue := range issues {
		bySeverity[issue.Severity]++
		byCategory[issue.Category]++
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%s%s  Summary — %s%s\n", colorBold, colorCyan, sess.ProjectName, colorReset))
	b.WriteString(fmt.Sprintf("%s  ─────────────────────────────────%s\n", colorDim, colorReset))
	b.WriteString(fmt.Sprintf("  Total issues: %s%d%s\n\n", issueColor(len(issues)), len(issues), colorReset))

	b.WriteString(fmt.Sprintf("  %sBY SEVERITY%s\n", colorBold, colorReset))
	for _, sev := range []string{"HIGH", "MEDIUM", "LOW"} {
		if n, ok := bySeverity[sev]; ok && n > 0 {
			b.WriteString(fmt.Sprintf("  %s  %-10s %d%s\n", severityColor(sev), sev, n, colorReset))
		}
	}

	b.WriteString(fmt.Sprintf("\n  %sBY CATEGORY%s\n", colorBold, colorReset))
	for cat, n := range byCategory {
		b.WriteString(fmt.Sprintf("  %s  %-20s %d%s\n", colorCyan, cat, n, colorReset))
	}
	return b.String()
}

func handleShowRules(sess *Session) string {
	reg := loader.DefaultRegistry(sess.Config)
	allRules := reg.All()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%s%s  Loaded Rules (%d)%s\n", colorBold, colorCyan, len(allRules), colorReset))
	b.WriteString(fmt.Sprintf("%s  ─────────────────────────────────────────────────────%s\n", colorDim, colorReset))

	byCategory := make(map[string][]rules.Rule)
	for _, r := range allRules {
		cat := string(r.Category())
		byCategory[cat] = append(byCategory[cat], r)
	}

	for cat, catRules := range byCategory {
		b.WriteString(fmt.Sprintf("\n  %s%s%s\n", colorBold, cat, colorReset))
		for _, r := range catRules {
			b.WriteString(fmt.Sprintf("  %s%-20s%s %s%s%s\n",
				colorGreen, r.ID(), colorReset, colorDim, r.Name(), colorReset))
		}
	}
	return b.String()
}

func handleExplain(cmd ParsedCommand) string {
	target := strings.ToUpper(strings.TrimSpace(cmd.Args["target"]))
	if target == "" {
		return colorYellow + "Usage: explain <rule-id>  e.g. explain NEXUS-SEC-001" + colorReset
	}

	reg := loader.DefaultRegistry(rules.DefaultConfig())
	rule, ok := reg.Get(target)
	if !ok {
		return colorRed + "Rule not found: " + target + colorReset
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%s%s  %s%s\n", colorBold, colorCyan, rule.ID(), colorReset))
	b.WriteString(fmt.Sprintf("%s  ─────────────────────────────────%s\n", colorDim, colorReset))
	b.WriteString(fmt.Sprintf("  %sName:%s     %s\n", colorBold, colorReset, rule.Name()))
	b.WriteString(fmt.Sprintf("  %sSeverity:%s  %s%s%s\n", colorBold, colorReset,
		severityColor(string(rule.Severity())), rule.Severity(), colorReset))
	b.WriteString(fmt.Sprintf("  %sCategory:%s  %s\n", colorBold, colorReset, rule.Category()))
	b.WriteString(fmt.Sprintf("  %sDetects:%s   %s\n", colorBold, colorReset, rule.Description()))
	b.WriteString(fmt.Sprintf("  %sFix:%s       %s\n", colorBold, colorReset, rule.Remediation()))
	return b.String()
}

func handleFilter(cmd ParsedCommand, sess *Session) string {
	target := strings.ToLower(cmd.Args["target"])
	if target == "" {
		sess.SeverityFilter = ""
		sess.CategoryFilter = ""
		return colorGreen + "  Filters cleared." + colorReset
	}

	for _, sev := range []string{"high", "medium", "low", "critical"} {
		if strings.Contains(target, sev) {
			sess.SeverityFilter = strings.ToUpper(sev)
			return fmt.Sprintf("%s  Filter set: severity=%s%s", colorGreen, sess.SeverityFilter, colorReset)
		}
	}

	// Try as category
	sess.CategoryFilter = strings.ToUpper(strings.TrimSpace(cmd.Args["target"]))
	return fmt.Sprintf("%s  Filter set: category=%s%s", colorGreen, sess.CategoryFilter, colorReset)
}

// ── HELPERS ───────────────────────────────────────────

func issueColor(n int) string {
	if n == 0 {
		return colorGreen
	}
	if n < 5 {
		return colorYellow
	}
	return colorRed
}

func severityColor(sev string) string {
	switch strings.ToUpper(sev) {
	case "HIGH", "CRITICAL":
		return colorRed
	case "MEDIUM":
		return colorYellow
	default:
		return colorDim
	}
}

func severityBadge(sev string) string {
	switch strings.ToUpper(sev) {
	case "HIGH", "CRITICAL":
		return colorRed + "[HIGH]  " + colorReset
	case "MEDIUM":
		return colorYellow + "[MED]   " + colorReset
	default:
		return colorDim + "[LOW]   " + colorReset
	}
}
