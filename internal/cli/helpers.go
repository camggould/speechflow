// Shared helpers used across CLI command files: state lookups, error
// translation, decoration with computed fields.
package cli

import (
	"errors"

	"github.com/camggould/speechflow/internal/core"
	"github.com/camggould/speechflow/internal/coverage"
	"github.com/camggould/speechflow/internal/state"
	"github.com/camggould/speechflow/internal/store"
)

// translateStoreErr maps store errors to ExitErrors with the appropriate code.
func translateStoreErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return Exit(ExitNotFound, "%v", err)
	}
	if errors.Is(err, store.ErrConstraint) {
		return Exit(ExitConstraint, "%v", err)
	}
	return Exit(ExitGeneric, "%v", err)
}

// activeSession returns the slug of the currently active session, or an
// ExitError with code 4 (constraint) if none is set.
func activeSession() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", Exit(ExitGeneric, "%v", err)
	}
	st, err := state.Load(dir)
	if err != nil {
		return "", Exit(ExitGeneric, "load state: %v", err)
	}
	if st.ActiveSession == "" {
		return "", Exit(ExitConstraint, "no active session; run `speechflow session use <slug>`")
	}
	return st.ActiveSession, nil
}

// activeIteration returns the slug of the currently active iteration.
func activeIteration() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", Exit(ExitGeneric, "%v", err)
	}
	st, err := state.Load(dir)
	if err != nil {
		return "", Exit(ExitGeneric, "load state: %v", err)
	}
	if st.ActiveIteration == "" {
		return "", Exit(ExitConstraint, "no active iteration; run `speechflow iteration use <slug>` or start a new one")
	}
	return st.ActiveIteration, nil
}

// decorateSession fills latest_coverage_pct using the most recent iteration.
func decorateSession(s *store.Store, sess *core.Session) error {
	iters, err := s.ListIterations(sess.ID)
	if err != nil {
		return err
	}
	if len(iters) == 0 {
		return nil
	}
	rows, err := coverage.Compute(s, iters[0].ID)
	if err != nil {
		return err
	}
	sess.LatestCoveragePct = coverage.Percent(rows)
	return nil
}

// decorateIteration fills coverage_pct.
func decorateIteration(s *store.Store, it *core.Iteration) error {
	rows, err := coverage.Compute(s, it.ID)
	if err != nil {
		return err
	}
	it.CoveragePct = coverage.Percent(rows)
	return nil
}

