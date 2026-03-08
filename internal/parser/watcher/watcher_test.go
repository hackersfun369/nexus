package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hackersfun369/nexus/internal/parser/watcher"
)

// ── HELPERS ───────────────────────────────────────────

// makeTempProject creates a temporary directory with source files
func makeTempProject(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "nexus-watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(dir) })

	// Create initial files
	writeFile(t, filepath.Join(dir, "main.py"), `def main(): pass`)
	writeFile(t, filepath.Join(dir, "service.ts"), `export class Service {}`)

	// Create a subdirectory
	subDir := filepath.Join(dir, "src")
	os.MkdirAll(subDir, 0755)
	writeFile(t, filepath.Join(subDir, "utils.py"), `def util(): pass`)

	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func newTestWatcher(t *testing.T) *watcher.Watcher {
	t.Helper()
	config := watcher.DefaultConfig()
	config.DebounceDuration = 20 * time.Millisecond // Fast for tests

	w, err := watcher.New(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	return w
}

// waitForEvent waits for an event matching the predicate
func waitForEvent(
	t *testing.T,
	w *watcher.Watcher,
	timeout time.Duration,
	predicate func(watcher.FileEvent) bool,
) (watcher.FileEvent, bool) {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case event := <-w.Events():
			if predicate(event) {
				return event, true
			}
		case <-deadline:
			return watcher.FileEvent{}, false
		}
	}
}

// ── CONFIG TESTS ──────────────────────────────────────

func TestDefaultConfig(t *testing.T) {
	config := watcher.DefaultConfig()

	if config.DebounceDuration == 0 {
		t.Error("Expected non-zero debounce duration")
	}
	if len(config.Extensions) == 0 {
		t.Error("Expected non-empty extensions")
	}
	if len(config.IgnoreDirs) == 0 {
		t.Error("Expected non-empty ignore dirs")
	}
	t.Logf("✅ DefaultConfig: debounce=%v extensions=%v",
		config.DebounceDuration, config.Extensions)
}

// ── LANGUAGE DETECTION ────────────────────────────────

func TestDetectLanguage_Python(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)

	// Modify Python file
	time.Sleep(30 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "main.py"), `def main(): return 42`)

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return e.Language == watcher.LangPython
	})

	if !found {
		t.Fatal("Expected Python file event")
	}
	if event.Language != watcher.LangPython {
		t.Errorf("Expected LangPython, got %s", event.Language)
	}
	t.Logf("✅ Python detected: %s", event.FilePath)
}

func TestDetectLanguage_TypeScript(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)

	time.Sleep(30 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "service.ts"), `export class Service { id = 1 }`)

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return e.Language == watcher.LangTypeScript
	})

	if !found {
		t.Fatal("Expected TypeScript file event")
	}
	t.Logf("✅ TypeScript detected: %s", event.FilePath)
}

// ── EVENT KINDS ───────────────────────────────────────

func TestEventKind_Modified(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)

	time.Sleep(30 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "main.py"), `def main(): return 99`)

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return e.Kind == watcher.EventModified &&
			filepath.Base(e.FilePath) == "main.py"
	})

	if !found {
		t.Fatal("Expected MODIFIED event for main.py")
	}
	if event.Kind != watcher.EventModified {
		t.Errorf("Expected MODIFIED, got %s", event.Kind)
	}
	t.Logf("✅ Modified event: %s", filepath.Base(event.FilePath))
}

func TestEventKind_Created(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)

	time.Sleep(30 * time.Millisecond)
	newFile := filepath.Join(dir, "new_module.py")
	writeFile(t, newFile, `def new_func(): pass`)

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return (e.Kind == watcher.EventCreated ||
			e.Kind == watcher.EventModified) &&
			filepath.Base(e.FilePath) == "new_module.py"
	})

	if !found {
		t.Fatal("Expected event for new_module.py")
	}
	t.Logf("✅ Created event: %s kind=%s",
		filepath.Base(event.FilePath), event.Kind)
}

