import type { ApolloClient } from '@apollo/client';
import type { NodeTypes } from '@xyflow/react';

import { useQuery, useSubscription } from '@apollo/client/react';
import { Crosshair } from 'lucide-react';
import { useCallback, useMemo } from 'react';

import type { ExplorationUpdatedSubscription } from '@/graphql/types';

import { ErrorState } from '@/components/shared/error-state';
import { LoadingState } from '@/components/shared/loading-state';
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty';
import { Progress } from '@/components/ui/progress';
import { ExplorationUpdatedDocument, FlowCoverageDocument, FlowExplorationGraphDocument } from '@/graphql/types';
import { useLatestRef } from '@/hooks/use-latest-ref';
import { cn } from '@/lib/utils';

import ExplorationNode, { type ExplorationFlowNode } from './exploration-node';
import GraphCanvas from './graph-canvas';
import { relationEdges } from './graph-edges';
import { layoutGraph } from './graph-layout';

const nodeTypes = { exploration: ExplorationNode } satisfies NodeTypes;

function ExplorationGraphView({ className, flowId }: { className?: string; flowId: string }) {
    const {
        data: graphData,
        error: graphError,
        loading: graphLoading,
        refetch: refetchGraph,
    } = useQuery(FlowExplorationGraphDocument, { variables: { flowId } });

    const {
        data: coverageData,
        error: coverageError,
        refetch: refetchCoverage,
    } = useQuery(FlowCoverageDocument, { variables: { flowId } });

    // The subscription `onData` fires from WebSocket messages long after the render that
    // registered it, so it reads the current flow and refetch through refs.
    const flowIdRef = useLatestRef(flowId);
    const refetchGraphRef = useLatestRef(refetchGraph);

    // The event carries the new coverage; the nodes and edges behind it are refetched.
    const onExplorationUpdated = useCallback(
        ({ client, data }: { client: ApolloClient; data: { data?: ExplorationUpdatedSubscription | null } }) => {
            const coverage = data.data?.explorationUpdated;

            if (!coverage) {
                return;
            }

            client.writeQuery({
                data: { flowCoverage: coverage },
                query: FlowCoverageDocument,
                variables: { flowId: flowIdRef.current },
            });
            void refetchGraphRef.current();
        },
        [flowIdRef, refetchGraphRef],
    );

    useSubscription(ExplorationUpdatedDocument, { onData: onExplorationUpdated, variables: { flowId } });

    const graph = graphData?.flowExplorationGraph;
    const nodesSource = useMemo(() => graph?.nodes ?? [], [graph]);
    const relations = useMemo(() => graph?.edges ?? [], [graph]);
    const coverage = coverageData?.flowCoverage;

    const nodes = useMemo<ExplorationFlowNode[]>(() => {
        const positions = layoutGraph(
            nodesSource.map((node) => ({ id: node.id })),
            relations.map((edge) => ({ source: edge.fromNodeId, target: edge.toNodeId })),
        );

        return nodesSource.map((node) => ({
            data: {
                anchorCount: node.anchors.length,
                description: node.description,
                kind: node.kind,
                nodeId: node.id,
                status: node.status,
                title: node.title,
            },
            id: node.id,
            position: positions.get(node.id) ?? { x: 0, y: 0 },
            type: 'exploration' as const,
        }));
    }, [nodesSource, relations]);

    const edges = useMemo(
        () =>
            relationEdges(
                relations.map((edge) => ({
                    id: `${edge.fromNodeId}-${edge.relation}-${edge.toNodeId}`,
                    label: edge.relation,
                    source: edge.fromNodeId,
                    target: edge.toNodeId,
                })),
            ),
        [relations],
    );

    const refetch = () => {
        void refetchGraph();
        void refetchCoverage();
    };

    if (graphLoading && !graph) {
        return (
            <div className={cn('flex h-full min-h-0 flex-col', className)}>
                <LoadingState
                    description="Please wait while we fetch the exploration graph"
                    title="Loading exploration..."
                />
            </div>
        );
    }

    if (graphError && !graph) {
        return (
            <div className={cn('flex h-full min-h-0 flex-col', className)}>
                <ErrorState
                    message={graphError.message}
                    onRetry={refetch}
                    title="Error loading the exploration graph"
                />
            </div>
        );
    }

    const completedPercent =
        coverage && coverage.totalNodes > 0 ? Math.round((coverage.completedNodes / coverage.totalNodes) * 100) : 0;

    return (
        <div className={cn('flex h-full min-h-0 flex-col gap-3', className)}>
            <div
                className="border-border flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border px-3 py-2"
                data-testid="flow-coverage"
            >
                <span className="text-xs font-medium">Coverage</span>
                <Progress
                    aria-label="Exploration coverage"
                    className="h-1.5 w-40"
                    value={completedPercent}
                />
                <span className="text-muted-foreground text-xs">
                    {coverage ? `${coverage.completedNodes}/${coverage.totalNodes}` : '—'} explored
                </span>
                <span className="text-muted-foreground text-xs">
                    {coverage ? `${coverage.anchoredAssets}/${coverage.totalAssets}` : '—'} anchored
                </span>
                {coverageError && !coverage ? (
                    <span className="text-destructive text-xs">Coverage unavailable</span>
                ) : null}
            </div>

            <div className="border-border min-h-0 flex-1 overflow-hidden rounded-lg border">
                <GraphCanvas
                    ariaLabel="Flow exploration graph"
                    edges={edges}
                    empty={
                        <Empty>
                            <EmptyHeader>
                                <EmptyMedia variant="icon">
                                    <Crosshair />
                                </EmptyMedia>
                                <EmptyTitle>No exploration recorded yet</EmptyTitle>
                                <EmptyDescription>
                                    Agents record targets, hypotheses and findings with the exploration_record tool as
                                    they work through this flow
                                </EmptyDescription>
                            </EmptyHeader>
                        </Empty>
                    }
                    nodes={nodes}
                    nodeTypes={nodeTypes}
                />
            </div>
        </div>
    );
}

export default ExplorationGraphView;
