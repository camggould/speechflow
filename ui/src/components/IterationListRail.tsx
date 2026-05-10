import { useState } from "react";
import { Link, useLocation } from "wouter";
import {
  Button,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
} from "@heroui/react";
import { Trash2, LayoutGrid } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import type { Iteration } from "@/api/types.gen";
import { useDeleteIteration } from "@/api/query";
import { useAppStore } from "@/store/app";
import { ROUTES } from "@/routes";

interface IterationListRailProps {
  sessionId: string;
  iterations: Iteration[];
  selectedIterationId?: string;
}

export function IterationListRail({
  sessionId,
  iterations,
  selectedIterationId,
}: IterationListRailProps) {
  const matrixOpen = useAppStore((s) => s.coverageMatrixOpen);
  const setMatrixOpen = useAppStore((s) => s.setCoverageMatrixOpen);
  const [confirmDelete, setConfirmDelete] = useState<Iteration | null>(null);
  const deleteIteration = useDeleteIteration(sessionId);
  const [, navigate] = useLocation();

  // Newest first by started_at.
  const sorted = [...iterations].sort(
    (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime(),
  );

  return (
    <aside className="w-72 border-r border-divider flex flex-col">
      <div className="p-3 border-b border-divider flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-default-500">
          Iterations
        </h3>
        <Button
          size="sm"
          variant={matrixOpen ? "solid" : "flat"}
          color={matrixOpen ? "primary" : "default"}
          onPress={() => setMatrixOpen(!matrixOpen)}
          startContent={<LayoutGrid size={12} />}
          className="h-7 text-xs"
        >
          Matrix
        </Button>
      </div>
      <div className="flex-1 overflow-y-auto">
        {sorted.length === 0 && (
          <div className="text-xs text-default-400 p-3">
            No iterations yet. Run <code>speechflow iteration start</code>.
          </div>
        )}
        {sorted.map((it) => {
          const selected = it.id === selectedIterationId;
          const active = it.ended_at == null;
          return (
            <div
              key={it.id}
              className={`group border-b border-divider/50 ${
                selected ? "bg-primary-50 dark:bg-primary-900/20" : ""
              }`}
            >
              <Link
                href={ROUTES.iteration(sessionId, it.id)}
                className="block px-3 py-2 hover:bg-default-100 dark:hover:bg-default-800/40"
              >
                <div className="flex items-center gap-2">
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium truncate">
                      {it.title}
                    </div>
                    <div className="text-[10px] text-default-400 flex items-center gap-2">
                      <span>
                        {formatDistanceToNow(new Date(it.started_at), {
                          addSuffix: true,
                        })}
                      </span>
                      {active && (
                        <span className="text-success-600 dark:text-success-400">
                          ● live
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="text-xs tabular-nums text-default-500">
                    {Math.round((it.coverage_pct ?? 0) * 100)}%
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      setConfirmDelete(it);
                    }}
                    className="opacity-0 group-hover:opacity-100 text-danger-500 p-1"
                    aria-label="Delete iteration"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              </Link>
            </div>
          );
        })}
      </div>

      <Modal
        isOpen={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
      >
        <ModalContent>
          <ModalHeader>Delete iteration?</ModalHeader>
          <ModalBody>
            <p className="text-sm">
              This will permanently delete the iteration "
              <strong>{confirmDelete?.title}</strong>" and all its nodes, edges,
              and transcript. This cannot be undone.
            </p>
          </ModalBody>
          <ModalFooter>
            <Button variant="light" onPress={() => setConfirmDelete(null)}>
              Cancel
            </Button>
            <Button
              color="danger"
              isLoading={deleteIteration.isPending}
              onPress={() => {
                if (!confirmDelete) return;
                const id = confirmDelete.id;
                deleteIteration.mutate(id, {
                  onSuccess: () => {
                    setConfirmDelete(null);
                    // If we just deleted the selected iteration, bounce back
                    // to the session view so the URL stays valid.
                    if (id === selectedIterationId) {
                      navigate(ROUTES.session(sessionId));
                    }
                  },
                });
              }}
            >
              Delete
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </aside>
  );
}
