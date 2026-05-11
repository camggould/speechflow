-- Adds the 'takeaway' kind to the nodes.kind CHECK constraint. SQLite can't
-- modify a CHECK constraint in place, so we recreate the table.
--
-- Foreign keys are toggled off for the rename dance: edges and node_tags
-- reference nodes(id), and the self-referencing FK on resolved_by_node_id
-- would otherwise prevent the temporary drop of the old table. The PRAGMAs
-- and DDL all run in one Exec — `mattn/go-sqlite3` accepts multi-statement
-- payloads — and the migration runner records this file in
-- `schema_migrations` so it only ever runs once per database.

PRAGMA foreign_keys=OFF;

CREATE TABLE nodes_new (
    id                   TEXT PRIMARY KEY,
    iteration_id         TEXT NOT NULL REFERENCES iterations(id) ON DELETE CASCADE,
    kind                 TEXT NOT NULL CHECK (kind IN ('root_ref','concept','curiosity','takeaway')),
    title                TEXT NOT NULL,
    quote                TEXT,
    transcript_start     INTEGER,
    transcript_end       INTEGER,
    root_id              TEXT REFERENCES roots(id) ON DELETE SET NULL,
    resolved_by_node_id  TEXT REFERENCES nodes(id) ON DELETE SET NULL,
    source               TEXT NOT NULL DEFAULT 'agent',
    created_at           TEXT NOT NULL
);

INSERT INTO nodes_new(
    id, iteration_id, kind, title, quote,
    transcript_start, transcript_end, root_id,
    resolved_by_node_id, source, created_at
)
SELECT
    id, iteration_id, kind, title, quote,
    transcript_start, transcript_end, root_id,
    resolved_by_node_id, source, created_at
FROM nodes;

DROP TABLE nodes;
ALTER TABLE nodes_new RENAME TO nodes;

CREATE INDEX IF NOT EXISTS idx_nodes_iteration   ON nodes(iteration_id);
CREATE INDEX IF NOT EXISTS idx_nodes_resolved_by ON nodes(resolved_by_node_id);

PRAGMA foreign_keys=ON;
