import { useLocation } from "wouter";
import { Check, X } from "lucide-react";
import { useSessionCoverage } from "@/api/query";
import { useAppStore } from "@/store/app";
import { ROUTES } from "@/routes";

interface CoverageMatrixProps {
  sessionId: string;
}

export function CoverageMatrix({ sessionId }: CoverageMatrixProps) {
  const { data, isLoading } = useSessionCoverage(sessionId);
  const focusNode = useAppStore((s) => s.focusNode);
  const [, navigate] = useLocation();

  if (isLoading) {
    return (
      <div className="p-6 text-sm text-default-500">Loading coverage…</div>
    );
  }

  if (!data || data.iterations.length === 0 || data.roots.length === 0) {
    return (
      <div className="p-6 text-sm text-default-400">
        No coverage to show yet. Declare roots with{" "}
        <code>speechflow root add</code> and start an iteration.
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto p-6">
      <h2 className="text-lg font-semibold mb-4">Coverage matrix</h2>
      <table className="border-collapse text-sm">
        <thead>
          <tr>
            <th className="text-left p-2 sticky left-0 bg-background z-10">
              Iteration
            </th>
            {data.roots.map((r) => (
              <th
                key={r.id}
                className="text-left p-2 align-bottom whitespace-nowrap"
              >
                <span className="block max-w-32 truncate">{r.title}</span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.iterations.map((it) => (
            <tr key={it.iteration_id} className="border-t border-divider/50">
              <td className="p-2 sticky left-0 bg-background z-10 max-w-48 truncate">
                {it.iteration_title}
              </td>
              {data.roots.map((r) => {
                const row = it.rows.find((x) => x.root_id === r.id);
                const covered = row?.covered ?? false;
                const supporting = row?.supporting_node_ids ?? [];
                return (
                  <td key={r.id} className="p-2">
                    <button
                      type="button"
                      disabled={!covered || supporting.length === 0}
                      onClick={() => {
                        if (supporting[0]) {
                          focusNode(supporting[0]);
                          navigate(
                            ROUTES.iteration(sessionId, it.iteration_id),
                          );
                        }
                      }}
                      className={`p-1 rounded ${
                        covered
                          ? "text-success hover:bg-success-100 dark:hover:bg-success-900/30"
                          : "text-default-300"
                      }`}
                      aria-label={covered ? "covered" : "not covered"}
                    >
                      {covered ? <Check size={16} /> : <X size={16} />}
                    </button>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
