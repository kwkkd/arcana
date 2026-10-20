import type { ReactNode } from 'react';

import {
    Background,
    BackgroundVariant,
    Controls,
    type Edge,
    MiniMap,
    type Node,
    type NodeTypes,
    ReactFlow,
} from '@xyflow/react';
import { useEffect, useMemo, useRef } from 'react';

import { useTheme } from '@/hooks/use-theme';
import { cn } from '@/lib/utils';

import '@xyflow/react/dist/style.css';

interface GraphCanvasProps {
    ariaLabel: string;
    className?: string;
    edges: Edge[];
    empty?: ReactNode;
    nodes: Node[];
    nodeTypes: NodeTypes;
}

interface InteractiveFlowProps {
    colorMode: 'dark' | 'light' | 'system';
    edges: Edge[];
    nodes: Node[];
    nodeTypes: NodeTypes;
}

function GraphCanvas({ ariaLabel, className, edges, empty, nodes, nodeTypes }: GraphCanvasProps) {
    const { theme } = useTheme();

    // Remounting on a different node set re-runs fitView, which is what a company or flow
    // switch wants; a refetch of the same set keeps the operator's pan and zoom.
    const signature = useMemo(() => nodes.map((node) => node.id).join(','), [nodes]);

    const isEmpty = nodes.length === 0;

    return (
        <div
            aria-label={ariaLabel}
            className={cn('h-full w-full', className)}
            role="region"
        >
            {isEmpty ? (
                empty
            ) : (
                <InteractiveFlow
                    colorMode={theme}
                    edges={edges}
                    key={signature}
                    nodes={nodes}
                    nodeTypes={nodeTypes}
                />
            )}
        </div>
    );
}

function InteractiveFlow({ colorMode, edges, nodes, nodeTypes }: InteractiveFlowProps) {
    const containerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        // Style.css sets a light placeholder behind the canvas; the wrapper follows the theme.
        const container = containerRef.current;

        if (!container) {
            return;
        }

        container.style.setProperty('--xy-background-color', 'var(--background)');
    }, []);

    return (
        <div
            className="h-full w-full"
            ref={containerRef}
        >
            <ReactFlow
                colorMode={colorMode}
                edges={edges}
                fitView
                fitViewOptions={{ maxZoom: 1.2, padding: 0.2 }}
                minZoom={0.1}
                nodes={nodes}
                nodesConnectable={false}
                nodeTypes={nodeTypes}
            >
                <Background
                    gap={20}
                    size={1.5}
                    variant={BackgroundVariant.Dots}
                />
                <Controls
                    position="bottom-left"
                    showInteractive={false}
                />
                <MiniMap
                    pannable
                    position="bottom-right"
                    zoomable
                />
            </ReactFlow>
        </div>
    );
}

export default GraphCanvas;
