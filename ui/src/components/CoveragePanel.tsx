import { Check, X } from "lucide-react";
import { useCoverage } from "@/api/query";
import { useAppStore } from "@/store/app";

interface CoveragePanelProps {
  iterationId: string;
}

export function CoveragePanel({ iterationId }: CoveragePanelProps) {
  const { data, isLoading } = useCoverage(iterationId);
  const focusNode = useAppStore((s) => s.focusNode);

  return (
    <div className="h-full w-72 border-l border-divider flex flex-col">
      <div className="p-3 border-b border-divider">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-default-500">
          Coverage
        </h3>
      </div>
      <div className="flex-1 overflow-y-auto p-2">
        {isLoading && <div className="text-xs text-default-400 p-2">Loading…</div>}
        {!isLoading && (!data || data.length === 0) && (
          <div className="text-xs text-default-400 p-2">
            No roots declared. Use <code>speechflow root add</code>.
          </div>
        )}
        {data?.map((row) => (
          <button
            key={row.root_id}
            type="button"
            disabled={!row.covered || row.supporting_node_ids.length === 0}
            onClick={() => {
              if (row.supporting_node_ids.length > 0) {
                focusNode(row.supporting_node_ids[0]);
              }
            }}
            className="w-full text-left p-2 rounded hover:bg-default-100 dark:hover:bg-default-800/40 flex items-start gap-2 disabled:hover:bg-transparent"
          >
            <span
              className={`mt-0.5 ${row.covered ? "text-success" : "text-default-400"}`}
            >
              {row.covered ? <Check size={14} /> : <X size={14} />}
            </span>
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium truncate">{row.root_title}</div>
              {row.covered && row.first_touched_at && (
                <div className="text-[10px] text-default-400">
                  touched {new Date(row.first_touched_at).toLocaleTimeString()}
                </div>
              )}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
