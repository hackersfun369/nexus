package chat_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hackersfun369/nexus/internal/chat"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

var ctx = context.Background()

func newTestSession(t *testing.T) *chat.Session {
	t.Helper()
	dir, err := os.MkdirTemp("", "nexus-chat-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	s, err := store.NewSQLiteStore(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	s.CreateProject(ctx, store.Project{
		ID: "proj-001", Name: "myapp",
		RootPath: "/home/user/myapp", Platform: "web",
		PrimaryLanguage: "python",
	})
	return chat.NewSession(s, rules.DefaultConfig())
}

// ── INTENT PARSING ────────────────────────────────────

func TestParseCommand_Help(t *testing.T) {
	for _, input := range []string{"help", "?", "commands"} {
		cmd := chat.ParseCommand(input)
		if cmd.Intent != chat.IntentHelp {
			t.Errorf("Expected IntentHelp for %q, got %d", input, cmd.Intent)
		}
	}
	t.Logf("✅ ParseCommand: help intents")
}

func TestParseCommand_Quit(t *testing.T) {
	for _, input := range []string{"quit", "exit", "bye", "q"} {
		cmd := chat.ParseCommand(input)
		if cmd.Intent != chat.IntentQuit {
			t.Errorf("Expected IntentQuit for %q, got %d", input, cmd.Intent)
		}
	}
	t.Logf("✅ ParseCommand: quit intents")
}

func TestParseCommand_Analyze(t *testing.T) {
	for _, input := range []string{"analyze", "scan", "run", "check"} {
		cmd := chat.ParseCommand(input)
		if cmd.Intent != chat.IntentAnalyze {
			t.Errorf("Expected IntentAnalyze for %q, got %d", input, cmd.Intent)
		}
	}
	t.Logf("✅ ParseCommand: analyze intents")
}

func TestParseCommand_Issues(t *testing.T) {
	for _, input := range []string{"issues", "problems", "findings"} {
		cmd := chat.ParseCommand(input)
		if cmd.Intent != chat.IntentShowIssues {
			t.Errorf("Expected IntentShowIssues for %q, got %d", input, cmd.Intent)
		}
	}
	t.Logf("✅ ParseCommand: issues intents")
}

func TestParseCommand_SetProject(t *testing.T) {
	cmd := chat.ParseCommand("use project myapp")
	if cmd.Intent != chat.IntentSetProject {
		t.Errorf("Expected IntentSetProject, got %d", cmd.Intent)
	}
	if cmd.Args["target"] != "myapp" {
		t.Errorf("Expected target=myapp, got %q", cmd.Args["target"])
	}
	t.Logf("✅ ParseCommand: set project with target")
}

func TestParseCommand_Explain(t *testing.T) {
	cmd := chat.ParseCommand("explain NEXUS-SEC-001")
	if cmd.Intent != chat.IntentExplain {
		t.Errorf("Expected IntentExplain, got %d", cmd.Intent)
	}
	t.Logf("✅ ParseCommand: explain intent")
}

func TestParseCommand_Unknown(t *testing.T) {
	cmd := chat.ParseCommand("random gibberish xyz")
	if cmd.Intent != chat.IntentUnknown {
		t.Errorf("Expected IntentUnknown, got %d", cmd.Intent)
	}
	t.Logf("✅ ParseCommand: unknown intent")
}

func TestParseCommand_Empty(t *testing.T) {
	cmd := chat.ParseCommand("")
	if cmd.Intent != chat.IntentUnknown {
		t.Errorf("Expected IntentUnknown for empty, got %d", cmd.Intent)
	}
	t.Logf("✅ ParseCommand: empty input")
}

// ── SESSION ───────────────────────────────────────────

func TestSession_HasProject(t *testing.T) {
	sess := newTestSession(t)
	if sess.HasProject() {
		t.Error("Expected no project initially")
	}
	sess.SetProject(store.Project{ID: "p1", Name: "app", RootPath: "/tmp"})
	if !sess.HasProject() {
		t.Error("Expected project after SetProject")
	}
	sess.ClearProject()
	if sess.HasProject() {
		t.Error("Expected no project after ClearProject")
	}
	t.Logf("✅ Session: HasProject/SetProject/ClearProject")
}

func TestSession_Prompt(t *testing.T) {
	sess := newTestSession(t)
	if sess.Prompt() != "nexus > " {
		t.Errorf("Expected 'nexus > ', got %q", sess.Prompt())
	}
	sess.SetProject(store.Project{ID: "p1", Name: "myapp", RootPath: "/tmp"})
	if sess.Prompt() != "nexus [myapp] > " {
		t.Errorf("Expected 'nexus [myapp] > ', got %q", sess.Prompt())
	}
	t.Logf("✅ Session: Prompt")
}

