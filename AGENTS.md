# speechflow — Agent Contract

This is the operating contract for an LLM agent driving `speechflow` during a
spoken (or written) conversation with a user. It expands the TL;DR in
`README.md` into a complete, opinionated playbook.

If you only have time to read one thing: **you record, you do not judge.** You
turn the conversation into a graph of nodes and edges, faithfully attached to
the literal transcript. You do not compute coverage. You do not grade the
speech. You do not decide whether the user "did well." Those are downstream
questions answered by the user, the `coverage` command, or the UI.

---

## 1. The conversational loop

Your job is divided into three phases per session: **session start**,
**during the conversation**, and **session end**. Within "during the
conversation" you operate a tight per-utterance loop.

All write commands emit a single JSON object on stdout describing the
created/updated record. **Read it.** You will need its `id` (a slug) to chain
follow-up calls (`--from`, `--by`, `edge add`, `node tag`, etc.). If you call
a write command and discard its output, you have nothing to attach to.

### 1.1 Session start (once per topic)

1. Decide whether this is a new topic or a continuation.
   - New topic: `speechflow session new --title "<short title>" [--description "..."]`.
     Read the returned `id` — that slug is now the active session.
   - Resuming: `speechflow session list` to find it, then
     `speechflow session use <session-slug>` to make it active.
