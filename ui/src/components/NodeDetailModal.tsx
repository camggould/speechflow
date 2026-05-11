import { useMemo } from "react";
import { Modal, ModalContent, ModalBody } from "@heroui/react";
import { useAppStore } from "@/store/app";
import type { Graph } from "@/api/types.gen";
import { NodeDetailContent } from "@/components/NodeDetailContent";

interface NodeDetailModalProps {
  iterationId: string;
  graph: Graph | undefined;
}

// Standalone node detail modal. Only opens when the transcript modal is
// closed — when the transcript is open, the same detail content renders
// inside that modal's sidebar to avoid stacking modals.
export function NodeDetailModal({ iterationId, graph }: NodeDetailModalProps) {
  const focusedNodeId = useAppStore((s) => s.focusedNodeId);
  const focusNode = useAppStore((s) => s.focusNode);
  const transcriptOpen = useAppStore((s) => s.transcriptOpen);
  const openTranscript = useAppStore((s) => s.openTranscript);

  const node = useMemo(
    () => graph?.nodes.find((n) => n.id === focusedNodeId) ?? null,
    [graph, focusedNodeId],
  );

  const open = focusedNodeId != null && node != null && !transcriptOpen;

  return (
    <Modal
      isOpen={open}
      onClose={() => focusNode(null)}
      size="3xl"
      scrollBehavior="inside"
    >
      <ModalContent>
        {node && (
          <ModalBody className="py-6">
            <NodeDetailContent
              iterationId={iterationId}
              graph={graph}
              node={node}
              onOpenTranscript={openTranscript}
            />
          </ModalBody>
        )}
      </ModalContent>
    </Modal>
  );
}
