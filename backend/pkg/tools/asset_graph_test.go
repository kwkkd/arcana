package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/kwkkd/arcana/pkg/assets"
	"github.com/kwkkd/arcana/pkg/database"
)

type stubAssetGraph struct {
	observe    func(ctx context.Context, company, notes string, observations []assets.Observation, relations []assets.AssetRelation) (*assets.ObserveResult, error)
	recordNode func(ctx context.Context, flowID int64, kind, title, description, status string, anchors []assets.AnchorRef) (*database.ExplorationNode, error)
	setStatus  func(ctx context.Context, flowID, nodeID int64, status, description string) (*database.ExplorationNode, error)
	recordEdge func(ctx context.Context, flowID, fromNodeID, toNodeID int64, relation string) (*database.ExplorationEdge, error)
	flowGraph  func(ctx context.Context, flowID int64) (*assets.FlowGraph, error)
	coverage   func(ctx context.Context, flowID int64) (assets.Coverage, error)
	companies  func(ctx context.Context) ([]database.GetAssetCompaniesRow, error)
	assetGraph func(ctx context.Context, companyID int64) (*assets.AssetGraph, error)
}

func (s *stubAssetGraph) Observe(ctx context.Context, company, notes string, observations []assets.Observation, relations []assets.AssetRelation) (*assets.ObserveResult, error) {
	if s.observe == nil {
		return nil, fmt.Errorf("unexpected Observe call")
	}
	return s.observe(ctx, company, notes, observations, relations)
}

func (s *stubAssetGraph) RecordNode(ctx context.Context, flowID int64, kind, title, description, status string, anchors []assets.AnchorRef) (*database.ExplorationNode, error) {
	if s.recordNode == nil {
		return nil, fmt.Errorf("unexpected RecordNode call")
	}
	return s.recordNode(ctx, flowID, kind, title, description, status, anchors)
}

func (s *stubAssetGraph) SetNodeStatus(ctx context.Context, flowID, nodeID int64, status, description string) (*database.ExplorationNode, error) {
	if s.setStatus == nil {
		return nil, fmt.Errorf("unexpected SetNodeStatus call")
	}
	return s.setStatus(ctx, flowID, nodeID, status, description)
}

func (s *stubAssetGraph) RecordEdge(ctx context.Context, flowID, fromNodeID, toNodeID int64, relation string) (*database.ExplorationEdge, error) {
	if s.recordEdge == nil {
		return nil, fmt.Errorf("unexpected RecordEdge call")
	}
	return s.recordEdge(ctx, flowID, fromNodeID, toNodeID, relation)
}

func (s *stubAssetGraph) FlowGraph(ctx context.Context, flowID int64) (*assets.FlowGraph, error) {
	if s.flowGraph == nil {
		return nil, fmt.Errorf("unexpected FlowGraph call")
	}
	return s.flowGraph(ctx, flowID)
}

func (s *stubAssetGraph) Coverage(ctx context.Context, flowID int64) (assets.Coverage, error) {
	if s.coverage == nil {
		return assets.Coverage{}, fmt.Errorf("unexpected Coverage call")
	}
	return s.coverage(ctx, flowID)
}

func (s *stubAssetGraph) Companies(ctx context.Context) ([]database.GetAssetCompaniesRow, error) {
	if s.companies == nil {
		return nil, fmt.Errorf("unexpected Companies call")
	}
	return s.companies(ctx)
}

func (s *stubAssetGraph) AssetGraph(ctx context.Context, companyID int64) (*assets.AssetGraph, error) {
	if s.assetGraph == nil {
		return nil, fmt.Errorf("unexpected AssetGraph call")
	}
	return s.assetGraph(ctx, companyID)
}

func newAssetGraphToolForTest(flowID int64, stub *stubAssetGraph) *assetGraphTool {
	return &assetGraphTool{flowID: flowID, service: stub}
}

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("result does not contain %q:\n%s", want, got)
		}
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err.Error(), want)
	}
}

