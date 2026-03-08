package pipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/pipeline"
)

func newTestPipeline(t *testing.T) (*pipeline.Pipeline, store.GraphStore, string) {
	t.Helper()

	dir, err := os.MkdirTemp("", "nexus-pipeline-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := store.NewSQLiteStore(filepath.Join(dir, "nexus.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	s.CreateProject(context.Background(), store.Project{
		ID: "proj-001", Name: "test-app",
		RootPath: dir, Platform: "web",
		PrimaryLanguage: "python",
	})

	p := pipeline.New("proj-001", s)
	return p, s, dir
}

func writePythonFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

const simplePython = `
def greet(name: str) -> str:
    return "Hello " + name

def farewell(name: str) -> str:
    return "Goodbye " + name
`

const complexPython = `
class UserService:
    def get_user(self, user_id: int):
        return user_id

    def create_user(self, name: str, email: str):
        return {"name": name, "email": email}
`

// ── TESTS ─────────────────────────────────────────────

func TestProcessFile_Python_IndexesModule(t *testing.T) {
	p, s, dir := newTestPipeline(t)
	ctx := context.Background()

	path := writePythonFile(t, dir, "service.py", simplePython)

	result, err := p.ProcessFile(ctx, path)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}
	if result.Skipped {
		t.Error("Expected result not skipped")
	}

	modules, err := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryModules: %v", err)
	}
	if len(modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(modules))
	}

	t.Logf("✅ Module indexed: %s", modules[0].FilePath)
}

func TestProcessFile_Python_IndexesFunctions(t *testing.T) {
	p, s, dir := newTestPipeline(t)
	ctx := context.Background()

	path := writePythonFile(t, dir, "service.py", simplePython)
	p.ProcessFile(ctx, path)

	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryFunctions: %v", err)
	}
	if len(fns) < 2 {
		t.Errorf("Expected >= 2 functions, got %d", len(fns))
	}

	t.Logf("✅ Functions indexed: %d", len(fns))
}

func TestProcessFile_Python_IndexesClass(t *testing.T) {
	p, s, dir := newTestPipeline(t)
	ctx := context.Background()

	path := writePythonFile(t, dir, "service.py", complexPython)
	p.ProcessFile(ctx, path)

	classes, err := s.QueryClasses(ctx, store.ClassFilter{ProjectID: "proj-001"})
	if err != nil {
		t.Fatalf("QueryClasses: %v", err)
	}
	if len(classes) != 1 {
		t.Errorf("Expected 1 class, got %d", len(classes))
	}
	if classes[0].Name != "UserService" {
		t.Errorf("Expected UserService, got %s", classes[0].Name)
	}

	t.Logf("✅ Class indexed: %s", classes[0].Name)
}

func TestProcessFile_Idempotent(t *testing.T) {
	p, s, dir := newTestPipeline(t)
	ctx := context.Background()

	path := writePythonFile(t, dir, "service.py", simplePython)

	p.ProcessFile(ctx, path)
	p.ProcessFile(ctx, path)
	p.ProcessFile(ctx, path)

	modules, _ := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if len(modules) != 1 {
		t.Errorf("Expected 1 module after 3 runs, got %d", len(modules))
	}

	t.Logf("✅ Idempotent: %d modules after 3 runs", len(modules))
}

func TestProcessFile_Stats_Tracked(t *testing.T) {
	p, _, dir := newTestPipeline(t)
	ctx := context.Background()

	path := writePythonFile(t, dir, "service.py", simplePython)
	p.ProcessFile(ctx, path)

	stats := p.Stats()
	if stats.FilesProcessed != 1 {
		t.Errorf("Expected 1 file processed, got %d", stats.FilesProcessed)
	}
	if stats.NodesAdded == 0 {
		t.Error("Expected nodes to be added")
	}

	t.Logf("✅ Stats: files=%d nodes=%d", stats.FilesProcessed, stats.NodesAdded)
}

func TestIndexDirectory_MultipleFiles(t *testing.T) {
	p, s, dir := newTestPipeline(t)
	ctx := context.Background()

	writePythonFile(t, dir, "a.py", simplePython)
	writePythonFile(t, dir, "b.py", complexPython)

	if err := p.IndexDirectory(ctx, dir); err != nil {
		t.Fatalf("IndexDirectory: %v", err)
	}

	modules, _ := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if len(modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(modules))
	}

	t.Logf("✅ IndexDirectory: %d modules indexed", len(modules))
}

func TestIndexDirectory_IgnoresNonPython(t *testing.T) {
	p, s, dir := newTestPipeline(t)
	ctx := context.Background()

	writePythonFile(t, dir, "service.py", simplePython)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0644)

	p.IndexDirectory(ctx, dir)

	modules, _ := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if len(modules) != 1 {
		t.Errorf("Expected 1 module (only .py), got %d", len(modules))
	}

	t.Logf("✅ IgnoresNonPython: %d modules", len(modules))
}

func TestRun_WatchesAndProcessesChanges(t *testing.T) {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("SKIP_WATCHER_TEST") != "" {
		t.Skip("Skipping watcher test on WSL2 — inotify unreliable")
	}

	p, s, dir := newTestPipeline(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- p.Run(ctx, dir)
	}()

	time.Sleep(500 * time.Millisecond)
	writePythonFile(t, dir, "service.py", simplePython)
	time.Sleep(1500 * time.Millisecond)

	cancel()
	<-done

	modules, _ := s.QueryModules(ctx, store.ModuleFilter{ProjectID: "proj-001"})
	if len(modules) == 0 {
		t.Error("Expected at least 1 module to be indexed by watcher")
	}

	t.Logf("✅ Watcher triggered: %d modules indexed", len(modules))
}