func TestEventKind_Deleted(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)

	time.Sleep(30 * time.Millisecond)
	os.Remove(filepath.Join(dir, "main.py"))

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return (e.Kind == watcher.EventDeleted ||
			e.Kind == watcher.EventRenamed) &&
			filepath.Base(e.FilePath) == "main.py"
	})

	if !found {
		t.Fatal("Expected DELETED event for main.py")
	}
	t.Logf("✅ Deleted event: %s kind=%s",
		filepath.Base(event.FilePath), event.Kind)
}

// ── FILTERING ─────────────────────────────────────────

func TestFilter_IgnoresUnsupportedFiles(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	w.Watch(ctx, dir)

	time.Sleep(30 * time.Millisecond)

	// Write unsupported files — should not trigger events
	writeFile(t, filepath.Join(dir, "README.md"), `# README`)
	writeFile(t, filepath.Join(dir, "config.json"), `{}`)
	writeFile(t, filepath.Join(dir, "style.css"), `.body {}`)

	// Write one supported file after
	time.Sleep(50 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "main.py"), `def x(): pass`)

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return true // Any event
	})

	if !found {
		t.Fatal("Expected at least one event")
	}

	// The only event should be for main.py
	if filepath.Base(event.FilePath) == "README.md" ||
		filepath.Base(event.FilePath) == "config.json" ||
		filepath.Base(event.FilePath) == "style.css" {
		t.Errorf("Should not have received event for: %s", event.FilePath)
	}
	t.Logf("✅ Unsupported files filtered: first event is %s",
		filepath.Base(event.FilePath))
}

func TestFilter_IgnoresNodeModules(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Create node_modules directory
	nmDir := filepath.Join(dir, "node_modules", "some-package")
	os.MkdirAll(nmDir, 0755)

	w.Watch(ctx, dir)
	time.Sleep(30 * time.Millisecond)

	// Write to node_modules — should be ignored
	writeFile(t, filepath.Join(nmDir, "index.ts"), `export {}`)

	// Write real file
	time.Sleep(50 * time.Millisecond)
	writeFile(t, filepath.Join(dir, "main.py"), `def x(): pass`)

	event, found := waitForEvent(t, w, time.Second, func(e watcher.FileEvent) bool {
		return true
	})

	if found && strings.Contains(event.FilePath, "node_modules") {
		t.Error("Should not receive events from node_modules")
	}
	t.Logf("✅ node_modules ignored")
}

// ── DEBOUNCE ──────────────────────────────────────────

func TestDebounce_RapidChanges_EmitsSingleEvent(t *testing.T) {
	dir := makeTempProject(t)

	config := watcher.DefaultConfig()
	config.DebounceDuration = 100 * time.Millisecond

	w, _ := watcher.New(config)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	w.Watch(ctx, dir)
	time.Sleep(50 * time.Millisecond)

	// Write same file 5 times rapidly
	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(dir, "main.py"),
			`def main(): return `+string(rune('0'+i)))
		time.Sleep(10 * time.Millisecond)
	}

	// Collect events for 500ms
	eventCount := 0
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case e := <-w.Events():
			if filepath.Base(e.FilePath) == "main.py" {
				eventCount++
			}
		case <-timeout:
			goto done
		}
	}
done:

	// Should be 1-2 events (debounced), not 5
	if eventCount > 3 {
		t.Errorf("Debounce failed: got %d events for 5 rapid writes", eventCount)
	}
	t.Logf("✅ Debounce: %d events for 5 rapid writes", eventCount)
}

// ── STOP ──────────────────────────────────────────────

func TestWatcher_Stop(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)
	time.Sleep(30 * time.Millisecond)

	err := w.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}

	// Stop again should not panic
	err = w.Stop()
	if err != nil {
		t.Errorf("Second Stop returned error: %v", err)
	}

	t.Logf("✅ Stop: watcher stopped cleanly")
}

// ── WATCHED PATHS ─────────────────────────────────────

func TestWatcher_WatchedPaths(t *testing.T) {
	dir := makeTempProject(t)
	w := newTestWatcher(t)
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w.Watch(ctx, dir)
	time.Sleep(30 * time.Millisecond)

	paths := w.WatchedPaths()
	if len(paths) == 0 {
		t.Error("Expected at least one watched path")
	}
	t.Logf("✅ WatchedPaths: %d paths watched", len(paths))
}