func TestAssetGraphTool_Handle_ObservesAssetsAndFormatsTheResult(t *testing.T) {
	t.Parallel()

	var sawObservations []assets.Observation
	var sawRelations []assets.AssetRelation
	stub := &stubAssetGraph{
		observe: func(_ context.Context, company, notes string, observations []assets.Observation, relations []assets.AssetRelation) (*assets.ObserveResult, error) {
			if company != "Acme" {
				t.Errorf("company = %q; want %q", company, "Acme")
			}
			if notes != "" {
				t.Errorf("notes = %q; want empty", notes)
			}
			sawObservations = observations
			sawRelations = relations
			return &assets.ObserveResult{
				Company: &database.AssetCompany{ID: 3, Name: "acme"},
				Assets: []assets.ObservedAsset{
					{Asset: database.Asset{ID: 11, Type: "domain", Value: "dev.acme.com"}, Created: true},
					{Asset: database.Asset{ID: 12, Type: "domain", Value: "acme.com"}},
				},
				Edges: []database.AssetEdge{{SrcAssetID: 11, DstAssetID: 12, Relation: "subdomain_of"}},
			}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	args := json.RawMessage(`{
		"company": "Acme",
		"assets": [{"type": "domain", "value": "dev.acme.com", "properties": {"title": "dev portal"}}],
		"relations": [{"source": {"type": "domain", "value": "dev.acme.com"}, "target": {"type": "domain", "value": "acme.com"}, "relation": "subdomain_of"}],
		"message": "recording what nmap showed"
	}`)

	result, err := tool.Handle(context.Background(), AssetObserveToolName, args)
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	if len(sawObservations) != 1 {
		t.Fatalf("service saw %d observations; want 1", len(sawObservations))
	}
	if sawObservations[0].Type != "domain" || sawObservations[0].Value != "dev.acme.com" {
		t.Errorf("observation = %+v; want domain dev.acme.com", sawObservations[0])
	}
	if title, _ := sawObservations[0].Properties["title"].(string); title != "dev portal" {
		t.Errorf("observation properties title = %v; want \"dev portal\"", sawObservations[0].Properties["title"])
	}
	if len(sawRelations) != 1 || sawRelations[0].Relation != "subdomain_of" {
		t.Fatalf("service saw %+v; want one subdomain_of relation", sawRelations)
	}
	if sawRelations[0].Source.Value != "dev.acme.com" || sawRelations[0].Target.Value != "acme.com" {
		t.Errorf("relation = %+v; want dev.acme.com -[subdomain_of]-> acme.com", sawRelations[0])
	}

	assertContainsAll(t, result,
		`Company "acme" (id 3): 2 assets (1 new), 1 relations.`,
		"- [domain] dev.acme.com (new)",
		"- [domain] acme.com",
		"- dev.acme.com -[subdomain_of]-> acme.com",
	)
}

func TestAssetGraphTool_Handle_PropagatesObservationErrors(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		observe: func(context.Context, string, string, []assets.Observation, []assets.AssetRelation) (*assets.ObserveResult, error) {
			return nil, fmt.Errorf(`unsupported asset type "banana"`)
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	_, err := tool.Handle(context.Background(), AssetObserveToolName,
		json.RawMessage(`{"company": "Acme", "assets": [{"type": "banana", "value": "x"}], "message": "m"}`))
	assertErrorContains(t, err, `unsupported asset type "banana"`)
}

func TestAssetGraphTool_Handle_RequiresItsActionFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		args    string
		wantErr string
	}{
		{
			name:    "observing nothing",
			tool:    AssetObserveToolName,
			args:    `{"company": "Acme", "assets": [], "message": "m"}`,
			wantErr: "nothing to observe",
		},
		{
			name:    "recording a node without a title",
			tool:    ExplorationRecordToolName,
			args:    `{"action": "record_node", "kind": "target", "message": "m"}`,
			wantErr: "title is required",
		},
		{
			name:    "updating a status without a node id",
			tool:    ExplorationRecordToolName,
			args:    `{"action": "update_status", "status": "explored", "message": "m"}`,
			wantErr: "node_id is required",
		},
		{
			name:    "updating a status without a status",
			tool:    ExplorationRecordToolName,
			args:    `{"action": "update_status", "node_id": 7, "message": "m"}`,
			wantErr: "status is required",
		},
		{
			name:    "linking without both nodes",
			tool:    ExplorationRecordToolName,
			args:    `{"action": "link_nodes", "from_node_id": 7, "relation": "derives", "message": "m"}`,
			wantErr: "from_node_id and to_node_id are required",
		},
		{
			name:    "linking without a relation",
			tool:    ExplorationRecordToolName,
			args:    `{"action": "link_nodes", "from_node_id": 7, "to_node_id": 9, "message": "m"}`,
			wantErr: "relation is required",
		},
		{
			name:    "an unknown exploration action",
			tool:    ExplorationRecordToolName,
			args:    `{"action": "explode", "message": "m"}`,
			wantErr: "unsupported action",
		},
		{
			name:    "an unknown graph view",
			tool:    GraphQueryToolName,
			args:    `{"view": "everything", "message": "m"}`,
			wantErr: "unsupported view",
		},
		{
			name:    "the assets view without a company id",
			tool:    GraphQueryToolName,
			args:    `{"view": "assets", "message": "m"}`,
			wantErr: "company_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tool := newAssetGraphToolForTest(42, &stubAssetGraph{})
			_, err := tool.Handle(context.Background(), tt.tool, json.RawMessage(tt.args))
			assertErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAssetGraphTool_Handle_RecordsANodeWithItsAnchors(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		recordNode: func(_ context.Context, flowID int64, kind, title, description, status string, anchors []assets.AnchorRef) (*database.ExplorationNode, error) {
			if flowID != 42 {
				t.Errorf("flowID = %d; want 42", flowID)
			}
			if kind != "hypothesis" || title != "Default credentials on the VPN portal" || status != "active" {
				t.Errorf("kind/title/status = %q/%q/%q", kind, title, status)
			}
			if len(anchors) != 1 || anchors[0].Company != "Acme" || anchors[0].Value != "vpn.acme.com" {
				t.Errorf("anchors = %+v; want one Acme anchor on vpn.acme.com", anchors)
			}
			return &database.ExplorationNode{ID: 7, Kind: kind, Title: title, Status: status}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	args := json.RawMessage(`{
		"action": "record_node",
		"kind": "hypothesis",
		"title": "Default credentials on the VPN portal",
		"status": "active",
		"anchors": [{"company": "Acme", "type": "domain", "value": "vpn.acme.com"}],
		"message": "m"
	}`)

	result, err := tool.Handle(context.Background(), ExplorationRecordToolName, args)
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	assertContainsAll(t, result, `Node 7 "Default credentials on the VPN portal" [hypothesis, active] is in the graph, anchored to 1 asset(s).`)
}

func TestAssetGraphTool_Handle_UpdatesNodeStatus(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		setStatus: func(_ context.Context, flowID, nodeID int64, status, description string) (*database.ExplorationNode, error) {
			if flowID != 42 || nodeID != 7 {
				t.Errorf("flowID/nodeID = %d/%d; want 42/7", flowID, nodeID)
			}
			if status != "dead_end" || description != "the portal rejects both default pairs" {
				t.Errorf("status/description = %q/%q", status, description)
			}
			return &database.ExplorationNode{ID: nodeID, Title: "Default credentials", Status: status}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	args := json.RawMessage(`{
		"action": "update_status",
		"node_id": 7,
		"status": "dead_end",
		"description": "the portal rejects both default pairs",
		"message": "m"
	}`)

	result, err := tool.Handle(context.Background(), ExplorationRecordToolName, args)
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	assertContainsAll(t, result, `Node 7 "Default credentials" is now 'dead_end'.`)
}

func TestAssetGraphTool_Handle_LinksNodes(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		recordEdge: func(_ context.Context, flowID, fromNodeID, toNodeID int64, relation string) (*database.ExplorationEdge, error) {
			if flowID != 42 || fromNodeID != 3 || toNodeID != 5 {
				t.Errorf("flowID/from/to = %d/%d/%d; want 42/3/5", flowID, fromNodeID, toNodeID)
			}
			if relation != "derives" {
				t.Errorf("relation = %q; want derives", relation)
			}
			return &database.ExplorationEdge{FromNodeID: fromNodeID, ToNodeID: toNodeID, Relation: relation}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	args := json.RawMessage(`{"action": "link_nodes", "from_node_id": 3, "to_node_id": 5, "relation": "derives", "message": "m"}`)

	result, err := tool.Handle(context.Background(), ExplorationRecordToolName, args)
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	assertContainsAll(t, result, "Node 3 -[derives]-> node 5 recorded.")
}

func TestAssetGraphTool_Handle_QueriesCoverage(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		coverage: func(_ context.Context, flowID int64) (assets.Coverage, error) {
			if flowID != 42 {
				t.Errorf("flowID = %d; want 42", flowID)
			}
			return assets.Coverage{TotalNodes: 20, CompletedNodes: 5, AnchoredAssets: 3, TotalAssets: 6}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	result, err := tool.Handle(context.Background(), GraphQueryToolName, json.RawMessage(`{"view": "coverage", "message": "m"}`))
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	assertContainsAll(t, result, "5 of 20 exploration nodes resolved (25%)", "3 of 6 company assets anchored (50%)")
}

func TestAssetGraphTool_Handle_QueriesCompanies(t *testing.T) {
	t.Parallel()

	t.Run("with companies", func(t *testing.T) {
		t.Parallel()

		stub := &stubAssetGraph{
			companies: func(context.Context) ([]database.GetAssetCompaniesRow, error) {
				return []database.GetAssetCompaniesRow{
					{ID: 3, Name: "acme", AssetCount: 12, Notes: "main engagement target"},
					{ID: 4, Name: "globex", AssetCount: 2},
				}, nil
			},
		}
		tool := newAssetGraphToolForTest(42, stub)

		result, err := tool.Handle(context.Background(), GraphQueryToolName, json.RawMessage(`{"view": "companies", "message": "m"}`))
		if err != nil {
			t.Fatalf("Handle returned an error: %v", err)
		}

		assertContainsAll(t, result,
			"Companies (2):",
			`- id 3 "acme" — 12 assets (notes: main engagement target)`,
			`- id 4 "globex" — 2 assets`,
		)
	})

	t.Run("with none", func(t *testing.T) {
		t.Parallel()

		stub := &stubAssetGraph{
			companies: func(context.Context) ([]database.GetAssetCompaniesRow, error) {
				return nil, nil
			},
		}
		tool := newAssetGraphToolForTest(42, stub)

		result, err := tool.Handle(context.Background(), GraphQueryToolName, json.RawMessage(`{"view": "companies", "message": "m"}`))
		if err != nil {
			t.Fatalf("Handle returned an error: %v", err)
		}

		assertContainsAll(t, result, "No companies are in the asset graph yet")
	})
}

func TestAssetGraphTool_Handle_QueriesTheExplorationGraph(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		flowGraph: func(_ context.Context, flowID int64) (*assets.FlowGraph, error) {
			if flowID != 42 {
				t.Errorf("flowID = %d; want 42", flowID)
			}
			return &assets.FlowGraph{
				Nodes: []database.ExplorationNode{
					{ID: 7, Kind: "target", Status: "explored", Title: "Enumerate subdomains", Description: "pull the certificate transparency logs"},
					{ID: 9, Kind: "action", Status: "pending", Title: "Scan vhosts"},
				},
				Edges: []database.ExplorationEdge{
					{FromNodeID: 7, ToNodeID: 9, Relation: "derives"},
				},
				Anchors: []database.GetExplorationAnchorsByFlowRow{
					{NodeID: 7, Value: "acme.com"},
					{NodeID: 7, Value: "dev.acme.com"},
				},
			}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	result, err := tool.Handle(context.Background(), GraphQueryToolName, json.RawMessage(`{"view": "exploration", "message": "m"}`))
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	assertContainsAll(t, result,
		"Exploration graph of flow 42: 2 nodes, 1 links.",
		`- 7 [target/explored] Enumerate subdomains — anchored to: acme.com, dev.acme.com`,
		"pull the certificate transparency logs",
		"- 9 [action/pending] Scan vhosts",
		`- 7 "Enumerate subdomains" -[derives]-> 9 "Scan vhosts"`,
	)
}

func TestAssetGraphTool_Handle_QueriesCompanyAssets(t *testing.T) {
	t.Parallel()

	stub := &stubAssetGraph{
		assetGraph: func(_ context.Context, companyID int64) (*assets.AssetGraph, error) {
			if companyID != 3 {
				t.Errorf("companyID = %d; want 3", companyID)
			}
			return &assets.AssetGraph{
				Company: database.AssetCompany{ID: 3, Name: "acme"},
				Assets: []database.Asset{
					{ID: 11, Type: "domain", Value: "dev.acme.com"},
					{ID: 12, Type: "domain", Value: "acme.com"},
				},
				Edges: []database.AssetEdge{
					{SrcAssetID: 11, DstAssetID: 12, Relation: "subdomain_of"},
					{SrcAssetID: 99, DstAssetID: 12, Relation: "related_to"},
				},
			}, nil
		},
	}
	tool := newAssetGraphToolForTest(42, stub)

	result, err := tool.Handle(context.Background(), GraphQueryToolName, json.RawMessage(`{"view": "assets", "company_id": 3, "message": "m"}`))
	if err != nil {
		t.Fatalf("Handle returned an error: %v", err)
	}

	assertContainsAll(t, result,
		`Company "acme" (id 3): 2 assets, 2 relations.`,
		"- [domain] acme.com",
		"- dev.acme.com -[subdomain_of]-> acme.com",
		"- asset #99 -[related_to]-> acme.com",
	)
}

func TestAssetGraphTool_Handle_RejectsUnknownToolNames(t *testing.T) {
	t.Parallel()

	tool := newAssetGraphToolForTest(42, &stubAssetGraph{})
	_, err := tool.Handle(context.Background(), "asset_magic", json.RawMessage(`{}`))
	assertErrorContains(t, err, "unknown asset graph tool")
}

func TestAssetGraphTool_IsAvailable(t *testing.T) {
	t.Parallel()

	if (&assetGraphTool{flowID: 42}).IsAvailable() {
		t.Error("a tool without a service reports itself available")
	}
	if !newAssetGraphToolForTest(42, &stubAssetGraph{}).IsAvailable() {
		t.Error("a tool with a service reports itself unavailable")
	}
}

type recordingExplorationPublisher struct {
	coverages []assets.Coverage
}

func (p *recordingExplorationPublisher) ExplorationUpdated(_ context.Context, coverage assets.Coverage) {
	p.coverages = append(p.coverages, coverage)
}

func TestAssetGraphTool_Handle_PublishesCoverageAfterEveryMutation(t *testing.T) {
	t.Parallel()

	coverage := assets.Coverage{TotalNodes: 4, CompletedNodes: 1, AnchoredAssets: 2, TotalAssets: 9}
	stub := &stubAssetGraph{
		observe: func(context.Context, string, string, []assets.Observation, []assets.AssetRelation) (*assets.ObserveResult, error) {
			return &assets.ObserveResult{Company: &database.AssetCompany{ID: 3, Name: "acme"}}, nil
		},
		recordNode: func(_ context.Context, _ int64, kind, title, _, status string, _ []assets.AnchorRef) (*database.ExplorationNode, error) {
			return &database.ExplorationNode{ID: 7, Kind: kind, Title: title, Status: status}, nil
		},
		setStatus: func(_ context.Context, _, nodeID int64, status, _ string) (*database.ExplorationNode, error) {
			return &database.ExplorationNode{ID: nodeID, Status: status}, nil
		},
		recordEdge: func(_ context.Context, _, fromNodeID, toNodeID int64, relation string) (*database.ExplorationEdge, error) {
			return &database.ExplorationEdge{FromNodeID: fromNodeID, ToNodeID: toNodeID, Relation: relation}, nil
		},
		coverage: func(context.Context, int64) (assets.Coverage, error) {
			return coverage, nil
		},
	}

	for _, tc := range []struct {
		name string
		tool string
		args string
	}{
		{"asset observe", AssetObserveToolName, `{"company": "Acme", "assets": [{"type": "domain", "value": "acme.com"}], "message": "m"}`},
		{"exploration record node", ExplorationRecordToolName, `{"action": "record_node", "kind": "target", "title": "Login form", "message": "m"}`},
		{"exploration update status", ExplorationRecordToolName, `{"action": "update_status", "node_id": 7, "status": "explored", "message": "m"}`},
		{"exploration link nodes", ExplorationRecordToolName, `{"action": "link_nodes", "from_node_id": 7, "to_node_id": 9, "relation": "derives", "message": "m"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pub := &recordingExplorationPublisher{}
			tool := &assetGraphTool{flowID: 42, publisher: pub, service: stub}

			if _, err := tool.Handle(context.Background(), tc.tool, json.RawMessage(tc.args)); err != nil {
				t.Fatalf("Handle returned an error: %v", err)
			}

			if len(pub.coverages) != 1 {
				t.Fatalf("published %d coverages; want 1", len(pub.coverages))
			}
			if pub.coverages[0] != coverage {
				t.Errorf("published %+v; want %+v", pub.coverages[0], coverage)
			}
		})
	}
}

func TestAssetGraphTool_Handle_DoesNotPublishOnReadsOrFailures(t *testing.T) {
	t.Parallel()

	t.Run("a rejected mutation", func(t *testing.T) {
		t.Parallel()

		pub := &recordingExplorationPublisher{}
		stub := &stubAssetGraph{
			recordNode: func(context.Context, int64, string, string, string, string, []assets.AnchorRef) (*database.ExplorationNode, error) {
				return nil, fmt.Errorf("node kind %q is not accepted", "banana")
			},
		}
		tool := &assetGraphTool{flowID: 42, publisher: pub, service: stub}

		_, err := tool.Handle(context.Background(), ExplorationRecordToolName,
			json.RawMessage(`{"action": "record_node", "kind": "banana", "title": "x", "message": "m"}`))
		assertErrorContains(t, err, `node kind "banana" is not accepted`)

		if len(pub.coverages) != 0 {
			t.Errorf("published %d coverages for a rejected mutation; want 0", len(pub.coverages))
		}
	})

	t.Run("an unreadable coverage", func(t *testing.T) {
		t.Parallel()

		pub := &recordingExplorationPublisher{}
		stub := &stubAssetGraph{
			recordNode: func(_ context.Context, _ int64, kind, title, _, status string, _ []assets.AnchorRef) (*database.ExplorationNode, error) {
				return &database.ExplorationNode{ID: 7, Kind: kind, Title: title, Status: status}, nil
			},
			coverage: func(context.Context, int64) (assets.Coverage, error) {
				return assets.Coverage{}, fmt.Errorf("coverage query failed")
			},
		}
		tool := &assetGraphTool{flowID: 42, publisher: pub, service: stub}

		if _, err := tool.Handle(context.Background(), ExplorationRecordToolName,
			json.RawMessage(`{"action": "record_node", "kind": "target", "title": "Login form", "message": "m"}`)); err != nil {
			t.Fatalf("Handle returned an error after a committed write: %v", err)
		}

		if len(pub.coverages) != 0 {
			t.Errorf("published %d coverages although the coverage read failed; want 0", len(pub.coverages))
		}
	})

	t.Run("a graph query", func(t *testing.T) {
		t.Parallel()

		pub := &recordingExplorationPublisher{}
		stub := &stubAssetGraph{
			coverage: func(context.Context, int64) (assets.Coverage, error) {
				return assets.Coverage{TotalNodes: 4}, nil
			},
		}
		tool := &assetGraphTool{flowID: 42, publisher: pub, service: stub}

		if _, err := tool.Handle(context.Background(), GraphQueryToolName, json.RawMessage(`{"view": "coverage", "message": "m"}`)); err != nil {
			t.Fatalf("Handle returned an error: %v", err)
		}

		if len(pub.coverages) != 0 {
			t.Errorf("a read published %d coverages; want 0", len(pub.coverages))
		}
	})
}
