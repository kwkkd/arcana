import type { Edge } from '@xyflow/react';

import { MarkerType } from '@xyflow/react';

export interface RelationedEdge {
    id: string;
    label: string;
    source: string;
    target: string;
}

export function relationEdges(edges: RelationedEdge[]): Edge[] {
    return edges.map((edge) => ({
        id: edge.id,
        label: edge.label,
        labelBgPadding: [2, 1] as [number, number],
        labelBgStyle: { fill: 'var(--card)', fillOpacity: 0.9 },
        labelStyle: { fill: 'var(--muted-foreground)', fontSize: 9 },
        markerEnd: { height: 14, type: MarkerType.ArrowClosed, width: 14 },
        source: edge.source,
        style: { stroke: 'var(--border)' },
        target: edge.target,
        type: 'smoothstep',
    }));
}
