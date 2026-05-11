import { useEffect, useMemo, useRef } from "react";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
} from "@heroui/react";
import { useGraph, useIteration, useTranscript } from "@/api/query";
import { useAppStore } from "@/store/app";
import type { Transcript } from "@/api/types.gen";
import { NodeDetailContent } from "@/components/NodeDetailContent";

interface TranscriptModalProps {
  iterationId: string;
}

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
                    const highlighted =
                      focusedNodeId != null && seg.nodeIds.includes(focusedNodeId);
                    const hasAny = seg.nodeIds.length > 0;
                    const className = highlighted
                      ? "bg-warning-200/60 dark:bg-warning-700/40 rounded px-0.5 cursor-pointer"
                      : hasAny
                      ? "bg-default-100 dark:bg-default-800/50 cursor-pointer hover:bg-default-200 dark:hover:bg-default-700"
                      : "";
                    return (
                      <span
                        key={i}
                        ref={highlighted ? highlightRef : undefined}
                        className={className}
                        onClick={() => {
                          if (hasAny) focusNode(seg.nodeIds[0]);
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
