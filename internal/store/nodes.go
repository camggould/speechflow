// Node, tag, and edge CRUD lives here.
package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/camggould/speechflow/internal/core"
)

// NodeInput collects the optional fields used when creating a node.
type NodeInput struct {
	ID              string
	IterationID     string
	Kind            core.NodeKind
	Title           string
	Quote           *string
	TranscriptStart *int
	TranscriptEnd   *int
	RootID          *string
	Source          core.NodeSource
	Tags            []string
}

// CreateNode inserts a node and (optionally) its tags. Returns the new row
// with tags populated. Validates kind-specific constraints:
//   - kind=root_ref must carry root_id
//   - root_id is optional on takeaway (associates a takeaway with the root
//     it was supposed to land on) and forbidden on concept/curiosity
func (s *Store) CreateNode(in NodeInput) (*core.Node, error) {
	if in.Kind == core.NodeKindRootRef && (in.RootID == nil || *in.RootID == "") {
		return nil, fmt.Errorf("%w: root_ref node requires root_id", ErrConstraint)
	}
	if in.Kind != core.NodeKindRootRef && in.Kind != core.NodeKindTakeaway && in.RootID != nil {
		return nil, fmt.Errorf("%w: root_id only allowed on root_ref or takeaway nodes", ErrConstraint)
	}
	if in.Source == "" {
		in.Source = core.SourceAgent
	}
	now := nowUTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(
		`INSERT INTO nodes(id, iteration_id, kind, title, quote, transcript_start, transcript_end,
		                   root_id, resolved_by_node_id, source, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)`,
		in.ID, in.IterationID, string(in.Kind), in.Title,
		nullStr(in.Quote), nullInt(in.TranscriptStart), nullInt(in.TranscriptEnd),
		nullStr(in.RootID), string(in.Source), rfc3339(now),
	)
	if err != nil {
		return nil, fmt.Errorf("store: insert node: %w", err)
	}
	for _, tag := range in.Tags {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO node_tags(node_id, tag) VALUES(?, ?)`,
			in.ID, tag,
		); err != nil {
			return nil, fmt.Errorf("store: insert tag: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetNode(in.ID)
}

// GetNode returns the node with the given ID, with tags populated.
func (s *Store) GetNode(id string) (*core.Node, error) {
	row := s.db.QueryRow(
		`SELECT id, iteration_id, kind, title, quote, transcript_start, transcript_end,
		        root_id, resolved_by_node_id, source, created_at
		   FROM nodes WHERE id = ?`,
		id,
	)
	var (
		n               core.Node
		kind, source, ca string
		quote, rootID, resolved sql.NullString
		ts, te          sql.NullInt64
	)
	err := row.Scan(&n.ID, &n.IterationID, &kind, &n.Title, &quote, &ts, &te, &rootID, &resolved, &source, &ca)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	n.Kind = core.NodeKind(kind)
	n.Source = core.NodeSource(source)
	n.Quote = strPtr(quote)
	n.TranscriptStart = intPtr(ts)
	n.TranscriptEnd = intPtr(te)
	n.RootID = strPtr(rootID)
	n.ResolvedByNodeID = strPtr(resolved)
	if n.CreatedAt, err = parseTime(ca); err != nil {
		return nil, err
	}
	tags, err := s.listTags(n.ID)
	if err != nil {
		return nil, err
	}
	n.Tags = tags
	return &n, nil
}

func (s *Store) listTags(nodeID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT tag FROM node_tags WHERE node_id = ? ORDER BY tag ASC`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// ListNodes returns every node in an iteration ordered by creation time.
func (s *Store) ListNodes(iterationID string) ([]core.Node, error) {
	rows, err := s.db.Query(
		`SELECT id, iteration_id, kind, title, quote, transcript_start, transcript_end,
		        root_id, resolved_by_node_id, source, created_at
		   FROM nodes WHERE iteration_id = ? ORDER BY created_at ASC`,
		iterationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Node
	for rows.Next() {
		var (
			n               core.Node
			kind, source, ca string
			quote, rootID, resolved sql.NullString
			ts, te          sql.NullInt64
		)
		if err := rows.Scan(&n.ID, &n.IterationID, &kind, &n.Title, &quote, &ts, &te, &rootID, &resolved, &source, &ca); err != nil {
			return nil, err
		}
		n.Kind = core.NodeKind(kind)
		n.Source = core.NodeSource(source)
		n.Quote = strPtr(quote)
		n.TranscriptStart = intPtr(ts)
		n.TranscriptEnd = intPtr(te)
		n.RootID = strPtr(rootID)
		n.ResolvedByNodeID = strPtr(resolved)
		if n.CreatedAt, err = parseTime(ca); err != nil {
			return nil, err
		}
		tags, err := s.listTags(n.ID)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
		out = append(out, n)
	}
	return out, rows.Err()
}

// DeleteNode removes a node; FK cascades delete its tags and any edges
// touching it.
func (s *Store) DeleteNode(id string) error {
	res, err := s.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveCuriosity sets resolved_by_node_id on a curiosity node. Returns
// ErrConstraint if the target is not of kind curiosity.
func (s *Store) ResolveCuriosity(curiosityID, byNodeID string) (*core.Node, error) {
	c, err := s.GetNode(curiosityID)
	if err != nil {
		return nil, err
	}
	if c.Kind != core.NodeKindCuriosity {
		return nil, fmt.Errorf("%w: %s is not a curiosity", ErrConstraint, curiosityID)
	}
	if _, err := s.GetNode(byNodeID); err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(
		`UPDATE nodes SET resolved_by_node_id = ? WHERE id = ?`,
		byNodeID, curiosityID,
	); err != nil {
		return nil, err
	}
	return s.GetNode(curiosityID)
}

// AddTags inserts (node_id, tag) rows. Existing rows are ignored.
func (s *Store) AddTags(nodeID string, tags []string) (*core.Node, error) {
	if _, err := s.GetNode(nodeID); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, t := range tags {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO node_tags(node_id, tag) VALUES(?, ?)`,
			nodeID, t,
		); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetNode(nodeID)
}

// RemoveTag deletes a single tag from a node. Missing rows are a no-op.
func (s *Store) RemoveTag(nodeID, tag string) (*core.Node, error) {
	if _, err := s.db.Exec(
		`DELETE FROM node_tags WHERE node_id = ? AND tag = ?`,
		nodeID, tag,
	); err != nil {
		return nil, err
	}
	return s.GetNode(nodeID)
}

// --- edges ----------------------------------------------------------------

// CreateEdge inserts an edge between two nodes within the same iteration
// and returns the new row. The edge ID is generated by the caller; the
// store does not pick IDs.
func (s *Store) CreateEdge(id, iterationID, from, to string, kind core.EdgeKind) (*core.Edge, error) {
	// Sanity-check that both endpoints exist and live in the iteration.
	for _, nid := range []string{from, to} {
		var iter string
		err := s.db.QueryRow(`SELECT iteration_id FROM nodes WHERE id = ?`, nid).Scan(&iter)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: node %s not found", ErrNotFound, nid)
		}
		if err != nil {
			return nil, err
		}
		if iter != iterationID {
			return nil, fmt.Errorf("%w: node %s is not in iteration %s", ErrConstraint, nid, iterationID)
		}
	}

	now := nowUTC()
	_, err := s.db.Exec(
		`INSERT INTO edges(id, iteration_id, from_node, to_node, kind, created_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		id, iterationID, from, to, string(kind), rfc3339(now),
	)
	if err != nil {
		return nil, fmt.Errorf("store: insert edge: %w", err)
	}
	return &core.Edge{
		ID:          id,
		IterationID: iterationID,
		FromNode:    from,
		ToNode:      to,
		Kind:        kind,
		CreatedAt:   now,
	}, nil
}

// ListEdges returns every edge in an iteration in creation order.
func (s *Store) ListEdges(iterationID string) ([]core.Edge, error) {
	rows, err := s.db.Query(
		`SELECT id, iteration_id, from_node, to_node, kind, created_at
		   FROM edges WHERE iteration_id = ? ORDER BY created_at ASC`,
		iterationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Edge
	for rows.Next() {
		var e core.Edge
		var kind, ca string
		if err := rows.Scan(&e.ID, &e.IterationID, &e.FromNode, &e.ToNode, &kind, &ca); err != nil {
			return nil, err
		}
		e.Kind = core.EdgeKind(kind)
		if e.CreatedAt, err = parseTime(ca); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteEdge removes an edge by ID.
func (s *Store) DeleteEdge(id string) error {
	res, err := s.db.Exec(`DELETE FROM edges WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// NextEdgeID returns a small, monotonically-increasing edge ID for an
// iteration. Edges aren't slug-addressed in the CLI README but they do
// need a stable ID for `edge delete`.
func (s *Store) NextEdgeID(iterationID string) (string, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM edges WHERE iteration_id = ?`, iterationID).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-edge-%d", iterationID, n+1), nil
}
