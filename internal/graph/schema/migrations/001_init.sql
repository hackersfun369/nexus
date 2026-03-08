-- ─────────────────────────────────────────
-- NEXUS Knowledge Graph — Initial Schema
-- Migration: 001_init
-- ─────────────────────────────────────────

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA synchronous = NORMAL;

-- ─────────────────────────────────────────
-- SCHEMA VERSION TRACKING
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS schema_versions (
    version     TEXT     NOT NULL PRIMARY KEY,
    applied_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    description TEXT     NOT NULL
);

-- ─────────────────────────────────────────
-- PROJECT
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS project (
    id                TEXT NOT NULL PRIMARY KEY,
    name              TEXT NOT NULL,
    root_path         TEXT NOT NULL,
    platform          TEXT NOT NULL DEFAULT 'unknown',
    primary_language  TEXT NOT NULL DEFAULT 'unknown',
    created_at        DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at        DATETIME NOT NULL DEFAULT (datetime('now')),
    last_analyzed     DATETIME,
    version           TEXT NOT NULL DEFAULT '0.1.0',
    description       TEXT
);

-- ─────────────────────────────────────────
-- NODES
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS nodes (
    id          TEXT    NOT NULL PRIMARY KEY,
    node_type   TEXT    NOT NULL,
    project_id  TEXT    NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    version     INTEGER  NOT NULL DEFAULT 1,
    checksum    TEXT     NOT NULL DEFAULT '',
    is_deleted  BOOLEAN  NOT NULL DEFAULT FALSE,
    tags        TEXT     NOT NULL DEFAULT '[]',

    CHECK (node_type IN (
        'MODULE','FUNCTION','CLASS','INTERFACE',
        'VARIABLE','FIELD','PARAMETER','CONSTANT',
        'ISSUE','IMPORT','DEPENDENCY'
    ))
);

CREATE INDEX IF NOT EXISTS idx_nodes_project  ON nodes(project_id);
CREATE INDEX IF NOT EXISTS idx_nodes_type     ON nodes(node_type);
CREATE INDEX IF NOT EXISTS idx_nodes_updated  ON nodes(updated_at);
CREATE INDEX IF NOT EXISTS idx_nodes_checksum ON nodes(checksum);

-- ─────────────────────────────────────────
-- MODULES
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS modules (
    id              TEXT NOT NULL PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    file_path       TEXT NOT NULL,
    qualified_name  TEXT NOT NULL,
    language        TEXT NOT NULL,
    lines_of_code   INTEGER NOT NULL DEFAULT 0,
    parse_status    TEXT NOT NULL DEFAULT 'OK',
    parse_errors    TEXT NOT NULL DEFAULT '[]',
    cycle_risk      REAL NOT NULL DEFAULT 0.0,

    UNIQUE(file_path),
    CHECK (parse_status IN ('OK','ERROR','PARTIAL'))
);

CREATE INDEX IF NOT EXISTS idx_modules_file     ON modules(file_path);
CREATE INDEX IF NOT EXISTS idx_modules_language ON modules(language);

-- ─────────────────────────────────────────
-- FUNCTIONS
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS functions (
    id                      TEXT NOT NULL PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    name                    TEXT NOT NULL,
    qualified_name          TEXT NOT NULL DEFAULT '',
    module_id               TEXT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    language                TEXT NOT NULL,
    start_line              INTEGER NOT NULL DEFAULT 0,
    start_col               INTEGER NOT NULL DEFAULT 0,
    end_line                INTEGER NOT NULL DEFAULT 0,
    end_col                 INTEGER NOT NULL DEFAULT 0,
    visibility              TEXT NOT NULL DEFAULT 'PUBLIC',
    parameters              TEXT NOT NULL DEFAULT '[]',
    return_type             TEXT NOT NULL DEFAULT '{}',
    is_async                BOOLEAN NOT NULL DEFAULT FALSE,
    is_static               BOOLEAN NOT NULL DEFAULT FALSE,
    is_abstract             BOOLEAN NOT NULL DEFAULT FALSE,
    is_constructor          BOOLEAN NOT NULL DEFAULT FALSE,
    cyclomatic_complexity   INTEGER NOT NULL DEFAULT 1,
    cognitive_complexity    INTEGER NOT NULL DEFAULT 0,
    lines_of_code           INTEGER NOT NULL DEFAULT 0,
    parameter_count         INTEGER NOT NULL DEFAULT 0,
    nesting_depth           INTEGER NOT NULL DEFAULT 0,
    fan_in                  INTEGER NOT NULL DEFAULT 0,
    fan_out                 INTEGER NOT NULL DEFAULT 0,
    test_coverage           REAL,
    doc_comment             TEXT,
    annotations             TEXT NOT NULL DEFAULT '[]',

    CHECK (visibility IN ('PUBLIC','PROTECTED','PRIVATE','INTERNAL','PACKAGE'))
);

