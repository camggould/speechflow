import {
  QueryClient,
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryResult,
} from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiFetch } from "@/api/client";
import type {
  CoverageMatrix,
  CoverageRow,
  Graph,
  IterationDetail,
  Session,
  SessionDetail,
  TimelineEvent,
  Transcript,
} from "@/api/types.gen";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 30,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

export const keys = {
  sessions: () => ["sessions"] as const,
  session: (id: string) => ["sessions", id] as const,
  sessionCoverage: (id: string) => ["sessions", id, "coverage"] as const,
  iteration: (id: string) => ["iterations", id] as const,
  graph: (id: string) => ["iterations", id, "graph"] as const,
  transcript: (id: string) => ["iterations", id, "transcript"] as const,
  timeline: (id: string) => ["iterations", id, "timeline"] as const,
  coverage: (id: string) => ["iterations", id, "coverage"] as const,
};

// --- Hooks ---

export function useSessions(): UseQueryResult<Session[]> {
  return useQuery({
    queryKey: keys.sessions(),
    queryFn: () => apiFetch<Session[]>("/sessions"),
  });
}

export function useSession(id: string): UseQueryResult<SessionDetail> {
  return useQuery({
    queryKey: keys.session(id),
    queryFn: () => apiFetch<SessionDetail>(`/sessions/${id}`),
    enabled: Boolean(id),
  });
}

export function useSessionCoverage(id: string): UseQueryResult<CoverageMatrix> {
  return useQuery({
    queryKey: keys.sessionCoverage(id),
    queryFn: () => apiFetch<CoverageMatrix>(`/sessions/${id}/coverage`),
    enabled: Boolean(id),
  });
}

export function useIteration(id: string): UseQueryResult<IterationDetail> {
  return useQuery({
    queryKey: keys.iteration(id),
    queryFn: () => apiFetch<IterationDetail>(`/iterations/${id}`),
    enabled: Boolean(id),
  });
}

/**
 * Subscribes to `document.visibilityState` so we can pause polling while the
 * tab is hidden — keeps the dev server (and a future backend) from getting
 * hammered when the UI is in a background tab.
 */
function useDocumentVisible(): boolean {
  const [visible, setVisible] = useState(
    typeof document === "undefined" ? true : document.visibilityState === "visible",
  );
  useEffect(() => {
    if (typeof document === "undefined") return;
    const onChange = () => setVisible(document.visibilityState === "visible");
    document.addEventListener("visibilitychange", onChange);
    return () => document.removeEventListener("visibilitychange", onChange);
  }, []);
  return visible;
}

/**
 * Graph poller. Refetches every 1s while the iteration is still active
 * (ended_at == null) AND the tab is visible. Pass `endedAt` from the parent
 * so we don't double-fetch the iteration just to know its state.
 */
export function useGraph(
  iterationId: string,
  endedAt: string | null | undefined,
): UseQueryResult<Graph> {
  const visible = useDocumentVisible();
  const live = endedAt == null;
  return useQuery({
    queryKey: keys.graph(iterationId),
    queryFn: () => apiFetch<Graph>(`/iterations/${iterationId}/graph`),
    enabled: Boolean(iterationId),
    refetchInterval: live && visible ? 1000 : false,
    refetchIntervalInBackground: false,
  });
}

export function useTranscript(iterationId: string): UseQueryResult<Transcript> {
  return useQuery({
    queryKey: keys.transcript(iterationId),
    queryFn: () => apiFetch<Transcript>(`/iterations/${iterationId}/transcript`),
    enabled: Boolean(iterationId),
  });
}

export function useTimeline(iterationId: string): UseQueryResult<TimelineEvent[]> {
  return useQuery({
    queryKey: keys.timeline(iterationId),
    queryFn: () => apiFetch<TimelineEvent[]>(`/iterations/${iterationId}/timeline`),
    enabled: Boolean(iterationId),
  });
}

export function useCoverage(iterationId: string): UseQueryResult<CoverageRow[]> {
  return useQuery({
    queryKey: keys.coverage(iterationId),
    queryFn: () => apiFetch<CoverageRow[]>(`/iterations/${iterationId}/coverage`),
    enabled: Boolean(iterationId),
  });
}

/**
 * Deletes an iteration via DELETE /iterations/:id. Invalidates the parent
 * session cache so its iteration list updates without a manual refetch.
 */
export function useDeleteIteration(sessionId?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (iterationId: string) =>
      apiFetch<void>(`/iterations/${iterationId}`, { method: "DELETE" }),
    onSuccess: () => {
      if (sessionId) {
        qc.invalidateQueries({ queryKey: keys.session(sessionId) });
        qc.invalidateQueries({ queryKey: keys.sessionCoverage(sessionId) });
      }
      qc.invalidateQueries({ queryKey: keys.sessions() });
    },
  });
}
