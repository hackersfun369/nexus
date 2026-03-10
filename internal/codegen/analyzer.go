package codegen

import (
	"context"
	"strings"
)

// Issue is a quality issue found in generated code
type Issue struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
}

// QualityReport is the result of analyzing generated files
type QualityReport struct {
	Issues     []Issue `json:"issues"`
	IssueCount int     `json:"issue_count"`
	Score      int     `json:"score"` // 0-100
	Passed     bool    `json:"passed"`
}

// Analyze runs static analysis on generated files
func Analyze(ctx context.Context, files []GeneratedFile) QualityReport {
	var issues []Issue

	for _, f := range files {
		fileIssues := analyzeFile(f)
		issues = append(issues, fileIssues...)
	}

	score := calcScore(len(issues), len(files))
	return QualityReport{
		Issues:     issues,
		IssueCount: len(issues),
		Score:      score,
		Passed:     score >= 70,
	}
}

func analyzeFile(f GeneratedFile) []Issue {
	var issues []Issue
	lines := strings.Split(f.Content, "\n")

	for i, line := range lines {
		lineNum := i + 1

		// SEC-001: hardcoded secrets
		if containsSecret(line) {
			issues = append(issues, Issue{
				RuleID:   "NEXUS-SEC-001",
				Severity: "critical",
				Category: "security",
				File:     f.Path,
				Line:     lineNum,
				Message:  "Potential hardcoded secret or credential detected",
				Fix:      "Move secrets to environment variables or a secrets manager",
			})
		}

		// SEC-002: SQL injection risk
		if hasSQLInjection(line, f.Language) {
			issues = append(issues, Issue{
				RuleID:   "NEXUS-SEC-002",
				Severity: "high",
				Category: "security",
				File:     f.Path,
				Line:     lineNum,
				Message:  "Possible SQL injection vulnerability — use parameterized queries",
				Fix:      "Replace string-formatted SQL with parameterized queries",
			})
		}

		// COMP-001: function too long (rough heuristic)
		if isLongFunction(lines, i, f.Language) {
			issues = append(issues, Issue{
				RuleID:   "NEXUS-COMP-001",
				Severity: "medium",
				Category: "complexity",
				File:     f.Path,
				Line:     lineNum,
				Message:  "Function exceeds recommended length (>50 lines)",
				Fix:      "Break this function into smaller, focused functions",
			})
		}

		// DOC-001: missing docstring on public function
		if missingDocstring(lines, i, f.Language) {
			issues = append(issues, Issue{
				RuleID:   "NEXUS-DOC-001",
				Severity: "low",
				Category: "documentation",
				File:     f.Path,
				Line:     lineNum,
				Message:  "Public function is missing a documentation comment",
				Fix:      "Add a docstring describing the function's purpose",
			})
		}

		// PERF-001: N+1 query risk
		if hasNPlusOne(line, f.Language) {
			issues = append(issues, Issue{
				RuleID:   "NEXUS-PERF-001",
				Severity: "medium",
				Category: "performance",
				File:     f.Path,
				Line:     lineNum,
				Message:  "Possible N+1 query pattern inside a loop",
				Fix:      "Use batch queries or eager loading instead",
			})
		}
	}

	return issues
}

func containsSecret(line string) bool {
	lower := strings.ToLower(line)
	secretPatterns := []string{
		"password = \"", "password=\"", "secret = \"", "secret=\"",
		"api_key = \"", "api_key=\"", "token = \"", "token=\"",
		"private_key", "aws_secret", "database_url = \"postgres",
	}
	for _, p := range secretPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func hasSQLInjection(line, lang string) bool {
	if lang != "python" && lang != "go" && lang != "javascript" {
		return false
	}
	lower := strings.ToLower(line)
	return (strings.Contains(lower, "execute(") || strings.Contains(lower, "query(")) &&
		(strings.Contains(lower, "f\"") || strings.Contains(lower, "f'") ||
			strings.Contains(lower, "%s") || strings.Contains(lower, "sprintf"))
}

func isLongFunction(lines []string, idx int, lang string) bool {
	// Only trigger once at function start
	line := strings.TrimSpace(lines[idx])
	isFuncStart := false
	switch lang {
	case "python":
		isFuncStart = strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "async def ")
	case "go":
		isFuncStart = strings.HasPrefix(line, "func ")
	case "typescript", "javascript":
		isFuncStart = strings.Contains(line, "function ") || strings.Contains(line, "=> {")
	case "kotlin":
		isFuncStart = strings.HasPrefix(line, "fun ")
	}
	if !isFuncStart {
		return false
	}

	// Count lines until function ends
	depth := 0
	count := 0
	for j := idx; j < len(lines) && j < idx+200; j++ {
		l := lines[j]
		depth += strings.Count(l, "{") - strings.Count(l, "}")
		if lang == "python" {
			if j > idx && len(l) > 0 && l[0] != ' ' && l[0] != '\t' && l[0] != '#' {
				break
			}
		}
		count++
		if depth < 0 {
			break
		}
	}
	return count > 50
}

func missingDocstring(lines []string, idx int, lang string) bool {
	line := strings.TrimSpace(lines[idx])
	isPublic := false
	switch lang {
	case "python":
		isPublic = (strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "async def ")) &&
			len(line) > 4 && line[4] != '_'
		if isPublic && idx > 0 {
			prev := strings.TrimSpace(lines[idx-1])
			if strings.HasPrefix(prev, "#") || strings.HasSuffix(prev, `"""`) {
				return false
			}
		}
	case "go":
		isPublic = strings.HasPrefix(line, "func ") &&
			len(line) > 5 && line[5] >= 'A' && line[5] <= 'Z'
		if isPublic && idx > 0 {
			prev := strings.TrimSpace(lines[idx-1])
			if strings.HasPrefix(prev, "//") {
				return false
			}
		}
	}
	return isPublic
}

func hasNPlusOne(line, lang string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	dbCalls := []string{"db.query", "db.execute", "session.query", ".find(", ".get(", "fetch("}
	loopKeywords := []string{"for ", "while "}

	isInLoop := false
	for _, kw := range loopKeywords {
		if strings.HasPrefix(lower, kw) {
			isInLoop = true
			break
		}
	}
	_ = isInLoop

	// Simplified: flag DB calls that appear indented (likely inside loop)
	isIndented := strings.HasPrefix(line, "\t\t") || strings.HasPrefix(line, "        ")
	if !isIndented {
		return false
	}
	for _, call := range dbCalls {
		if strings.Contains(lower, call) {
			return true
		}
	}
	return false
}

func calcScore(issueCount, fileCount int) int {
	if fileCount == 0 {
		return 100
	}
	base := 100
	for _, issue := range []int{} {
		_ = issue
	}
	// Deduct points based on issue density
	deduction := issueCount * 5
	if deduction > 80 {
		deduction = 80
	}
	score := base - deduction
	if score < 0 {
		return 0
	}
	return score
}