CREATE INDEX IF NOT EXISTS idx_functions_module     ON functions(module_id);
CREATE INDEX IF NOT EXISTS idx_functions_name       ON functions(name);
CREATE INDEX IF NOT EXISTS idx_functions_complexity ON functions(cyclomatic_complexity);

-- ─────────────────────────────────────────
-- CLASSES
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS classes (
    id                          TEXT NOT NULL PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    name                        TEXT NOT NULL,
    qualified_name              TEXT NOT NULL DEFAULT '',
    module_id                   TEXT NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    language                    TEXT NOT NULL,
    kind                        TEXT NOT NULL DEFAULT 'CLASS',
    start_line                  INTEGER NOT NULL DEFAULT 0,
    start_col                   INTEGER NOT NULL DEFAULT 0,
    end_line                    INTEGER NOT NULL DEFAULT 0,
    end_col                     INTEGER NOT NULL DEFAULT 0,
    visibility                  TEXT NOT NULL DEFAULT 'PUBLIC',
    method_count                INTEGER NOT NULL DEFAULT 0,
    field_count                 INTEGER NOT NULL DEFAULT 0,
    lines_of_code               INTEGER NOT NULL DEFAULT 0,
    lack_of_cohesion            REAL NOT NULL DEFAULT 0.0,
    coupling_between_objects    INTEGER NOT NULL DEFAULT 0,
    doc_comment                 TEXT,
    annotations                 TEXT NOT NULL DEFAULT '[]',
    is_abstract                 BOOLEAN NOT NULL DEFAULT FALSE,

    CHECK (kind IN ('CLASS','INTERFACE','ABSTRACT_CLASS','ENUM','RECORD','OBJECT')),
    CHECK (visibility IN ('PUBLIC','PROTECTED','PRIVATE','INTERNAL','PACKAGE'))
);

CREATE INDEX IF NOT EXISTS idx_classes_module  ON classes(module_id);
CREATE INDEX IF NOT EXISTS idx_classes_name    ON classes(name);

-- ─────────────────────────────────────────
-- ISSUES
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS issues (
    id              TEXT NOT NULL PRIMARY KEY,
    node_id         TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id      TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    rule_id         TEXT NOT NULL,
    severity        TEXT NOT NULL,
    category        TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    file_path       TEXT NOT NULL,
    start_line      INTEGER NOT NULL DEFAULT 0,
    start_col       INTEGER NOT NULL DEFAULT 0,
    evidence        TEXT NOT NULL DEFAULT '',
    remediation     TEXT NOT NULL DEFAULT '',
    inference_chain TEXT NOT NULL DEFAULT '[]',
    cwe             TEXT,
    owasp           TEXT,
    status          TEXT NOT NULL DEFAULT 'OPEN',
    detected_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    resolved_at     DATETIME,
    resolved_by     TEXT,
    false_positive_reason TEXT,

    CHECK (severity IN ('CRITICAL','HIGH','MEDIUM','LOW','INFO')),
    CHECK (category IN (
        'SECURITY','CORRECTNESS','PERFORMANCE',
        'MAINTAINABILITY','ARCHITECTURE','STYLE',
        'DOCUMENTATION','TEST_COVERAGE'
    )),
    CHECK (status IN ('OPEN','ACKNOWLEDGED','RESOLVED','FALSE_POSITIVE'))
);

