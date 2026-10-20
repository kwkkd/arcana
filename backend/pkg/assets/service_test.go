package assets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/kwkkd/arcana/pkg/database"
)

type fakeQuerier struct {
	database.Querier

	nextID    int64
	companies map[string]database.AssetCompany
	assets    map[string]database.Asset
	assetByID map[int64]database.Asset
	edges     map[string]database.AssetEdge
	nodes     map[string]database.ExplorationNode
	nodeByID  map[int64]database.ExplorationNode
	explEdges map[string]database.ExplorationEdge
	anchors   map[string]bool
	coverage  database.GetFlowCoverageRow
}

func newFakeQuerier() *fakeQuerier {
	return &fakeQuerier{
		companies: map[string]database.AssetCompany{},
		assets:    map[string]database.Asset{},
		assetByID: map[int64]database.Asset{},
		edges:     map[string]database.AssetEdge{},
		nodes:     map[string]database.ExplorationNode{},
		nodeByID:  map[int64]database.ExplorationNode{},
		explEdges: map[string]database.ExplorationEdge{},
		anchors:   map[string]bool{},
	}
}

func (f *fakeQuerier) UpsertAssetCompany(ctx context.Context, arg database.UpsertAssetCompanyParams) (database.AssetCompany, error) {
	if err := ctx.Err(); err != nil {
		return database.AssetCompany{}, err
	}
	if c, ok := f.companies[arg.DedupKey]; ok {
		c.Name = arg.Name
		if arg.Notes != "" {
			c.Notes = arg.Notes
		}
		f.companies[arg.DedupKey] = c
		return c, nil
	}
	f.nextID++
	c := database.AssetCompany{ID: f.nextID, Name: arg.Name, DedupKey: arg.DedupKey, Notes: arg.Notes}
	f.companies[arg.DedupKey] = c
	return c, nil
}

func (f *fakeQuerier) GetAssetCompany(ctx context.Context, id int64) (database.AssetCompany, error) {
	if err := ctx.Err(); err != nil {
		return database.AssetCompany{}, err
	}
	for _, c := range f.companies {
		if c.ID == id {
			return c, nil
		}
	}
	return database.AssetCompany{}, sql.ErrNoRows
}

