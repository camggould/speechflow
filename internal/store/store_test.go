package store

import (
	"path/filepath"
	"testing"

	"github.com/camggould/speechflow/internal/core"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "speechflow.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSessionRoundTrip(t *testing.T) {
	s := newTestStore(t)
	desc := "Q4 strategy review"
	sess, err := s.CreateSession("q4-review", "Q4 Review", &desc)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.ID != "q4-review" || sess.Title != "Q4 Review" {
		t.Errorf("unexpected session: %+v", sess)
	}
	if sess.Description == nil || *sess.Description != desc {
		t.Errorf("description mismatch")
	}

	got, err := s.GetSession("q4-review")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("get returned %q", got.ID)
	}

	all, err := s.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("list returned %d sessions", len(all))
	}

	if err := s.DeleteSession("q4-review"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetSession("q4-review"); err == nil {
		t.Errorf("expected not-found after delete")
	}
}

func TestCascadingDelete(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateSession("a", "A", nil); err != nil {
		t.Fatalf("session: %v", err)
	}
	if _, err := s.CreateRoot("pricing", "a", "Pricing"); err != nil {
		t.Fatalf("root: %v", err)
	}
	if _, err := s.CreateIteration("iter-1", "a", "First"); err != nil {
		t.Fatalf("iter: %v", err)
	}
	rootID := "pricing"
	if _, err := s.CreateNode(NodeInput{
		ID: "node-1", IterationID: "iter-1", Kind: core.NodeKindRootRef,
		Title: "touch", RootID: &rootID,
	}); err != nil {
		t.Fatalf("node: %v", err)
	}
	if err := s.DeleteSession("a"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetIteration("iter-1"); err == nil {
		t.Errorf("expected iteration cascaded")
	}
	if _, err := s.GetNode("node-1"); err == nil {
		t.Errorf("expected node cascaded")
	}
}

func TestNodeAndEdgeRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateSession("a", "A", nil); err != nil {
		t.Fatalf("session: %v", err)
	}
	if _, err := s.CreateIteration("iter-1", "a", "First"); err != nil {
		t.Fatalf("iter: %v", err)
	}
	quote := "this is the quote"
	concept, err := s.CreateNode(NodeInput{
		ID: "c1", IterationID: "iter-1", Kind: core.NodeKindConcept,
		Title: "Pricing matters", Quote: &quote, Tags: []string{"key"},
	})
	if err != nil {
		t.Fatalf("concept: %v", err)
	}
	if len(concept.Tags) != 1 || concept.Tags[0] != "key" {
		t.Errorf("tags: %+v", concept.Tags)
	}
	cur, err := s.CreateNode(NodeInput{
		ID: "q1", IterationID: "iter-1", Kind: core.NodeKindCuriosity,
		Title: "How do we price?",
	})
	if err != nil {
		t.Fatalf("curiosity: %v", err)
	}

	edge, err := s.CreateEdge("e1", "iter-1", cur.ID, concept.ID, core.EdgeBranchesFrom)
	if err != nil {
		t.Fatalf("edge: %v", err)
	}
	if edge.Kind != core.EdgeBranchesFrom {
		t.Errorf("edge kind: %s", edge.Kind)
	}

	// Resolve curiosity.
	resolved, err := s.ResolveCuriosity(cur.ID, concept.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.ResolvedByNodeID == nil || *resolved.ResolvedByNodeID != concept.ID {
		t.Errorf("not resolved")
	}

	// Constraint: resolving a non-curiosity must fail.
	if _, err := s.ResolveCuriosity(concept.ID, cur.ID); err == nil {
		t.Errorf("expected constraint error resolving concept")
	}

	// Tag add/remove.
	tagged, err := s.AddTags(cur.ID, []string{"tangent", "evidence"})
	if err != nil {
		t.Fatalf("add tags: %v", err)
	}
	if len(tagged.Tags) != 2 {
		t.Errorf("expected 2 tags, got %+v", tagged.Tags)
	}
	untagged, err := s.RemoveTag(cur.ID, "tangent")
	if err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	if len(untagged.Tags) != 1 || untagged.Tags[0] != "evidence" {
		t.Errorf("tags after remove: %+v", untagged.Tags)
	}
}

func TestRootRefConstraints(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateSession("a", "A", nil); err != nil {
		t.Fatalf("session: %v", err)
	}
	if _, err := s.CreateIteration("iter-1", "a", "First"); err != nil {
		t.Fatalf("iter: %v", err)
	}
	if _, err := s.CreateRoot("pricing", "a", "Pricing"); err != nil {
		t.Fatalf("root: %v", err)
	}

	if _, err := s.CreateNode(NodeInput{
		ID: "rr1", IterationID: "iter-1", Kind: core.NodeKindRootRef, Title: "touch",
	}); err == nil {
		t.Errorf("expected error: root_ref without root_id")
	}

	rid := "pricing"
	if _, err := s.CreateNode(NodeInput{
		ID: "c1", IterationID: "iter-1", Kind: core.NodeKindConcept, Title: "x", RootID: &rid,
	}); err == nil {
		t.Errorf("expected error: concept with root_id")
	}
}

func TestTranscriptAppendAndSet(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateSession("a", "A", nil); err != nil {
		t.Fatalf("session: %v", err)
	}
	if _, err := s.CreateIteration("iter-1", "a", "First"); err != nil {
		t.Fatalf("iter: %v", err)
	}
	if _, err := s.AppendTranscript("iter-1", "hello"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := s.AppendTranscript("iter-1", "world"); err != nil {
		t.Fatalf("append: %v", err)
	}
	tr, err := s.GetTranscript("iter-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if tr.Text != "hello world" {
		t.Errorf("got transcript %q", tr.Text)
	}
	if _, err := s.SetTranscript("iter-1", "rewritten"); err != nil {
		t.Fatalf("set: %v", err)
	}
	tr, _ = s.GetTranscript("iter-1")
	if tr.Text != "rewritten" {
		t.Errorf("got %q after set", tr.Text)
	}
}

func TestSlugExists(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateSession("a", "A", nil); err != nil {
		t.Fatalf("session: %v", err)
	}
	got, err := s.SlugExists("sessions", "", "a")
	if err != nil || !got {
		t.Errorf("expected exists, got %v err=%v", got, err)
	}
	got, err = s.SlugExists("sessions", "", "nope")
	if err != nil || got {
		t.Errorf("expected not-exists, got %v err=%v", got, err)
	}
}
