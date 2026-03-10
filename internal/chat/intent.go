package chat

import (
	"strings"
)

// Intent represents what the user wants to do
type Intent int

const (
	IntentUnknown Intent = iota
	IntentHelp
	IntentVersion
	IntentAnalyze
	IntentShowIssues
	IntentShowSummary
	IntentShowRules
	IntentSetProject
	IntentShowProject
	IntentClearScreen
	IntentQuit
	IntentExplain
	IntentFilter
)

// ParsedCommand holds the detected intent and extracted args
type ParsedCommand struct {
	Intent  Intent
	Args    map[string]string
	Raw     string
}

// keywords maps trigger words to intents — symbolic NLP
var keywords = []struct {
	intent   Intent
	triggers []string
}{
	{IntentQuit,        []string{"quit", "exit", "bye", "q"}},
	{IntentHelp,        []string{"help", "?", "commands", "what can you do"}},
	{IntentVersion,     []string{"version", "ver"}},
	{IntentClearScreen, []string{"clear", "cls"}},
	{IntentAnalyze,     []string{"analyze", "analyse", "scan", "check", "run", "inspect"}},
	{IntentShowIssues,  []string{"issues", "problems", "findings", "errors", "warnings", "list issues", "show issues"}},
	{IntentShowSummary, []string{"summary", "overview", "report", "stats", "statistics"}},
	{IntentShowRules,   []string{"rules", "show rules", "list rules", "what rules"}},
	{IntentSetProject,  []string{"use project", "set project", "open project", "switch project", "project "}},
	{IntentShowProject, []string{"current project", "which project", "show project"}},
	{IntentExplain,     []string{"explain", "what is", "what does", "tell me about", "describe"}},
	{IntentFilter,      []string{"filter", "only show", "severity", "category"}},
}

// ParseCommand detects intent from natural language input
func ParseCommand(input string) ParsedCommand {
	raw := strings.TrimSpace(input)
	lower := strings.ToLower(raw)
	cmd := ParsedCommand{
		Raw:  raw,
		Args: make(map[string]string),
	}

	if lower == "" {
		return cmd
	}

	// Match intent by keywords
	for _, k := range keywords {
		for _, trigger := range k.triggers {
			if lower == trigger || strings.HasPrefix(lower, trigger+" ") || strings.Contains(lower, trigger) {
				cmd.Intent = k.intent
				// Extract trailing argument
				if idx := strings.Index(lower, trigger); idx >= 0 {
					rest := strings.TrimSpace(raw[idx+len(trigger):])
					if rest != "" {
						cmd.Args["target"] = rest
					}
				}
				return cmd
			}
		}
	}

	// If no intent matched but looks like a project path, treat as set project
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") || strings.HasPrefix(raw, ".") {
		cmd.Intent = IntentSetProject
		cmd.Args["target"] = raw
		return cmd
	}

	cmd.Intent = IntentUnknown
	return cmd
}

// intentName returns a human readable name for logging
func intentName(i Intent) string {
	names := map[Intent]string{
		IntentHelp:        "help",
		IntentVersion:     "version",
		IntentAnalyze:     "analyze",
		IntentShowIssues:  "show_issues",
		IntentShowSummary: "summary",
		IntentShowRules:   "show_rules",
		IntentSetProject:  "set_project",
		IntentShowProject: "show_project",
		IntentClearScreen: "clear",
		IntentQuit:        "quit",
		IntentExplain:     "explain",
		IntentFilter:      "filter",
	}
	if name, ok := names[i]; ok {
		return name
	}
	return "unknown"
}
