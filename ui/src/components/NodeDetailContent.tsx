import { useMemo } from "react";
import { Chip, Divider } from "@heroui/react";
import {
  Quote,
  MessageCircleQuestion,
  CheckCircle2,
  FileText,
  Compass,
  AlertTriangle,
  Sparkles,
  GitBranch,
  Lightbulb,
} from "lucide-react";
import { useCoverage, useTranscript } from "@/api/query";
import { useAppStore } from "@/store/app";
import type { Graph, Node, NodeKind } from "@/api/types.gen";

// Highlight palette per node kind, mirrors the transcript-modal palette
// so a quote excerpt's tint matches the colour of the node it belongs to.
const KIND_MARK: Record<NodeKind, string> = {
  root_ref: "bg-amber-200 text-amber-900 dark:bg-amber-700/70 dark:text-amber-50",
  concept: "bg-blue-200 text-blue-900 dark:bg-blue-700/70 dark:text-blue-50",
  curiosity:
    "bg-purple-200 text-purple-900 dark:bg-purple-700/70 dark:text-purple-50",
  takeaway:
    "bg-emerald-200 text-emerald-900 dark:bg-emerald-700/70 dark:text-emerald-50",
};

interface NodeDetailContentProps {
  iterationId: string;
  graph: Graph | undefined;
  node: Node;
  // Optional caller-controlled hook used by the sidebar variant to also
  // toggle the parent transcript modal — e.g. so "Open transcript" doesn't
  // try to open a modal that's already open.
  onOpenTranscript?: () => void;
  // When true, hide the "Open transcript" button (already in transcript view).
  hideOpenTranscript?: boolean;
}

// Pull the transcript window around [start, end] with a little context so
// quotes don't read mid-sentence.
function transcriptExcerpt(
  text: string,
  start: number | null,
  end: number | null,
): { before: string; highlight: string; after: string } | null {
  if (start == null || end == null) return null;
  const clamp = (n: number) => Math.max(0, Math.min(text.length, n));
  const s = clamp(start);
  const e = clamp(end);
  if (e <= s) return null;
  // Walk back to sentence/line start, forward to sentence end.
  const dot = text.lastIndexOf(". ", s);
  const nl = text.lastIndexOf("\n", s);
  const beforeStart = Math.max(0, dot >= 0 ? dot + 2 : 0, nl >= 0 ? nl + 1 : 0);
  let afterEnd = text.length;
  const dotIdx = text.indexOf(". ", e);
  const nlIdx = text.indexOf("\n", e);
  if (dotIdx >= 0) afterEnd = Math.min(afterEnd, dotIdx + 1);
  if (nlIdx >= 0) afterEnd = Math.min(afterEnd, nlIdx);
  return {
    before: text.slice(beforeStart, s),
    highlight: text.slice(s, e),
    after: text.slice(e, afterEnd),
  };
}

