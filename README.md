# speechflow

A CLI tool (with an embedded web UI) that lets an LLM agent structure a spoken or written conversation as a **concept graph** — ideas, concepts, and curiosities branching off one another over time — so you can review how well you covered the topics you intended to cover.

The CLI is deterministic. The LLM does the judgment (what is a concept, when is a curiosity resolved, what quote to attach). `speechflow` just stores it and renders it.

---

## Install

One-liner — detects OS/arch, downloads the matching pre-built binary from the latest GitHub Release, verifies the SHA-256 checksum, and installs to `/usr/local/bin` (falling back to `~/.local/bin` if that's not writable):

```sh
curl -fsSL https://raw.githubusercontent.com/camggould/speechflow/main/install.sh | sh
```

Optional flags (pass via `sh -s --`):

```sh
curl -fsSL .../install.sh | sh -s -- --version v0.1.0       # pin a specific release
curl -fsSL .../install.sh | sh -s -- --prefix ~/.local/bin  # override install dir
```

After install:

```sh
speechflow init               # one-time: create ~/.speechflow/ and run migrations
speechflow serve --open       # start the local UI + JSON API at 127.0.0.1:7777
```

The binary ships everything it needs — UI is embedded, SQLite is statically linked, no runtime deps beyond libc.

---

## Mental model

```
session       ─ a speech / presentation you're working on
 ├─ roots     ─ the topics you intend to cover (declared up front, evolvable)
 ├─ iteration ─ one rehearsal / pass / dictation run
 │   ├─ transcript    ─ full text of what you said
 │   ├─ nodes         ─ root_ref | concept | curiosity
 │   ├─ edges         ─ branches_from | references | returns_to
 │   └─ coverage      ─ structural check: did this iteration touch each root?
 ├─ iteration ...
 └─ iteration ...
```

You practice a speech across many **iterations** of the same **session**. Each iteration gets its own graph and transcript, judged against the session's shared **roots**. The UI lets you flip between iterations, scrub a playback timeline, and see coverage across all rehearsals.

### Node kinds

| Kind         | Meaning                                                                                                | Notes |
|--------------|--------------------------------------------------------------------------------------------------------|-------|
| `root_ref`   | An in-iteration node that records "I touched root X at time T."                                        | One per (iteration, root_touch); created with `node touch-root`. |
| `concept`    | An idea or claim you introduced.                                                                       | May carry a quote and tags. |
| `curiosity`  | An open question or branch the agent (or you) noticed. Branches off a concept; can reference others.   | Can be `resolved_by` a later node. |
| `takeaway`   | Leaf-of-chain synthesis: what the listener actually walked away with.                                  | Branches from the last concept in a chain. May optionally `--root` to the root it was aiming at, so the UI can render intended vs. actual side-by-side. |

### Edge kinds

| Kind            | Direction        | Meaning                                                       |
|-----------------|------------------|---------------------------------------------------------------|
| `branches_from` | child → parent   | The child concept/curiosity developed from the parent.        |
| `references`    | from → to        | The from-node leaned on the to-node without being a child of it. |
| `returns_to`    | from → to        | The speaker explicitly looped back to an earlier idea.        |
| `supports`      | evidence → claim | The from-node provides evidence, an example, or an analogy underpinning the to-node's claim. Surfaces structurally as "this claim has support." |
| `contrasts`     | from → to        | The from-node pushes against the to-node — steel-manning, "but X", or self-contradiction. |

### Tags

Tags are the agent's primary instrument for annotating speech *quality*; the **Speech Health** panel groups them. Canonical strengths: `key`, `hook`, `signpost`, `exposition`, `analogy`, `example`, `callback`, `definition`, `pivot`, `closing`. Canonical weaknesses: `tangent`, `unsupported-claim`, `dropped-thread`, `filler`, `abrupt-transition`, `contradiction`. Anything else is rendered as a chip but doesn't appear in the dashboard. Full when-to-apply guidance is in `AGENTS.md` §4.

---

## Storage

- **Location:** `~/.speechflow/speechflow.db` (per-user, SQLite).
- **State file:** `~/.speechflow/state.json` holds the active session and active iteration so the agent doesn't pass IDs on every call.
- **IDs:** two schemes coexist.
  - **Slugs** (sessions, roots, nodes): derived from titles (`pricing-strategy`; collisions get `-2`, `-3`, …).
  - **Random tokens** (iterations): `it_<16-hex>` — opaque, not derived from the title. Iterations are globally unique in the DB and titles like `"Rehearsal 1"` recur freely across sessions; random IDs side-step what would otherwise be cross-session slug collisions.
  Read every ID from the JSON response of the call that creates it. Never construct one client-side.
- **Deletes:** cascading. Deleting an iteration drops its nodes/edges/transcript. Deleting a session drops all its iterations + roots.

### Schema (SQLite)

```sql
CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,           -- slug
    title        TEXT NOT NULL,
    description  TEXT,
    created_at   TEXT NOT NULL,              -- RFC3339
    updated_at   TEXT NOT NULL
);

CREATE TABLE roots (
    id          TEXT PRIMARY KEY,            -- slug, unique within session
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE TABLE iterations (
    id          TEXT PRIMARY KEY,            -- slug, unique within session
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    transcript  TEXT NOT NULL DEFAULT '',
    started_at  TEXT NOT NULL,
    ended_at    TEXT                          -- null while active
);

CREATE TABLE nodes (
    id                   TEXT PRIMARY KEY,   -- slug, unique within iteration
    iteration_id         TEXT NOT NULL REFERENCES iterations(id) ON DELETE CASCADE,
    kind                 TEXT NOT NULL CHECK (kind IN ('root_ref','concept','curiosity','takeaway')),
    title                TEXT NOT NULL,
    quote                TEXT,
    transcript_start     INTEGER,             -- char offset into iteration.transcript
    transcript_end       INTEGER,
    root_id              TEXT REFERENCES roots(id) ON DELETE SET NULL,   -- root_ref (required) or takeaway (optional, "intended root")
    resolved_by_node_id  TEXT REFERENCES nodes(id) ON DELETE SET NULL,   -- only for kind=curiosity
    source               TEXT NOT NULL DEFAULT 'agent',                  -- 'agent' | 'user'
    created_at           TEXT NOT NULL
);

CREATE TABLE node_tags (
    node_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY (node_id, tag)
);

CREATE TABLE edges (
    id           TEXT PRIMARY KEY,
    iteration_id TEXT NOT NULL REFERENCES iterations(id) ON DELETE CASCADE,
    from_node    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node      TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('branches_from','references','returns_to','supports','contrasts')),
    created_at   TEXT NOT NULL
);

CREATE INDEX idx_iterations_session    ON iterations(session_id);
CREATE INDEX idx_roots_session         ON roots(session_id);
CREATE INDEX idx_nodes_iteration       ON nodes(iteration_id);
CREATE INDEX idx_edges_iteration       ON edges(iteration_id);
CREATE INDEX idx_nodes_resolved_by     ON nodes(resolved_by_node_id);
```

---

## CLI

All write commands return JSON of the created/updated object on stdout (so the agent can read the assigned slug). Use `--pretty` to render as a human-readable table/tree.

```
speechflow init                                          # one-time: ensure ~/.speechflow/ exists; run migrations
speechflow version

# Sessions ----------------------------------------------------------------
speechflow session new --title "Q4 review" [--description "..."]
speechflow session list                                  # all sessions
speechflow session show [<session-slug>]                 # active session if omitted
speechflow session use <session-slug>                    # set active session
speechflow session delete <session-slug>                 # cascades to iterations

# Roots (session-scoped) ---------------------------------------------------
speechflow root add "Pricing" "Roadmap" "Hiring"         # one or many at once
speechflow root list [<session-slug>]
speechflow root delete <root-slug>

# Iterations (per session) -------------------------------------------------
speechflow iteration start [--title "Rehearsal 3"]       # uses active session; sets active iteration
speechflow iteration end [<iteration-slug>]              # uses active iteration if omitted
speechflow iteration list [<session-slug>]
speechflow iteration use <iteration-slug>                # set active iteration
speechflow iteration delete <iteration-slug>

# Transcript ---------------------------------------------------------------
speechflow transcript append "..."                       # appends to active iteration
speechflow transcript set --file ./script.txt            # replace whole transcript
speechflow transcript show

# Nodes --------------------------------------------------------------------
speechflow node add concept    --title "..." [--quote "..."] [--tag key] [--span 1240,1380] [--from <slug>]
speechflow node add curiosity  --title "..." --from <slug> [--refs <slug,slug>] [--quote "..."] [--span 1240,1380]
speechflow node add takeaway   --title "..." --from <slug> [--root <root-slug>] [--quote "..."]
speechflow node touch-root     <root-slug> [--span 1240,1380]   # records a root_ref node
speechflow node resolve        <curiosity-slug> --by <node-slug>
speechflow node tag            <node-slug> <tag> [<tag>...]
speechflow node untag          <node-slug> <tag>
speechflow node delete         <node-slug>

# Edges --------------------------------------------------------------------
speechflow edge add <from-slug> <to-slug> --kind references|branches_from|returns_to
speechflow edge delete <edge-id>

# Reporting ----------------------------------------------------------------
speechflow coverage                                      # active iteration vs session roots
speechflow coverage --session                            # matrix across all iterations of active session
speechflow timeline                                      # ordered events for active iteration (playback feed)
speechflow export json [--iteration <slug>]              # full dump
speechflow export graphml --iteration <slug>

# UI -----------------------------------------------------------------------
speechflow serve [--port 7777] [--open]                  # serve embedded UI + JSON API
```

### Global flags

| Flag         | Default | Notes                                                                  |
|--------------|---------|------------------------------------------------------------------------|
| `--pretty`   | off     | Human-readable output instead of JSON.                                 |
| `--db PATH`  | `~/.speechflow/speechflow.db` | Override DB location (mainly for tests).         |

### Exit codes

- `0` success
- `1` generic error
- `2` usage error (bad flags / args)
- `3` not found (slug doesn't exist)
- `4` constraint violation (e.g. resolving a non-curiosity, no active session set)

---

## HTTP API

Mounted at `/api/v1`, served by `speechflow serve`. Bound to `127.0.0.1` only.

```
GET    /api/v1/health
GET    /api/v1/sessions                                   # dashboard list (incl. last_activity_at, iteration_count, latest_coverage_pct)
GET    /api/v1/sessions/:id                               # session + roots + iterations (no graph)
GET    /api/v1/sessions/:id/coverage                      # matrix: iterations × roots
DELETE /api/v1/sessions/:id

GET    /api/v1/iterations/:id                             # iteration meta + roots-at-the-time
GET    /api/v1/iterations/:id/graph                       # { nodes, edges, tags } — UI polls this
GET    /api/v1/iterations/:id/timeline                    # ordered event stream for playback
GET    /api/v1/iterations/:id/transcript                  # { text, spans: [{node_id,start,end}] }
GET    /api/v1/iterations/:id/coverage                    # per-root: covered, supporting_node_ids, first_touched_at
DELETE /api/v1/iterations/:id
```

All responses are JSON. Errors follow `{ "error": "<machine-code>", "message": "<human-readable>" }`. v1 is read-only over HTTP — mutations are CLI-only (keeps the contract simple for the agent).

### Response shapes (canonical)

```ts
type Session = {
  id: string;
  title: string;
  description: string | null;
  created_at: string;          // RFC3339
  updated_at: string;
  iteration_count: number;
  last_activity_at: string;
  latest_coverage_pct: number; // 0..1; null if no iterations
};

type Root = { id: string; session_id: string; title: string; created_at: string };

type Iteration = {
  id: string;
  session_id: string;
  title: string;
  started_at: string;
  ended_at: string | null;
  node_count: number;
  coverage_pct: number;
};

type Node = {
  id: string;
  iteration_id: string;
  kind: "root_ref" | "concept" | "curiosity" | "takeaway";
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

type Edge = {
  id: string;
  iteration_id: string;
  from_node: string;
  to_node: string;
  kind: "branches_from" | "references" | "returns_to" | "supports" | "contrasts";
  created_at: string;
};

type Graph = { nodes: Node[]; edges: Edge[] };

type CoverageRow = {
  root_id: string;
  root_title: string;
  covered: boolean;
  supporting_node_ids: string[];
  first_touched_at: string | null;
};

type CoverageMatrix = {
  session_id: string;
  roots: Root[];
  iterations: Array<{
    iteration_id: string;
    iteration_title: string;
    started_at: string;
    rows: CoverageRow[];
  }>;
};

type TimelineEvent = {
  ts: string;
  kind: "node_added" | "edge_added" | "curiosity_resolved" | "root_touched" | "tag_added";
  node_id?: string;
  edge_id?: string;
  payload?: Record<string, unknown>;
};
```

Generated automatically from Go via `tygo` into `ui/src/api/types.gen.ts`.

---

## UI

Built with **React 19 + HeroUI + @xyflow/react + @dagrejs/dagre + framer-motion + zustand + wouter + @tanstack/react-query**, compiled by Vite, embedded into the binary via `go:embed` (mirrors dtree's `internal/uifs`). Served at `/ui/` by `speechflow serve`.

### Routes

| Path                                  | View              |
|---------------------------------------|-------------------|
| `/`                                   | Dashboard         |
| `/sessions/:sessionId`                | Session detail    |
| `/sessions/:sessionId/iterations/:id` | Iteration view    |

### Dashboard (`/`)

- Grid of session cards: title, # iterations, latest coverage %, last-activity timestamp.
- Side panel: **Recent activity** — sessions sorted by `last_activity_at` desc, top 10.
- Click a card → session detail.

### Session detail (`/sessions/:sessionId`)

- **Left rail:** iterations sorted newest → oldest. Each row: title, started_at, coverage %, delete affordance (confirm modal).
- **Right pane:** empty-state copy until an iteration is clicked. Then the iteration view loads in-place (same URL pattern, replaces `:iterationId`).
- **Coverage matrix toggle** (above iterations rail): swaps the right pane for a matrix (rows = iterations, cols = roots, ✓/✗ cells); click a cell to focus the supporting node in the iteration view.

### Iteration view (`/sessions/:sessionId/iterations/:iterationId`)

- **Center:** React Flow graph, dagre auto-layout (relayout on each new node insertion). Node colors by kind: root_ref (gold), concept (blue), curiosity (purple). Border style by tag (`key` solid, `tangent` dashed). Resolved curiosities render dimmed with a dotted edge to the resolving node.
- **Top bar:**
  - **Transcript** button → modal with the full transcript; clicking a span highlights the corresponding node (and vice-versa).
  - **Live / Playback** toggle.
  - **Timeline scrubber** (visible in playback mode) with play/pause/speed (0.5×/1×/2×/4×). In playback mode, nodes/edges past the cursor render with fog-of-war (interpolated `opacity` + `filter: blur()`).
- **Right panel:** coverage panel listing declared roots with ✓/✗; click to focus a supporting node on the graph.
- **Live mode:** polls `/api/v1/iterations/:id/graph` every 1s while `ended_at` is null. Pauses polling when tab is hidden.

### Theme

Light + dark via `documentElement.classList.toggle("dark")`, matching dtree's pattern. HeroUI provider mounted at the root; `@tailwindcss/vite` plugin.

---

## Repo layout

```
speechflow/
├── cmd/
│   └── speechflow/
│       └── main.go                  # cobra entrypoint
├── internal/
│   ├── cli/                         # cobra command definitions, one file per command group
│   ├── core/                        # domain types (Session, Iteration, Node, Edge, Coverage); source for tygo
│   ├── store/                       # SQLite layer: migrations, queries, repository methods
│   ├── coverage/                    # structural coverage algorithm
│   ├── slug/                        # title → slug + collision suffixing
│   ├── state/                       # ~/.speechflow/state.json reader/writer (active session/iteration)
│   ├── server/                      # chi HTTP server + handlers; consumes core + store
│   └── uifs/
│       ├── dist/                    # populated by `make ui`; embedded via go:embed
│       └── uifs.go
├── ui/
│   ├── src/
│   │   ├── api/                     # client.ts, query.ts, types.gen.ts (tygo output)
│   │   ├── components/              # Layout, GraphCanvas, TranscriptModal, CoveragePanel, …
│   │   ├── views/                   # Dashboard, SessionView, IterationView
│   │   ├── store/                   # zustand stores (playback, theme, ui-state)
│   │   ├── styles/globals.css
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   └── routes.ts
│   ├── hero.ts
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── tygo.yaml                        # Go → TS type generation
├── install.sh                       # curl-pipe installer (multi-arch tarballs from GH Releases)
├── Makefile
├── go.mod / go.sum
├── AGENTS.md                        # agent usage contract — when to create what kind of node
├── SKILL.md                         # human-readable skill description for hooking the agent up
└── README.md                        # this file
```

---

## Coverage algorithm (structural)

For a given iteration:

1. Gather all session-scoped roots whose `created_at <= iteration.ended_at` (or `now()` if iteration is still active).
2. For each root, compute the set of nodes that "support" it:
   - any `root_ref` node where `root_id == root.id`, **OR**
   - any node `n` such that there is a directed path of edges (any kind) from `n` to a supporting `root_ref` node for that root.
3. A root is **covered** iff its supporting set is non-empty.
4. `first_touched_at` = min `created_at` across the supporting set.

Orphan detection (UI-only): concepts with no path to any `root_ref` are flagged as potential tangents. The LLM, not the CLI, decides whether to actually tag them `tangent`.

---

## Build & run

```bash
make setup          # go mod download + npm install + tygo
make ui             # compile UI into internal/uifs/dist/
make build          # build the speechflow binary (UI must be built first)
make dev            # ui + build + ./speechflow serve
make test           # go test ./...
```

UI dev with HMR:

```bash
make ui-dev         # in one terminal — Vite at :5173
./speechflow serve  # in another  — API at :7777; UI proxy lives in vite.config.ts
```

For pre-built binary installation, see [Install](#install) at the top of this README.

---

## Agent contract (TL;DR — full version in `AGENTS.md`)

While conversing with the user, the agent should:

1. **Once per session:** `speechflow session new` (if new topic) or `speechflow session use <slug>` (resuming).
2. **At start of each rehearsal:** `speechflow root add ...` (if the user declared topics they intend to cover), then `speechflow iteration start`.
3. **As the user speaks:**
   - `speechflow transcript append "..."` for each utterance (so spans line up).
   - When the user touches a declared root: `speechflow node touch-root <root-slug>`.
   - For each introduced idea: `speechflow node add concept --title "..." --quote "..." --span S,E [--tag key]`.
   - For each open question the agent notices: `speechflow node add curiosity --from <concept-slug> --title "..."`.
   - When a later concept resolves a curiosity: `speechflow node resolve <curiosity-slug> --by <concept-slug>`.
   - `speechflow edge add <from> <to> --kind ...` to connect non-parent relationships.
   - When a chain of concepts wraps up and the listener has a clear synthesis: `speechflow node add takeaway --from <leaf-concept> [--root <intended-root>] --title "..."`. One per coherent chain, not per concept.
4. **At end:** `speechflow iteration end`.
5. The agent **never** decides coverage — it just records data. `speechflow coverage` is run by the user or the UI.

All writes return JSON; the agent reads the returned `id` to chain subsequent calls.

---

## Conventions

- All timestamps are RFC3339 UTC.
- IDs in CLI args and API paths are slugs (sessions/roots/nodes) or random `it_<hex>` tokens (iterations). Both are opaque strings to consumers — always read them from the JSON response of the call that created them.
- The CLI is the **only** way to mutate state. HTTP is read-only.
- JSON output keys are `snake_case`. The UI maps to camelCase at the boundary.
- No backwards-compat shims pre-1.0 — schema migrations may be destructive.

---

## License

MIT
