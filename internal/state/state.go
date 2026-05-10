// Package state persists the user's active session and active iteration
// to ~/.speechflow/state.json so that CLI commands don't have to be
// passed those IDs on every call. The file format is plain JSON; absent
// or malformed files are treated as "no active selection".
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the basename of the state file inside the speechflow data dir.
const FileName = "state.json"

// State holds the persisted active-selection pointers.
type State struct {
	ActiveSession   string `json:"active_session,omitempty"`
	ActiveIteration string `json:"active_iteration,omitempty"`
}

// Path returns the absolute path to the state file inside dir.
func Path(dir string) string {
	return filepath.Join(dir, FileName)
}

// Load reads the state file from dir. A missing file is not an error;
// it returns a zero-value State.
func Load(dir string) (*State, error) {
	p := Path(dir)
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state: read %s: %w", p, err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("state: parse %s: %w", p, err)
	}
	return &s, nil
}

// Save writes s to dir/state.json, creating dir if missing. Writes are
// atomic via tmp+rename so concurrent readers never see a half-written file.
func Save(dir string, s *State) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state: mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}
	p := Path(dir)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("state: write tmp: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		return fmt.Errorf("state: rename: %w", err)
	}
	return nil
}

// SetActiveSession updates the active session pointer in dir.
// Setting iteration to "" also clears the active iteration, since
// iterations are session-scoped.
func SetActiveSession(dir, session string) error {
	s, err := Load(dir)
	if err != nil {
		return err
	}
	if s.ActiveSession != session {
		s.ActiveIteration = ""
	}
	s.ActiveSession = session
	return Save(dir, s)
}

// SetActiveIteration updates the active iteration pointer in dir.
func SetActiveIteration(dir, iteration string) error {
	s, err := Load(dir)
	if err != nil {
		return err
	}
	s.ActiveIteration = iteration
	return Save(dir, s)
}
