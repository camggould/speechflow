CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    description  TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS roots (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS iterations (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    transcript  TEXT NOT NULL DEFAULT '',
    started_at  TEXT NOT NULL,
    ended_at    TEXT
);

CREATE TABLE IF NOT EXISTS nodes (
    id                   TEXT PRIMARY KEY,
    iteration_id         TEXT NOT NULL REFERENCES iterations(id) ON DELETE CASCADE,
    kind                 TEXT NOT NULL CHECK (kind IN ('root_ref','concept','curiosity')),
    title                TEXT NOT NULL,
    quote                TEXT,
    transcript_start     INTEGER,
    transcript_end       INTEGER,
    root_id              TEXT REFERENCES roots(id) ON DELETE SET NULL,
    resolved_by_node_id  TEXT REFERENCES nodes(id) ON DELETE SET NULL,
    source               TEXT NOT NULL DEFAULT 'agent',
    created_at           TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS node_tags (
    node_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY (node_id, tag)
);

CREATE TABLE IF NOT EXISTS edges (
    id           TEXT PRIMARY KEY,
    iteration_id TEXT NOT NULL REFERENCES iterations(id) ON DELETE CASCADE,
    from_node    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node      TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('branches_from','references','returns_to')),
    created_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_iterations_session    ON iterations(session_id);
CREATE INDEX IF NOT EXISTS idx_roots_session         ON roots(session_id);
CREATE INDEX IF NOT EXISTS idx_nodes_iteration       ON nodes(iteration_id);
CREATE INDEX IF NOT EXISTS idx_edges_iteration       ON edges(iteration_id);
CREATE INDEX IF NOT EXISTS idx_nodes_resolved_by     ON nodes(resolved_by_node_id);
