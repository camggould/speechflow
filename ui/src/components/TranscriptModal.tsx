import { useEffect, useMemo, useRef } from "react";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
} from "@heroui/react";
import { useGraph, useIteration, useTranscript } from "@/api/query";
import { useAppStore } from "@/store/app";
import type { NodeKind, Transcript } from "@/api/types.gen";
import { NodeDetailContent } from "@/components/NodeDetailContent";

interface TranscriptModalProps {
  iterationId: string;
}

// Highlight palette per node kind, matched to the graph node colors so a
// reader can scan the transcript and recognise which kind anchored each
// span without looking back at the canvas. Saturation is bumped in light
// mode (so it reads on white) and dropped in dark mode (so it doesn't
// overpower the surrounding text).
const KIND_HIGHLIGHT: Record<
  NodeKind,
  { base: string; focused: string }
> = {
  root_ref: {
    base:
      "bg-amber-200 text-amber-900 hover:bg-amber-300 " +
      "dark:bg-amber-700/70 dark:text-amber-50 dark:hover:bg-amber-600/80",
    focused:
      "bg-amber-300 text-amber-950 ring-2 ring-amber-500 " +
      "dark:bg-amber-500/80 dark:text-amber-50 dark:ring-amber-300",
  },
  concept: {
    base:
      "bg-blue-200 text-blue-900 hover:bg-blue-300 " +
      "dark:bg-blue-700/70 dark:text-blue-50 dark:hover:bg-blue-600/80",
    focused:
      "bg-blue-300 text-blue-950 ring-2 ring-blue-500 " +
      "dark:bg-blue-500/80 dark:text-blue-50 dark:ring-blue-300",
  },
  curiosity: {
    base:
      "bg-purple-200 text-purple-900 hover:bg-purple-300 " +
      "dark:bg-purple-700/70 dark:text-purple-50 dark:hover:bg-purple-600/80",
    focused:
      "bg-purple-300 text-purple-950 ring-2 ring-purple-500 " +
      "dark:bg-purple-500/80 dark:text-purple-50 dark:ring-purple-300",
  },
  takeaway: {
    base:
      "bg-emerald-200 text-emerald-900 hover:bg-emerald-300 " +
      "dark:bg-emerald-700/70 dark:text-emerald-50 dark:hover:bg-emerald-600/80",
    focused:
      "bg-emerald-300 text-emerald-950 ring-2 ring-emerald-500 " +
      "dark:bg-emerald-500/80 dark:text-emerald-50 dark:ring-emerald-300",
  },
};

// When a transcript span is covered by nodes of multiple kinds (rare —
// happens when the agent records overlapping anchors), pick the one most
// structurally distinctive so the colour is informative. Root anchors win
// because they're the structural backbone; takeaways are leaf syntheses;
// curiosities are open questions; concepts are the workhorse default.
const KIND_PRIORITY: NodeKind[] = ["root_ref", "takeaway", "curiosity", "concept"];

interface Segment {
  start: number;
  end: number;
  text: string;
  nodeIds: string[];
}

// Split transcript into non-overlapping segments tagged with the node ids
// whose spans cover that segment. Linear scan: collect every span boundary,
// then walk left-to-right emitting one segment per gap.
function segmentTranscript(t: Transcript): Segment[] {
  if (!t.text) return [];
  const boundaries = new Set<number>([0, t.text.length]);
  for (const s of t.spans) {
    boundaries.add(Math.max(0, Math.min(t.text.length, s.start)));
    boundaries.add(Math.max(0, Math.min(t.text.length, s.end)));
  }
  const sorted = Array.from(boundaries).sort((a, b) => a - b);
  const out: Segment[] = [];
  for (let i = 0; i < sorted.length - 1; i++) {
    const start = sorted[i];
    const end = sorted[i + 1];
    if (start === end) continue;
    const nodeIds = t.spans
      .filter((s) => s.start <= start && s.end >= end)
      .map((s) => s.node_id);
    out.push({ start, end, text: t.text.slice(start, end), nodeIds });
  }
  return out;
}

