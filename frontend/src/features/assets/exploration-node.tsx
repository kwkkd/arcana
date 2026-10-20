import type { Node, NodeProps } from '@xyflow/react';

import { Handle, Position } from '@xyflow/react';
import { Anchor, Crosshair, Flag, Lightbulb, StickyNote, Zap } from 'lucide-react';

import type { BadgeVariant } from '@/components/ui/badge';

import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

const kindIcons: Record<string, typeof Crosshair> = {
    action: Zap,
    finding: Flag,
    hypothesis: Lightbulb,
    note: StickyNote,
    target: Crosshair,
};

export const statusVariants: Record<string, BadgeVariant> = {
    active: 'blue',
    blocked: 'yellow',
    dead_end: 'red',
    explored: 'green',
    pending: 'outline',
};

export type ExplorationFlowNode = Node<ExplorationNodeFields, 'exploration'>;

export interface ExplorationNodeFields extends Record<string, unknown> {
    anchorCount: number;
    description: string;
    kind: string;
    nodeId: string;
    status: string;
    title: string;
}

function ExplorationNode({ data, selected }: NodeProps<ExplorationFlowNode>) {
    const Icon = kindIcons[data.kind] ?? StickyNote;

    return (
        <div
            className={cn(
                'border-border bg-card text-card-foreground w-64 rounded-lg border shadow-sm transition-shadow',
                selected && 'ring-primary ring-2',
            )}
            data-testid={`exploration-node-${data.nodeId}`}
        >
            <Handle
                position={Position.Left}
                style={{ opacity: 0 }}
                type="target"
            />
            <div className="flex items-center gap-2 border-b px-3 py-2">
                <Icon className="text-muted-foreground size-3.5 shrink-0" />
                <Badge
                    className="shrink-0"
                    variant={statusVariants[data.status] ?? 'outline'}
                >
                    {data.status.replace('_', ' ')}
                </Badge>
                <span className="text-muted-foreground ml-auto truncate text-[10px]">{data.kind}</span>
            </div>
            <div className="space-y-1 px-3 py-2">
                <div
                    className="line-clamp-2 text-xs font-medium"
                    title={data.title}
                >
                    {data.title}
                </div>
                {data.description ? (
                    <div
                        className="text-muted-foreground line-clamp-2 text-[10px]"
                        title={data.description}
                    >
                        {data.description}
                    </div>
                ) : null}
                {data.anchorCount > 0 ? (
                    <div className="text-muted-foreground flex items-center gap-1 pt-0.5 text-[10px]">
                        <Anchor className="size-3" />
                        {data.anchorCount === 1 ? '1 anchor' : `${data.anchorCount} anchors`}
                    </div>
                ) : null}
            </div>
            <Handle
                position={Position.Right}
                style={{ opacity: 0 }}
                type="source"
            />
        </div>
    );
}

export default ExplorationNode;
