import type { NodeTypes } from '@xyflow/react';

import { useQuery } from '@apollo/client/react';
import { Network } from 'lucide-react';
import { useMemo, useState } from 'react';

import { ErrorState } from '@/components/shared/error-state';
import { LoadingState } from '@/components/shared/loading-state';
import { Badge } from '@/components/ui/badge';
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty';
import { AssetCompaniesDocument, type AssetFragmentFragment, AssetGraphDocument } from '@/graphql/types';
import { cn } from '@/lib/utils';

import AssetNode, { type AssetFlowNode } from './asset-node';
import GraphCanvas from './graph-canvas';
import { relationEdges } from './graph-edges';
import { layoutGraph } from './graph-layout';

const nodeTypes = { asset: AssetNode } satisfies NodeTypes;

function AssetGraphView({ className }: { className?: string }) {
    const {
        data: companiesData,
        error: companiesError,
        loading: companiesLoading,
        refetch: refetchCompanies,
    } = useQuery(AssetCompaniesDocument);

    const companies = useMemo(() => companiesData?.assetCompanies ?? [], [companiesData]);
    const [selectedCompanyId, setSelectedCompanyId] = useState<null | string>(null);

    // The selection is derived rather than stored, so a deleted company falls back to the
    // first one on the next render instead of leaving the graph on a stale id.
    const companyId = companies.some((company) => company.id === selectedCompanyId)
        ? selectedCompanyId
        : (companies[0]?.id ?? null);

    const {
        data: graphData,
        error: graphError,
        loading: graphLoading,
        refetch: refetchGraph,
    } = useQuery(AssetGraphDocument, {
        skip: !companyId,
        variables: { companyId: companyId ?? '' },
    });

    const graph = graphData?.assetGraph;
    const assets = useMemo(() => graph?.assets ?? [], [graph]);
    const relations = useMemo(() => graph?.edges ?? [], [graph]);

    const nodes = useMemo<AssetFlowNode[]>(() => {
        const positions = layoutGraph(
            assets.map((asset) => ({ id: asset.id })),
            relations.map((edge) => ({ source: edge.srcAssetId, target: edge.dstAssetId })),
        );

        return assets.map((asset) => ({
            data: {
                assetId: asset.id,
                summary: summarizeProperties(asset),
                type: asset.type,
                value: asset.value,
            },
            id: asset.id,
            position: positions.get(asset.id) ?? { x: 0, y: 0 },
            type: 'asset' as const,
        }));
    }, [assets, relations]);

    const edges = useMemo(
        () =>
            relationEdges(
                relations.map((edge) => ({
                    id: `${edge.srcAssetId}-${edge.relation}-${edge.dstAssetId}`,
                    label: edge.relation,
                    source: edge.srcAssetId,
                    target: edge.dstAssetId,
                })),
            ),
        [relations],
    );

    if (companiesLoading && companies.length === 0) {
        return (
            <div className={cn('flex h-full min-h-0 flex-col p-4', className)}>
                <LoadingState
                    description="Please wait while we fetch the engagement companies"
                    title="Loading assets..."
                />
            </div>
        );
    }

    if (companiesError && companies.length === 0) {
        return (
            <div className={cn('flex h-full min-h-0 flex-col p-4', className)}>
                <ErrorState
                    message={companiesError.message}
                    onRetry={refetchCompanies}
                    title="Error loading engagement companies"
                />
            </div>
        );
    }

    if (companies.length === 0) {
        return (
            <div className={cn('flex h-full min-h-0 flex-col p-4', className)}>
                <Empty>
                    <EmptyHeader>
                        <EmptyMedia variant="icon">
                            <Network />
                        </EmptyMedia>
                        <EmptyTitle>No engagement assets yet</EmptyTitle>
                        <EmptyDescription>
                            Agents record companies and assets with the asset_observe tool as they work through a flow
                        </EmptyDescription>
                    </EmptyHeader>
                </Empty>
            </div>
        );
    }

    return (
        <div className={cn('flex h-full min-h-0 flex-col gap-4 md:flex-row', className)}>
            <div
                aria-label="Engagement companies"
                className="flex max-h-40 shrink-0 flex-col gap-1 overflow-y-auto md:max-h-none md:w-64"
                role="listbox"
            >
                {companies.map((company) => (
                    <button
                        aria-selected={company.id === companyId}
                        className={cn(
                            'hover:bg-accent hover:text-accent-foreground flex w-full flex-col gap-1 rounded-md border border-transparent px-3 py-2 text-left transition-colors',
                            company.id === companyId && 'bg-accent text-accent-foreground border-border',
                        )}
                        key={company.id}
                        onClick={() => setSelectedCompanyId(company.id)}
                        role="option"
                        type="button"
                    >
                        <span className="flex items-center gap-2">
                            <span className="truncate text-sm font-medium">{company.name}</span>
                            <Badge
                                className="ml-auto shrink-0"
                                variant="secondary"
                            >
                                {company.assetCount}
                            </Badge>
                        </span>
                        {company.notes ? (
                            <span className="text-foreground/80 truncate text-xs">{company.notes}</span>
                        ) : null}
                    </button>
                ))}
            </div>

            <div className="border-border min-h-0 flex-1 overflow-hidden rounded-lg border">
                {graphLoading && !graph ? (
                    <LoadingState
                        description="Please wait while we fetch the company graph"
                        title="Loading graph..."
                    />
                ) : graphError && !graph ? (
                    <ErrorState
                        message={graphError.message}
                        onRetry={refetchGraph}
                        title="Error loading the asset graph"
                    />
                ) : (
                    <GraphCanvas
                        ariaLabel={`Asset graph for ${graph?.company.name ?? 'the selected company'}`}
                        edges={edges}
                        empty={
                            <Empty>
                                <EmptyHeader>
                                    <EmptyMedia variant="icon">
                                        <Network />
                                    </EmptyMedia>
                                    <EmptyTitle>No assets for this company yet</EmptyTitle>
                                    <EmptyDescription>
                                        Observed assets and their relations will appear here
                                    </EmptyDescription>
                                </EmptyHeader>
                            </Empty>
                        }
                        nodes={nodes}
                        nodeTypes={nodeTypes}
                    />
                )}
            </div>
        </div>
    );
}

function summarizeProperties(asset: AssetFragmentFragment): string[] {
    if (typeof asset.properties !== 'object' || asset.properties === null) {
        return [];
    }

    return Object.entries(asset.properties)
        .filter((entry): entry is [string, boolean | number | string] => {
            const value = entry[1];

            return typeof value === 'boolean' || typeof value === 'number' || typeof value === 'string';
        })
        .slice(0, 3)
        .map(([key, value]) => `${key}: ${value}`);
}

export default AssetGraphView;
