package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SaveGeneratedProject persists a generated project and its files to SQLite
func (s *Server) saveGeneratedProject(ctx context.Context, prompt string, resp GenerateResponse) (string, error) {
	db, err := s.openProjectsDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	if err := migrateProjectsDB(db); err != nil {
		return "", err
	}

	// Generate stable ID from prompt + timestamp
	id := projectID(prompt)

	featuresJSON, _ := json.Marshal(resp.Features)
	entitiesJSON, _ := json.Marshal(resp.Entities)

	score := 0
	if resp.Quality != nil {
		score = resp.Quality.Score
	}

	_, err = db.ExecContext(ctx, `
		INSERT OR REPLACE INTO generated_project
			(id, name, platform, language, framework, plugin_id, prompt, features, entities, file_count, score, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		id, resp.AppName, resp.Platform, resp.Language, resp.Framework,
		resp.PluginID, prompt, string(featuresJSON), string(entitiesJSON),
		resp.FileCount, score,
	)
	if err != nil {
		return "", err
	}

	// Delete old files for this project
	_, err = db.ExecContext(ctx, `DELETE FROM generated_file WHERE project_id = ?`, id)
	if err != nil {
		return "", err
	}

	// Insert files
	for _, f := range resp.Files {
		fileID := fmt.Sprintf("%s:%s", id, f.Path)
		_, err = db.ExecContext(ctx, `
			INSERT INTO generated_file (id, project_id, path, content, language, size)
			VALUES (?, ?, ?, ?, ?, ?)`,
			fileID, id, f.Path, f.Content, f.Lang, f.Size,
		)
		if err != nil {
			return "", err
		}
	}

	return id, nil
}

// handleListGeneratedProjects returns all saved generated projects
func (s *Server) handleListGeneratedProjects(w http.ResponseWriter, r *http.Request) {
	db, err := s.openProjectsDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	if err := migrateProjectsDB(db); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := db.QueryContext(r.Context(), `
		SELECT id, name, platform, language, framework, plugin_id, prompt,
		       features, entities, file_count, score, created_at
		FROM generated_project
		ORDER BY created_at DESC
		LIMIT 50`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type ProjectRow struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Platform  string   `json:"platform"`
		Language  string   `json:"language"`
		Framework string   `json:"framework"`
		PluginID  string   `json:"plugin_id"`
		Prompt    string   `json:"prompt"`
		Features  []string `json:"features"`
		Entities  []string `json:"entities"`
		FileCount int      `json:"file_count"`
		Score     int      `json:"score"`
		CreatedAt string   `json:"created_at"`
	}

	var projects []ProjectRow
	for rows.Next() {
		var p ProjectRow
		var featuresJSON, entitiesJSON string
		err := rows.Scan(&p.ID, &p.Name, &p.Platform, &p.Language, &p.Framework,
			&p.PluginID, &p.Prompt, &featuresJSON, &entitiesJSON,
			&p.FileCount, &p.Score, &p.CreatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(featuresJSON), &p.Features)
		json.Unmarshal([]byte(entitiesJSON), &p.Entities)
		projects = append(projects, p)
	}

	if projects == nil {
		projects = []ProjectRow{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
	})
}

// handleGetGeneratedProject returns a single project with all its files
func (s *Server) handleGetGeneratedProject(w http.ResponseWriter, r *http.Request) {
	// Extract id from URL: /api/v1/generated/{id}
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]
	if id == "" {
		writeError(w, http.StatusBadRequest, "project id required")
		return
	}

	db, err := s.openProjectsDB()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	// Get files
	rows, err := db.QueryContext(r.Context(),
		`SELECT path, content, language, size FROM generated_file WHERE project_id = ? ORDER BY path`,
		id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var files []FileEntry
	for rows.Next() {
		var f FileEntry
		if err := rows.Scan(&f.Path, &f.Content, &f.Lang, &f.Size); err != nil {
			continue
		}
		files = append(files, f)
	}

	if files == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":    id,
		"files": files,
		"count": len(files),
	})
}

func (s *Server) openProjectsDB() (*sql.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	path := filepath.Join(home, ".nexus", "nexus.db")
	return sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL")
}

func migrateProjectsDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS generated_project (
			id TEXT PRIMARY KEY, name TEXT NOT NULL,
			platform TEXT NOT NULL DEFAULT '', language TEXT NOT NULL DEFAULT '',
			framework TEXT NOT NULL DEFAULT '', plugin_id TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '', features TEXT NOT NULL DEFAULT '[]',
			entities TEXT NOT NULL DEFAULT '[]', file_count INTEGER NOT NULL DEFAULT 0,
			score INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS generated_file (
			id TEXT PRIMARY KEY, project_id TEXT NOT NULL,
			path TEXT NOT NULL, content TEXT NOT NULL,
			language TEXT NOT NULL DEFAULT '', size INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (project_id) REFERENCES generated_project(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_gf_project ON generated_file(project_id);
		CREATE INDEX IF NOT EXISTS idx_gp_created ON generated_project(created_at DESC);
	`)
	return err
}

func projectID(prompt string) string {
	h := sha256.Sum256([]byte(prompt + time.Now().Format("2006-01-02T15:04")))
	return fmt.Sprintf("gp-%x", h[:8])
}
