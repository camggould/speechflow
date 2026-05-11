// Hand-written bootstrap of speechflow API types.
// tygo will regenerate this from internal/core when the Go backend lands.
// Shapes mirror README "Response shapes (canonical)" exactly.

export type Session = {
  id: string;
  title: string;
  description: string | null;
  created_at: string; // RFC3339
  updated_at: string;
  iteration_count: number;
  last_activity_at: string;
  latest_coverage_pct: number; // 0..1; null if no iterations
};

export type Root = {
  id: string;
  session_id: string;
  title: string;
  created_at: string;
};

export type Iteration = {
  id: string;
  session_id: string;
  title: string;
  started_at: string;
  ended_at: string | null;
  node_count: number;
  coverage_pct: number;
};

export type NodeKind = "root_ref" | "concept" | "curiosity" | "takeaway";

export type Node = {
  id: string;
  iteration_id: string;
  kind: NodeKind;
  title: string;
  quote: string | null;
  transcript_start: number | null;
  transcript_end: number | null;
  root_id: string | null;
  resolved_by_node_id: string | null;
  tags: string[];
  source: "agent" | "user";
  created_at: string;
};

export type EdgeKind = "branches_from" | "references" | "returns_to";

export type Edge = {
  id: string;
  iteration_id: string;
  from_node: string;
  to_node: string;
  kind: EdgeKind;
  created_at: string;
};

export type Graph = { nodes: Node[]; edges: Edge[] };

export type CoverageRow = {
  root_id: string;
  root_title: string;
  covered: boolean;
  supporting_node_ids: string[];
  first_touched_at: string | null;
};

export type CoverageMatrix = {
  session_id: string;
  roots: Root[];
  iterations: Array<{
    iteration_id: string;
    iteration_title: string;
    started_at: string;
    rows: CoverageRow[];
  }>;
};

export type TimelineEvent = {
  ts: string;
  kind:
    | "node_added"
    | "edge_added"
    | "curiosity_resolved"
    | "root_touched"
    | "tag_added";
  node_id?: string;
  edge_id?: string;
  payload?: Record<string, unknown>;
};

// Composite shapes returned by endpoints that augment a base type with
// embedded relations. Not part of the canonical README list, but the only
// way to type the HTTP responses for /sessions/:id and /iterations/:id.
export type SessionDetail = Session & {
  roots: Root[];
  iterations: Iteration[];
};

export type IterationDetail = Iteration & {
  roots: Root[];
};

export type Transcript = {
  text: string;
  spans: Array<{ node_id: string; start: number; end: number }>;
};

export type ApiError = {
  error: string; // machine-code
  message: string; // human-readable
};
