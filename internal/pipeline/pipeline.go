package pipeline

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/applier"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/parser/extractor"
	"github.com/hackersfun369/nexus/internal/parser/normalizer"
	"github.com/hackersfun369/nexus/internal/parser/symbols"
	"github.com/hackersfun369/nexus/internal/parser/treesitter"
	"github.com/hackersfun369/nexus/internal/parser/watcher"
)

// Stats tracks pipeline activity
type Stats struct {
	FilesProcessed int
	FilesErrored   int
	NodesAdded     int
	NodesModified  int
	NodesDeleted   int
	EdgesAdded     int
	LastRunAt      time.Time
	mu             sync.Mutex
}

func (s *Stats) record(r applier.Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesProcessed++
	s.NodesAdded += r.NodesAdded
	s.NodesModified += r.NodesModified
	s.NodesDeleted += r.NodesDeleted
	s.EdgesAdded += r.EdgesAdded
	s.LastRunAt = time.Now()
}

func (s *Stats) recordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesErrored++
}

func (s *Stats) Snapshot() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{
		FilesProcessed: s.FilesProcessed,
		FilesErrored:   s.FilesErrored,
		NodesAdded:     s.NodesAdded,
		NodesModified:  s.NodesModified,
		NodesDeleted:   s.NodesDeleted,
		EdgesAdded:     s.EdgesAdded,
		LastRunAt:      s.LastRunAt,
	}
}

// Pipeline connects the file watcher to the parser and graph store
type Pipeline struct {
	projectID string
	store     store.GraphStore
	applier   *applier.Applier
	symbols   *symbols.SymbolTable
	stats     Stats
}

// New creates a new Pipeline
func New(projectID string, s store.GraphStore) *Pipeline {
	return &Pipeline{
		projectID: projectID,
		store:     s,
		applier:   applier.New(s),
		symbols:   symbols.New(),
	}
}

// Run starts the pipeline — watches rootPath and processes file events
func (p *Pipeline) Run(ctx context.Context, rootPath string) error {
	w, _ := watcher.New(watcher.DefaultConfig())

	if err := w.Watch(ctx, rootPath); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer w.Stop()

	log.Printf("[pipeline] watching %s for project %s", rootPath, p.projectID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[pipeline] stopped: %v", ctx.Err())
			return nil

		case err := <-w.Errors():
			if err != nil {
				log.Printf("[pipeline] watcher error: %v", err)
			}

		case event := <-w.Events():
			if event.Kind == watcher.EventDeleted {
				p.handleDelete(ctx, event)
			} else {
				p.handleChange(ctx, event)
			}
		}
	}
}

// ProcessFile parses and applies a single file synchronously.
// Used for initial indexing and direct calls.
func (p *Pipeline) ProcessFile(ctx context.Context, filePath string) (applier.Result, error) {
	// Read file
	source, err := os.ReadFile(filePath)
	if err != nil {
		p.stats.recordError()
		return applier.Result{}, fmt.Errorf("read %s: %w", filePath, err)
	}

	// Detect language
	lang := normalizer.DetectLanguage(filePath)
	if lang == "" {
		return applier.Result{Skipped: true}, nil
	}

	// Parse
	parser := treesitter.New()
	defer parser.Close()

	tsLang := treesitter.Language(lang)
	result, err := parser.Parse(ctx, source, tsLang)
	if err != nil {
		p.stats.recordError()
		return applier.Result{}, fmt.Errorf("parse %s: %w", filePath, err)
	}

	// Normalize
	var astNode normalizer.ASTNode
	switch lang {
	case normalizer.LangPython:
		astNode = normalizer.NewPythonNormalizer(source, filePath).Normalize(result.Tree.RootNode())
	case normalizer.LangTypeScript:
		astNode = normalizer.NewTypeScriptNormalizer(source, filePath).Normalize(result.Tree.RootNode())
	case normalizer.LangJava:
		astNode = normalizer.NewJavaNormalizer(source, filePath).Normalize(result.Tree.RootNode())
	default:
		return applier.Result{Skipped: true}, nil
	}

	// Build delta
	existing := p.existingChecksums(ctx, filePath)
	delta := extractor.NewBuilder(p.projectID).
		WithExistingChecksums(existing).
		Build(astNode)

	// Apply delta
	applied, err := p.applier.Apply(ctx, delta)
	if err != nil {
		p.stats.recordError()
		return applier.Result{}, fmt.Errorf("apply %s: %w", filePath, err)
	}

	p.stats.record(applied)
	return applied, nil
}

// IndexDirectory processes all supported files in a directory
func (p *Pipeline) IndexDirectory(ctx context.Context, rootPath string) error {
	w, _ := watcher.New(watcher.DefaultConfig())
	paths, err := w.SupportedFiles(rootPath)
	if err != nil {
		return fmt.Errorf("scan %s: %w", rootPath, err)
	}

	log.Printf("[pipeline] indexing %d files in %s", len(paths), rootPath)

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if _, err := p.ProcessFile(ctx, path); err != nil {
				log.Printf("[pipeline] error processing %s: %v", path, err)
			}
		}
	}

	log.Printf("[pipeline] indexed %d files", p.stats.FilesProcessed)
	return nil
}

// Stats returns a snapshot of pipeline activity
func (p *Pipeline) Stats() Stats {
	return p.stats.Snapshot()
}

// ── INTERNAL ──────────────────────────────────────────

func (p *Pipeline) handleChange(ctx context.Context, event watcher.FileEvent) {
	if _, err := p.ProcessFile(ctx, event.FilePath); err != nil {
		log.Printf("[pipeline] error processing %s: %v", event.FilePath, err)
	}
}

func (p *Pipeline) handleDelete(ctx context.Context, event watcher.FileEvent) {
	mod, err := p.store.GetModuleByPath(ctx, p.projectID, event.FilePath)
	if err != nil {
		return // not in store — nothing to do
	}
	if err := p.store.DeleteModule(ctx, mod.ID); err != nil {
		log.Printf("[pipeline] error deleting module %s: %v", event.FilePath, err)
	}
}

// existingChecksums fetches current node checksums for a file
// so the delta builder can detect what actually changed
func (p *Pipeline) existingChecksums(ctx context.Context, filePath string) map[string]string {
	checksums := map[string]string{}

	mod, err := p.store.GetModuleByPath(ctx, p.projectID, filePath)
	if err != nil {
		return checksums // file not indexed yet
	}
	checksums[mod.ID] = mod.Checksum

	fns, err := p.store.QueryFunctions(ctx, store.FunctionFilter{
		ProjectID: p.projectID,
		ModuleID:  mod.ID,
	})
	if err == nil {
		for _, fn := range fns {
			checksums[fn.ID] = fn.Checksum
		}
	}

	classes, err := p.store.QueryClasses(ctx, store.ClassFilter{
		ProjectID: p.projectID,
		ModuleID:  mod.ID,
	})
	if err == nil {
		for _, c := range classes {
			checksums[c.ID] = c.Checksum
		}
	}

	return checksums
}
