import type { Node, NodeProps } from '@xyflow/react';

import { Handle, Position } from '@xyflow/react';
import { Boxes, Cog, Cpu, Globe, Link, Mail, Network, Server } from 'lucide-react';

import type { BadgeVariant } from '@/components/ui/badge';

import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

const assetIcons: Record<string, typeof Globe> = {
    cidr: Network,
    domain: Globe,
    email: Mail,
    ip: Server,
    service: Cog,
    technology: Cpu,
    url: Link,
};

const assetVariants: Record<string, BadgeVariant> = {
    cidr: 'purple',
    domain: 'blue',
    email: 'pink',
    ip: 'orange',
    service: 'yellow',
    technology: 'secondary',
    url: 'green',
};

export type AssetFlowNode = Node<AssetNodeFields, 'asset'>;

export interface AssetNodeFields extends Record<string, unknown> {
    assetId: string;
    summary: string[];
    type: string;
    value: string;
}

function AssetNode({ data, selected }: NodeProps<AssetFlowNode>) {
    const Icon = assetIcons[data.type] ?? Boxes;

    return (
        <div
            className={cn(
                'border-border bg-card text-card-foreground w-60 rounded-lg border shadow-sm transition-shadow',
                selected && 'ring-primary ring-2',
            )}
            data-testid={`asset-node-${data.assetId}`}
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
                    variant={assetVariants[data.type] ?? 'outline'}
                >
                    {data.type}
                </Badge>
            </div>
            <div className="space-y-1 px-3 py-2">
                <div
                    className="truncate font-mono text-xs"
                    title={data.value}
                >
                    {data.value}
                </div>
                {data.summary.map((line) => (
                    <div
                        className="text-muted-foreground truncate text-[10px]"
                        key={line}
                        title={line}
                    >
                        {line}
                    </div>
                ))}
            </div>
            <Handle
                position={Position.Right}
                style={{ opacity: 0 }}
                type="source"
            />
        </div>
    );
}

export default AssetNode;
