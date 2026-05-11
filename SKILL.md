---
name: speechflow
description: Use speechflow to turn a speech transcript (or a live spoken conversation) into a concept graph with rhetorical-quality annotations — declared topics, ideas, supporting evidence, takeaways, and flagged weaknesses. Invoke when the user hands you a transcript and wants it evaluated, when they're rehearsing a talk aloud and want it captured, or when they ask to see how their reasoning was structured. Records via the speechflow CLI; produces a local web UI for review.
---

# speechflow skill

speechflow is a local CLI with an embedded web UI. It models a speech as a
typed graph (roots, concepts, curiosities, takeaways) with edges
(`branches_from`, `references`, `returns_to`, `supports`, `contrasts`) and
a canonical vocabulary of quality tags. **You** read the transcript and
make the structural calls; speechflow records them deterministically and
renders them in a graph + transcript + health dashboard.

This skill handles two modes:

- **Transcript mode** (most common) — the user hands you a full
  transcript and wants it evaluated. You read it end-to-end first, plan
  the structure, then populate the graph in one pass.
- **Live mode** — the user is speaking aloud and you record as they go.
  Same vocabulary, different pacing. Covered in `AGENTS.md` §1.

---

## Install

Check first:

```sh
speechflow version
```

If missing, install and initialise (ask the user before running):

```sh
curl -fsSL https://raw.githubusercontent.com/camggould/speechflow/main/install.sh | sh
speechflow init
```

`speechflow init` creates `~/.speechflow/` and runs schema migrations.
One-time.

---

## Transcript workflow

The user gives you a transcript — paste, file, dictation result. Follow
this order. Skipping the planning step (1) and going straight to nodes
produces a graph that misses structure.

### 1. Read the transcript end-to-end first

Before any CLI call: scan the full text. You're looking for:

- **Declared topics** — the speaker's explicit "I want to cover X, Y, Z."
  These become **roots**.
- **The arc** — opening hook, exposition, claim chains, examples,
  curiosities, callbacks, closing.
- **Structural moments** — pivots, signposts, transitions, returns.
- **Weaknesses** — claims without evidence, dropped threads, abrupt
  jumps, tangents, contradictions.

Don't write yet. Think.

### 2. Create the session and declare roots

```sh
speechflow session new --title "<short title for the talk>" \
    --description "<optional one-line summary>"
# returns {"id":"<session-slug>","title":"...",...}
```

Then declare every root the speaker explicitly named:

```sh
speechflow root add "First topic" "Second topic" "Third topic"
```

If the speaker didn't declare roots explicitly, infer 2–4 from the
opening/structure (e.g. "the talk has three pillars: …"). Don't invent
roots that aren't in the speech.

### 3. Start the iteration and set the transcript

```sh
speechflow iteration start --title "Initial evaluation"
# returns {"id":"it_<hex>",...} — read and remember this opaque ID
```

For a static transcript, set the whole text once:

```sh
speechflow transcript set --file /path/to/transcript.txt
```

(If the transcript isn't a file, write it to a temp file first — the CLI
takes `--file`, not stdin.)

### 4. Walk the transcript and create nodes

Iterate through the text in order. For each substantive moment, create a
node and attach a `--span S,E` (character offsets into the transcript).
Compute spans by searching the transcript text for the relevant
substring; `str.find()`-equivalent is fine.

For each declared root, when the speaker first lands on it:

```sh
speechflow node touch-root <root-slug> --span <S>,<E>
```

For each idea the speaker introduces:

```sh
speechflow node add concept \
    --title "<your short paraphrase>" \
    --quote "<verbatim load-bearing fragment>" \
    --span <S>,<E> \
    [--from <parent-slug>] \
    [--tag key]
```

`--from` parents the new concept under whichever node it developed from.
For the first concept of a chain, point `--from` at the relevant
`touch-<root-slug>` node so the structural coverage algorithm can reach
the root.

For each open question the speaker raises but doesn't resolve:

```sh
speechflow node add curiosity \
    --from <parent-concept-slug> \
    --title "<the question>" \
    --quote "<the hedge>" \
    --span <S>,<E>
```

When a later concept clearly resolves a curiosity:

```sh
speechflow node resolve <curiosity-slug> --by <resolver-slug>
```

At the end of each coherent chain, create a takeaway — the synthesis of
what the listener actually walked away with:

```sh
speechflow node add takeaway \
    --from <leaf-concept-slug> \
    --root <intended-root-slug> \
    --title "<one-sentence synthesis>"
```

One takeaway per chain. Skip it if you can't articulate a synthesis
crisper than the chain itself.

### 5. Add non-parent edges

Beyond the `branches_from` edges that `--from` creates, attach:

```sh
speechflow edge add <evidence-slug> <claim-slug>  --kind supports
speechflow edge add <a-slug> <b-slug>             --kind references
speechflow edge add <callback-slug> <earlier-slug> --kind returns_to
speechflow edge add <new-slug> <prior-slug>       --kind contrasts
```

