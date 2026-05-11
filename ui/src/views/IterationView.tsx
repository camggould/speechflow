import { useMemo } from "react";
import { Button } from "@heroui/react";
import { FileText, Radio, History } from "lucide-react";
import { useGraph, useIteration, useSession } from "@/api/query";
import { useAppStore } from "@/store/app";
import { IterationListRail } from "@/components/IterationListRail";
import { GraphCanvas } from "@/components/GraphCanvas";
import { InsightsPanel } from "@/components/InsightsPanel";
import { TimelineScrubber } from "@/components/TimelineScrubber";
import { TranscriptModal } from "@/components/TranscriptModal";
import { NodeDetailModal } from "@/components/NodeDetailModal";
import { CoverageMatrix } from "@/components/CoverageMatrix";

interface IterationViewProps {
  sessionId: string;
  iterationId: string;
}

export function IterationView({ sessionId, iterationId }: IterationViewProps) {
  const session = useSession(sessionId);
  const iteration = useIteration(iterationId);
  const matrixOpen = useAppStore((s) => s.coverageMatrixOpen);
  const mode = useAppStore((s) => s.playback.mode);
  const setMode = useAppStore((s) => s.setMode);
  const openTranscript = useAppStore((s) => s.openTranscript);

  const graph = useGraph(iterationId, iteration.data?.ended_at ?? null);

  // Latest event timestamp across nodes + edges. Used by the scrubber to
  // bound an active iteration's playback window — we can't scrub past what
  // actually happened.
  const latestEventAt = useMemo(() => {
    if (!graph.data) return null;
    const all = [
      ...graph.data.nodes.map((n) => n.created_at),
      ...graph.data.edges.map((e) => e.created_at),
    ];
    if (all.length === 0) return null;
    return all.reduce((a, b) => (a > b ? a : b));
  }, [graph.data]);

  if (session.isLoading || iteration.isLoading) {
    return <div className="p-6 text-sm text-default-500">Loading…</div>;
  }
  if (session.error || !session.data || iteration.error || !iteration.data) {
    return (
      <div className="p-6 text-sm text-default-500">
        Couldn't load iteration. Check the backend is running.
      </div>
    );
  }

  const it = iteration.data;

  return (
    <div className="h-full flex min-h-0">
      <IterationListRail
        sessionId={sessionId}
        iterations={session.data.iterations}
        selectedIterationId={iterationId}
      />

      <div className="flex-1 min-w-0 flex flex-col min-h-0">
        {matrixOpen ? (
          <CoverageMatrix sessionId={sessionId} />
        ) : (
          <>
            <div className="px-4 py-2 border-b border-divider flex items-center gap-3">
              <div className="min-w-0 flex-1">
                <h1 className="text-sm font-semibold truncate">{it.title}</h1>
                <div className="text-[10px] text-default-400">
                  {it.ended_at ? "Ended" : "Live"} ·{" "}
                  {it.node_count} node{it.node_count === 1 ? "" : "s"} ·{" "}
                  {Math.round((it.coverage_pct ?? 0) * 100)}% coverage
                </div>
              </div>
              <Button
                variant="flat"
                size="sm"
                startContent={<FileText size={14} />}
                onPress={openTranscript}
              >
                Transcript
              </Button>
              <Button
                variant={mode === "live" ? "solid" : "flat"}
                color={mode === "live" ? "primary" : "default"}
                size="sm"
                startContent={<Radio size={14} />}
                onPress={() => setMode("live")}
              >
                Live
              </Button>
              <Button
                variant={mode === "playback" ? "solid" : "flat"}
                color={mode === "playback" ? "primary" : "default"}
                size="sm"
                startContent={<History size={14} />}
                onPress={() => setMode("playback")}
              >
                Playback
              </Button>
            </div>

            {mode === "playback" && (
              <div className="px-4 py-2 border-b border-divider flex items-center">
                <TimelineScrubber
                  startedAt={it.started_at}
                  endedAt={it.ended_at}
                  latestEventAt={latestEventAt}
                />
              </div>
            )}

            <div className="flex-1 min-h-0 flex">
              <div className="flex-1 min-w-0 min-h-0">
                {graph.isLoading && !graph.data && (
                  <div className="h-full flex items-center justify-center text-sm text-default-500">
                    Loading graph…
                  </div>
                )}
                {graph.data && (
                  <GraphCanvas
                    graph={graph.data}
                    iterationStartedAt={it.started_at}
                    iterationEndedAt={it.ended_at}
                  />
                )}
              </div>
              <InsightsPanel iterationId={iterationId} />
            </div>
          </>
        )}
      </div>

      <TranscriptModal iterationId={iterationId} />
      <NodeDetailModal iterationId={iterationId} graph={graph.data} />
    </div>
  );
}