func (f *fakeQuerier) GetAssetCompanies(ctx context.Context) ([]database.GetAssetCompaniesRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows := make([]database.GetAssetCompaniesRow, 0, len(f.companies))
	for _, c := range f.companies {
		row := database.GetAssetCompaniesRow{ID: c.ID, Name: c.Name, DedupKey: c.DedupKey, Notes: c.Notes}
		for _, a := range f.assetByID {
			if a.CompanyID == c.ID {
				row.AssetCount++
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

func (f *fakeQuerier) GetAssetByDedupKey(ctx context.Context, arg database.GetAssetByDedupKeyParams) (database.Asset, error) {
	if err := ctx.Err(); err != nil {
		return database.Asset{}, err
	}
	if a, ok := f.assets[assetKey(fmt.Sprint(arg.CompanyID), arg.DedupKey)]; ok {
		return a, nil
	}
	return database.Asset{}, sql.ErrNoRows
}

func (f *fakeQuerier) GetAssetsByDedupKey(ctx context.Context, dedupKey string) ([]database.Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []database.Asset
	for _, a := range f.assetByID {
		if a.DedupKey == dedupKey {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeQuerier) GetAssetsFiltered(ctx context.Context, arg database.GetAssetsFilteredParams) ([]database.Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []database.Asset
	for _, a := range f.assetByID {
		if a.CompanyID != arg.CompanyID {
			continue
		}
		if arg.Type.Valid && a.Type != arg.Type.String {
			continue
		}
		if arg.Search.Valid && !strings.Contains(a.Value, arg.Search.String) {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeQuerier) UpsertAsset(ctx context.Context, arg database.UpsertAssetParams) (database.Asset, error) {
	if err := ctx.Err(); err != nil {
		return database.Asset{}, err
	}
	key := assetKey(fmt.Sprint(arg.CompanyID), arg.DedupKey)
	if a, ok := f.assets[key]; ok {
		a.Value = arg.Value
		f.assets[key] = a
		f.assetByID[a.ID] = a
		return a, nil
	}
	f.nextID++
	a := database.Asset{
		ID:         f.nextID,
		CompanyID:  arg.CompanyID,
		Type:       arg.Type,
		Value:      arg.Value,
		DedupKey:   arg.DedupKey,
		Properties: arg.Properties,
	}
	f.assets[key] = a
	f.assetByID[a.ID] = a
	return a, nil
}

func (f *fakeQuerier) UpsertAssetEdge(ctx context.Context, arg database.UpsertAssetEdgeParams) (database.AssetEdge, error) {
	if err := ctx.Err(); err != nil {
		return database.AssetEdge{}, err
	}
	key := fmt.Sprintf("%d|%d|%s", arg.SrcAssetID, arg.DstAssetID, arg.Relation)
	if e, ok := f.edges[key]; ok {
		return e, nil
	}
	f.nextID++
	e := database.AssetEdge{
		ID:         f.nextID,
		SrcAssetID: arg.SrcAssetID,
		DstAssetID: arg.DstAssetID,
		Relation:   arg.Relation,
	}
	f.edges[key] = e
	return e, nil
}

func (f *fakeQuerier) GetAssetEdgesByCompany(ctx context.Context, companyID int64) ([]database.AssetEdge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []database.AssetEdge
	for _, e := range f.edges {
		if src, ok := f.assetByID[e.SrcAssetID]; ok && src.CompanyID == companyID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeQuerier) CreateExplorationNode(ctx context.Context, arg database.CreateExplorationNodeParams) (database.ExplorationNode, error) {
	if err := ctx.Err(); err != nil {
		return database.ExplorationNode{}, err
	}
	key := fmt.Sprintf("%d|%s", arg.FlowID, arg.DedupKey)
	if n, ok := f.nodes[key]; ok {
		return n, nil
	}
	f.nextID++
	n := database.ExplorationNode{
		ID:          f.nextID,
		FlowID:      arg.FlowID,
		Kind:        arg.Kind,
		Title:       arg.Title,
		Description: arg.Description,
		Status:      arg.Status,
		DedupKey:    arg.DedupKey,
	}
	f.nodes[key] = n
	f.nodeByID[n.ID] = n
	return n, nil
}

func (f *fakeQuerier) GetExplorationNode(ctx context.Context, arg database.GetExplorationNodeParams) (database.ExplorationNode, error) {
	if err := ctx.Err(); err != nil {
		return database.ExplorationNode{}, err
	}
	if n, ok := f.nodeByID[arg.ID]; ok && n.FlowID == arg.FlowID {
		return n, nil
	}
	return database.ExplorationNode{}, sql.ErrNoRows
}

func (f *fakeQuerier) UpdateExplorationNodeStatus(ctx context.Context, arg database.UpdateExplorationNodeStatusParams) (database.ExplorationNode, error) {
	if err := ctx.Err(); err != nil {
		return database.ExplorationNode{}, err
	}
	n, ok := f.nodeByID[arg.ID]
	if !ok || n.FlowID != arg.FlowID {
		return database.ExplorationNode{}, sql.ErrNoRows
	}
	n.Status = arg.Status
	if arg.Description != "" {
		n.Description = arg.Description
	}
	f.nodeByID[n.ID] = n
	f.nodes[fmt.Sprintf("%d|%s", n.FlowID, n.DedupKey)] = n
	return n, nil
}

func (f *fakeQuerier) GetExplorationNodesByFlow(ctx context.Context, flowID int64) ([]database.ExplorationNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []database.ExplorationNode
	for _, n := range f.nodes {
		if n.FlowID == flowID {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeQuerier) CreateExplorationEdge(ctx context.Context, arg database.CreateExplorationEdgeParams) (database.ExplorationEdge, error) {
	if err := ctx.Err(); err != nil {
		return database.ExplorationEdge{}, err
	}
	key := fmt.Sprintf("%d|%d|%s", arg.FromNodeID, arg.ToNodeID, arg.Relation)
	if e, ok := f.explEdges[key]; ok {
		return e, nil
	}
	f.nextID++
	e := database.ExplorationEdge{
		ID:         f.nextID,
		FlowID:     arg.FlowID,
		FromNodeID: arg.FromNodeID,
		ToNodeID:   arg.ToNodeID,
		Relation:   arg.Relation,
	}
	f.explEdges[key] = e
	return e, nil
}

func (f *fakeQuerier) GetExplorationEdgesByFlow(ctx context.Context, flowID int64) ([]database.ExplorationEdge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []database.ExplorationEdge
	for _, e := range f.explEdges {
		if e.FlowID == flowID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeQuerier) AnchorExplorationNode(ctx context.Context, arg database.AnchorExplorationNodeParams) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.anchors[fmt.Sprintf("%d|%d", arg.NodeID, arg.AssetID)] = true
	return nil
}

func (f *fakeQuerier) GetExplorationAnchorsByFlow(ctx context.Context, flowID int64) ([]database.GetExplorationAnchorsByFlowRow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []database.GetExplorationAnchorsByFlowRow
	for key := range f.anchors {
		var nodeID, assetID int64
		if _, err := fmt.Sscanf(key, "%d|%d", &nodeID, &assetID); err != nil {
			return nil, fmt.Errorf("bad anchor key %q: %w", key, err)
		}
		node, ok := f.nodeByID[nodeID]
		if !ok || node.FlowID != flowID {
			continue
		}
		asset := f.assetByID[assetID]
		out = append(out, database.GetExplorationAnchorsByFlowRow{
			NodeID:    nodeID,
			ID:        asset.ID,
			CompanyID: asset.CompanyID,
			Type:      asset.Type,
			Value:     asset.Value,
			DedupKey:  asset.DedupKey,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NodeID < out[j].NodeID })
	return out, nil
}

func (f *fakeQuerier) GetFlowCoverage(ctx context.Context, flowID int64) (database.GetFlowCoverageRow, error) {
	if err := ctx.Err(); err != nil {
		return database.GetFlowCoverageRow{}, err
	}
	return f.coverage, nil
}

func TestService_Observe_DeduplicatesSubdomainsUnderRegistrableParent(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	first, err := svc.Observe(ctx, "Acme Corp", "", []Observation{
		{Type: TypeDomain, Value: "Dev.Acme.com"},
		{Type: TypeDomain, Value: "acme.com"},
	}, nil)
	if err != nil {
		t.Fatalf("Observe returned an error: %v", err)
	}
	if len(first.Assets) != 2 {
		t.Fatalf("first Observe recorded %d assets; want 2", len(first.Assets))
	}
	if first.Assets[0].Asset.Value != "dev.acme.com" || first.Assets[1].Asset.Value != "acme.com" {
		t.Errorf("asset values = %q, %q; want dev.acme.com, acme.com", first.Assets[0].Asset.Value, first.Assets[1].Asset.Value)
	}
	if !first.Assets[0].Created || !first.Assets[1].Created {
		t.Error("first Observe did not report both assets as created")
	}
	if len(first.Edges) != 1 {
		t.Fatalf("first Observe recorded %d edges; want 1 subdomain_of edge", len(first.Edges))
	}
	edge := first.Edges[0]
	if edge.Relation != RelationSubdomainOf {
		t.Errorf("edge relation = %q; want %q", edge.Relation, RelationSubdomainOf)
	}
	if edge.SrcAssetID != first.Assets[0].Asset.ID || edge.DstAssetID != first.Assets[1].Asset.ID {
		t.Errorf("edge %d -> %d does not point from dev.acme.com to acme.com", edge.SrcAssetID, edge.DstAssetID)
	}

	second, err := svc.Observe(ctx, "acme corp", "", []Observation{{Type: TypeDomain, Value: "dev.acme.com"}}, nil)
	if err != nil {
		t.Fatalf("second Observe returned an error: %v", err)
	}
	if second.Company.ID != first.Company.ID {
		t.Errorf("company IDs differ between calls: %d and %d", first.Company.ID, second.Company.ID)
	}
	for _, a := range second.Assets {
		if a.Created {
			t.Errorf("second Observe reported %q as created", a.Asset.Value)
		}
	}
	if len(second.Edges) != 1 {
		t.Errorf("second Observe recorded %d edges; want the one already there", len(second.Edges))
	}
}

func TestService_Observe_LinksRelationEndpoints(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	result, err := svc.Observe(ctx, "Acme", "", []Observation{
		{Type: TypeURL, Value: "HTTPS://App.Acme.com:443/login"},
		{Type: TypeService, Value: "app.acme.com:443"},
	}, []AssetRelation{{
		Source:   Observation{Type: TypeURL, Value: "https://app.acme.com/login"},
		Target:   Observation{Type: TypeService, Value: "app.acme.com:443"},
		Relation: RelationServes,
	}})
	if err != nil {
		t.Fatalf("Observe returned an error: %v", err)
	}
	if len(result.Assets) != 2 {
		t.Fatalf("Observe recorded %d assets; want 2", len(result.Assets))
	}
	if result.Assets[0].Asset.Value != "https://app.acme.com/login" {
		t.Errorf("url asset = %q; want https://app.acme.com/login", result.Assets[0].Asset.Value)
	}
	if result.Assets[1].Asset.Value != "app.acme.com:443/tcp" {
		t.Errorf("service asset = %q; want app.acme.com:443/tcp", result.Assets[1].Asset.Value)
	}
	if len(result.Edges) != 1 || result.Edges[0].Relation != RelationServes {
		t.Fatalf("edges = %+v; want one %q edge", result.Edges, RelationServes)
	}
	if result.Edges[0].SrcAssetID != result.Assets[0].Asset.ID || result.Edges[0].DstAssetID != result.Assets[1].Asset.ID {
		t.Error("serves edge does not run from the url to the service")
	}
}

func TestService_Observe_RejectsUnknownRelationBeforeWriting(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)

	_, err := svc.Observe(context.Background(), "Acme", "",
		[]Observation{{Type: TypeDomain, Value: "acme.com"}},
		[]AssetRelation{{Relation: "owns"}})
	if err == nil || !strings.Contains(err.Error(), "owns") {
		t.Fatalf("Observe error = %v; want one naming the unsupported relation", err)
	}
	if len(fake.companies) != 0 || len(fake.assets) != 0 {
		t.Error("Observe wrote rows although the input was invalid")
	}
}

func TestService_Observe_RejectsInvalidObservationBeforeWriting(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)

	_, err := svc.Observe(context.Background(), "Acme", "",
		[]Observation{{Type: TypeIP, Value: "999.1.1.1"}}, nil)
	if err == nil {
		t.Fatal("Observe accepted an invalid ip address")
	}
	if len(fake.companies) != 0 || len(fake.assets) != 0 {
		t.Error("Observe wrote rows although the input was invalid")
	}
}

func TestService_Observe_CancelledContextWritesNothing(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Observe(ctx, "Acme", "", []Observation{{Type: TypeDomain, Value: "acme.com"}}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Observe error = %v; want context.Canceled", err)
	}
	if len(fake.companies) != 0 || len(fake.assets) != 0 {
		t.Error("Observe wrote rows on a cancelled context")
	}
}

func TestService_RecordNode_DeduplicatesByNormalizedTitle(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	first, err := svc.RecordNode(ctx, 7, "", "Enumerate  subdomains", "", "", nil)
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}
	if first.Kind != KindTarget {
		t.Errorf("kind = %q; want the default %q", first.Kind, KindTarget)
	}
	if first.Status != StatusPending {
		t.Errorf("status = %q; want the default %q", first.Status, StatusPending)
	}

	second, err := svc.RecordNode(ctx, 7, KindTarget, "enumerate SUBDOMAINS", "", "", nil)
	if err != nil {
		t.Fatalf("second RecordNode returned an error: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("RecordNode created a second node (%d) for the same title (%d)", second.ID, first.ID)
	}

	if _, err := svc.RecordNode(ctx, 7, "todo", "Check panel", "", "", nil); err == nil {
		t.Error("RecordNode accepted an unsupported kind")
	}
	if _, err := svc.RecordNode(ctx, 7, "", "Check panel", "", "open", nil); err == nil {
		t.Error("RecordNode accepted an unsupported status")
	}
}

func TestService_RecordNode_AnchorsToAssetByUniqueCompany(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	observed, err := svc.Observe(ctx, "Acme", "", []Observation{{Type: TypeDomain, Value: "acme.com"}}, nil)
	if err != nil {
		t.Fatalf("Observe returned an error: %v", err)
	}

	node, err := svc.RecordNode(ctx, 3, KindTarget, "Recon acme.com", "", StatusActive,
		[]AnchorRef{{Type: TypeDomain, Value: "ACME.com."}})
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}
	want := fmt.Sprintf("%d|%d", node.ID, observed.Assets[0].Asset.ID)
	if len(fake.anchors) != 1 || !fake.anchors[want] {
		t.Errorf("anchors = %v; want exactly %s", fake.anchors, want)
	}
}

func TestService_RecordNode_RejectsAmbiguousAndUnknownAnchors(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	for _, company := range []string{"Acme", "Globex"} {
		if _, err := svc.Observe(ctx, company, "", []Observation{{Type: TypeDomain, Value: "acme.com"}}, nil); err != nil {
			t.Fatalf("Observe(%q) returned an error: %v", company, err)
		}
	}
	_, err := svc.RecordNode(ctx, 3, "", "Recon", "", "", []AnchorRef{{Type: TypeDomain, Value: "acme.com"}})
	if err == nil || !strings.Contains(err.Error(), "name the one") {
		t.Errorf("ambiguous anchor error = %v; want one asking for the company", err)
	}

	_, err = svc.RecordNode(ctx, 3, "", "Other", "", "", []AnchorRef{{Type: TypeDomain, Value: "unknown.example"}})
	if err == nil || !strings.Contains(err.Error(), "not observed") {
		t.Errorf("unknown anchor error = %v; want one saying the asset is unknown", err)
	}
}

func TestService_RecordEdge_GuardsTheFlow(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	a, err := svc.RecordNode(ctx, 1, "", "A", "", "", nil)
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}
	b, err := svc.RecordNode(ctx, 2, "", "B", "", "", nil)
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}
	c, err := svc.RecordNode(ctx, 1, "", "C", "", "", nil)
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}

	if _, err := svc.RecordEdge(ctx, 1, a.ID, b.ID, ExplorationRelationDerives); err == nil || !strings.Contains(err.Error(), "not in flow") {
		t.Errorf("cross-flow edge error = %v; want one saying the node is not in the flow", err)
	}
	if _, err := svc.RecordEdge(ctx, 1, a.ID, a.ID, ExplorationRelationDerives); err == nil {
		t.Error("RecordEdge accepted an edge from a node to itself")
	}
	if _, err := svc.RecordEdge(ctx, 1, a.ID, c.ID, "owns"); err == nil || !strings.Contains(err.Error(), "owns") {
		t.Errorf("unknown relation error = %v; want one naming the relation", err)
	}

	edge, err := svc.RecordEdge(ctx, 1, a.ID, c.ID, ExplorationRelationDerives)
	if err != nil {
		t.Fatalf("RecordEdge returned an error: %v", err)
	}
	if edge.FromNodeID != a.ID || edge.ToNodeID != c.ID || edge.Relation != ExplorationRelationDerives {
		t.Errorf("edge = %+v; want %d -[derives]-> %d", edge, a.ID, c.ID)
	}
}

func TestService_SetNodeStatus_UpdatesOnlyTheFlowNode(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	node, err := svc.RecordNode(ctx, 5, "", "Probe login panel", "", "", nil)
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}

	updated, err := svc.SetNodeStatus(ctx, 5, node.ID, StatusExplored, "no rate limiting on /admin")
	if err != nil {
		t.Fatalf("SetNodeStatus returned an error: %v", err)
	}
	if updated.Status != StatusExplored {
		t.Errorf("status = %q; want %q", updated.Status, StatusExplored)
	}
	if updated.Description != "no rate limiting on /admin" {
		t.Errorf("description = %q; want the one just set", updated.Description)
	}

	if _, err := svc.SetNodeStatus(ctx, 6, node.ID, StatusExplored, ""); err == nil || !strings.Contains(err.Error(), "not in flow") {
		t.Errorf("cross-flow update error = %v; want one saying the node is not in the flow", err)
	}
	if _, err := svc.SetNodeStatus(ctx, 5, node.ID, "open", ""); err == nil {
		t.Error("SetNodeStatus accepted an unsupported status")
	}
}

func TestService_FlowGraph_ReturnsNodesEdgesAndAnchors(t *testing.T) {
	fake := newFakeQuerier()
	svc := NewService(fake)
	ctx := context.Background()

	observed, err := svc.Observe(ctx, "Acme", "", []Observation{{Type: TypeDomain, Value: "acme.com"}}, nil)
	if err != nil {
		t.Fatalf("Observe returned an error: %v", err)
	}
	a, err := svc.RecordNode(ctx, 4, "", "Recon", "", "", []AnchorRef{{Company: "Acme", Type: TypeDomain, Value: "acme.com"}})
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}
	b, err := svc.RecordNode(ctx, 4, "", "Scan 443", "", "", nil)
	if err != nil {
		t.Fatalf("RecordNode returned an error: %v", err)
	}
	if _, err := svc.RecordEdge(ctx, 4, a.ID, b.ID, ExplorationRelationDerives); err != nil {
		t.Fatalf("RecordEdge returned an error: %v", err)
	}

	graph, err := svc.FlowGraph(ctx, 4)
	if err != nil {
		t.Fatalf("FlowGraph returned an error: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Errorf("nodes = %d; want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("edges = %d; want 1", len(graph.Edges))
	}
	if len(graph.Anchors) != 1 {
		t.Fatalf("anchors = %d; want 1", len(graph.Anchors))
	}
	if graph.Anchors[0].NodeID != a.ID || graph.Anchors[0].ID != observed.Assets[0].Asset.ID {
		t.Errorf("anchor = %+v; want node %d anchored to asset %d", graph.Anchors[0], a.ID, observed.Assets[0].Asset.ID)
	}
}

func TestService_Coverage_MapsTheRow(t *testing.T) {
	fake := newFakeQuerier()
	fake.coverage = database.GetFlowCoverageRow{
		TotalNodes:     10,
		CompletedNodes: 4,
		AnchoredAssets: 3,
		TotalAssets:    6,
	}
	svc := NewService(fake)

	coverage, err := svc.Coverage(context.Background(), 7)
	if err != nil {
		t.Fatalf("Coverage returned an error: %v", err)
	}
	want := Coverage{TotalNodes: 10, CompletedNodes: 4, AnchoredAssets: 3, TotalAssets: 6}
	if coverage != want {
		t.Errorf("Coverage = %+v; want %+v", coverage, want)
	}
}
