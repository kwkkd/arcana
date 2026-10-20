package assets

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kwkkd/arcana/pkg/database"
)

type Service struct {
	db database.Querier
}

func NewService(db database.Querier) *Service {
	return &Service{db: db}
}

type Observation struct {
	Type       string
	Value      string
	Properties map[string]any
}

type AssetRelation struct {
	Source   Observation
	Target   Observation
	Relation string
}

type ObservedAsset struct {
	Asset   database.Asset
	Created bool
}

type ObserveResult struct {
	Company *database.AssetCompany
	Assets  []ObservedAsset
	Edges   []database.AssetEdge
}

// Observe records observations and relations under a company, creating the
// company if it is new. Every value is normalized and deduplicated against the
// company's existing assets, so the same call twice adds nothing the second
// time. A domain observation whose registrable domain differs from the value
// also links the value under its registrable domain.
func (s *Service) Observe(
	ctx context.Context, company, notes string, observations []Observation, relations []AssetRelation,
) (*ObserveResult, error) {
	normalized := make([]normalizedObservation, 0, len(observations))
	for _, o := range observations {
		n, err := normalize(o)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, n)
	}
	for _, r := range relations {
		if !ValidRelation(r.Relation) {
			return nil, fmt.Errorf("unsupported asset relation %q", r.Relation)
		}
	}

	entity, err := s.ensureCompany(ctx, company, notes)
	if err != nil {
		return nil, err
	}

	result := &ObserveResult{Company: entity}
	cache := make(map[string]ObservedAsset)

	ensured := make([]ObservedAsset, len(normalized))
	for i, n := range normalized {
		entry, err := s.ensureObservation(ctx, entity.ID, n, cache, result)
		if err != nil {
			return nil, err
		}
		ensured[i] = entry
	}

	for i, n := range normalized {
		if n.Type != TypeDomain {
			continue
		}
		parent := RegistrableDomain(n.Canonical)
		if parent == n.Canonical || parent == "" {
			continue
		}
		parentObservation := normalizedObservation{Type: TypeDomain, Canonical: parent, DedupKey: parent}
		parentAsset, err := s.ensureObservation(ctx, entity.ID, parentObservation, cache, result)
		if err != nil {
			return nil, err
		}
		if err := s.upsertEdge(ctx, ensured[i].Asset.ID, parentAsset.Asset.ID, RelationSubdomainOf, result); err != nil {
			return nil, err
		}
	}

	for _, r := range relations {
		source, err := normalize(r.Source)
		if err != nil {
			return nil, err
		}
		target, err := normalize(r.Target)
		if err != nil {
			return nil, err
		}
		sourceAsset, err := s.ensureObservation(ctx, entity.ID, source, cache, result)
		if err != nil {
			return nil, err
		}
		targetAsset, err := s.ensureObservation(ctx, entity.ID, target, cache, result)
		if err != nil {
			return nil, err
		}
		if err := s.upsertEdge(ctx, sourceAsset.Asset.ID, targetAsset.Asset.ID, r.Relation, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

type AnchorRef struct {
	Company    string
	Type       string
	Value      string
	Properties map[string]any
}

// RecordNode creates the exploration node for a title or returns the one
// already recorded for it, then anchors it to the assets it concerns. An
// anchor that names no company is resolved against the known assets and must
// match exactly one; an anchor that names one is created if it does not exist.
func (s *Service) RecordNode(
	ctx context.Context, flowID int64, kind, title, description, status string, anchors []AnchorRef,
) (*database.ExplorationNode, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("node title is empty")
	}
	if kind == "" {
		kind = KindTarget
	}
	if !ValidKind(kind) {
		return nil, fmt.Errorf("unsupported node kind %q", kind)
	}
	if status == "" {
		status = StatusPending
	}
	if !ValidStatus(status) {
		return nil, fmt.Errorf("unsupported node status %q", status)
	}

	node, err := s.db.CreateExplorationNode(ctx, database.CreateExplorationNodeParams{
		FlowID:      flowID,
		Kind:        kind,
		Title:       title,
		Description: description,
		Status:      status,
		DedupKey:    NodeDedupKey(title),
	})
	if err != nil {
		return nil, fmt.Errorf("create exploration node %q: %w", title, err)
	}

	for _, anchor := range anchors {
		asset, err := s.resolveAnchor(ctx, anchor)
		if err != nil {
			return nil, err
		}
		if err := s.db.AnchorExplorationNode(ctx, database.AnchorExplorationNodeParams{
			NodeID:  node.ID,
			AssetID: asset.ID,
		}); err != nil {
			return nil, fmt.Errorf("anchor node %d to asset %d: %w", node.ID, asset.ID, err)
		}
	}

	return &node, nil
}

func (s *Service) SetNodeStatus(ctx context.Context, flowID, nodeID int64, status, description string) (*database.ExplorationNode, error) {
	if !ValidStatus(status) {
		return nil, fmt.Errorf("unsupported node status %q", status)
	}
	node, err := s.db.UpdateExplorationNodeStatus(ctx, database.UpdateExplorationNodeStatusParams{
		FlowID:      flowID,
		ID:          nodeID,
		Status:      status,
		Description: description,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("node %d is not in flow %d", nodeID, flowID)
		}
		return nil, fmt.Errorf("update node %d: %w", nodeID, err)
	}
	return &node, nil
}

func (s *Service) RecordEdge(ctx context.Context, flowID, fromNodeID, toNodeID int64, relation string) (*database.ExplorationEdge, error) {
	if !ValidExplorationRelation(relation) {
		return nil, fmt.Errorf("unsupported exploration relation %q", relation)
	}
	if fromNodeID == toNodeID {
		return nil, errors.New("an exploration edge cannot connect a node to itself")
	}
	for _, id := range []int64{fromNodeID, toNodeID} {
		if _, err := s.db.GetExplorationNode(ctx, database.GetExplorationNodeParams{FlowID: flowID, ID: id}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("node %d is not in flow %d", id, flowID)
			}
			return nil, fmt.Errorf("get node %d: %w", id, err)
		}
	}
	edge, err := s.db.CreateExplorationEdge(ctx, database.CreateExplorationEdgeParams{
		FlowID:     flowID,
		FromNodeID: fromNodeID,
		ToNodeID:   toNodeID,
		Relation:   relation,
	})
	if err != nil {
		return nil, fmt.Errorf("create exploration edge: %w", err)
	}
	return &edge, nil
}

