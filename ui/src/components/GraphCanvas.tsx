import { useCallback, useEffect, useMemo } from "react";
import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  type Edge as RFEdge,
  type Node as RFNode,
  type NodeProps,
} from "@xyflow/react";
import dagre from "@dagrejs/dagre";
import { motion } from "framer-motion";
import type { Edge, Graph, Node } from "@/api/types.gen";
import { useAppStore } from "@/store/app";

const NODE_W = 200;
const NODE_H = 64;

const KIND_STYLES: Record<
  Node["kind"],
  { bg: string; ring: string; text: string; label: string }
> = {
  root_ref: {
    bg: "bg-amber-100 dark:bg-amber-900/40",
    ring: "ring-amber-400 dark:ring-amber-500",
    text: "text-amber-900 dark:text-amber-100",
    label: "root",
  },
  concept: {
    bg: "bg-blue-100 dark:bg-blue-900/40",
    ring: "ring-blue-400 dark:ring-blue-500",
    text: "text-blue-900 dark:text-blue-100",
    label: "concept",
  },
  curiosity: {
    bg: "bg-purple-100 dark:bg-purple-900/40",
    ring: "ring-purple-400 dark:ring-purple-500",
    text: "text-purple-900 dark:text-purple-100",
    label: "curiosity",
  },
};

type NodeData = {
  node: Node;
  opacity: number;
  blur: boolean;
  focused: boolean;
};

function dagreLayout(nodes: Node[], edges: Edge[]): Map<string, { x: number; y: number }> {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "TB", nodesep: 40, ranksep: 60 });
  g.setDefaultEdgeLabel(() => ({}));

  for (const n of nodes) {
    g.setNode(n.id, { width: NODE_W, height: NODE_H });
  }
  for (const e of edges) {
    if (g.hasNode(e.from_node) && g.hasNode(e.to_node)) {
      g.setEdge(e.from_node, e.to_node);
    }
  }

  dagre.layout(g);

  const positions = new Map<string, { x: number; y: number }>();
  for (const n of nodes) {
    const pos = g.node(n.id);
    if (pos) {
      // dagre returns centre coords; ReactFlow wants the top-left.
      positions.set(n.id, { x: pos.x - NODE_W / 2, y: pos.y - NODE_H / 2 });
    }
  }
  return positions;
}

function GraphNode(props: NodeProps) {
  const data = props.data as unknown as NodeData;
  const selected = props.selected;
  const { node, opacity, blur, focused } = data;
  const style = KIND_STYLES[node.kind];
  const hasKey = node.tags.includes("key");
  const hasTangent = node.tags.includes("tangent");
  const resolved = node.kind === "curiosity" && node.resolved_by_node_id != null;

  // Tag-driven border. `key` overrides `tangent` if both are set — `key`
  // is the stronger signal of "this is on-script".
  const borderStyle = hasKey
    ? "border-solid border-2"
    : hasTangent
    ? "border-dashed border-2"
    : "border border-default-300 dark:border-default-700";

  const ring = focused || selected ? `ring-2 ring-offset-2 ${style.ring}` : "";
  const effectiveOpacity = resolved ? Math.min(opacity, 0.4) : opacity;

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.92 }}
      animate={{
        opacity: effectiveOpacity,
        scale: 1,
        filter: blur ? "blur(2px)" : "blur(0px)",
      }}
      transition={{ duration: 0.35 }}
      className={`px-3 py-2 rounded-md ${style.bg} ${style.text} ${borderStyle} ${ring} shadow-sm`}
      style={{ width: NODE_W, minHeight: NODE_H }}
    >
      <div className="text-[10px] uppercase tracking-wider opacity-60">
        {style.label}
      </div>
      <div className="text-sm font-medium line-clamp-2">{node.title}</div>
      {node.tags.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-1">
          {node.tags
            .filter((t) => t !== "key" && t !== "tangent")
            .slice(0, 3)
            .map((t) => (
              <span
                key={t}
                className="text-[9px] px-1 rounded bg-default-200/60 dark:bg-default-700/60"
              >
                {t}
              </span>
            ))}
        </div>
      )}
    </motion.div>
  );
}

const nodeTypes = { sf: GraphNode };

interface GraphCanvasProps {
  graph: Graph;
  iterationStartedAt: string;
  iterationEndedAt: string | null;
}

