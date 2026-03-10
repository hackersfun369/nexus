package plugin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store manages plugin persistence
type Store struct {
	db *sql.DB
}

// NewStore opens the plugin store using an existing DB connection
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Save inserts or updates a plugin record
func (s *Store) Save(ctx context.Context, p Plugin) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO plugin (
			id, name, version, kind, platform, language,
			description, author, registry_url, download_url,
			sha256, install_path, status, manifest,
			installed_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			version=excluded.version,
			kind=excluded.kind,
			platform=excluded.platform,
			language=excluded.language,
			description=excluded.description,
			author=excluded.author,
			registry_url=excluded.registry_url,
			download_url=excluded.download_url,
			sha256=excluded.sha256,
			install_path=excluded.install_path,
			status=excluded.status,
			manifest=excluded.manifest,
			installed_at=excluded.installed_at,
			updated_at=excluded.updated_at`,
		p.ID, p.Name, p.Version, string(p.Kind), p.Platform, p.Language,
		p.Description, p.Author, p.RegistryURL, p.DownloadURL,
		p.SHA256, p.InstallPath, string(p.Status), p.Manifest,
		p.InstalledAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("Save plugin: %w", err)
	}
	return nil
}

// Get retrieves a plugin by ID
func (s *Store) Get(ctx context.Context, id string) (Plugin, error) {
	var p Plugin
	var kind, status string
	var installedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, version, kind, platform, language,
		       description, author, registry_url, download_url,
		       sha256, install_path, status, manifest,
		       installed_at, updated_at
		FROM plugin WHERE id = ?`, id,
	).Scan(
		&p.ID, &p.Name, &p.Version, &kind, &p.Platform, &p.Language,
		&p.Description, &p.Author, &p.RegistryURL, &p.DownloadURL,
		&p.SHA256, &p.InstallPath, &status, &p.Manifest,
		&installedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Plugin{}, fmt.Errorf("plugin not found: %s", id)
	}
	if err != nil {
		return Plugin{}, fmt.Errorf("Get plugin: %w", err)
	}
	p.Kind = Kind(kind)
	p.Status = Status(status)
	if installedAt.Valid {
		p.InstalledAt = &installedAt.Time
	}
	return p, nil
}

// List returns plugins matching the filter
func (s *Store) List(ctx context.Context, f Filter) ([]Plugin, error) {
	query := `
		SELECT id, name, version, kind, platform, language,
		       description, author, registry_url, download_url,
		       sha256, install_path, status, manifest,
		       installed_at, updated_at
		FROM plugin WHERE 1=1`
	args := []interface{}{}

	if f.Kind != "" {
		query += " AND kind = ?"
		args = append(args, string(f.Kind))
	}
	if f.Platform != "" {
		query += " AND (platform = ? OR platform = 'all')"
		args = append(args, f.Platform)
	}
	if f.Language != "" {
		query += " AND language = ?"
		args = append(args, f.Language)
	}
	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, string(f.Status))
	}
	query += " ORDER BY name ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("List plugins: %w", err)
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		var p Plugin
		var kind, status string
		var installedAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Version, &kind, &p.Platform, &p.Language,
			&p.Description, &p.Author, &p.RegistryURL, &p.DownloadURL,
			&p.SHA256, &p.InstallPath, &status, &p.Manifest,
			&installedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("List scan: %w", err)
		}
		p.Kind = Kind(kind)
		p.Status = Status(status)
		if installedAt.Valid {
			p.InstalledAt = &installedAt.Time
		}
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

// SetStatus updates a plugin's status
func (s *Store) SetStatus(ctx context.Context, id string, status Status) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE plugin SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now(), id,
	)
	return err
}

// Delete removes a plugin record
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM plugin WHERE id=?`, id)
	return err
}

// SaveCapabilities stores plugin capabilities
func (s *Store) SaveCapabilities(ctx context.Context, pluginID string, caps []string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM plugin_capability WHERE plugin_id=?`, pluginID)
	if err != nil {
		return err
	}
	for _, cap := range caps {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO plugin_capability (plugin_id, capability) VALUES (?,?)`,
			pluginID, cap)
		if err != nil {
			return err
		}
	}
	return nil
}

// FindByCapability returns plugins that provide a capability
func (s *Store) FindByCapability(ctx context.Context, capability string) ([]Plugin, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id FROM plugin p
		JOIN plugin_capability pc ON pc.plugin_id = p.id
		WHERE pc.capability = ? AND p.status = 'installed'`, capability)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		p, err := s.Get(ctx, id)
		if err != nil {
			continue
		}
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

// ManifestToPlugin converts a Manifest to a Plugin record
func ManifestToPlugin(m Manifest, registryURL string) Plugin {
	data, _ := json.Marshal(m)
	return Plugin{
		ID:          m.ID,
		Name:        m.Name,
		Version:     m.Version,
		Kind:        m.Kind,
		Platform:    m.Platform,
		Language:    m.Language,
		Description: m.Description,
		Author:      m.Author,
		RegistryURL: registryURL,
		DownloadURL: m.DownloadURL,
		SHA256:      m.SHA256,
		Status:      StatusAvailable,
		Manifest:    string(data),
		UpdatedAt:   time.Now(),
	}
}
