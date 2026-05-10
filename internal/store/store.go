// Package store is speechflow's SQLite persistence layer. It owns the
// database connection, embedded migrations, and CRUD methods over the
// domain types declared in internal/core. The CLI is the only writer;
// the HTTP server only reads. All write methods return the row they
// produced so the caller can echo it as JSON.
package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// ErrNotFound is returned by lookup methods when no row matches the slug.
var ErrNotFound = errors.New("not found")

// ErrConstraint is returned when a write violates a domain constraint
// (e.g. resolving a non-curiosity, missing FK).
var ErrConstraint = errors.New("constraint violation")

// Store wraps the SQLite handle plus migration state.
type Store struct {
	db   *sql.DB
	path string
}

// Open initializes (or opens) the SQLite database at path and applies
// pending migrations. The file is created if missing.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	s := &Store{db: conn, path: path}
	if err := s.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

// DB returns the underlying *sql.DB for read-only use by handlers
// that need to compose their own queries.
func (s *Store) DB() *sql.DB { return s.db }

// migrate applies every embedded migration file in lexical order.
func (s *Store) migrate() error {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("store: read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		data, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("store: read migration %s: %w", name, err)
		}
		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("store: apply migration %s: %w", name, err)
		}
	}
	return nil
}

// --- helpers ---------------------------------------------------------------

func nowUTC() time.Time { return time.Now().UTC() }

func rfc3339(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	return time.Parse(time.RFC3339, s)
}

func nullStr(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func strPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func timePtr(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := parseTime(s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// SlugExists is a generic existence check used by the slug.Unique helper.
// table must be one of the known speechflow tables. When scope is non-empty
// the check is restricted to the given scope (e.g. session_id for iterations
// or iteration_id for nodes) so slugs only need to be unique within scope.
func (s *Store) SlugExists(table, scope, candidate string) (bool, error) {
	var q string
	args := []any{candidate}
	switch table {
	case "sessions":
		q = `SELECT 1 FROM sessions WHERE id = ? LIMIT 1`
	case "roots":
		if scope == "" {
			return false, fmt.Errorf("store: roots scope required")
		}
		q = `SELECT 1 FROM roots WHERE id = ? AND session_id = ? LIMIT 1`
		args = append(args, scope)
	case "iterations":
		if scope == "" {
			return false, fmt.Errorf("store: iterations scope required")
		}
		q = `SELECT 1 FROM iterations WHERE id = ? AND session_id = ? LIMIT 1`
		args = append(args, scope)
	case "nodes":
		if scope == "" {
			return false, fmt.Errorf("store: nodes scope required")
		}
		q = `SELECT 1 FROM nodes WHERE id = ? AND iteration_id = ? LIMIT 1`
		args = append(args, scope)
	default:
		return false, fmt.Errorf("store: unknown slug table %q", table)
	}
	var one int
	err := s.db.QueryRow(q, args...).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
