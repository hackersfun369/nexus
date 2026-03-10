-- Plugin registry table
CREATE TABLE IF NOT EXISTS plugin (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    version         TEXT NOT NULL,
    kind            TEXT NOT NULL,  -- 'language', 'framework', 'platform', 'rule'
    platform        TEXT,           -- 'android', 'web', 'windows', 'mac', 'cli', 'backend', 'all'
    language        TEXT,           -- 'python', 'kotlin', 'typescript', etc
    description     TEXT,
    author          TEXT,
    registry_url    TEXT,
    download_url    TEXT,
    sha256          TEXT,
    install_path    TEXT,
    status          TEXT NOT NULL DEFAULT 'available',  -- 'available', 'installed', 'disabled'
    manifest        TEXT,           -- full JSON manifest
    installed_at    DATETIME,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Plugin dependencies
CREATE TABLE IF NOT EXISTS plugin_dependency (
    plugin_id       TEXT NOT NULL,
    depends_on      TEXT NOT NULL,
    min_version     TEXT,
    PRIMARY KEY (plugin_id, depends_on),
    FOREIGN KEY (plugin_id) REFERENCES plugin(id)
);

-- Plugin capability index
CREATE TABLE IF NOT EXISTS plugin_capability (
    plugin_id       TEXT NOT NULL,
    capability      TEXT NOT NULL,
    PRIMARY KEY (plugin_id, capability),
    FOREIGN KEY (plugin_id) REFERENCES plugin(id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_kind     ON plugin(kind);
CREATE INDEX IF NOT EXISTS idx_plugin_language ON plugin(language);
CREATE INDEX IF NOT EXISTS idx_plugin_platform ON plugin(platform);
CREATE INDEX IF NOT EXISTS idx_plugin_status   ON plugin(status);
