package coverage

import (
	"testing"
	"time"

	"github.com/camggould/speechflow/internal/core"
)

func ptr(s string) *string { return &s }

func TestComputeFromData_DirectTouch(t *testing.T) {
	now := time.Now().UTC()
	it := &core.Iteration{ID: "iter", SessionID: "s", StartedAt: now}
	roots := []core.Root{
		{ID: "pricing", SessionID: "s", Title: "Pricing", CreatedAt: now.Add(-time.Hour)},
		{ID: "roadmap", SessionID: "s", Title: "Roadmap", CreatedAt: now.Add(-time.Hour)},
	}
	nodes := []core.Node{
		{ID: "n1", IterationID: "iter", Kind: core.NodeKindRootRef, RootID: ptr("pricing"), CreatedAt: now},
	}
	rows := computeFromData(it, roots, nodes, nil)
	if len(rows) != 2 {
		t.Fatalf("rows len: %d", len(rows))
	}
	var pricing, roadmap core.CoverageRow
	for _, r := range rows {
		if r.RootID == "pricing" {
			pricing = r
		}
		if r.RootID == "roadmap" {
			roadmap = r
		}
	}
	if !pricing.Covered {
		t.Errorf("pricing should be covered")
	}
	if len(pricing.SupportingNodeIDs) != 1 || pricing.SupportingNodeIDs[0] != "n1" {
		t.Errorf("pricing supporting: %+v", pricing.SupportingNodeIDs)
	}
	if pricing.FirstTouchedAt == nil {
		t.Errorf("expected first_touched_at set")
	}
	if roadmap.Covered {
		t.Errorf("roadmap should not be covered")
	}
}

func TestComputeFromData_TransitiveReachability(t *testing.T) {
	now := time.Now().UTC()
	it := &core.Iteration{ID: "iter", SessionID: "s", StartedAt: now}
	roots := []core.Root{
		{ID: "pricing", SessionID: "s", Title: "Pricing", CreatedAt: now.Add(-time.Hour)},
	}
	// chain: c2 -> c1 -> rr (rr is the root_ref).
	// computeFromData walks edges in reverse from rr; c1 supports rr, c2 supports c1 → c2 supports pricing.
	nodes := []core.Node{
		{ID: "rr", IterationID: "iter", Kind: core.NodeKindRootRef, RootID: ptr("pricing"), CreatedAt: now.Add(2 * time.Second)},
		{ID: "c1", IterationID: "iter", Kind: core.NodeKindConcept, CreatedAt: now.Add(time.Second)},
		{ID: "c2", IterationID: "iter", Kind: core.NodeKindConcept, CreatedAt: now},
		{ID: "orphan", IterationID: "iter", Kind: core.NodeKindConcept, CreatedAt: now},
	}
	edges := []core.Edge{
		{ID: "e1", IterationID: "iter", FromNode: "c1", ToNode: "rr", Kind: core.EdgeBranchesFrom, CreatedAt: now},
		{ID: "e2", IterationID: "iter", FromNode: "c2", ToNode: "c1", Kind: core.EdgeBranchesFrom, CreatedAt: now},
	}
	rows := computeFromData(it, roots, nodes, edges)
	if len(rows) != 1 || !rows[0].Covered {
		t.Fatalf("expected pricing covered, got %+v", rows)
	}
	got := rows[0].SupportingNodeIDs
	// Expect rr, c1, c2 (in creation order). orphan must be absent.
	want := map[string]bool{"rr": true, "c1": true, "c2": true}
	if len(got) != 3 {
		t.Errorf("supporting count = %d, want 3, got %v", len(got), got)
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected supporter %q", id)
		}
		if id == "orphan" {
			t.Errorf("orphan should not appear")
		}
	}
}

func TestComputeFromData_RootCreatedAfterCutoff(t *testing.T) {
	now := time.Now().UTC()
	end := now
	it := &core.Iteration{ID: "iter", SessionID: "s", StartedAt: now.Add(-time.Hour), EndedAt: &end}
	roots := []core.Root{
		{ID: "old", SessionID: "s", Title: "Old", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "future", SessionID: "s", Title: "Future", CreatedAt: now.Add(time.Hour)},
	}
	rows := computeFromData(it, roots, nil, nil)
	if len(rows) != 1 || rows[0].RootID != "old" {
		t.Errorf("expected only old root, got %+v", rows)
	}
}

func TestPercent(t *testing.T) {
	cases := []struct {
		rows []core.CoverageRow
		want float64
	}{
		{nil, 0},
		{[]core.CoverageRow{{Covered: true}, {Covered: true}}, 1.0},
		{[]core.CoverageRow{{Covered: true}, {Covered: false}}, 0.5},
		{[]core.CoverageRow{{Covered: false}, {Covered: false}}, 0.0},
	}
	for _, c := range cases {
		got := Percent(c.rows)
		if got != c.want {
			t.Errorf("Percent(%+v) = %v, want %v", c.rows, got, c.want)
		}
	}
}