Use `supports` aggressively — every evidence/example/analogy under a
claim should have one. Without it, the Health panel will flag the claim
as `unsupported-claim` during the end-of-iteration sweep.

### 6. Tag for the Speech Health panel

Apply tags at node creation (`--tag <tag>`) or after the fact
(`speechflow node tag <slug> <tag>`). Canonical vocabulary:

| Strengths (apply when you see it)                                | Weaknesses (apply when you detect the issue)                        |
|------------------------------------------------------------------|---------------------------------------------------------------------|
| `key`, `hook`, `signpost`, `exposition`, `analogy`, `example`,   | `tangent`, `unsupported-claim`, `dropped-thread`, `filler`,         |
| `callback`, `definition`, `pivot`, `closing`                     | `abrupt-transition`, `contradiction`                                |

Apply eagerly. False positives are cheaper than false negatives — the
user can scan and untag, but they can't see what you never flagged.

### 7. End-of-iteration sweep

Before closing, do one retroactive pass:

- For each `concept` with no children, no incoming `references` /
  `returns_to`, and no takeaway: tag `dropped-thread`.
- For each `key`-tagged concept with no `supports` edge and no
  `example`/`analogy`-tagged child: tag `unsupported-claim`.

These are the only retroactive tags. Everything else should already be
applied at creation.

### 8. Close the iteration

```sh
speechflow iteration end
```

After this, `ended_at` is set, the timeline is finite, and the Coverage /
Health panels render against a frozen graph.

### 9. Launch the UI

```sh
speechflow serve --open
```

Binds `127.0.0.1:7777` and opens the user's browser to the dashboard.
The session you just populated will appear; clicking it shows the
iteration list, and clicking the iteration shows the graph + transcript
modal + tabbed insights panel (Coverage / Health).

If the user prefers not to auto-open, drop `--open` and tell them to
visit `http://127.0.0.1:7777/`.

---

## A complete worked example

User pastes a 2-paragraph speech with three claims and one open question.

```sh
# 1. Plan: read it. Identify roots = ["Pricing", "Roadmap"].
# 2. Session + roots
speechflow session new --title "Q4 review"           # → q4-review
speechflow root add "Pricing" "Roadmap"

# 3. Iteration + transcript
ITER=$(speechflow iteration start --title "Eval 1" | jq -r .id)
speechflow transcript set --file /tmp/speech.txt

# 4. Walk and record (spans computed from the transcript text)
speechflow node touch-root pricing --span 0,12
speechflow node add concept \
    --title "Seat-based pricing next quarter" \
    --quote "we're moving to seat-based tiers" \
    --span 38,86 \
    --tag key \
    --from touch-pricing
speechflow node add concept \
    --title "Existing customers grandfathered" \
    --quote "existing annuals stay on legacy rate" \
    --span 90,130 \
    --tag example \
    --from seat-based-pricing-next-quarter
speechflow edge add existing-customers-grandfathered \
    seat-based-pricing-next-quarter --kind supports

speechflow node add curiosity \
    --from seat-based-pricing-next-quarter \
    --title "What about mid-year upgrades?" \
    --quote "I haven't worked out mid-year upgrade pricing"

# (… continue for the second root …)

# 5. Add a takeaway per chain
speechflow node add takeaway \
    --from existing-customers-grandfathered \
    --root pricing \
    --title "Pricing shifts to seats but doesn't break existing contracts"

# 6. Sweep, then close
speechflow node tag unaddressed-roadmap-slip dropped-thread  # if any
speechflow iteration end

# 7. Show it
speechflow serve --open
```

---

## What you do NOT do

- Do not call `speechflow coverage` to score the speech in chat. The
  Health panel exists for that; the user owns the evaluation.
- Do not invent roots the speaker didn't declare or strongly imply.
- Do not promote a tangent into a root because it seemed important.
- Do not `transcript set` again mid-iteration — it invalidates every
  span you've recorded.
- Do not mutate via the HTTP API. `/api/v1` is read-only by design.

---

## Reference

- **Node kinds**: `root_ref` (root anchor) · `concept` (idea) ·
  `curiosity` (open question) · `takeaway` (chain leaf synthesis).
- **Edge kinds**: `branches_from` (child → parent, set by `--from`) ·
  `references` (peer cross-link) · `returns_to` (explicit callback) ·
  `supports` (evidence → claim) · `contrasts` (steel-man / contradiction).
- **ID schemes**: sessions / roots / nodes use slugs derived from
  titles. Iterations use opaque random `it_<16-hex>` tokens. Always read
  IDs from the JSON output of the call that created them; never
  construct an iteration ID.
- **Live UI updates**: while an iteration is active (`ended_at` is null)
  the UI polls `/api/v1/iterations/:id/graph` every second, so the user
  can watch the graph build as you record.

For the full opinionated contract — every "when to use this exact
node/edge/tag", the curiosity resolution rules, the agent's negative
space — read `AGENTS.md` in the speechflow repo.
