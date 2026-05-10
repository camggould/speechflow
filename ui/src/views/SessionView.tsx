import { useSession } from "@/api/query";
import { useAppStore } from "@/store/app";
import { IterationListRail } from "@/components/IterationListRail";
import { CoverageMatrix } from "@/components/CoverageMatrix";

interface SessionViewProps {
  sessionId: string;
}

export function SessionView({ sessionId }: SessionViewProps) {
  const { data, isLoading, error } = useSession(sessionId);
  const matrixOpen = useAppStore((s) => s.coverageMatrixOpen);

  if (isLoading) {
    return <div className="p-6 text-sm text-default-500">Loading…</div>;
  }
  if (error || !data) {
    return (
      <div className="p-6 text-sm text-default-500">
        Couldn't load session. Check the backend is running.
      </div>
    );
  }

  return (
    <div className="h-full flex">
      <IterationListRail sessionId={sessionId} iterations={data.iterations} />
      <div className="flex-1 min-w-0 flex flex-col">
        <div className="px-6 py-4 border-b border-divider">
          <h1 className="text-xl font-bold">{data.title}</h1>
          {data.description && (
            <p className="text-sm text-default-500 mt-1">{data.description}</p>
          )}
        </div>
        <div className="flex-1 min-h-0">
          {matrixOpen ? (
            <CoverageMatrix sessionId={sessionId} />
          ) : (
            <div className="h-full flex items-center justify-center p-6 text-sm text-default-400">
              Select an iteration from the left to view its graph, or toggle the matrix.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
