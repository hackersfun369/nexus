package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/hackersfun369/nexus/main/plugins/registry.json"
	httpTimeout        = 30 * time.Second
)

// Registry fetches and caches the plugin registry manifest
type Registry struct {
	url        string
	httpClient *http.Client
	store      *Store
	pluginsDir string
}

// NewRegistry creates a registry client
func NewRegistry(store *Store, pluginsDir string) *Registry {
	return &Registry{
		url:        DefaultRegistryURL,
		httpClient: &http.Client{Timeout: httpTimeout},
		store:      store,
		pluginsDir: pluginsDir,
	}
}

// NewRegistryWithURL creates a registry client with a custom URL
func NewRegistryWithURL(store *Store, pluginsDir, url string) *Registry {
	r := NewRegistry(store, pluginsDir)
	r.url = url
	return r
}

// Sync fetches the registry manifest and updates the local store
func (r *Registry) Sync(ctx context.Context) (*RegistryManifest, error) {
	manifest, err := r.fetchManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("Sync: %w", err)
	}
	for _, m := range manifest.Plugins {
		p := ManifestToPlugin(m, r.url)
		// Keep installed status if already installed
		existing, err := r.store.Get(ctx, m.ID)
		if err == nil && existing.Status == StatusInstalled {
			p.Status = StatusInstalled
			p.InstallPath = existing.InstallPath
			p.InstalledAt = existing.InstalledAt
		}
		if err := r.store.Save(ctx, p); err != nil {
			return nil, fmt.Errorf("Sync save %s: %w", m.ID, err)
		}
		if err := r.store.SaveCapabilities(ctx, m.ID, m.Capabilities); err != nil {
			return nil, fmt.Errorf("Sync capabilities %s: %w", m.ID, err)
		}
	}
	return manifest, nil
}

// Install downloads and installs a plugin
func (r *Registry) Install(ctx context.Context, id string) (*Plugin, error) {
	p, err := r.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("plugin not found: %s — run 'nexus plugin sync' first", id)
	}
	if p.Status == StatusInstalled {
		return &p, nil
	}
	if p.DownloadURL == "" {
		return nil, fmt.Errorf("plugin %s has no download URL", id)
	}

	// Download
	data, err := r.download(ctx, p.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("Install download %s: %w", id, err)
	}

	// Verify checksum
	if p.SHA256 != "" {
		if err := verifyChecksum(data, p.SHA256); err != nil {
			return nil, fmt.Errorf("Install checksum %s: %w", id, err)
		}
	}

	// Write to disk
	installDir := filepath.Join(r.pluginsDir, id+"@"+p.Version)
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("Install mkdir: %w", err)
	}

	pluginFile := filepath.Join(installDir, "plugin.json")
	if err := os.WriteFile(pluginFile, data, 0644); err != nil {
		return nil, fmt.Errorf("Install write: %w", err)
	}

	// Update store
	now := time.Now()
	p.Status = StatusInstalled
	p.InstallPath = installDir
	p.InstalledAt = &now
	p.UpdatedAt = now

	if err := r.store.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("Install store: %w", err)
	}

	return &p, nil
}

// Remove uninstalls a plugin
func (r *Registry) Remove(ctx context.Context, id string) error {
	p, err := r.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("plugin not found: %s", id)
	}
	if p.Status != StatusInstalled {
		return fmt.Errorf("plugin %s is not installed", id)
	}

	// Remove from disk
	if p.InstallPath != "" {
		if err := os.RemoveAll(p.InstallPath); err != nil {
			return fmt.Errorf("Remove disk: %w", err)
		}
	}

	// Update store
	return r.store.SetStatus(ctx, id, StatusAvailable)
}

// Resolve returns the best plugin for a given requirement
func (r *Registry) Resolve(ctx context.Context, platform, language string) ([]Plugin, error) {
	f := Filter{
		Kind:     KindLanguage,
		Status:   StatusInstalled,
	}
	if platform != "" {
		f.Platform = platform
	}
	if language != "" {
		f.Language = language
	}
	return r.store.List(ctx, f)
}

// fetchManifest fetches the registry JSON
func (r *Registry) fetchManifest(ctx context.Context) (*RegistryManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest: HTTP %d", resp.StatusCode)
	}

	var manifest RegistryManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return &manifest, nil
}

// download fetches raw bytes from a URL
func (r *Registry) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum checks sha256 of data against expected hex string
func verifyChecksum(data []byte, expected string) error {
	h := sha256.Sum256(data)
	actual := hex.EncodeToString(h[:])
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s got %s", expected, actual)
	}
	return nil
}

// SyncFromFile loads a registry manifest from a local JSON file
func (r *Registry) SyncFromFile(ctx context.Context, path string) (*RegistryManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("SyncFromFile: %w", err)
	}
	var manifest RegistryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("SyncFromFile parse: %w", err)
	}
	for _, m := range manifest.Plugins {
		p := ManifestToPlugin(m, path)
		existing, err := r.store.Get(ctx, m.ID)
		if err == nil && existing.Status == StatusInstalled {
			p.Status = StatusInstalled
			p.InstallPath = existing.InstallPath
			p.InstalledAt = existing.InstalledAt
		}
		if err := r.store.Save(ctx, p); err != nil {
			return nil, fmt.Errorf("SyncFromFile save %s: %w", m.ID, err)
		}
		if err := r.store.SaveCapabilities(ctx, m.ID, m.Capabilities); err != nil {
			return nil, fmt.Errorf("SyncFromFile capabilities %s: %w", m.ID, err)
		}
	}
	return &manifest, nil
}
