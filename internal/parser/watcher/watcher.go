package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventKind represents the type of file system event
type EventKind string

const (
	EventCreated  EventKind = "CREATED"
	EventModified EventKind = "MODIFIED"
	EventDeleted  EventKind = "DELETED"
	EventRenamed  EventKind = "RENAMED"
)

// Language represents a supported language
type Language string

const (
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
	LangJava       Language = "java"
	LangUnknown    Language = "unknown"
)

// FileEvent represents a single file change event
type FileEvent struct {
	FilePath  string
	Kind      EventKind
	Language  Language
	Timestamp time.Time
}

// Config holds watcher configuration
type Config struct {
	// Debounce duration — coalesces rapid changes
	// e.g. saving a file triggers multiple events
	DebounceDuration time.Duration

	// File extensions to watch
	// defaults to .py .ts .tsx .js .jsx .java
	Extensions []string

	// Directories to ignore
	IgnoreDirs []string
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		DebounceDuration: 100 * time.Millisecond,
		Extensions: []string{
			".py",
			".ts", ".tsx",
			".js", ".jsx",
			".java",
		},
		IgnoreDirs: []string{
			"node_modules",
			".git",
			"__pycache__",
			".pytest_cache",
			"build",
			"dist",
			"out",
			".gradle",
			".idea",
			".vscode",
		},
	}
}

// Watcher monitors a project directory for file changes
type Watcher struct {
	config  Config
	fsw     *fsnotify.Watcher
	events  chan FileEvent
	errors  chan error
	mu      sync.Mutex
	pending map[string]*pendingEvent // debounce buffer
	stopCh  chan struct{}
	stopped bool
}

// pendingEvent holds a debounced event
type pendingEvent struct {
	event FileEvent
	timer *time.Timer
}

// New creates a new Watcher
func New(config Config) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if len(config.Extensions) == 0 {
		config = DefaultConfig()
	}

	return &Watcher{
		config:  config,
		fsw:     fsw,
		events:  make(chan FileEvent, 100),
		errors:  make(chan error, 10),
		pending: make(map[string]*pendingEvent),
		stopCh:  make(chan struct{}),
	}, nil
}

// Watch starts watching a directory recursively
func (w *Watcher) Watch(ctx context.Context, rootPath string) error {
	// Walk directory and add all subdirectories
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if info.IsDir() {
			// Skip ignored directories
			if w.shouldIgnoreDir(info.Name()) {
				return filepath.SkipDir
			}
			return w.fsw.Add(path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Start processing events
	go w.processEvents(ctx)

	return nil
}

// Events returns the channel of file change events
func (w *Watcher) Events() <-chan FileEvent {
	return w.events
}

// Errors returns the channel of watcher errors
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// Stop shuts down the watcher
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	w.stopped = true
	close(w.stopCh)
	return w.fsw.Close()
}

// processEvents reads raw fsnotify events and emits debounced FileEvents
func (w *Watcher) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return

		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}

			// Filter to supported languages only
			if !w.isSupportedFile(event.Name) {
				continue
			}

			// Ignore ignored directories
			if w.isInIgnoredDir(event.Name) {
				continue
			}

			kind := w.mapEventKind(event.Op)
			if kind == "" {
				continue
			}

			fe := FileEvent{
				FilePath:  event.Name,
				Kind:      kind,
				Language:  detectLanguage(event.Name),
				Timestamp: time.Now(),
			}

			w.debounce(fe)

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			if err != nil {
				select {
				case w.errors <- err:
				default:
					log.Printf("watcher error dropped: %v", err)
				}
			}
		}
	}
}

// debounce coalesces rapid events for the same file
func (w *Watcher) debounce(fe FileEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if pending, exists := w.pending[fe.FilePath]; exists {
		// Reset timer — file still changing
		pending.timer.Reset(w.config.DebounceDuration)
		// Keep latest event kind
		pending.event = fe
		return
	}

	// New pending event
	pe := &pendingEvent{event: fe}
	pe.timer = time.AfterFunc(w.config.DebounceDuration, func() {
		w.mu.Lock()
		delete(w.pending, fe.FilePath)
		w.mu.Unlock()

		select {
		case w.events <- pe.event:
		case <-w.stopCh:
		}
	})

	w.pending[fe.FilePath] = pe
}

// mapEventKind converts fsnotify Op to EventKind
func (w *Watcher) mapEventKind(op fsnotify.Op) EventKind {
	switch {
	case op.Has(fsnotify.Create):
		return EventCreated
	case op.Has(fsnotify.Write):
		return EventModified
	case op.Has(fsnotify.Remove):
		return EventDeleted
	case op.Has(fsnotify.Rename):
		return EventRenamed
	default:
		return ""
	}
}

// isSupportedFile checks if a file has a supported extension
func (w *Watcher) isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range w.config.Extensions {
		if ext == supported {
			return true
		}
	}
	return false
}

// shouldIgnoreDir checks if a directory should be skipped
func (w *Watcher) shouldIgnoreDir(name string) bool {
	for _, ignored := range w.config.IgnoreDirs {
		if name == ignored {
			return true
		}
	}
	return false
}

// isInIgnoredDir checks if a file path contains an ignored directory
func (w *Watcher) isInIgnoredDir(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if w.shouldIgnoreDir(part) {
			return true
		}
	}
	return false
}

// detectLanguage detects language from file extension
func detectLanguage(path string) Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".py":
		return LangPython
	case ".ts", ".tsx":
		return LangTypeScript
	case ".js", ".jsx":
		return LangTypeScript // Treat JS as TS
	case ".java":
		return LangJava
	default:
		return LangUnknown
	}
}

// WatchedPaths returns all currently watched paths
func (w *Watcher) WatchedPaths() []string {
	return w.fsw.WatchList()
}
