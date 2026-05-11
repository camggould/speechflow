-- Adds 'supports' and 'contrasts' to the edges.kind CHECK constraint.
-- Same recreate-and-rename dance as 002_takeaway.sql, scoped to edges.

PRAGMA foreign_keys=OFF;

CREATE TABLE edges_new (
    id           TEXT PRIMARY KEY,
    iteration_id TEXT NOT NULL REFERENCES iterations(id) ON DELETE CASCADE,
    from_node    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node      TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN (
                     'branches_from','references','returns_to','supports','contrasts'
                 )),
    created_at   TEXT NOT NULL
);

INSERT INTO edges_new(id, iteration_id, from_node, to_node, kind, created_at)
SELECT id, iteration_id, from_node, to_node, kind, created_at FROM edges;

DROP TABLE edges;
ALTER TABLE edges_new RENAME TO edges;

CREATE INDEX IF NOT EXISTS idx_edges_iteration ON edges(iteration_id);

PRAGMA foreign_keys=ON;