function GraphCanvasInner({
  graph,
  iterationStartedAt,
  iterationEndedAt,
}: GraphCanvasProps) {
  const mode = useAppStore((s) => s.playback.mode);
  const cursor = useAppStore((s) => s.playback.cursor);
  const focusedNodeId = useAppStore((s) => s.focusedNodeId);
  const focusNode = useAppStore((s) => s.focusNode);

  // In live mode the cursor effectively == now. In playback we use the
  // explicit cursor stored in the store. Effective cursor is the threshold
  // beyond which nodes haven't "happened yet" from the viewer's perspective.
  const effectiveCursor = useMemo(() => {
    if (mode === "live") {
      return iterationEndedAt ?? new Date().toISOString();
    }
    return cursor;
  }, [mode, cursor, iterationEndedAt]);

  const layout = useMemo(
    () => dagreLayout(graph.nodes, graph.edges),
    [graph],
  );

  const rfNodes: RFNode[] = useMemo(() => {
    const cursorMs = new Date(effectiveCursor).getTime();
    return graph.nodes.map((n) => {
      const pos = layout.get(n.id) ?? { x: 0, y: 0 };
      const createdMs = new Date(n.created_at).getTime();

      // Fog-of-war: future nodes hidden entirely; nodes that arrived in the
      // last ~3s before the cursor get blurred + dimmed. Live mode short-
      // circuits to full opacity — we don't fog the present.
      let opacity = 1;
      let blur = false;
      if (mode === "playback") {
        if (createdMs > cursorMs) {
          opacity = 0;
        } else if (cursorMs - createdMs < 3000) {
          opacity = 0.65;
          blur = true;
        }
      }

      return {
        id: n.id,
        type: "sf",
        position: pos,
        data: {
          node: n,
          opacity,
          blur,
          focused: focusedNodeId === n.id,
        } satisfies NodeData,
        // Future nodes shouldn't take pointer events.
        selectable: opacity > 0,
        draggable: false,
      };
    });
  }, [graph.nodes, layout, effectiveCursor, mode, focusedNodeId]);

  const rfEdges: RFEdge[] = useMemo(() => {
    const cursorMs = new Date(effectiveCursor).getTime();
    const nodeById = new Map(graph.nodes.map((n) => [n.id, n]));
    return graph.edges
      .map((e): RFEdge | null => {
        const fromNode = nodeById.get(e.from_node);
        const toNode = nodeById.get(e.to_node);
        if (!fromNode || !toNode) return null;
        const createdMs = new Date(e.created_at).getTime();
        const future = mode === "playback" && createdMs > cursorMs;
        const isResolveLink =
          toNode.kind === "curiosity" &&
          toNode.resolved_by_node_id === fromNode.id;

        return {
          id: e.id,
          source: e.from_node,
          target: e.to_node,
          animated: e.kind === "returns_to",
          style: {
            opacity: future ? 0 : 1,
            strokeDasharray:
              isResolveLink || e.kind === "references" ? "4 4" : undefined,
          },
        };
      })
      .filter((e): e is RFEdge => e !== null);
  }, [graph.edges, graph.nodes, mode, effectiveCursor]);

  // When iteration timestamps change (new iteration loaded) reset the
  // playback cursor to the iteration's start so the scrubber lines up.
  useEffect(() => {
    if (mode === "playback") {
      useAppStore.setState((s) => {
        const startMs = new Date(iterationStartedAt).getTime();
        const cursorMs = new Date(s.playback.cursor).getTime();
        const endMs = iterationEndedAt
          ? new Date(iterationEndedAt).getTime()
          : Date.now();
        if (cursorMs < startMs || cursorMs > endMs) {
          return { playback: { ...s.playback, cursor: iterationStartedAt } };
        }
        return s;
      });
    }
  }, [iterationStartedAt, iterationEndedAt, mode]);

  const onNodeClick = useCallback(
    (_: unknown, node: RFNode) => {
      focusNode(node.id);
    },
    [focusNode],
  );

  if (graph.nodes.length === 0) {
    return (
      <div className="h-full w-full flex items-center justify-center text-default-400 text-sm">
        No nodes yet. The agent will add concepts as the iteration progresses.
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={rfNodes}
      edges={rfEdges}
      nodeTypes={nodeTypes}
      fitView
      proOptions={{ hideAttribution: true }}
      onNodeClick={onNodeClick}
      nodesDraggable={false}
      nodesConnectable={false}
      elementsSelectable
    >
      <Background />
      <Controls showInteractive={false} />
    </ReactFlow>
  );
}

export function GraphCanvas(props: GraphCanvasProps) {
  return (
    <ReactFlowProvider>
      <GraphCanvasInner {...props} />
    </ReactFlowProvider>
  );
}
