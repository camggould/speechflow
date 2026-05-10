---
name: speechflow
description: Use speechflow to turn a spoken (or written) practice conversation into a concept graph — ideas, claims, and curiosities branching off the topics the user said they wanted to cover. Invoke when the user is rehearsing a talk, dictating a draft, brainstorming aloud, or any time the user wants their reasoning shaped into a reviewable structure rather than a flat transcript. Records via the speechflow CLI.
---

# speechflow skill

speechflow is a local CLI (with an embedded web UI) that lets an LLM agent
record a spoken conversation as a typed graph: **sessions** (a talk you're
working on), made of **iterations** (one rehearsal each), made of **nodes**
(`root_ref`, `concept`, `curiosity`) and **edges** (`branches_from`,
`references`, `returns_to`). The user declares roots up front ("today I
want to cover pricing, roadmap, hiring"); you record the rest as the
conversation unfolds. After the fact, the user can replay any iteration
in the UI and see structural coverage of their declared topics.

The CLI is deterministic. **You** do the judgment — what's a concept,
what's a curiosity, when something is resolved. speechflow just stores it.

Use this skill when:
- the user is rehearsing or dictating a talk, pitch, lecture, or essay
- the user wants their reasoning shaped into a reviewable map, not a
  flat transcript
- the user wants to come back across multiple rehearsals and see which
  declared topics they consistently hit (or miss)
- the user says something like "let's record this", "let me practise out
  loud", "I want to map this out as I think"

## Detect whether speechflow is installed

`speechflow version` should print a version string. If the command is
missing, suggest installing it:

```sh
curl -fsSL https://raw.githubusercontent.com/camggould/speechflow/main/install.sh | sh
```

Then run `speechflow init` once to create `~/.speechflow/` and run
migrations. Don't proactively install on the user's machine; ask first.

## The shape of the work

speechflow has a strict three-phase loop. Stick to it; don't improvise.

1. **Session start** — pick or create a session, declare roots (if the
   user named topics), start an iteration.
2. **During the conversation** — for each utterance: append transcript,
   then optionally add nodes / edges / tags / resolve curiosities.
3. **Session end** — close the iteration. Don't score it.

Full prescriptive contract: see `AGENTS.md` in the speechflow repo. Read
it before driving the CLI for the first time.

## Minimum viable session

```sh
# once per topic
speechflow session new --title "Q4 review"
speechflow root add "Pricing" "Roadmap" "Hiring"

# once per rehearsal
speechflow iteration start --title "Rehearsal 1"

# per utterance
speechflow transcript append "On pricing — we're moving to seat-based tiers."
speechflow node touch-root pricing
speechflow node add concept --title "Move to seat-based tiered pricing" --tag key

# wrap up
speechflow iteration end
```

Every write command prints a single JSON object with the new record's
`id` (a slug). **Read the id** and pass it to follow-up commands —
slugs are derived from titles but get suffixed (`-2`, `-3`, …) on
collision, so never guess them.

## The three node kinds

| Kind        | When                                                              |
|-------------|-------------------------------------------------------------------|
| `root_ref`  | The user materially touches one of the declared session roots.    |
| `concept`   | The user introduces a substantive idea, claim, definition, etc.   |
| `curiosity` | The user opens a question, hedge, or thread they don't resolve.   |

`concept` is the workhorse. `root_ref` is purely structural — the
coverage algorithm follows edges back to `root_ref` nodes to decide which
roots got touched. `curiosity` captures open threads — leave them open
unless a later concept clearly answers them, then call
`speechflow node resolve <curiosity-slug> --by <concept-slug>`.

## Tags

Two have UI meaning: `key` (the user signalled the idea is central; solid
border) and `tangent` (a digression; dashed border). Set them with
`--tag key` at creation or `speechflow node tag <slug> tangent` later.
Other tags (`evidence`, `example`, `definition`, `pivot`) render as
chips — use them when they fit.

## What you do NOT do

- Do **not** call `speechflow coverage`. The user owns that.
- Do **not** score the rehearsal in chat ("you did well", "you missed
  hiring") unless the user explicitly asks for an assessment.
- Do **not** mutate state via the HTTP API — `/api/v1` is read-only by
  design. All writes go through the CLI.
- Do **not** promote a stray topic into a root just because you think it
  should be covered. If the user didn't declare it, leave it as a
  concept (optionally tagged `tangent`).
- Do **not** `transcript set` mid-iteration — it invalidates every
  previously recorded span. Stick to `transcript append`.

## Reviewing past sessions

The web UI is the right surface for review. Launch it:

```sh
speechflow serve --open       # binds 127.0.0.1:7777, opens the browser
```

It renders each iteration as a React Flow graph, a transcript with span
highlights, and a coverage matrix across all rehearsals of the session.
If the user asks "how did I do across rehearsals?" — open the UI.

For programmatic inspection, the read-only HTTP API at
`http://127.0.0.1:7777/api/v1/...` exposes sessions, iterations, graph,
timeline, transcript, and coverage. JSON responses are `snake_case`.

## Quick reference

| Goal                              | CLI                                                                       |
|-----------------------------------|---------------------------------------------------------------------------|
| Initialise                        | `speechflow init`                                                         |
| Start a topic                     | `speechflow session new --title "..."`                                    |
| Resume a topic                    | `speechflow session use <slug>`                                           |
| Declare intended topics           | `speechflow root add "A" "B" "C"`                                         |
| Begin a rehearsal                 | `speechflow iteration start [--title "..."]`                              |
| Append to the transcript          | `speechflow transcript append "..."`                                      |
| Anchor a declared root            | `speechflow node touch-root <root-slug>`                                  |
| Add an idea                       | `speechflow node add concept --title "..." [--quote ...] [--span S,E]`    |
| Add an open question              | `speechflow node add curiosity --from <slug> --title "..."`               |
| Connect related nodes             | `speechflow edge add <from> <to> --kind references\|returns_to`           |
| Resolve an open question          | `speechflow node resolve <slug> --by <node-slug>`                         |
| Tag a node                        | `speechflow node tag <slug> key`                                          |
| End the rehearsal                 | `speechflow iteration end`                                                |
| Open the UI                       | `speechflow serve --open`                                                 |

For exhaustive flag details, run `speechflow <command> --help`. For the
opinionated contract (when to create what kind of node, how spans work,
what NOT to do), read `AGENTS.md`.