export function TranscriptModal({ iterationId }: TranscriptModalProps) {
  const open = useAppStore((s) => s.transcriptOpen);
  const close = useAppStore((s) => s.closeTranscript);
  const focusedNodeId = useAppStore((s) => s.focusedNodeId);
  const focusNode = useAppStore((s) => s.focusNode);

  const { data, isLoading } = useTranscript(iterationId);
  const iteration = useIteration(iterationId);
  const graph = useGraph(iterationId, iteration.data?.ended_at ?? null);

  const focusedNode = useMemo(
    () => graph.data?.nodes.find((n) => n.id === focusedNodeId) ?? null,
    [graph.data, focusedNodeId],
  );

  const segments = useMemo(
    () => (data ? segmentTranscript(data) : []),
    [data],
  );

  const nodesById = useMemo(
    () => new Map((graph.data?.nodes ?? []).map((n) => [n.id, n])),
    [graph.data],
  );

  const pickKind = (nodeIds: string[]): NodeKind | null => {
    for (const kind of KIND_PRIORITY) {
      if (nodeIds.some((id) => nodesById.get(id)?.kind === kind)) return kind;
    }
    return null;
  };

  // Auto-scroll the highlighted span into view when the user clicks a node.
  const highlightRef = useRef<HTMLSpanElement>(null);
  useEffect(() => {
    if (open && focusedNodeId && highlightRef.current) {
      highlightRef.current.scrollIntoView({ block: "center", behavior: "smooth" });
    }
  }, [open, focusedNodeId]);

  // Closing the transcript modal also clears the highlighted node so the
  // detail content doesn't reappear unexpectedly on next open.
  const handleClose = () => {
    focusNode(null);
    close();
  };

  return (
    <Modal
      isOpen={open}
      onClose={handleClose}
      size={focusedNode ? "5xl" : "3xl"}
      scrollBehavior="inside"
    >
      <ModalContent>
        <ModalHeader>Transcript</ModalHeader>
        <ModalBody className="pb-6">
          <div
            className={
              focusedNode
                ? "grid grid-cols-[1fr_360px] gap-6"
                : ""
            }
          >
            {/* Transcript pane */}
            <div className="min-w-0">
              {isLoading && (
                <div className="text-sm text-default-500">Loading…</div>
              )}
              {!isLoading && (!data || data.text === "") && (
                <div className="text-sm text-default-400">
                  No transcript yet. Use{" "}
                  <code>speechflow transcript append</code> to add text.
                </div>
              )}
              {data && data.text !== "" && (
                <div className="leading-relaxed whitespace-pre-wrap text-sm max-h-[70vh] overflow-y-auto pr-2">
                  {segments.map((seg, i) => {
                    const focused =
                      focusedNodeId != null && seg.nodeIds.includes(focusedNodeId);
                    const kind = seg.nodeIds.length > 0 ? pickKind(seg.nodeIds) : null;
                    const palette = kind ? KIND_HIGHLIGHT[kind] : null;
                    const className = palette
                      ? `rounded px-0.5 cursor-pointer transition-colors ${
                          focused ? palette.focused : palette.base
                        }`
                      : "";
                    return (
                      <span
                        key={i}
                        ref={focused ? highlightRef : undefined}
                        className={className}
                        onClick={() => {
                          if (seg.nodeIds.length > 0) focusNode(seg.nodeIds[0]);
                        }}
                      >
                        {seg.text}
                      </span>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Sidebar with focused node detail */}
            {focusedNode && (
              <aside className="border-l border-divider pl-6 max-h-[70vh] overflow-y-auto">
                <NodeDetailContent
                  iterationId={iterationId}
                  graph={graph.data}
                  node={focusedNode}
                  hideOpenTranscript
                />
              </aside>
            )}
          </div>
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}
