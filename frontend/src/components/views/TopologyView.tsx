import React, { useMemo } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  MarkerType,
  Node,
  Edge,
  BackgroundVariant,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import { Asset, Finding } from '../../types';
import { useWorkspaceStore } from '../../store/useWorkspaceStore';
import { Shield, Globe, Server, AlertTriangle } from 'lucide-react';

interface TopologyViewProps {
  rootDomain?: string;
  assets: Asset[];
  findings: Finding[];
}

const nodeWidth = 200;
const nodeHeight = 50;

function getLayoutedElements(nodes: Node[], edges: Edge[]) {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: 'LR', ranksep: 80, nodesep: 30 });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      },
    };
  });

  return { nodes: layoutedNodes, edges };
}

export const TopologyView: React.FC<TopologyViewProps> = ({
  rootDomain = 'acme.org',
  assets,
  findings,
}) => {
  const { setSelectedAsset, setSelectedFinding } = useWorkspaceStore();

  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => {
    const nodes: Node[] = [];
    const edges: Edge[] = [];

    // 1. Root Domain Node
    const rootId = 'root-node';
    nodes.push({
      id: rootId,
      data: {
        label: (
          <div className="flex items-center gap-2 font-mono text-xs font-bold text-signal-accent">
            <Shield className="w-4 h-4 shrink-0" />
            <span>{rootDomain}</span>
          </div>
        ),
      },
      position: { x: 0, y: 0 },
      style: {
        background: '#0D0F14',
        border: '1px solid #38BDF8',
        borderRadius: '4px',
        padding: '10px 14px',
        boxShadow: '0 0 16px rgba(56, 189, 248, 0.15)',
      },
    });

    // 2. Subdomain Nodes (Level 1)
    assets.slice(0, 10).forEach((a) => {
      const assetNodeId = `asset-${a.id}`;
      const hostFindings = findings.filter((f) => f.fqdn === a.fqdn);
      const hasCritical = hostFindings.some((f) => f.severity === 'critical');

      nodes.push({
        id: assetNodeId,
        data: {
          label: (
            <div className="flex flex-col text-left font-mono text-[11px]">
              <div className="flex items-center gap-1.5 font-semibold text-typography-primary truncate">
                <Globe className="w-3 h-3 text-signal-accent shrink-0" />
                <span className="truncate">{a.fqdn}</span>
              </div>
              <div className="text-[10px] text-typography-muted truncate">
                HTTP {a.http_status || 'TCP'} • {a.open_ports?.slice(0, 3).join(', ')}
              </div>
            </div>
          ),
          asset: a,
        },
        position: { x: 0, y: 0 },
        style: {
          background: '#10121A',
          border: hasCritical ? '1px solid #F43F5E' : '1px solid #1E222D',
          borderRadius: '4px',
          padding: '8px 10px',
        },
      });

      edges.push({
        id: `e-${rootId}-${assetNodeId}`,
        source: rootId,
        target: assetNodeId,
        style: { stroke: '#333A4D', strokeWidth: 1.5 },
      });

      // 3. Open Ports / Services (Level 2)
      a.open_ports?.slice(0, 2).forEach((port) => {
        const portNodeId = `port-${a.id}-${port}`;
        const isCritical = port === 6379 || port === 22 || port === 3306;

        nodes.push({
          id: portNodeId,
          data: {
            label: (
              <div className="flex items-center gap-1 font-mono text-[10px]">
                <Server className="w-2.5 h-2.5 text-typography-muted" />
                <span className={isCritical ? 'text-signal-critical font-bold' : 'text-typography-secondary'}>
                  {port}/tcp
                </span>
              </div>
            ),
          },
          position: { x: 0, y: 0 },
          style: {
            background: '#0D0F14',
            border: isCritical ? '1px solid #F43F5E' : '1px solid #1E222D',
            borderRadius: '2px',
            padding: '4px 8px',
          },
        });

        edges.push({
          id: `e-${assetNodeId}-${portNodeId}`,
          source: assetNodeId,
          target: portNodeId,
          style: { stroke: '#1E222D', strokeWidth: 1 },
        });
      });

      // 4. Vulnerability Findings Nodes (Level 3)
      hostFindings.forEach((f) => {
        const findingNodeId = `finding-${f.id}`;
        nodes.push({
          id: findingNodeId,
          data: {
            label: (
              <div className="flex items-center gap-1.5 font-mono text-[10px] text-signal-critical">
                <AlertTriangle className="w-3 h-3 shrink-0" />
                <span className="font-semibold truncate max-w-[140px]">{f.title}</span>
              </div>
            ),
            finding: f,
          },
          position: { x: 0, y: 0 },
          style: {
            background: '#2A0C14',
            border: '1px solid #F43F5E',
            borderRadius: '3px',
            padding: '6px 8px',
            boxShadow: '0 0 10px rgba(244, 63, 94, 0.2)',
          },
        });

        edges.push({
          id: `e-${assetNodeId}-${findingNodeId}`,
          source: assetNodeId,
          target: findingNodeId,
          style: { stroke: '#F43F5E', strokeWidth: 1.5 },
          markerEnd: { type: MarkerType.ArrowClosed, color: '#F43F5E' },
        });
      });
    });

    return getLayoutedElements(nodes, edges);
  }, [rootDomain, assets, findings]);

  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, , onEdgesChange] = useEdgesState(initialEdges);

  return (
    <div className="flex-1 w-full h-full bg-carbon-canvas relative">
      <div className="absolute top-3 left-4 z-10 px-3 py-1.5 rounded-micro bg-carbon-panel/90 border border-carbon-border text-xs font-mono text-typography-secondary flex items-center gap-2">
        <span className="w-2 h-2 rounded-full bg-signal-accent animate-pulse" />
        <span>Root Boundary: <b className="text-typography-primary">{rootDomain}</b></span>
        <span className="text-typography-dim">•</span>
        <span>Graph Hierarchy: Target → Subdomains → Open Ports → Findings</span>
      </div>

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={(_, node) => {
          if (node.data?.asset) {
            setSelectedAsset(node.data.asset as Asset);
          }
          if (node.data?.finding) {
            setSelectedFinding(node.data.finding as Finding);
          }
        }}
        fitView
        className="bg-carbon-canvas"
      >
        <Background variant={BackgroundVariant.Dots} gap={16} size={1} color="#1E222D" />
        <Controls className="!bg-carbon-panel !border-carbon-border !text-typography-primary" />
      </ReactFlow>
    </div>
  );
};
