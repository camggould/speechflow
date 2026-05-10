// Session, root, and iteration CRUD lives here.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/camggould/speechflow/internal/core"
)

// CreateSession inserts a new session with the given slug ID and returns
// the freshly created row. CreatedAt and UpdatedAt are set to now() UTC.
func (s *Store) CreateSession(id, title string, description *string) (*core.Session, error) {
	now := nowUTC()
	_, err := s.db.Exec(
		`INSERT INTO sessions(id, title, description, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?)`,
		id, title, nullStr(description), rfc3339(now), rfc3339(now),
	)
	if err != nil {
		return nil, fmt.Errorf("store: create session: %w", err)
	}
	return s.GetSession(id)
}

// GetSession returns the session with the given ID, decorated with
// iteration_count, last_activity_at, and latest_coverage_pct fields.
// Coverage is computed lazily by the caller — store returns 0 here.
func (s *Store) GetSession(id string) (*core.Session, error) {
	row := s.db.QueryRow(
		`SELECT id, title, description, created_at, updated_at FROM sessions WHERE id = ?`,
		id,
	)
	var (
		sess core.Session
		desc sql.NullString
		ca, ua string
	)
	err := row.Scan(&sess.ID, &sess.Title, &desc, &ca, &ua)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get session: %w", err)
	}
	sess.Description = strPtr(desc)
	if sess.CreatedAt, err = parseTime(ca); err != nil {
		return nil, err
	}
	if sess.UpdatedAt, err = parseTime(ua); err != nil {
		return nil, err
	}
	if err := s.decorateSession(&sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// decorateSession fills iteration_count, last_activity_at, and
// latest_coverage_pct (which is left as 0 — set by the coverage package).
func (s *Store) decorateSession(sess *core.Session) error {
	var count int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM iterations WHERE session_id = ?`, sess.ID,
	).Scan(&count); err != nil {
		return fmt.Errorf("store: count iterations: %w", err)
	}
	sess.IterationCount = count

	// last_activity_at = max of (session.updated_at, max iteration.started_at, max iteration.ended_at)
	last := sess.UpdatedAt
	var maxStart, maxEnd sql.NullString
	if err := s.db.QueryRow(
		`SELECT MAX(started_at), MAX(ended_at) FROM iterations WHERE session_id = ?`, sess.ID,
	).Scan(&maxStart, &maxEnd); err != nil {
		return fmt.Errorf("store: max iteration times: %w", err)
	}
	if maxStart.Valid {
		if t, err := parseTime(maxStart.String); err == nil && t.After(last) {
			last = t
		}
	}
	if maxEnd.Valid {
		if t, err := parseTime(maxEnd.String); err == nil && t.After(last) {
			last = t
		}
	}
	sess.LastActivityAt = last
	return nil
}

// ListSessions returns every session sorted newest-first by last_activity.
func (s *Store) ListSessions() ([]core.Session, error) {
	rows, err := s.db.Query(
		`SELECT id, title, description, created_at, updated_at FROM sessions ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}
	defer rows.Close()

	var out []core.Session
	for rows.Next() {
		var (
			sess core.Session
			desc sql.NullString
			ca, ua string
		)
		if err := rows.Scan(&sess.ID, &sess.Title, &desc, &ca, &ua); err != nil {
			return nil, err
		}
		sess.Description = strPtr(desc)
		if sess.CreatedAt, err = parseTime(ca); err != nil {
			return nil, err
		}
		if sess.UpdatedAt, err = parseTime(ua); err != nil {
			return nil, err
		}
		if err := s.decorateSession(&sess); err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// DeleteSession removes a session and all dependent rows via FK cascade.
func (s *Store) DeleteSession(id string) error {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchSession bumps updated_at to now().
func (s *Store) TouchSession(id string) error {
	_, err := s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, rfc3339(nowUTC()), id)
	return err
}

// --- roots ----------------------------------------------------------------

// CreateRoot inserts a root scoped to sessionID and returns the new row.
func (s *Store) CreateRoot(id, sessionID, title string) (*core.Root, error) {
	now := nowUTC()
	_, err := s.db.Exec(
		`INSERT INTO roots(id, session_id, title, created_at) VALUES(?, ?, ?, ?)`,
		id, sessionID, title, rfc3339(now),
	)
	if err != nil {
		return nil, fmt.Errorf("store: create root: %w", err)
	}
	_ = s.TouchSession(sessionID)
	return &core.Root{ID: id, SessionID: sessionID, Title: title, CreatedAt: now}, nil
}

// GetRoot returns the root with the given ID.
func (s *Store) GetRoot(id string) (*core.Root, error) {
	row := s.db.QueryRow(`SELECT id, session_id, title, created_at FROM roots WHERE id = ?`, id)
	var r core.Root
	var ca string
	err := row.Scan(&r.ID, &r.SessionID, &r.Title, &ca)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if r.CreatedAt, err = parseTime(ca); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRoots returns roots for a session ordered by creation time.
func (s *Store) ListRoots(sessionID string) ([]core.Root, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, title, created_at FROM roots WHERE session_id = ? ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Root
	for rows.Next() {
		var r core.Root
		var ca string
		if err := rows.Scan(&r.ID, &r.SessionID, &r.Title, &ca); err != nil {
			return nil, err
		}
		if r.CreatedAt, err = parseTime(ca); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteRoot removes a root. Nodes referencing it have their root_id set NULL.
func (s *Store) DeleteRoot(id string) error {
	res, err := s.db.Exec(`DELETE FROM roots WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// --- iterations -----------------------------------------------------------

// CreateIteration inserts an iteration with started_at = now() and returns
// the new row. ended_at remains NULL until EndIteration is called.
func (s *Store) CreateIteration(id, sessionID, title string) (*core.Iteration, error) {
	now := nowUTC()
	_, err := s.db.Exec(
		`INSERT INTO iterations(id, session_id, title, transcript, started_at, ended_at)
		 VALUES(?, ?, ?, '', ?, NULL)`,
		id, sessionID, title, rfc3339(now),
	)
	if err != nil {
		return nil, fmt.Errorf("store: create iteration: %w", err)
	}
	_ = s.TouchSession(sessionID)
	return s.GetIteration(id)
}

// GetIteration returns the iteration with the given ID. node_count is
// populated; coverage_pct is left as 0 for the coverage package to fill.
func (s *Store) GetIteration(id string) (*core.Iteration, error) {
	row := s.db.QueryRow(
		`SELECT id, session_id, title, started_at, ended_at FROM iterations WHERE id = ?`,
		id,
	)
	var it core.Iteration
	var sa string
	var ea sql.NullString
	err := row.Scan(&it.ID, &it.SessionID, &it.Title, &sa, &ea)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if it.StartedAt, err = parseTime(sa); err != nil {
		return nil, err
	}
	if it.EndedAt, err = timePtr(ea); err != nil {
		return nil, err
	}

	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM nodes WHERE iteration_id = ?`, id,
	).Scan(&it.NodeCount); err != nil {
		return nil, err
	}
	return &it, nil
}

// ListIterations returns iterations for a session, newest first.
func (s *Store) ListIterations(sessionID string) ([]core.Iteration, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, title, started_at, ended_at FROM iterations WHERE session_id = ? ORDER BY started_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.Iteration
	for rows.Next() {
		var it core.Iteration
		var sa string
		var ea sql.NullString
		if err := rows.Scan(&it.ID, &it.SessionID, &it.Title, &sa, &ea); err != nil {
			return nil, err
		}
		if it.StartedAt, err = parseTime(sa); err != nil {
			return nil, err
		}
		if it.EndedAt, err = timePtr(ea); err != nil {
			return nil, err
		}
		var nc int
		if err := s.db.QueryRow(
			`SELECT COUNT(*) FROM nodes WHERE iteration_id = ?`, it.ID,
		).Scan(&nc); err != nil {
			return nil, err
		}
		it.NodeCount = nc
		out = append(out, it)
	}
	return out, rows.Err()
}

// EndIteration sets ended_at on an iteration to now() and returns the row.
// If already ended, ended_at is left as-is.
func (s *Store) EndIteration(id string) (*core.Iteration, error) {
	it, err := s.GetIteration(id)
	if err != nil {
		return nil, err
	}
	if it.EndedAt == nil {
		now := nowUTC()
		_, err := s.db.Exec(`UPDATE iterations SET ended_at = ? WHERE id = ?`, rfc3339(now), id)
		if err != nil {
			return nil, err
		}
		it.EndedAt = &now
		_ = s.TouchSession(it.SessionID)
	}
	return it, nil
}

// DeleteIteration removes an iteration and its nodes/edges via cascade.
func (s *Store) DeleteIteration(id string) error {
	res, err := s.db.Exec(`DELETE FROM iterations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendTranscript appends text (with a space delimiter if non-empty) to
// the iteration's transcript and returns the updated iteration.
func (s *Store) AppendTranscript(id, text string) (*core.Iteration, error) {
	// Concatenate in SQL so we don't round-trip the existing string.
	_, err := s.db.Exec(
		`UPDATE iterations
		   SET transcript = CASE WHEN transcript = '' THEN ? ELSE transcript || ' ' || ? END
		 WHERE id = ?`,
		text, text, id,
	)
	if err != nil {
		return nil, err
	}
	return s.GetIteration(id)
}

// SetTranscript replaces the entire transcript text.
func (s *Store) SetTranscript(id, text string) (*core.Iteration, error) {
	res, err := s.db.Exec(`UPDATE iterations SET transcript = ? WHERE id = ?`, text, id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.GetIteration(id)
}

// GetTranscript returns the iteration transcript text along with the
// (node_id, start, end) spans collected from every node that has a span.
func (s *Store) GetTranscript(id string) (*core.Transcript, error) {
	var text string
	err := s.db.QueryRow(`SELECT transcript FROM iterations WHERE id = ?`, id).Scan(&text)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT id, transcript_start, transcript_end FROM nodes
		 WHERE iteration_id = ? AND transcript_start IS NOT NULL AND transcript_end IS NOT NULL
		 ORDER BY transcript_start ASC`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var spans []core.TranscriptSpan
	for rows.Next() {
		var nid string
		var start, end int
		if err := rows.Scan(&nid, &start, &end); err != nil {
			return nil, err
		}
		spans = append(spans, core.TranscriptSpan{NodeID: nid, Start: start, End: end})
	}
	return &core.Transcript{Text: text, Spans: spans}, nil
}

// IterationStartedAt returns just the started_at timestamp without scanning the row.
func (s *Store) IterationStartedAt(id string) (time.Time, error) {
	var sa string
	if err := s.db.QueryRow(`SELECT started_at FROM iterations WHERE id = ?`, id).Scan(&sa); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, ErrNotFound
		}
		return time.Time{}, err
	}
	return parseTime(sa)
}