func TestSession_History(t *testing.T) {
	sess := newTestSession(t)
	sess.AddHistory("analyze", "found 3 issues")
	sess.AddHistory("issues", "...")
	if len(sess.History) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(sess.History))
	}
	t.Logf("✅ Session: AddHistory")
}

// ── HANDLERS ──────────────────────────────────────────

func TestHandle_Quit(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("quit")
	_, quit := chat.Handle(ctx, cmd, sess)
	if !quit {
		t.Error("Expected quit=true for quit command")
	}
	t.Logf("✅ Handle: quit returns true")
}

func TestHandle_Help(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("help")
	response, quit := chat.Handle(ctx, cmd, sess)
	if quit {
		t.Error("Expected quit=false for help")
	}
	if !strings.Contains(response, "analyze") {
		t.Error("Expected help to mention 'analyze'")
	}
	t.Logf("✅ Handle: help response")
}

func TestHandle_Analyze_NoProject(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("analyze")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !strings.Contains(strings.ToLower(response), "no project") {
		t.Errorf("Expected 'no project' message, got: %s", response)
	}
	t.Logf("✅ Handle: analyze without project")
}

func TestHandle_Analyze_WithProject(t *testing.T) {
	sess := newTestSession(t)
	sess.SetProject(store.Project{
		ID: "proj-001", Name: "myapp", RootPath: "/home/user/myapp",
	})
	cmd := chat.ParseCommand("analyze")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !strings.Contains(response, "Analysis Complete") {
		t.Errorf("Expected 'Analysis Complete', got: %s", response)
	}
	t.Logf("✅ Handle: analyze with project")
}

func TestHandle_ShowIssues_NoProject(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("issues")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !strings.Contains(strings.ToLower(response), "no project") {
		t.Errorf("Expected 'no project' message, got: %s", response)
	}
	t.Logf("✅ Handle: issues without project")
}

func TestHandle_SetProject_Found(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("use project proj-001")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !sess.HasProject() {
		t.Error("Expected project to be set after 'use project'")
	}
	if !strings.Contains(response, "myapp") {
		t.Errorf("Expected project name in response, got: %s", response)
	}
	t.Logf("✅ Handle: set project found")
}

func TestHandle_SetProject_NotFound(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("use project nonexistent")
	response, _ := chat.Handle(ctx, cmd, sess)
	if sess.HasProject() {
		t.Error("Expected no project after failed set")
	}
	if !strings.Contains(strings.ToLower(response), "not found") {
		t.Errorf("Expected 'not found' message, got: %s", response)
	}
	t.Logf("✅ Handle: set project not found")
}

func TestHandle_ShowRules(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("rules")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !strings.Contains(response, "NEXUS-") {
		t.Errorf("Expected rule IDs in response, got: %s", response)
	}
	t.Logf("✅ Handle: show rules")
}

func TestHandle_Explain(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("explain NEXUS-SEC-001")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !strings.Contains(response, "NEXUS-SEC-001") {
		t.Errorf("Expected rule ID in response, got: %s", response)
	}
	t.Logf("✅ Handle: explain rule")
}

func TestHandle_Unknown(t *testing.T) {
	sess := newTestSession(t)
	cmd := chat.ParseCommand("xyzzy plugh")
	response, _ := chat.Handle(ctx, cmd, sess)
	if !strings.Contains(strings.ToLower(response), "didn't understand") {
		t.Errorf("Expected unknown response, got: %s", response)
	}
	t.Logf("✅ Handle: unknown command")
}

// ── REPL ──────────────────────────────────────────────

func TestREPL_QuitOnEOF(t *testing.T) {
	sess := newTestSession(t)
	in := strings.NewReader("")
	var out strings.Builder
	repl := chat.NewREPLWithIO(sess, in, &out)
	repl.Run(ctx)
	// Should exit cleanly on EOF
	t.Logf("✅ REPL: exits on EOF")
}

func TestREPL_ProcessesCommands(t *testing.T) {
	sess := newTestSession(t)
	in := strings.NewReader("help\nquit\n")
	var out strings.Builder
	repl := chat.NewREPLWithIO(sess, in, &out)
	repl.Run(ctx)
	output := out.String()
	if !strings.Contains(output, "analyze") {
		t.Errorf("Expected help output, got: %s", output)
	}
	t.Logf("✅ REPL: processes help then quit")
}