type FlowGraph struct {
	Nodes   []database.ExplorationNode
	Edges   []database.ExplorationEdge
	Anchors []database.GetExplorationAnchorsByFlowRow
}

func (s *Service) FlowGraph(ctx context.Context, flowID int64) (*FlowGraph, error) {
	nodes, err := s.db.GetExplorationNodesByFlow(ctx, flowID)
	if err != nil {
		return nil, fmt.Errorf("get exploration nodes of flow %d: %w", flowID, err)
	}
	edges, err := s.db.GetExplorationEdgesByFlow(ctx, flowID)
	if err != nil {
		return nil, fmt.Errorf("get exploration edges of flow %d: %w", flowID, err)
	}
	anchors, err := s.db.GetExplorationAnchorsByFlow(ctx, flowID)
	if err != nil {
		return nil, fmt.Errorf("get exploration anchors of flow %d: %w", flowID, err)
	}
	return &FlowGraph{Nodes: nodes, Edges: edges, Anchors: anchors}, nil
}

func (s *Service) Coverage(ctx context.Context, flowID int64) (Coverage, error) {
	row, err := s.db.GetFlowCoverage(ctx, flowID)
	if err != nil {
		return Coverage{}, fmt.Errorf("get flow %d coverage: %w", flowID, err)
	}
	return Coverage{
		TotalNodes:     row.TotalNodes,
		CompletedNodes: row.CompletedNodes,
		AnchoredAssets: row.AnchoredAssets,
		TotalAssets:    row.TotalAssets,
	}, nil
}

type AssetGraph struct {
	Company database.AssetCompany
	Assets  []database.Asset
	Edges   []database.AssetEdge
}

func (s *Service) Companies(ctx context.Context) ([]database.GetAssetCompaniesRow, error) {
	companies, err := s.db.GetAssetCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("get companies: %w", err)
	}
	return companies, nil
}

func (s *Service) AssetGraph(ctx context.Context, companyID int64) (*AssetGraph, error) {
	company, err := s.db.GetAssetCompany(ctx, companyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("company %d does not exist", companyID)
		}
		return nil, fmt.Errorf("get company %d: %w", companyID, err)
	}
	assets, err := s.Assets(ctx, companyID, "", "")
	if err != nil {
		return nil, err
	}
	edges, err := s.db.GetAssetEdgesByCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("get asset edges of company %d: %w", companyID, err)
	}
	return &AssetGraph{Company: company, Assets: assets, Edges: edges}, nil
}

// Assets filters a company's assets; an empty assetType or search matches
// everything.
func (s *Service) Assets(ctx context.Context, companyID int64, assetType, search string) ([]database.Asset, error) {
	if assetType != "" && !ValidType(assetType) {
		return nil, fmt.Errorf("unsupported asset type %q", assetType)
	}
	assets, err := s.db.GetAssetsFiltered(ctx, database.GetAssetsFilteredParams{
		CompanyID: companyID,
		Type:      database.StringToNullString(assetType),
		Search:    database.StringToNullString(strings.TrimSpace(search)),
	})
	if err != nil {
		return nil, fmt.Errorf("get assets of company %d: %w", companyID, err)
	}
	return assets, nil
}

