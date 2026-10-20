import type { ResultOf } from '@graphql-typed-document-node/core';

import type { AssetCompaniesDocument, AssetFragmentFragment, AssetGraphDocument } from '@/graphql/types';

import type { Cassette } from '../cassette.ts';

import { entity, mergeCassettes } from '../cassette.ts';
import { baseQueries, baseRest } from './base.ts';

export const ASSETS_COMPANY_ID = '3';
export const ASSETS_COMPANY_NAME = 'E2E Engagement Corp';

export const makeAsset = (
    id: string,
    type: string,
    value: string,
    properties: Record<string, unknown> = {},
): AssetFragmentFragment =>
    entity('Asset', {
        companyId: ASSETS_COMPANY_ID,
        firstSeenAt: '2026-01-02T10:00:00Z',
        id,
        lastSeenAt: '2026-01-02T11:30:00Z',
        properties,
        type,
        value,
    });

export const ASSET_PARENT_DOMAIN = makeAsset('11', 'domain', 'acme.example', { title: 'Corporate site' });
export const ASSET_CHILD_DOMAIN = makeAsset('12', 'domain', 'dev.acme.example');
export const ASSET_HOST = makeAsset('13', 'ip', '10.0.0.5');

const assetCompanies: ResultOf<typeof AssetCompaniesDocument> = {
    assetCompanies: [
        entity('AssetCompany', {
            assetCount: 3,
            id: ASSETS_COMPANY_ID,
            name: ASSETS_COMPANY_NAME,
            notes: 'Primary engagement target',
        }),
    ],
};

const assetGraph: ResultOf<typeof AssetGraphDocument> = {
    assetGraph: {
        assets: [ASSET_PARENT_DOMAIN, ASSET_CHILD_DOMAIN, ASSET_HOST],
        company: entity('AssetCompany', {
            assetCount: 3,
            id: ASSETS_COMPANY_ID,
            name: ASSETS_COMPANY_NAME,
            notes: 'Primary engagement target',
        }),
        // Child first, parent second: the same direction the tools write, so a client that
        // reverses it renders a reversed hierarchy, not the seeded one.
        edges: [
            entity('AssetEdge', {
                dstAssetId: ASSET_PARENT_DOMAIN.id,
                relation: 'subdomain_of',
                srcAssetId: ASSET_CHILD_DOMAIN.id,
            }),
            entity('AssetEdge', {
                dstAssetId: ASSET_HOST.id,
                relation: 'resolves_to',
                srcAssetId: ASSET_PARENT_DOMAIN.id,
            }),
        ],
    },
};

export const assetsCassette = (override: Cassette = {}): Cassette =>
    mergeCassettes(
        {
            queries: {
                ...baseQueries(),
                assetCompanies: [{ data: assetCompanies }],
                assetGraph: [{ data: assetGraph, variables: { companyId: ASSETS_COMPANY_ID } }],
            },
            rest: baseRest(),
        },
        override,
    );
