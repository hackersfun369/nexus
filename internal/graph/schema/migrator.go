package schema

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrator runs SQL migrations against a SQLite database
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a new Migrator
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// Migrate runs all pending migrations in order
func (m *Migrator) Migrate() error {
	// Ensure schema_versions table exists first
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_versions (
			version     TEXT     NOT NULL PRIMARY KEY,
			applied_at  DATETIME NOT NULL DEFAULT (datetime('now')),
			description TEXT     NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_versions: %w", err)
	}

	// Get already applied migrations
	applied, err := m.appliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Read migration files
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Sort by filename to ensure order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Run pending migrations
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := strings.TrimSuffix(entry.Name(), ".sql")

		if applied[version] {
			continue // Already applied
		}

		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if err := m.runMigration(version, string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", version, err)
		}
	}

	return nil
}

// runMigration executes a single migration
func (m *Migrator) runMigration(version, sql string) error {
	_, err := m.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", version, err)
	}
	return nil
}

// appliedMigrations returns a set of already-applied migration versions
func (m *Migrator) appliedMigrations() (map[string]bool, error) {
	applied := map[string]bool{}

	rows, err := m.db.Query(`SELECT version FROM schema_versions`)
	if err != nil {
		// Table may not exist yet — that's OK
		return applied, nil
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// Version returns the current schema version
func (m *Migrator) Version() (string, error) {
	var version string
	err := m.db.QueryRow(`
		SELECT version FROM schema_versions
		ORDER BY applied_at DESC
		LIMIT 1
	`).Scan(&version)

	if err == sql.ErrNoRows {
		return "none", nil
	}
	return version, err
}

// OpenDB opens or creates a SQLite database at the given path
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(1) // SQLite supports one writer
	db.SetMaxIdleConns(1)
	return db, nil
}
