// Package core defines speechflow's pure domain types. These are the
// canonical response shapes for the HTTP API and the source for tygo's
// TypeScript generation. All JSON keys are snake_case; all timestamps
// are RFC3339 UTC and are mapped to string in TypeScript.
package core

import "time"

// NodeKind enumerates the three kinds of in-iteration nodes.
type NodeKind string

const (
	// NodeKindRootRef records "I touched root X at time T" in an iteration.
	NodeKindRootRef NodeKind = "root_ref"
	// NodeKindConcept is an idea or claim introduced during an iteration.
	NodeKindConcept NodeKind = "concept"
	// NodeKindCuriosity is an open question or branch the agent (or user) noticed.
	NodeKindCuriosity NodeKind = "curiosity"
	// NodeKindTakeaway is a leaf-of-chain synthesis: what the agent
	// concludes the listener actually walked away with, vs. the declared
	// root they were aiming to land. Always optionally pinned to a root_id.
	NodeKindTakeaway NodeKind = "takeaway"
)

// EdgeKind enumerates the three kinds of directed relationships between nodes.
type EdgeKind string

const (
	// EdgeBranchesFrom: child concept/curiosity developed from the parent.
	EdgeBranchesFrom EdgeKind = "branches_from"
	// EdgeReferences: the from-node leaned on the to-node without being a child of it.
	EdgeReferences EdgeKind = "references"
	// EdgeReturnsTo: the speaker explicitly looped back to an earlier idea.
	EdgeReturnsTo EdgeKind = "returns_to"
)

// NodeSource describes who created a node.
type NodeSource string

const (
	// SourceAgent indicates the node was created by an LLM agent.
	SourceAgent NodeSource = "agent"
	// SourceUser indicates the node was created directly by the user.
	SourceUser NodeSource = "user"
)

// Session is a speech / presentation the user is working on.
type Session struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Description       *string   `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	IterationCount    int       `json:"iteration_count"`
	LastActivityAt    time.Time `json:"last_activity_at"`
	LatestCoveragePct float64   `json:"latest_coverage_pct"`
}

// Root is a topic the user intends to cover in a session. Roots are
// session-scoped; they apply across every iteration of that session.
type Root struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// Iteration is one rehearsal / pass / dictation run of a session.
type Iteration struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"session_id"`
	Title       string     `json:"title"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
	NodeCount   int        `json:"node_count"`
	CoveragePct float64    `json:"coverage_pct"`
}

// Node is a single point in an iteration graph: a root_ref, concept, or curiosity.
type Node struct {
	ID                string     `json:"id"`
	IterationID       string     `json:"iteration_id"`
	Kind              NodeKind   `json:"kind"`
	Title             string     `json:"title"`
	Quote             *string    `json:"quote"`
	TranscriptStart   *int       `json:"transcript_start"`
	TranscriptEnd     *int       `json:"transcript_end"`
	RootID            *string    `json:"root_id"`
	ResolvedByNodeID  *string    `json:"resolved_by_node_id"`
	Tags              []string   `json:"tags"`
	Source            NodeSource `json:"source"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Edge is a directed relationship between two nodes within an iteration.
type Edge struct {
	ID          string    `json:"id"`
	IterationID string    `json:"iteration_id"`
	FromNode    string    `json:"from_node"`
	ToNode      string    `json:"to_node"`
	Kind        EdgeKind  `json:"kind"`
	CreatedAt   time.Time `json:"created_at"`
}

// Graph is the full node+edge view of one iteration, returned from
// GET /api/v1/iterations/:id/graph and polled by the UI.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// CoverageRow describes whether a single root was covered in an iteration,
// plus the supporting nodes and the timestamp of the earliest touch.
type CoverageRow struct {
	RootID            string    `json:"root_id"`
	RootTitle         string    `json:"root_title"`
	Covered           bool      `json:"covered"`
	SupportingNodeIDs []string  `json:"supporting_node_ids"`
	FirstTouchedAt    *time.Time `json:"first_touched_at"`
}

// CoverageMatrix is the session-level matrix view (iterations x roots).
type CoverageMatrix struct {
	SessionID  string                 `json:"session_id"`
	Roots      []Root                 `json:"roots"`
	Iterations []CoverageMatrixRow    `json:"iterations"`
}

// CoverageMatrixRow is one iteration's coverage across the session's roots.
type CoverageMatrixRow struct {
	IterationID    string        `json:"iteration_id"`
	IterationTitle string        `json:"iteration_title"`
	StartedAt      time.Time     `json:"started_at"`
	Rows           []CoverageRow `json:"rows"`
}

// TimelineEventKind enumerates the events emitted on the iteration playback feed.
type TimelineEventKind string

const (
	// TimelineNodeAdded marks creation of a node in the iteration.
	TimelineNodeAdded TimelineEventKind = "node_added"
	// TimelineEdgeAdded marks creation of an edge in the iteration.
	TimelineEdgeAdded TimelineEventKind = "edge_added"
	// TimelineCuriosityResolved marks a curiosity being resolved by a later node.
	TimelineCuriosityResolved TimelineEventKind = "curiosity_resolved"
	// TimelineRootTouched marks a root_ref being recorded.
	TimelineRootTouched TimelineEventKind = "root_touched"
	// TimelineTagAdded marks a tag being attached to a node.
	TimelineTagAdded TimelineEventKind = "tag_added"
)

// TimelineEvent is one event in the ordered playback stream for an iteration.
type TimelineEvent struct {
	Ts      time.Time              `json:"ts"`
	Kind    TimelineEventKind      `json:"kind"`
	NodeID  *string                `json:"node_id,omitempty"`
	EdgeID  *string                `json:"edge_id,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// TranscriptSpan is a (node_id, start, end) pointer into an iteration's
// transcript text, used by GET /api/v1/iterations/:id/transcript.
type TranscriptSpan struct {
	NodeID string `json:"node_id"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
}

// Transcript is the iteration transcript view returned by the HTTP API.
type Transcript struct {
	Text  string           `json:"text"`
	Spans []TranscriptSpan `json:"spans"`
}

// SessionDetail is the response shape for GET /api/v1/sessions/:id. The
// embedded Session is marshalled inline so the JSON object is flat
// (id, title, ..., roots, iterations), matching the UI's type contract.
type SessionDetail struct {
	Session
	Roots      []Root      `json:"roots"`
	Iterations []Iteration `json:"iterations"`
}

// IterationDetail is the response shape for GET /api/v1/iterations/:id.
// The embedded Iteration is marshalled inline (id, title, ..., roots).
type IterationDetail struct {
	Iteration
	Roots []Root `json:"roots"`
}
