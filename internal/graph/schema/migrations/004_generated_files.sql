-- Generated files from code generator
CREATE TABLE IF NOT EXISTS generated_file (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL,
    path        TEXT NOT NULL,
    content     TEXT NOT NULL,
    language    TEXT NOT NULL DEFAULT '',
    size        INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_generated_file_project ON generated_file(project_id);

-- Generated project metadata
CREATE TABLE IF NOT EXISTS generated_project (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    platform    TEXT NOT NULL DEFAULT '',
    language    TEXT NOT NULL DEFAULT '',
    framework   TEXT NOT NULL DEFAULT '',
    plugin_id   TEXT NOT NULL DEFAULT '',
    prompt      TEXT NOT NULL DEFAULT '',
    features    TEXT NOT NULL DEFAULT '[]',
    entities    TEXT NOT NULL DEFAULT '[]',
    file_count  INTEGER NOT NULL DEFAULT 0,
    score       INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_generated_project_created ON generated_project(created_at DESC);
