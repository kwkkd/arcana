-- name: UpsertAssetCompany :one
INSERT INTO asset_companies (name, dedup_key, notes)
VALUES ($1, $2, $3)
ON CONFLICT (dedup_key) DO UPDATE SET
  name = EXCLUDED.name,
  notes = CASE WHEN EXCLUDED.notes <> '' THEN EXCLUDED.notes ELSE asset_companies.notes END,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetAssetCompanyByDedupKey :one
SELECT * FROM asset_companies
WHERE dedup_key = $1;

-- name: GetAssetCompany :one
SELECT * FROM asset_companies
WHERE id = $1;

-- name: GetAssetCompanies :many
SELECT c.*, (SELECT count(*) FROM assets a WHERE a.company_id = c.id) AS asset_count
FROM asset_companies c
ORDER BY c.name;

-- The merge keeps keys this observation does not mention, and overwrites the ones
-- it does: an asset accumulates properties across the flows that observe it.
-- name: UpsertAsset :one
INSERT INTO assets (company_id, type, value, dedup_key, properties)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (company_id, dedup_key) DO UPDATE SET
  value = EXCLUDED.value,
  properties = assets.properties || EXCLUDED.properties,
  last_seen_at = CURRENT_TIMESTAMP,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetAssetByDedupKey :one
SELECT * FROM assets
WHERE company_id = $1 AND dedup_key = $2;

-- name: GetAssetsByDedupKey :many
SELECT * FROM assets
WHERE dedup_key = $1
ORDER BY id;

-- name: GetAssetsFiltered :many
SELECT *
FROM assets
WHERE company_id = sqlc.arg(company_id)
  AND (sqlc.narg(type)::text IS NULL OR type = sqlc.narg(type)::text)
  AND (sqlc.narg(search)::text IS NULL OR value ILIKE '%' || sqlc.narg(search)::text || '%')
ORDER BY type, value
LIMIT 1000;

-- name: UpsertAssetEdge :one
INSERT INTO asset_edges (src_asset_id, dst_asset_id, relation)
VALUES ($1, $2, $3)
ON CONFLICT (src_asset_id, dst_asset_id, relation) DO UPDATE SET
  last_seen_at = CURRENT_TIMESTAMP,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetAssetEdgesByCompany :many
SELECT e.* FROM asset_edges e
JOIN assets a ON a.id = e.src_asset_id
WHERE a.company_id = sqlc.arg(company_id)
ORDER BY e.id;

-- The conflict branch is deliberately a no-op: re-recording a node that exists
-- must not move its status back to the caller's default.
-- name: CreateExplorationNode :one
INSERT INTO exploration_nodes (flow_id, kind, title, description, status, dedup_key)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (flow_id, dedup_key) DO UPDATE SET
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetExplorationNode :one
SELECT * FROM exploration_nodes
WHERE flow_id = $1 AND id = $2;

-- name: UpdateExplorationNodeStatus :one
UPDATE exploration_nodes
SET status = sqlc.arg(status),
    description = CASE WHEN sqlc.arg(description)::text <> '' THEN sqlc.arg(description)::text ELSE description END
WHERE flow_id = sqlc.arg(flow_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: GetExplorationNodesByFlow :many
SELECT * FROM exploration_nodes
WHERE flow_id = $1
ORDER BY id;

-- name: CreateExplorationEdge :one
INSERT INTO exploration_edges (flow_id, from_node_id, to_node_id, relation)
VALUES ($1, $2, $3, $4)
ON CONFLICT (from_node_id, to_node_id, relation) DO UPDATE SET
  flow_id = EXCLUDED.flow_id
RETURNING *;

-- name: GetExplorationEdgesByFlow :many
SELECT * FROM exploration_edges
WHERE flow_id = $1
ORDER BY id;

-- name: AnchorExplorationNode :exec
INSERT INTO exploration_anchors (node_id, asset_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetExplorationAnchorsByFlow :many
SELECT an.node_id, a.* FROM exploration_anchors an
JOIN exploration_nodes n ON n.id = an.node_id
JOIN assets a ON a.id = an.asset_id
WHERE n.flow_id = $1
ORDER BY an.node_id, a.id;

-- name: GetFlowCoverage :one
SELECT
  (SELECT count(*) FROM exploration_nodes WHERE flow_id = sqlc.arg(flow_id)::bigint) AS total_nodes,
  (SELECT count(*) FROM exploration_nodes WHERE flow_id = sqlc.arg(flow_id)::bigint AND status IN ('explored', 'dead_end')) AS completed_nodes,
  (SELECT count(DISTINCT an.asset_id)
     FROM exploration_anchors an
     JOIN exploration_nodes n ON n.id = an.node_id
    WHERE n.flow_id = sqlc.arg(flow_id)::bigint) AS anchored_assets,
  (SELECT count(*)
     FROM assets a
    WHERE a.company_id IN (
      SELECT DISTINCT a2.company_id
        FROM exploration_anchors an
        JOIN exploration_nodes n ON n.id = an.node_id
        JOIN assets a2 ON a2.id = an.asset_id
       WHERE n.flow_id = sqlc.arg(flow_id)::bigint)) AS total_assets;
