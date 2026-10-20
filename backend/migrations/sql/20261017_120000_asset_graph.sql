-- +goose Up
-- +goose StatementBegin
INSERT INTO privileges (role_id, name) VALUES
  (1, 'assets.admin'),
  (1, 'assets.view'),
  (1, 'assets.subscribe'),
  (2, 'assets.view'),
  (2, 'assets.subscribe')
  ON CONFLICT DO NOTHING;

CREATE TABLE asset_companies (
  id         BIGINT      PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  name       TEXT        NOT NULL,
  dedup_key  TEXT        NOT NULL,
  notes      TEXT        NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT asset_companies_name_not_empty   CHECK (length(trim(name)) > 0),
  CONSTRAINT asset_companies_dedup_key_unique UNIQUE (dedup_key)
);

-- One asset row per company and normalized value: dedup_key is the type-specific
-- canonical form (lowercased FQDN, masked network, host:port/proto, ...) that
-- pkg/assets computes, so a re-observed value joins the row already there.
CREATE TABLE assets (
  id            BIGINT      PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  company_id    BIGINT      NOT NULL REFERENCES asset_companies(id) ON DELETE CASCADE,
  type          TEXT        NOT NULL,
  value         TEXT        NOT NULL,
  dedup_key     TEXT        NOT NULL,
  properties    JSONB       NOT NULL DEFAULT '{}'::jsonb,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT assets_type_supported            CHECK (type IN ('domain', 'ip', 'cidr', 'url', 'email', 'service', 'technology')),
  CONSTRAINT assets_value_not_empty           CHECK (length(trim(value)) > 0),
  CONSTRAINT assets_company_dedup_key_unique  UNIQUE (company_id, dedup_key)
);

CREATE INDEX assets_company_id_idx ON assets(company_id);
CREATE INDEX assets_type_idx ON assets(type);
CREATE INDEX assets_value_trgm_idx ON assets USING gin (value gin_trgm_ops);

CREATE TABLE asset_edges (
  id            BIGINT      PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  src_asset_id  BIGINT      NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  dst_asset_id  BIGINT      NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  relation      TEXT        NOT NULL,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT asset_edges_relation_supported CHECK (relation IN ('subdomain_of', 'resolves_to', 'member_of', 'serves', 'hosts', 'uses', 'same_as', 'related_to')),
  CONSTRAINT asset_edges_src_dst_relation_unique UNIQUE (src_asset_id, dst_asset_id, relation)
);

CREATE INDEX asset_edges_src_asset_id_idx ON asset_edges(src_asset_id);
CREATE INDEX asset_edges_dst_asset_id_idx ON asset_edges(dst_asset_id);

CREATE TABLE exploration_nodes (
  id          BIGINT      PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  flow_id     BIGINT      NOT NULL REFERENCES flows(id) ON DELETE CASCADE,
  kind        TEXT        NOT NULL DEFAULT 'target',
  title       TEXT        NOT NULL,
  description TEXT        NOT NULL DEFAULT '',
  status      TEXT        NOT NULL DEFAULT 'pending',
  dedup_key   TEXT        NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT exploration_nodes_kind_supported    CHECK (kind IN ('target', 'hypothesis', 'action', 'finding', 'note')),
  CONSTRAINT exploration_nodes_status_supported  CHECK (status IN ('pending', 'active', 'explored', 'dead_end', 'blocked')),
  CONSTRAINT exploration_nodes_title_not_empty   CHECK (length(trim(title)) > 0),
  CONSTRAINT exploration_nodes_flow_dedup_unique UNIQUE (flow_id, dedup_key)
);

CREATE INDEX exploration_nodes_flow_status_idx ON exploration_nodes(flow_id, status);

CREATE TABLE exploration_edges (
  id           BIGINT      PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
  flow_id      BIGINT      NOT NULL REFERENCES flows(id) ON DELETE CASCADE,
  from_node_id BIGINT      NOT NULL REFERENCES exploration_nodes(id) ON DELETE CASCADE,
  to_node_id   BIGINT      NOT NULL REFERENCES exploration_nodes(id) ON DELETE CASCADE,
  relation     TEXT        NOT NULL DEFAULT 'derives',
  created_at   TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT exploration_edges_relation_supported      CHECK (relation IN ('derives', 'tests', 'depends_on', 'supports')),
  CONSTRAINT exploration_edges_from_to_relation_unique UNIQUE (from_node_id, to_node_id, relation)
);

CREATE INDEX exploration_edges_flow_id_idx ON exploration_edges(flow_id);

CREATE TABLE exploration_anchors (
  node_id    BIGINT      NOT NULL REFERENCES exploration_nodes(id) ON DELETE CASCADE,
  asset_id   BIGINT      NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (node_id, asset_id)
);

CREATE INDEX exploration_anchors_asset_id_idx ON exploration_anchors(asset_id);

CREATE OR REPLACE TRIGGER update_asset_companies_modified
  BEFORE UPDATE ON asset_companies
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE OR REPLACE TRIGGER update_assets_modified
  BEFORE UPDATE ON assets
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE OR REPLACE TRIGGER update_asset_edges_modified
  BEFORE UPDATE ON asset_edges
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();

CREATE OR REPLACE TRIGGER update_exploration_nodes_modified
  BEFORE UPDATE ON exploration_nodes
  FOR EACH ROW EXECUTE PROCEDURE update_modified_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS exploration_anchors;
DROP TABLE IF EXISTS exploration_edges;
DROP TABLE IF EXISTS exploration_nodes;
DROP TABLE IF EXISTS asset_edges;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS asset_companies;

DELETE FROM privileges WHERE name IN ('assets.admin', 'assets.view', 'assets.subscribe');
-- +goose StatementEnd