type normalizedObservation struct {
	Type       string
	Canonical  string
	DedupKey   string
	Properties map[string]any
}

func normalize(o Observation) (normalizedObservation, error) {
	canonical, dedupKey, err := Normalize(o.Type, o.Value)
	if err != nil {
		return normalizedObservation{}, err
	}
	return normalizedObservation{Type: o.Type, Canonical: canonical, DedupKey: dedupKey, Properties: o.Properties}, nil
}

func assetKey(assetType, dedupKey string) string {
	return assetType + "|" + dedupKey
}

func (s *Service) ensureCompany(ctx context.Context, company, notes string) (*database.AssetCompany, error) {
	name, dedupKey, err := NormalizeCompanyName(company)
	if err != nil {
		return nil, err
	}
	entity, err := s.db.UpsertAssetCompany(ctx, database.UpsertAssetCompanyParams{
		Name:     name,
		DedupKey: dedupKey,
		Notes:    notes,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert company %q: %w", name, err)
	}
	return &entity, nil
}

func (s *Service) ensureObservation(
	ctx context.Context, companyID int64, n normalizedObservation,
	cache map[string]ObservedAsset, result *ObserveResult,
) (ObservedAsset, error) {
	key := assetKey(n.Type, n.DedupKey)
	if entry, ok := cache[key]; ok {
		return entry, nil
	}
	entry, err := s.upsertObservation(ctx, companyID, n)
	if err != nil {
		return ObservedAsset{}, err
	}
	cache[key] = entry
	result.Assets = append(result.Assets, entry)
	return entry, nil
}

func (s *Service) upsertObservation(ctx context.Context, companyID int64, n normalizedObservation) (ObservedAsset, error) {
	created := false
	_, err := s.db.GetAssetByDedupKey(ctx, database.GetAssetByDedupKeyParams{CompanyID: companyID, DedupKey: n.DedupKey})
	switch {
	case err == nil:
	case errors.Is(err, sql.ErrNoRows):
		created = true
	default:
		return ObservedAsset{}, fmt.Errorf("look up asset %q: %w", n.Canonical, err)
	}

	properties := json.RawMessage("{}")
	if len(n.Properties) > 0 {
		encoded, err := json.Marshal(n.Properties)
		if err != nil {
			return ObservedAsset{}, fmt.Errorf("encode properties of %q: %w", n.Canonical, err)
		}
		properties = encoded
	}

	asset, err := s.db.UpsertAsset(ctx, database.UpsertAssetParams{
		CompanyID:  companyID,
		Type:       n.Type,
		Value:      n.Canonical,
		DedupKey:   n.DedupKey,
		Properties: properties,
	})
	if err != nil {
		return ObservedAsset{}, fmt.Errorf("upsert asset %q: %w", n.Canonical, err)
	}
	return ObservedAsset{Asset: asset, Created: created}, nil
}

func (s *Service) upsertEdge(ctx context.Context, srcID, dstID int64, relation string, result *ObserveResult) error {
	edge, err := s.db.UpsertAssetEdge(ctx, database.UpsertAssetEdgeParams{
		SrcAssetID: srcID,
		DstAssetID: dstID,
		Relation:   relation,
	})
	if err != nil {
		return fmt.Errorf("upsert asset edge %d -[%s]-> %d: %w", srcID, relation, dstID, err)
	}
	result.Edges = append(result.Edges, edge)
	return nil
}

func (s *Service) resolveAnchor(ctx context.Context, anchor AnchorRef) (*database.Asset, error) {
	n, err := normalize(Observation{Type: anchor.Type, Value: anchor.Value, Properties: anchor.Properties})
	if err != nil {
		return nil, err
	}

	if anchor.Company != "" {
		company, err := s.ensureCompany(ctx, anchor.Company, "")
		if err != nil {
			return nil, err
		}
		entry, err := s.upsertObservation(ctx, company.ID, n)
		if err != nil {
			return nil, err
		}
		return &entry.Asset, nil
	}

	matches, err := s.db.GetAssetsByDedupKey(ctx, n.DedupKey)
	if err != nil {
		return nil, fmt.Errorf("look up asset %q: %w", n.Canonical, err)
	}
	switch len(matches) {
	case 1:
		return &matches[0], nil
	case 0:
		return nil, fmt.Errorf("asset %q (%s) is not observed under any company yet; observe it first or name its company", n.Canonical, n.Type)
	default:
		return nil, fmt.Errorf("asset %q (%s) exists under %d companies; name the one the node concerns", n.Canonical, n.Type, len(matches))
	}
}
