import { useCallback, useEffect, useMemo } from "react";
import {
  Background,
  Controls,
  Handle,
  MarkerType,
  Position,
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
  { bg: string; ring: string; text: string; label: string; stroke: string }
> = {
  root_ref: {
    bg: "bg-amber-100 dark:bg-amber-900/40",
    ring: "ring-amber-400 dark:ring-amber-500",
    text: "text-amber-900 dark:text-amber-100",
    label: "root",
    stroke: "#d97706",
  },
  concept: {
    bg: "bg-blue-100 dark:bg-blue-900/40",
    ring: "ring-blue-400 dark:ring-blue-500",
    text: "text-blue-900 dark:text-blue-100",
    label: "concept",
    stroke: "#2563eb",
  },
  curiosity: {
    bg: "bg-purple-100 dark:bg-purple-900/40",
    ring: "ring-purple-400 dark:ring-purple-500",
    text: "text-purple-900 dark:text-purple-100",
    label: "curiosity",
    stroke: "#9333ea",
  },
  takeaway: {
    bg: "bg-emerald-100 dark:bg-emerald-900/40",
    ring: "ring-emerald-400 dark:ring-emerald-500",
    text: "text-emerald-900 dark:text-emerald-100",
    label: "takeaway",
    stroke: "#059669",
  },
};

const EDGE_STROKE = {
  branches_from: "#64748b",
  references: "#94a3b8",
  returns_to: "#0891b2",
  supports: "#059669",
  contrasts: "#dc2626",
} as const;

type NodeData = {
  node: Node;
  opacity: number;
  blur: boolean;
  focused: boolean;
};

// Layout is a DAG with parents on top. We only feed branches_from edges to
// dagre (reversed so the parent is upstream). references/returns_to are
// peer/back-edges that would confuse the rank assignment, so we render them
// after layout but skip them here.
function dagreLayout(
  nodes: Node[],
  edges: Edge[],
): Map<string, { x: number; y: number }> {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "TB", nodesep: 60, ranksep: 80, marginx: 24, marginy: 24 });
  g.setDefaultEdgeLabel(() => ({}));

  for (const n of nodes) {
    g.setNode(n.id, { width: NODE_W, height: NODE_H });
  }
  for (const e of edges) {
    if (e.kind !== "branches_from") continue;
    if (g.hasNode(e.from_node) && g.hasNode(e.to_node)) {
      // branches_from stores from=child, to=parent. Reverse for layout so
      // parent ranks above child.
      g.setEdge(e.to_node, e.from_node);
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

  const borderStyle = hasKey
    ? "border-solid border-2"
    : hasTangent
    ? "border-dashed border-2"
    : "border border-default-300 dark:border-default-700";

  const ring = focused || selected ? `ring-2 ring-offset-2 ${style.ring}` : "";
  const effectiveOpacity = resolved ? Math.min(opacity, 0.55) : opacity;

  // Invisible handles anchor edges to the node's top (incoming) and
  // bottom (outgoing). Without these, React Flow has no SVG anchor for
  // an edge's endpoint and the line + arrowhead never render.
  const handleStyle = {
    width: 1,
    height: 1,
    minWidth: 0,
    minHeight: 0,
    background: "transparent",
    border: "none",
    opacity: 0,
  } as const;

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.92 }}
      animate={{
        opacity: effectiveOpacity,
        scale: 1,
        filter: blur ? "blur(2px)" : "blur(0px)",
      }}
      transition={{ duration: 0.35 }}
      className={`px-3 py-2 rounded-md ${style.bg} ${style.text} ${borderStyle} ${ring} shadow-sm cursor-pointer relative`}
      style={{ width: NODE_W, minHeight: NODE_H }}
    >
      <Handle
        type="target"
        position={Position.Top}
        style={handleStyle}
        isConnectable={false}
      />
      <Handle
        type="source"
        position={Position.Bottom}
        style={handleStyle}
        isConnectable={false}
      />
      <div className="text-[10px] uppercase tracking-wider opacity-60">
        {style.label}
        {resolved && " · resolved"}
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
                className="text-[9px] px-1 rounded bg-white/80 text-default-700 dark:bg-white/20 dark:text-default-700"
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
  const theme = useAppStore((s) => s.theme);

  // In live mode the effective cursor is the latest moment we care about,
  // so everything is visible. In playback we use the explicit store cursor.
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

      // Fog-of-war: in playback, future nodes are invisible and recently-
      // arrived nodes get a brief blur+dim to draw the eye.
      let opacity = 1;
      let blur = false;
      if (mode === "playback") {
        if (createdMs > cursorMs) {
          opacity = 0;
        } else if (cursorMs - createdMs < 600) {
          opacity = 0.7;
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

        // branches_from stores child → parent; render arrow parent → child.
        const [source, target] =
          e.kind === "branches_from"
            ? [e.to_node, e.from_node]
            : [e.from_node, e.to_node];

        const isResolveLink =
          e.kind === "branches_from" &&
          fromNode.kind === "curiosity" &&
          fromNode.resolved_by_node_id != null;

        const stroke = EDGE_STROKE[e.kind];
        const dashed =
          e.kind === "references" || e.kind === "contrasts" || isResolveLink;

        return {
          id: e.id,
          source,
          target,
          animated: e.kind === "returns_to",
          markerEnd: {
            type: MarkerType.ArrowClosed,
            width: 16,
            height: 16,
            color: stroke,
          },
          style: {
            opacity: future ? 0 : 1,
            stroke,
            strokeWidth: 1.5,
            strokeDasharray: dashed ? "5 4" : undefined,
          },
        };
      })
      .filter((e): e is RFEdge => e !== null);
  }, [graph.edges, graph.nodes, mode, effectiveCursor]);

  // When entering playback mode, snap the cursor to the iteration's start
  // so the user can scrub from the beginning. Otherwise the persisted
  // store cursor (potentially from a different iteration or live mode)
  // would leave the scrubber pointing at the wrong moment.
  useEffect(() => {
    if (mode === "playback") {
      useAppStore.setState((s) => ({
        playback: { ...s.playback, cursor: iterationStartedAt, playing: false },
      }));
    }
    // We only want this to fire on mode transition into playback or when
    // the iteration changes, not every cursor update.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [iterationStartedAt, mode]);

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
      fitViewOptions={{ padding: 0.2 }}
      proOptions={{ hideAttribution: true }}
      onNodeClick={onNodeClick}
      nodesDraggable={false}
      nodesConnectable={false}
      elementsSelectable
      colorMode={theme}
    >
      <Background />
      <Controls showInteractive={false} />
    </ReactFlow>
  );
}

export function GraphCanvas(props: GraphCanvasProps) {
  return (
    <div className="h-full w-full">
      <ReactFlowProvider>
        <GraphCanvasInner {...props} />
      </ReactFlowProvider>
    </div>
  );
}