export function NodeDetailContent({
  iterationId,
  graph,
  node,
  onOpenTranscript,
  hideOpenTranscript,
}: NodeDetailContentProps) {
  const focusNode = useAppStore((s) => s.focusNode);
  const { data: transcript } = useTranscript(iterationId);
  const { data: coverage } = useCoverage(iterationId);

  const byId = useMemo(
    () => new Map((graph?.nodes ?? []).map((n) => [n.id, n])),
    [graph],
  );

  // Direct children: nodes that branched FROM this node (incoming
  // branches_from edges where to=this.id, from=child).
  const children = useMemo(() => {
    if (!graph) return [] as Node[];
    return graph.edges
      .filter((e) => e.kind === "branches_from" && e.to_node === node.id)
      .map((e) => byId.get(e.from_node))
      .filter((n): n is Node => n != null);
  }, [graph, node, byId]);

  const childConcepts = children.filter((n) => n.kind === "concept");
  const childCuriosities = children.filter((n) => n.kind === "curiosity");

  // Connections (non-branches_from) and incoming branches_from (the parent).
  type Connection = {
    other: Node;
    label: string;
  };
  const otherConnections = useMemo<Connection[]>(() => {
    if (!graph) return [];
    const conns: Connection[] = [];
    for (const e of graph.edges) {
      if (e.kind === "branches_from") {
        // Outgoing branches_from from this node = "I am a child of X".
        if (e.from_node === node.id) {
          const parent = byId.get(e.to_node);
          if (parent) conns.push({ other: parent, label: "branches from" });
        }
      } else if (e.from_node === node.id) {
        const other = byId.get(e.to_node);
        if (other) conns.push({ other, label: `→ ${e.kind.replace("_", " ")}` });
      } else if (e.to_node === node.id) {
        const other = byId.get(e.from_node);
        if (other) conns.push({ other, label: `← ${e.kind.replace("_", " ")}` });
      }
    }
    return conns;
  }, [graph, node, byId]);

  const supportedRoots = useMemo(() => {
    if (!coverage) return [];
    return coverage.filter((row) => row.supporting_node_ids.includes(node.id));
  }, [coverage, node]);

  // For root_ref nodes, look up the real root title via the coverage rows.
  const ownRoot = useMemo(() => {
    if (node.kind !== "root_ref" || !node.root_id) return null;
    return coverage?.find((r) => r.root_id === node.root_id) ?? null;
  }, [coverage, node]);

  // For takeaway nodes, look up the root they were aiming at (if linked).
  const targetRoot = useMemo(() => {
    if (node.kind !== "takeaway" || !node.root_id) return null;
    return coverage?.find((r) => r.root_id === node.root_id) ?? null;
  }, [coverage, node]);

  const resolvedBy = useMemo(() => {
    if (node.kind !== "curiosity" || !node.resolved_by_node_id) return null;
    return byId.get(node.resolved_by_node_id) ?? null;
  }, [byId, node]);

  const resolvedHere = useMemo(() => {
    if (!graph) return [];
    return graph.nodes.filter(
      (n) => n.kind === "curiosity" && n.resolved_by_node_id === node.id,
    );
  }, [graph, node]);

  const excerpt = useMemo(() => {
    if (!transcript) return null;
    return transcriptExcerpt(
      transcript.text,
      node.transcript_start,
      node.transcript_end,
    );
  }, [transcript, node]);

  const classification = useMemo(() => {
    if (node.kind === "root_ref") {
      return {
        label: "Declared root",
        icon: <Compass size={14} />,
        tone: "text-amber-600 dark:text-amber-400",
      };
    }
    if (node.kind === "takeaway") {
      return {
        label: "Takeaway",
        icon: <Lightbulb size={14} />,
        tone: "text-emerald-600 dark:text-emerald-400",
      };
    }
    if (node.kind === "curiosity") {
      return node.resolved_by_node_id != null
        ? {
            label: "Resolved curiosity",
            icon: <CheckCircle2 size={14} />,
            tone: "text-success",
          }
        : {
            label: "Open question",
            icon: <MessageCircleQuestion size={14} />,
            tone: "text-purple-600 dark:text-purple-400",
          };
    }
    if (node.tags.includes("tangent")) {
      return {
        label: "Tangent",
        icon: <AlertTriangle size={14} />,
        tone: "text-warning",
      };
    }
    if (node.tags.includes("key")) {
      return {
        label: "Key concept",
        icon: <Sparkles size={14} />,
        tone: "text-primary",
      };
    }
    return {
      label: "Supporting concept",
      icon: <Quote size={14} />,
      tone: "text-default-500",
    };
  }, [node]);

  const displayTitle =
    node.kind === "root_ref" && ownRoot ? ownRoot.root_title : node.title;

  return (
    <div className="space-y-4">
      {/* Header */}
      <header className="space-y-2">
        <div className="flex items-center gap-2 flex-wrap">
          <Chip
            size="sm"
            variant="flat"
            color={
              node.kind === "root_ref"
                ? "warning"
                : node.kind === "curiosity"
                ? "secondary"
                : node.kind === "takeaway"
                ? "success"
                : "primary"
            }
          >
            {node.kind === "root_ref" ? "root" : node.kind}
          </Chip>
          <span
            className={`flex items-center gap-1 text-xs ${classification.tone}`}
          >
            {classification.icon}
            {classification.label}
          </span>
          {node.tags
            .filter((t) => t !== "key" && t !== "tangent")
            .map((t) => (
              <Chip key={t} size="sm" variant="flat">
                {t}
              </Chip>
            ))}
        </div>
        <h2 className="text-lg font-semibold leading-tight">{displayTitle}</h2>
        {node.kind === "root_ref" && (
          <p className="text-xs text-default-500">
            One of the topics declared up front. {ownRoot?.covered ? "Covered." : "Not yet covered."}
          </p>
        )}
        {node.kind === "takeaway" && (
          <p className="text-xs text-default-500">
            What the agent thinks the listener actually walked away with.
          </p>
        )}
      </header>

      {/* Takeaway-specific: what root was this aiming at? */}
      {node.kind === "takeaway" && targetRoot && (
        <section className="rounded-md border border-emerald-300/40 dark:border-emerald-700/40 bg-emerald-50/60 dark:bg-emerald-900/20 p-3">
          <div className="text-[10px] uppercase tracking-wider text-emerald-700 dark:text-emerald-300 mb-1">
            What you were going for
          </div>
          <div className="text-sm font-medium text-emerald-900 dark:text-emerald-100">
            {targetRoot.root_title}
          </div>
          <div className="text-[10px] text-emerald-700/80 dark:text-emerald-300/80 mt-1">
            Compare to the takeaway above. If they diverge, the chain landed
            somewhere different from where it was aimed.
          </div>
        </section>
      )}

      {/* Quote */}
      {node.quote && (
        <section>
          <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
            Quote
          </div>
          <blockquote className="border-l-2 border-default-300 dark:border-default-700 pl-3 italic text-sm text-default-700 dark:text-default-300">
            "{node.quote}"
          </blockquote>
        </section>
      )}

      {/* Transcript excerpt */}
      <section>
        <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1 flex items-center justify-between">
          <span>From the transcript</span>
          {!hideOpenTranscript && (
            <button
              type="button"
              onClick={() => onOpenTranscript?.()}
              className="text-[10px] text-primary hover:underline flex items-center gap-1"
            >
              <FileText size={10} />
              Open transcript
            </button>
          )}
        </div>
        {excerpt ? (
          <p className="text-sm leading-relaxed text-default-700 dark:text-default-300">
            <span className="text-default-400">{excerpt.before}</span>
            <mark className={`rounded px-0.5 ${KIND_MARK[node.kind]}`}>
              {excerpt.highlight}
            </mark>
            <span className="text-default-400">{excerpt.after}</span>
          </p>
        ) : (
          <p className="text-sm text-default-400">
            No transcript span attached to this node.
          </p>
        )}
      </section>

      {/* Root-specific: supporting concepts */}
      {node.kind === "root_ref" && childConcepts.length > 0 && (
        <section>
          <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
            Supporting concepts ({childConcepts.length})
          </div>
          <ul className="space-y-2">
            {childConcepts.map((c) => (
              <li key={c.id}>
                <button
                  type="button"
                  onClick={() => focusNode(c.id)}
                  className="w-full text-left p-2 rounded hover:bg-default-100 dark:hover:bg-default-800/40 border border-divider"
                >
                  <div className="flex items-center gap-1.5 mb-0.5">
                    <GitBranch size={11} className="text-default-400" />
                    <span className="text-sm font-medium">{c.title}</span>
                    {c.tags.includes("key") && (
                      <Chip size="sm" variant="flat" color="primary">
                        key
                      </Chip>
                    )}
                  </div>
                  {c.quote && (
                    <div className="text-xs text-default-500 italic line-clamp-2">
                      "{c.quote}"
                    </div>
                  )}
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Connections (parent + non-branches_from) */}
      {otherConnections.length > 0 && (
        <section>
          <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
            Connections
          </div>
          <ul className="space-y-0.5">
            {otherConnections.map((c, i) => (
              <li key={i}>
                <button
                  type="button"
                  onClick={() => focusNode(c.other.id)}
                  className="w-full text-left text-sm px-2 py-1 rounded hover:bg-default-100 dark:hover:bg-default-800/40 flex items-center gap-2"
                >
                  <span className="text-[10px] text-default-400 w-28 shrink-0">
                    {c.label}
                  </span>
                  <span className="truncate">{c.other.title}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      <Divider />

      {/* Coverage / status */}
      <section>
        <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
          Coverage
        </div>
        {node.kind === "root_ref" ? (
          <p className="text-xs text-default-700 dark:text-default-300 flex items-center gap-1">
            {ownRoot?.covered ? (
              <>
                <CheckCircle2 size={12} className="text-success" />
                Anchored — supported by {ownRoot.supporting_node_ids.length}{" "}
                node{ownRoot.supporting_node_ids.length === 1 ? "" : "s"}.
              </>
            ) : (
              <>
                <AlertTriangle size={12} className="text-warning" />
                Anchored but no supporting nodes recorded yet.
              </>
            )}
          </p>
        ) : supportedRoots.length > 0 ? (
          <ul className="space-y-1">
            {supportedRoots.map((r) => (
              <li
                key={r.root_id}
                className="text-xs flex items-center gap-1 text-success"
              >
                <CheckCircle2 size={12} />
                {r.root_title}
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-xs text-default-400">
            Doesn't reach any declared root. Possible tangent.
          </p>
        )}
      </section>

      {/* Curiosities raised */}
      {childCuriosities.length > 0 && (
        <section>
          <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
            Questions raised
          </div>
          <ul className="space-y-1">
            {childCuriosities.map((c) => (
              <li key={c.id}>
                <button
                  type="button"
                  onClick={() => focusNode(c.id)}
                  className="w-full text-left text-xs px-1 py-0.5 rounded hover:bg-default-100 dark:hover:bg-default-800/40 flex items-start gap-1"
                >
                  {c.resolved_by_node_id ? (
                    <CheckCircle2
                      size={12}
                      className="text-success mt-0.5 shrink-0"
                    />
                  ) : (
                    <MessageCircleQuestion
                      size={12}
                      className="text-purple-500 mt-0.5 shrink-0"
                    />
                  )}
                  <span>
                    {c.title}
                    <span className="text-default-400 ml-1">
                      {c.resolved_by_node_id ? "resolved" : "open"}
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* This-node-as-curiosity status */}
      {node.kind === "curiosity" && (
        <section>
          <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
            Status
          </div>
          {resolvedBy ? (
            <button
              type="button"
              onClick={() => focusNode(resolvedBy.id)}
              className="w-full text-left text-xs px-1 py-0.5 rounded hover:bg-default-100 dark:hover:bg-default-800/40 flex items-start gap-1 text-success"
            >
              <CheckCircle2 size={12} className="mt-0.5 shrink-0" />
              <span>
                Resolved by{" "}
                <span className="font-medium">{resolvedBy.title}</span>
              </span>
            </button>
          ) : (
            <p className="text-xs text-purple-600 dark:text-purple-400 flex items-center gap-1">
              <MessageCircleQuestion size={12} />
              Still open
            </p>
          )}
        </section>
      )}

      {/* Curiosities this node resolved */}
      {resolvedHere.length > 0 && (
        <section>
          <div className="text-[10px] uppercase tracking-wider text-default-500 mb-1">
            Resolves
          </div>
          <ul className="space-y-1">
            {resolvedHere.map((c) => (
              <li key={c.id}>
                <button
                  type="button"
                  onClick={() => focusNode(c.id)}
                  className="w-full text-left text-xs px-1 py-0.5 rounded hover:bg-default-100 dark:hover:bg-default-800/40 flex items-start gap-1 text-success"
                >
                  <CheckCircle2 size={12} className="mt-0.5 shrink-0" />
                  <span>{c.title}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
