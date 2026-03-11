package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hackersfun369/nexus/internal/api"
	"github.com/hackersfun369/nexus/internal/chat"
	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/plugin"
	"github.com/hackersfun369/nexus/internal/version"
	"github.com/hackersfun369/nexus/internal/rules"
	nexusweb "github.com/hackersfun369/nexus/internal/web"
	_ "github.com/mattn/go-sqlite3"
)

var (
	commit  = "none"
)

func main() {
	ctx := context.Background()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Println(version.String())
			return
		case "help", "--help", "-h":
			printUsage()
			return
		case "serve":
			addr := ":8080"
			if len(os.Args) > 2 {
				addr = os.Args[2]
			}
			runServe(ctx, addr)
			return
		case "plugin":
			runPlugin(ctx, os.Args[2:])
			return
		}
	}

	// Default: terminal REPL
	dbPath, err := initDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	cfg := rules.DefaultConfig()
	if loaded, err := rules.LoadConfig(filepath.Join(nexusDir(), "config.json")); err == nil {
		cfg = loaded
	}

	sess := chat.NewSession(s, cfg)
	repl := chat.NewREPL(sess)
	repl.Run(ctx)
}

func runPlugin(ctx context.Context, args []string) {
	db, err := openRawDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	pluginsDir := filepath.Join(nexusDir(), "plugins")
	ps := plugin.NewStore(db)
	reg := plugin.NewRegistry(ps, pluginsDir)
	cli := plugin.NewCLI(reg)
	cli.Run(ctx, args)
}

func runServe(ctx context.Context, addr string) {
	dbPath, err := initDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	cfg := rules.DefaultConfig()
	if loaded, err := rules.LoadConfig(filepath.Join(nexusDir(), "config.json")); err == nil {
		cfg = loaded
	}

	apiSrv := api.NewServer(s, cfg, addr)
	r := chi.NewRouter()
	webHandler := nexusweb.Handler()
	// Route: API requests go to apiSrv, everything else to webHandler
	apiHandler := apiSrv.Router()
	r.HandleFunc("/health", apiHandler.ServeHTTP)
	r.HandleFunc("/ready", apiHandler.ServeHTTP)
	r.HandleFunc("/api/v1/*", apiHandler.ServeHTTP)
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		webHandler.ServeHTTP(w, req)
	})

	host := addr
	if h, p, err := net.SplitHostPort(addr); err == nil {
		if h == "" {
			host = "localhost:" + p
		}
	}

	fmt.Printf("\n  NEXUS Web UI\n")
	fmt.Printf("  ─────────────────────────────\n")
	fmt.Printf("  Local:   http://%s\n", host)
	fmt.Printf("  API:     http://%s/api/v1\n", host)
	fmt.Printf("  ─────────────────────────────\n")
	fmt.Printf("  Press Ctrl+C to stop\n\n")

	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://" + host)
	}()

	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func openRawDB() (*sql.DB, error) {
	dbPath, err := initDB()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	// Run plugin migrations
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS plugin (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			kind TEXT NOT NULL,
			platform TEXT,
			language TEXT,
			description TEXT,
			author TEXT,
			registry_url TEXT,
			download_url TEXT,
			sha256 TEXT,
			install_path TEXT,
			status TEXT NOT NULL DEFAULT 'available',
			manifest TEXT,
			installed_at DATETIME,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS plugin_dependency (
			plugin_id TEXT NOT NULL,
			depends_on TEXT NOT NULL,
			min_version TEXT,
			PRIMARY KEY (plugin_id, depends_on)
		);
		CREATE TABLE IF NOT EXISTS plugin_capability (
			plugin_id TEXT NOT NULL,
			capability TEXT NOT NULL,
			PRIMARY KEY (plugin_id, capability)
		);
	`)
	return db, err
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}

func initDB() (string, error) {
	dir := nexusDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("MkdirAll: %w", err)
	}
	return filepath.Join(dir, "nexus.db"), nil
}

func nexusDir() string {
	if d := os.Getenv("NEXUS_HOME"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus")
}

func shortCommit() string {
	if len(commit) >= 7 {
		return commit[:7]
	}
	return commit
}

func printUsage() {
	fmt.Println("nexus — intelligent development system")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nexus                    Start terminal chat")
	fmt.Println("  nexus serve [:port]      Start web UI")
	fmt.Println("  nexus plugin <command>   Manage plugins")
	fmt.Println("  nexus version            Show version")
	fmt.Println("  nexus help               Show this message")
	fmt.Println()
	fmt.Println("Plugin commands:")
	fmt.Println("  nexus plugin list        List all plugins")
	fmt.Println("  nexus plugin sync        Sync from registry")
	fmt.Println("  nexus plugin install <id> Install a plugin")
	fmt.Println("  nexus plugin remove <id>  Remove a plugin")
	fmt.Println("  nexus plugin search <term> Search plugins")
	fmt.Println("  nexus plugin info <id>    Show plugin info")
}
