import { useMemo, useState } from "react";
import {
  Sparkles,
  AlertTriangle,
  MessageCircleQuestion,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { useCoverage, useGraph, useIteration, useTranscript } from "@/api/query";
import { useAppStore } from "@/store/app";
import type { Node } from "@/api/types.gen";

// Canonical tag vocabulary the agent applies in real time. Listed here so
// the panel can group findings into "strengths" vs. "weaknesses" without
// pulling unrelated free-form tags into the dashboard.
const POSITIVE_TAGS = [
  { tag: "hook", label: "Hook" },
  { tag: "signpost", label: "Signpost" },
  { tag: "exposition", label: "Exposition" },
  { tag: "analogy", label: "Analogy" },
  { tag: "example", label: "Example" },
  { tag: "callback", label: "Callback" },
  { tag: "definition", label: "Definition" },
  { tag: "pivot", label: "Pivot" },
  { tag: "closing", label: "Closing" },
  { tag: "key", label: "Key concept" },
] as const;

const WEAKNESS_TAGS = [
  { tag: "tangent", label: "Tangent" },
  { tag: "unsupported-claim", label: "Unsupported claim" },
  { tag: "dropped-thread", label: "Dropped thread" },
  { tag: "filler", label: "Filler" },
  { tag: "abrupt-transition", label: "Abrupt transition" },
  { tag: "contradiction", label: "Contradiction" },
] as const;

interface SpeechHealthPanelProps {
  iterationId: string;
}

interface RootAirtime {
  rootId: string;
  rootTitle: string;
  chars: number;
  pct: number;
}

// Partition the transcript by root_ref boundaries. Each root_ref starts
// owning the transcript window from its transcript_start until the next
// root_ref's transcript_start (or the transcript end). Chars before the
// first root_ref are "unattributed" — usually the intro/hook section.
function computeAirtime(
  rootRefs: Node[],
  rootTitleById: Map<string, string>,
  transcriptLen: number,
): { rows: RootAirtime[]; unattributed: number } {
  const anchored = rootRefs
    .filter((n) => n.transcript_start != null)
    .sort((a, b) => (a.transcript_start ?? 0) - (b.transcript_start ?? 0));

  if (anchored.length === 0) {
    return { rows: [], unattributed: transcriptLen };
  }

  const totals = new Map<string, number>();
  const unattributed = anchored[0].transcript_start ?? 0;

  for (let i = 0; i < anchored.length; i++) {
    const start = anchored[i].transcript_start ?? 0;
    const end =
      i + 1 < anchored.length
        ? anchored[i + 1].transcript_start ?? transcriptLen
        : transcriptLen;
    const rootId = anchored[i].root_id;
    if (!rootId) continue;
    totals.set(rootId, (totals.get(rootId) ?? 0) + Math.max(0, end - start));
  }

  const total = transcriptLen || 1;
  const rows: RootAirtime[] = Array.from(totals.entries()).map(
    ([rootId, chars]) => ({
      rootId,
      rootTitle: rootTitleById.get(rootId) ?? rootId,
      chars,
      pct: chars / total,
    }),
  );
  rows.sort((a, b) => b.chars - a.chars);
  return { rows, unattributed };
}

export function SpeechHealthPanel({ iterationId }: SpeechHealthPanelProps) {
  const iteration = useIteration(iterationId);
  const graph = useGraph(iterationId, iteration.data?.ended_at ?? null);
  const { data: transcript } = useTranscript(iterationId);
  const { data: coverage } = useCoverage(iterationId);
  const focusNode = useAppStore((s) => s.focusNode);

  const [openTag, setOpenTag] = useState<string | null>(null);
  const toggle = (tag: string) =>
    setOpenTag((current) => (current === tag ? null : tag));

  const rootTitleById = useMemo(() => {
    const m = new Map<string, string>();
    coverage?.forEach((r) => m.set(r.root_id, r.root_title));
    return m;
  }, [coverage]);

  const airtime = useMemo(() => {
    if (!graph.data || !transcript) {
      return { rows: [] as RootAirtime[], unattributed: 0 };
    }
    const rootRefs = graph.data.nodes.filter((n) => n.kind === "root_ref");
    return computeAirtime(rootRefs, rootTitleById, transcript.text.length);
  }, [graph.data, transcript, rootTitleById]);

  // Group nodes by every tag they carry, then render counts under the two
  // canonical buckets. Free-form tags are ignored here on purpose — this
  // dashboard is for the speech-quality vocabulary, not arbitrary labels.
  const nodesByTag = useMemo(() => {
    const m = new Map<string, Node[]>();
    if (!graph.data) return m;
    for (const n of graph.data.nodes) {
      for (const t of n.tags) {
        if (!m.has(t)) m.set(t, []);
        m.get(t)!.push(n);
      }
    }
    return m;
  }, [graph.data]);

  const openCuriosities = useMemo(() => {
    if (!graph.data) return [] as Node[];
    return graph.data.nodes.filter(
      (n) => n.kind === "curiosity" && n.resolved_by_node_id == null,
    );
  }, [graph.data]);

  const totalChars = transcript?.text.length ?? 0;
  const unattPct = totalChars > 0 ? airtime.unattributed / totalChars : 0;

  const positiveCount = POSITIVE_TAGS.reduce(
    (acc, t) => acc + (nodesByTag.get(t.tag)?.length ?? 0),
    0,
  );
  const weaknessCount = WEAKNESS_TAGS.reduce(
    (acc, t) => acc + (nodesByTag.get(t.tag)?.length ?? 0),
    0,
  );

  return (
    <div className="flex-1 overflow-y-auto p-3 space-y-5">
      {/* Airtime */}
      <section>
        <h3 className="text-[10px] uppercase tracking-wider text-default-500 mb-2">
          Airtime per root
        </h3>
        {airtime.rows.length === 0 ? (
          <p className="text-xs text-default-400">
            No anchored root_ref spans yet. Pass <code>--span S,E</code> when
            calling <code>node touch-root</code> so airtime can be computed.
          </p>
        ) : (
          <div className="space-y-1.5">
            {airtime.rows.map((r) => (
              <div key={r.rootId} className="text-xs">
                <div className="flex justify-between mb-0.5">
                  <span className="font-medium truncate">{r.rootTitle}</span>
                  <span className="tabular-nums text-default-400">
                    {Math.round(r.pct * 100)}%
                  </span>
                </div>
                <div className="h-1.5 bg-default-200/60 dark:bg-default-800/60 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-primary rounded-full"
                    style={{ width: `${Math.max(2, r.pct * 100)}%` }}
                  />
                </div>
              </div>
            ))}
            {unattPct > 0.01 && (
              <div className="text-xs pt-1">
                <div className="flex justify-between mb-0.5">
                  <span className="text-default-500 italic">
                    Intro / pre-roots
                  </span>
                  <span className="tabular-nums text-default-400">
                    {Math.round(unattPct * 100)}%
                  </span>
                </div>
                <div className="h-1.5 bg-default-200/60 dark:bg-default-800/60 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-default-400 rounded-full"
                    style={{ width: `${Math.max(2, unattPct * 100)}%` }}
                  />
                </div>
              </div>
            )}
          </div>
        )}
      </section>

      {/* Strengths */}
      <section>
        <h3 className="text-[10px] uppercase tracking-wider text-default-500 mb-2 flex items-center gap-1">
          <Sparkles size={12} className="text-success" />
          Strengths · {positiveCount}
        </h3>
        <TagBuckets
          tags={POSITIVE_TAGS}
          nodesByTag={nodesByTag}
          openTag={openTag}
          toggle={toggle}
          onFocus={focusNode}
          accent="success"
        />
      </section>

      {/* Weaknesses */}
      <section>
        <h3 className="text-[10px] uppercase tracking-wider text-default-500 mb-2 flex items-center gap-1">
          <AlertTriangle size={12} className="text-warning" />
          Weaknesses · {weaknessCount}
        </h3>
        <TagBuckets
          tags={WEAKNESS_TAGS}
          nodesByTag={nodesByTag}
          openTag={openTag}
          toggle={toggle}
          onFocus={focusNode}
          accent="warning"
        />
      </section>

      {/* Open curiosities — derived, not tag-based */}
      <section>
        <h3 className="text-[10px] uppercase tracking-wider text-default-500 mb-2 flex items-center gap-1">
          <MessageCircleQuestion size={12} className="text-purple-500" />
          Open curiosities · {openCuriosities.length}
        </h3>
        {openCuriosities.length === 0 ? (
          <p className="text-xs text-default-400">
            All curiosities resolved or none recorded.
          </p>
        ) : (
          <ul className="space-y-0.5">
            {openCuriosities.map((n) => (
              <li key={n.id}>
                <button
                  type="button"
                  onClick={() => focusNode(n.id)}
                  className="w-full text-left text-xs px-1.5 py-1 rounded hover:bg-default-100 dark:hover:bg-default-800/40 truncate"
                >
                  {n.title}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

interface TagBucketsProps {
  tags: ReadonlyArray<{ tag: string; label: string }>;
  nodesByTag: Map<string, Node[]>;
  openTag: string | null;
  toggle: (tag: string) => void;
  onFocus: (id: string) => void;
  accent: "success" | "warning";
}

function TagBuckets({ tags, nodesByTag, openTag, toggle, onFocus, accent }: TagBucketsProps) {
  const present = tags.filter((t) => (nodesByTag.get(t.tag)?.length ?? 0) > 0);
  if (present.length === 0) {
    return (
      <p className="text-xs text-default-400">
        None flagged. The agent will tag these in real time as it records.
      </p>
    );
  }
  return (
    <ul className="space-y-0.5">
      {present.map(({ tag, label }) => {
        const nodes = nodesByTag.get(tag) ?? [];
        const isOpen = openTag === tag;
        return (
          <li key={tag}>
            <button
              type="button"
              onClick={() => toggle(tag)}
              className="w-full flex items-center gap-1 text-xs px-1.5 py-1 rounded hover:bg-default-100 dark:hover:bg-default-800/40 text-default-700 dark:text-default-200"
            >
              {isOpen ? (
                <ChevronDown size={11} className="text-default-500 dark:text-default-300" />
              ) : (
                <ChevronRight size={11} className="text-default-500 dark:text-default-300" />
              )}
              <span className={`text-${accent}`}>{label}</span>
              <span className="ml-auto tabular-nums text-default-500 dark:text-default-300">
                {nodes.length}
              </span>
            </button>
            {isOpen && (
              <ul className="pl-4 mt-0.5 space-y-0.5">
                {nodes.map((n) => (
                  <li key={n.id}>
                    <button
                      type="button"
                      onClick={() => onFocus(n.id)}
                      className="w-full text-left text-xs px-1.5 py-0.5 rounded hover:bg-default-100 dark:hover:bg-default-800/40 truncate text-default-700 dark:text-default-700"
                    >
                      {n.title}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </li>
        );
      })}
    </ul>
  );
}