2. If the user named the topics they intend to cover (explicitly: "today I
   want to talk about pricing, roadmap, and hiring"), declare them as roots:
   `speechflow root add "Pricing" "Roadmap" "Hiring"`. You may add roots
   later if new ones surface, but earlier is better because coverage only
   considers roots whose `created_at <= iteration.ended_at`.
3. Start the iteration: `speechflow iteration start [--title "Rehearsal 3"]`.
   This sets the active iteration and is required before any node or
   transcript call. Iteration IDs are opaque random tokens of the form
   `it_<16-hex>` — **not** slugs derived from the title. Two iterations
   titled "Rehearsal 1" in different sessions both succeed and get
   different IDs. Read the returned `id` from the JSON response and store
   it; never construct it.

### 1.2 During the conversation (the per-utterance loop)

For each utterance the user produces, do this in order:

1. **Append the transcript first.**
   `speechflow transcript append "<exact text of what they said>"`.
   Always before adding a node that points at this utterance — spans are
   character offsets into the iteration's accumulated transcript, so the
   transcript needs to contain the relevant text before any node can
   reference it.
2. **Compute the span** for the utterance you just appended. The span is
   `[start, end)` character offsets into `iteration.transcript` — start is
   the length of the transcript *before* your append; end is the length
   *after*. (If you used `transcript set`, recompute against the new whole.)
3. **Recognise and record nodes** triggered by the utterance (see §2 for
   when to create each kind).
4. **Add edges** for any relationship that is not the parent edge already
   implied by `--from` (see §3).
5. **Resolve curiosities** that this utterance answers (see §5).
6. **Tag** newly created nodes that warrant it (see §4).

You do not need a node per sentence. Reach for a node when the user does
one of:
- introduces a substantive idea or claim (→ `concept`)
- visibly touches one of the declared roots (→ `root_ref` via `touch-root`)
- raises an open question, hedge, or "I should come back to this" (→
  `curiosity`)
- explicitly returns to an earlier point (→ a `returns_to` edge, often plus
  a fresh concept)

When in doubt, prefer fewer, higher-signal nodes over a thicket of
near-duplicates. You can always add a `references` edge from a new concept
to an existing one instead of re-creating it.

### 1.3 Session end

When the user signals they are done (or stops talking and the session is
being closed):

1. `speechflow iteration end` — closes the active iteration. After this,
   `ended_at` is set and coverage can be computed against it.
2. Do **not** call `speechflow coverage` automatically. That is the user's
   tool, not yours. (If the user explicitly asks "how did I do?" you can
   run it on their behalf and read it back.)

You may begin a fresh iteration in the same session at any time with
another `speechflow iteration start` — each rehearsal of the same topic
gets its own iteration, scored against the same roots.

---

## 2. When to create each node kind

There are exactly four node kinds. Pick the right one.

### `root_ref` — "I touched a declared root just now."

Create with `speechflow node touch-root <root-slug> [--span S,E]`. Use this
whenever the user materially engages with one of the session's declared
roots. The intent is structural, not topical: a `root_ref` is the
in-iteration anchor that lets the coverage algorithm say "yes, root X was
touched here."

- Add a `--span` if you can; coverage doesn't require it, but the UI uses
  spans to align nodes with the transcript.
- It is fine — expected, even — to create the same `root_ref` multiple
  times in one iteration if the user returns to that root. Each instance
  pins the moment.
- Do NOT use `root_ref` for "this concept is loosely related to a root."
  That relationship belongs in edges (a `concept` with an edge to a
  `root_ref`).

### `concept` — "An idea or claim the user introduced."

Create with
`speechflow node add concept --title "<short>" [--quote "<verbatim>"] [--span S,E] [--from <parent-slug>] [--tag key]`.

- `--title` is your own short paraphrase (≤ ~80 chars).
- `--quote` is the verbatim utterance fragment that made you create the
  node, if any. Quote sparingly — only the load-bearing words.
- `--from` is the parent concept (creates an implicit `branches_from`
  edge). Use it when the new concept genuinely developed from a previous
  one. Omit it for the first concept of an iteration or for a concept that
  is freestanding.
- Add `--tag key` if the user explicitly framed the idea as central
  ("the whole point is…", "the headline is…"). See §4.

Use `concept` for: claims, definitions, examples, evidence, framings,
pivots, summaries.

### `curiosity` — "An open question or thread to revisit."

Create with
`speechflow node add curiosity --from <concept-slug> --title "<short>" [--refs <slug,slug>] [--quote "..."] [--span S,E]`.

- `--from` is required: every curiosity branches from a concept (or another
  curiosity). Pick the most specific anchor — the concept that *raises*
  the question.
- `--refs` is for additional nodes the curiosity leans on without being a
  child of them. Equivalent to adding `references` edges after the fact.
- A curiosity is open by default. It becomes "resolved" only when you
  later call `speechflow node resolve` (see §5).

Use `curiosity` for: "I don't know yet", "we should check X", "this
contradicts Y, why?", explicit hedges, half-formed pivots the user did not
flesh out.

Do NOT use `curiosity` for rhetorical questions the user immediately
answers themselves — that's a `concept` (the answer), optionally tagged
`pivot`.

### `takeaway` — "What the listener actually walked away with."

Create with
`speechflow node add takeaway --from <parent-slug> --title "<synthesis>" [--root <root-slug>] [--quote "..."]`.

A takeaway is a **leaf-of-chain synthesis**. It is the agent's one-sentence
distillation of what the audience would, in fact, take from a chain of
concepts — irrespective of whether that matches the root the chain was
aimed at. Takeaways are how the user later answers: "did I land the point
I thought I was landing?"

- `--from` is required and should be the last substantive node in the
  chain — usually the most recent `concept` under the relevant
  `root_ref`. The takeaway hangs off the tip of the chain.
- `--root` optionally pins the takeaway to the declared root it was
  *trying* to land. The UI uses this for the "what you were going for"
  panel — it surfaces the comparison directly. Set it whenever the chain
  was structurally anchored to a root.
- `--quote` is unusual on a takeaway (it's synthesis, not transcript) but
  permitted when the user themselves voiced the takeaway verbatim.
- Do NOT create a takeaway after every concept. One per coherent chain is
  the rule. A chain is "coherent" when the user has moved on — explicitly
  pivoted to another topic, ended a section, or signalled they're done
  with that idea.
- Do NOT create a takeaway if you cannot articulate a synthesis crisper
  than the chain itself. A weak takeaway is worse than no takeaway.

Use `takeaway` for: chain-of-thought endings, "the point is…" moments the
user articulates, summary lines, and (most usefully) synthesis the agent
notices the user didn't quite say out loud but clearly implied.

---

## 3. When to create each edge kind

Most parent-child structure should come from the `--from` flag on
`node add` (which records a `branches_from` edge for you). Use explicit
`edge add` for non-parent links.

| Kind            | Use when…                                                                                                  |
|-----------------|------------------------------------------------------------------------------------------------------------|
| `branches_from` | Created automatically by `--from`. Only call `edge add ... --kind branches_from` to reparent or fix data. |
| `references`    | A node leans on another without descending from it. e.g. concept B cites concept A. Frequent, low-cost.    |
| `returns_to`    | The speaker explicitly loops back to an earlier idea ("as I was saying about pricing…"). High signal.      |

`speechflow edge add <from-slug> <to-slug> --kind references|branches_from|returns_to`.

Edges point in the direction stated in the table — `branches_from` is
child → parent; `references` and `returns_to` are from the new node to the
older one. Get this wrong and the coverage graph traversal still works,
but the UI's arrows look backwards.

`returns_to` is structurally what unlocks orphan-looking-but-actually-on-topic
detection: a `references`-rich graph with `returns_to` edges to `root_ref`
nodes will mark a digression as still on-topic for the right root.

---

## 4. Tags: `key`, `tangent`, and the rest

Tags are free-form, but a few have UI meaning:

- **`key`** — the user explicitly framed the node as central. Solid border
  in the UI. Reserve for moments the user signals "this is the point" or
  "this is the thesis."
- **`tangent`** — the agent (or user) recognised the node as a digression.
  Dashed border in the UI. Tag a concept `tangent` when it isn't traceable
  to any root *and* the user did not signal a `returns_to`.
- **`evidence`**, **`example`**, **`definition`**, **`pivot`** — render as
  chips. Use them when they fit; don't force.

Set tags at creation time with `--tag key` (repeatable) or after the fact
with `speechflow node tag <node-slug> <tag> [<tag>...]`.

Remove tags with `speechflow node untag <node-slug> <tag>`. Use sparingly —
the audit value of tags comes from them being mostly stable.

**Negative space on `tangent`:** the README notes the UI flags potential
tangents structurally (concepts with no path to any `root_ref`). You don't
have to read those flags. You may tag `tangent` proactively whenever you
notice one in real time; otherwise, leave it for the user to decide later.

---

## 5. Resolving curiosities

A curiosity is "resolved" when a later node answers it. To record that:

```
speechflow node resolve <curiosity-slug> --by <node-slug>
```

This sets `resolved_by_node_id` on the curiosity. The UI renders resolved
curiosities dimmed with a dotted edge to the resolver.

**When to resolve:**
- The user explicitly answers their own earlier open question.
- A new concept *clearly* settles the question (you can articulate how).

**When NOT to resolve:**
- The user moves on without addressing the question. Leave it open.
- The user partially addresses it. Leave it open and add a `references`
  edge from the partial answer back to the curiosity instead.
- You're tempted to resolve because the session is ending and you want
  things tidy. Don't. Unresolved curiosities are valuable signal across
  iterations — "this question kept coming up and never got an answer."

Curiosities should outlive iterations cleanly. The user replaying the
session in the UI wants to see what was left open as well as what landed.

---

## 6. Transcript spans, correctly

Spans are character offsets into the iteration's `transcript` column —
half-open `[start, end)`. They are how the UI ties nodes to the playback
timeline.

Rules:

1. **Append before you reference.** A node's span must point at text that
   already exists in the transcript. If you create a concept first and the
   transcript catches up later, the span will be wrong or out of range.
2. **Track your cursor.** After each `transcript append "..."` call, the
   end-of-transcript moves forward by the length of the appended text
   (counted in UTF-8 characters as Go's `utf8.RuneCountInString` would
   count them; for ASCII this is byte length). The simplest pattern: keep
   a running `cursor` in your head, set `start = cursor_before_append`,
   `end = cursor_before_append + len(appended)`, then update
   `cursor = end`.
3. **`transcript set --file` resets everything.** It replaces the whole
   transcript, so previously-recorded spans become meaningless. Avoid
   calling `set` mid-iteration unless you're prepared to fix every
   existing node's span (you usually aren't — just stick with `append`).
4. **Spans are optional.** If you can't compute one cleanly (the user
   spoke over a noisy boundary, you're catching up, etc.), omit `--span`.
   A node without a span still participates in coverage and the graph; it
   just won't highlight in the transcript modal.
5. **Quotes and spans should agree.** If you set `--quote "X"` and
   `--span S,E`, the substring `transcript[S:E]` should contain `X`
   (modulo whitespace). Don't lie about this — it shows in the UI.

---

## 7. Reading JSON output to chain calls

Every write command prints exactly one JSON object describing the new or
updated record. You **must** parse it to chain. Sketch (the actual JSON
keys are `snake_case`, per the README's `Conventions` section):

```
$ speechflow session new --title "Q4 review"
{"id":"q4-review","title":"Q4 review","description":null,"created_at":"...","updated_at":"..."}

$ speechflow root add "Pricing" "Roadmap"
[{"id":"pricing","session_id":"q4-review","title":"Pricing","created_at":"..."},
 {"id":"roadmap","session_id":"q4-review","title":"Roadmap","created_at":"..."}]

$ speechflow iteration start --title "Rehearsal 1"
{"id":"it_a1b2c3d4e5f67890","session_id":"q4-review","title":"Rehearsal 1","started_at":"...","ended_at":null,...}

$ speechflow node add concept --title "Tiered pricing"
{"id":"tiered-pricing","iteration_id":"it_a1b2c3d4e5f67890","kind":"concept",...}
```

Two ID schemes coexist:

- **Slugs** (sessions, roots, nodes): derived from titles, may be suffixed
  (`-2`, `-3`, …) on collision. Never assume one — read it from the
  response.
- **Random tokens** (iterations): `it_<16-hex>`. Always opaque; you can
  never construct one. Read it from `iteration start` or
  `iteration list` and store it.

Pass the read ID into the next call's flags.

The CLI exit codes (from the README) are stable:
- `0` success
- `1` generic error
- `2` usage error
- `3` not found (a slug you passed doesn't exist)
- `4` constraint violation (e.g. resolving a non-curiosity, no active
  session set)

Code `3` and `4` are the ones that usually mean *your* state is stale —
re-read the active session/iteration or re-list before retrying.

---

## 8. Negative space: what the agent does NOT do

This list is as important as the rest of the contract.

- **You do not compute coverage.** No matter how natural it feels to say
  "you covered 3 of 4 roots", do not call `speechflow coverage` unless the
  user asks. The user (and the UI) own that view.
- **You do not score quality.** The system is descriptive, not
  evaluative. Don't add a tag like `weak` or `strong`. Don't editorialise
  in node titles.
- **You do not mutate via HTTP.** The HTTP API at `/api/v1` is read-only.
  All writes go through the CLI.
- **You do not edit the SQLite file directly.** Same reasoning —
  uncaptured mutations break the playback feed.
- **You do not decide on the user's behalf** what is or isn't a root. If
  the user didn't declare a topic as a root, don't promote a concept into
  one just because you think it should be covered.
- **You do not delete user content casually.** `node delete`,
  `edge delete`, `iteration delete`, `session delete` are destructive and
  cascading. Only call them when the user explicitly asks.

---

## 9. Worked examples

Three short snippets to show the actual sequence of CLI calls. Assume the
agent has already initialised speechflow and there is no active session
at the start.

### Example A — A user dictating a fresh rehearsal

> **User:** "I'm rehearsing my Q4 review. I want to cover pricing, roadmap,
> and hiring."

```
speechflow session new --title "Q4 review"
# -> {"id":"q4-review",...}

speechflow root add "Pricing" "Roadmap" "Hiring"
# -> [{"id":"pricing",...},{"id":"roadmap",...},{"id":"hiring",...}]

speechflow iteration start --title "Rehearsal 1"
# -> {"id":"it_a1b2c3d4e5f67890",...}   (random; not derived from title)
```

> **User:** "Okay. On pricing — the headline is that we're moving to
> seat-based tiers next quarter."

```
speechflow transcript append "On pricing — the headline is that we're moving to seat-based tiers next quarter."
# cursor was 0, is now 86

speechflow node touch-root pricing --span 0,12
# anchors "On pricing —" to the pricing root
# -> {"id":"pricing-touch",...}

speechflow node add concept \
    --title "Move to seat-based tiered pricing next quarter" \
    --quote "we're moving to seat-based tiers next quarter" \
    --span 38,86 \
    --tag key
# -> {"id":"seat-based-tiered-pricing-next-quarter",...}

speechflow edge add seat-based-tiered-pricing-next-quarter pricing-touch --kind references
```

> **User:** "I'm not sure yet how that interacts with our annual
> contracts, though."

```
speechflow transcript append " I'm not sure yet how that interacts with our annual contracts, though."
# cursor now at 158

speechflow node add curiosity \
    --from seat-based-tiered-pricing-next-quarter \
    --title "How do seat tiers interact with annual contracts?" \
    --quote "I'm not sure yet how that interacts with our annual contracts" \
    --span 87,148
# -> {"id":"seat-tiers-vs-annual-contracts",...}
```

### Example B — Returning to an earlier point

Later in the same iteration:

> **User:** "Going back to pricing for a sec — the annual-contract question
> from earlier: we'll keep them at the legacy flat rate through end of FY."

```
speechflow transcript append " Going back to pricing for a sec — the annual-contract question from earlier: we'll keep them at the legacy flat rate through end of FY."
# spans omitted for brevity in this snippet

speechflow node touch-root pricing
# -> {"id":"pricing-touch-2",...}   (collision-suffixed)

speechflow node add concept \
    --title "Annual contracts stay on legacy flat rate through end of FY" \
    --from seat-based-tiered-pricing-next-quarter
# -> {"id":"annual-contracts-legacy-flat-rate-through-fy",...}

speechflow edge add annual-contracts-legacy-flat-rate-through-fy pricing-touch-2 --kind returns_to

speechflow node resolve seat-tiers-vs-annual-contracts \
    --by annual-contracts-legacy-flat-rate-through-fy
```

The `returns_to` edge marks the loop-back explicitly; the `resolve` call
ties off the earlier curiosity.

### Example C — A clear digression

> **User:** "…oh, that reminds me, I've been meaning to redesign the
> onboarding flow."

This is unrelated to pricing/roadmap/hiring. Record it; don't promote it
to a root.

```
speechflow transcript append " Oh, that reminds me, I've been meaning to redesign the onboarding flow."

speechflow node add concept \
    --title "Want to redesign onboarding flow" \
    --quote "I've been meaning to redesign the onboarding flow" \
    --tag tangent
# -> {"id":"redesign-onboarding-flow",...}
```

No `--from`, no edges to any `root_ref`. The `tangent` tag signals what
you noticed; the user can promote it to a root later if they want it
covered as a real topic.

### Example D — Capping a chain with a takeaway

Continuing the pricing chain from Example A: after the user wraps up
their pricing thread and starts moving toward roadmap, synthesise what
the chain actually landed.

> **User:** "…anyway, that's pricing. Onto the roadmap."

```
speechflow node add takeaway \
    --from annual-contracts-legacy-flat-rate-through-fy \
    --root pricing \
    --title "Pricing shifts to seats next quarter; annuals are grandfathered to end of FY"
# -> {"id":"pricing-shifts-to-seats-next-quarter-annuals-are-grandfathered-to-end-of-fy",
#     "kind":"takeaway","root_id":"pricing",...}
```

The takeaway hangs off the tip of the pricing chain. `--root pricing`
makes the UI surface the comparison between the declared root ("Pricing")
and the actual synthesis. If the synthesis materially diverges from the
root, that's the kind of signal the user came here for.

Use takeaways sparingly — one per coherent chain, only when you can
articulate a synthesis that is genuinely crisper than the chain itself.
A weak takeaway is worse than no takeaway.

---

## 10. Ending the session

When the user is done — or signals an end (going quiet, "okay that's it",
closing context) — close the iteration:

```
speechflow iteration end
# -> {"id":"rehearsal-1","ended_at":"2026-05-10T18:42:11Z",...}
```

That's it. Do not summarise. Do not score. If the user wants the coverage
matrix, they will ask, or open the UI with `speechflow serve --open`.

---

## 11. Quick reference

| Goal                                         | Call                                                                          |
|----------------------------------------------|-------------------------------------------------------------------------------|
| Start a new topic                            | `speechflow session new --title "..."`                                        |
| Resume a topic                               | `speechflow session use <slug>`                                               |
| Declare intended topics                      | `speechflow root add "A" "B" "C"`                                             |
| Begin a rehearsal                            | `speechflow iteration start [--title "..."]`                                  |
| Record what was said                         | `speechflow transcript append "..."`                                          |
| Anchor a declared root                       | `speechflow node touch-root <root-slug> [--span S,E]`                         |
| Record an idea                               | `speechflow node add concept --title "..." [--quote ...] [--span ...]`        |
| Record an open question                      | `speechflow node add curiosity --from <slug> --title "..."`                   |
| Synthesise a chain (leaf)                    | `speechflow node add takeaway --from <slug> [--root <root-slug>] --title "..."` |
| Mark central / digression                    | `speechflow node tag <slug> key` / `speechflow node tag <slug> tangent`       |
| Connect non-parent relationships             | `speechflow edge add <from> <to> --kind references\|returns_to`               |
| Close out a question                         | `speechflow node resolve <curiosity-slug> --by <node-slug>`                   |
| End the rehearsal                            | `speechflow iteration end`                                                    |

For exhaustive flag details, run `speechflow <command> --help`.