CREATE INDEX IF NOT EXISTS idx_issues_project   ON issues(project_id);
CREATE INDEX IF NOT EXISTS idx_issues_node      ON issues(node_id);
CREATE INDEX IF NOT EXISTS idx_issues_rule      ON issues(rule_id);
CREATE INDEX IF NOT EXISTS idx_issues_severity  ON issues(severity);
CREATE INDEX IF NOT EXISTS idx_issues_status    ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_detected  ON issues(detected_at);

-- ─────────────────────────────────────────
-- EDGES
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS edges (
    id          TEXT NOT NULL PRIMARY KEY,
    edge_type   TEXT NOT NULL,
    from_node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node_id   TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    created_at   DATETIME NOT NULL DEFAULT (datetime('now')),
    properties   TEXT NOT NULL DEFAULT '{}',

    CHECK (edge_type IN (
        'CALLS','CONTAINS','IMPORTS','INHERITS',
        'USES_TYPE','OVERRIDES','DATA_FLOWS_TO',
        'HAS_ISSUE','DEPENDS_ON','TESTS',
        'DOCUMENTED_BY','VERSION_OF'
    ))
);

CREATE INDEX IF NOT EXISTS idx_edges_from    ON edges(from_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_to      ON edges(to_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_type    ON edges(edge_type);
CREATE INDEX IF NOT EXISTS idx_edges_project ON edges(project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_edges_unique
    ON edges(edge_type, from_node_id, to_node_id);

-- ─────────────────────────────────────────
-- NODE HISTORY
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS node_history (
    id          TEXT NOT NULL PRIMARY KEY,
    node_id     TEXT NOT NULL,
    project_id  TEXT NOT NULL,
    version     INTEGER NOT NULL,
    snapshot    TEXT NOT NULL,
    changed_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    change_type TEXT NOT NULL,
    commit_hash TEXT,

    UNIQUE(node_id, version),
    CHECK (change_type IN ('CREATED','MODIFIED','DELETED'))
);

CREATE INDEX IF NOT EXISTS idx_history_node    ON node_history(node_id);
CREATE INDEX IF NOT EXISTS idx_history_project ON node_history(project_id);

-- ─────────────────────────────────────────
-- BUILDS
-- ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS builds (
    id              TEXT NOT NULL PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES project(id) ON DELETE CASCADE,
    platform        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    started_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    completed_at    DATETIME,
    output_path     TEXT,
    size_bytes      INTEGER,
    error_log       TEXT,
    test_passed     INTEGER NOT NULL DEFAULT 0,
    test_failed     INTEGER NOT NULL DEFAULT 0,
    test_coverage   REAL,

    CHECK (status IN ('PENDING','RUNNING','SUCCESS','FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_builds_project ON builds(project_id);
CREATE INDEX IF NOT EXISTS idx_builds_status  ON builds(status);

-- ─────────────────────────────────────────
-- VIEWS
-- ─────────────────────────────────────────

CREATE VIEW IF NOT EXISTS v_urgent_issues AS
SELECT
    i.id, i.rule_id, i.severity, i.category,
    i.title, i.file_path, i.start_line,
    i.evidence, i.remediation, i.detected_at
FROM issues i
WHERE i.status = 'OPEN'
  AND i.severity IN ('CRITICAL','HIGH')
ORDER BY
    CASE i.severity WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 END,
    i.detected_at DESC;

CREATE VIEW IF NOT EXISTS v_project_health AS
SELECT
    (SELECT COUNT(*) FROM issues WHERE status='OPEN' AND severity='CRITICAL') AS critical_issues,
    (SELECT COUNT(*) FROM issues WHERE status='OPEN' AND severity='HIGH')     AS high_issues,
    (SELECT COUNT(*) FROM issues WHERE status='OPEN' AND severity='MEDIUM')   AS medium_issues,
    (SELECT COUNT(*) FROM functions WHERE cyclomatic_complexity > 15)         AS complex_functions,
    (SELECT COUNT(*) FROM functions)                                           AS total_functions,
    (SELECT COUNT(*) FROM classes)                                             AS total_classes,
    (SELECT COUNT(*) FROM modules)                                             AS total_modules;

-- Record this migration
INSERT OR IGNORE INTO schema_versions VALUES
    ('001', datetime('now'), 'Initial schema — nodes, edges, issues, builds');