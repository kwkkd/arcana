package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kwkkd/arcana/pkg/assets"
	"github.com/kwkkd/arcana/pkg/database"
	"github.com/sirupsen/logrus"
)

// AssetGraphService is the slice of pkg/assets the graph tools consume.
type AssetGraphService interface {
	Observe(ctx context.Context, company, notes string, observations []assets.Observation, relations []assets.AssetRelation) (*assets.ObserveResult, error)
	RecordNode(ctx context.Context, flowID int64, kind, title, description, status string, anchors []assets.AnchorRef) (*database.ExplorationNode, error)
	SetNodeStatus(ctx context.Context, flowID, nodeID int64, status, description string) (*database.ExplorationNode, error)
	RecordEdge(ctx context.Context, flowID, fromNodeID, toNodeID int64, relation string) (*database.ExplorationEdge, error)
	FlowGraph(ctx context.Context, flowID int64) (*assets.FlowGraph, error)
	Coverage(ctx context.Context, flowID int64) (assets.Coverage, error)
	Companies(ctx context.Context) ([]database.GetAssetCompaniesRow, error)
	AssetGraph(ctx context.Context, companyID int64) (*assets.AssetGraph, error)
}

// ExplorationPublisher announces that a flow's exploration graph changed.
type ExplorationPublisher interface {
	ExplorationUpdated(ctx context.Context, coverage assets.Coverage)
}

type assetGraphTool struct {
	flowID    int64
	publisher ExplorationPublisher
	service   AssetGraphService
}

func NewAssetGraphTool(flowID int64, db database.Querier, publisher ExplorationPublisher) Tool {
	return &assetGraphTool{flowID: flowID, publisher: publisher, service: assets.NewService(db)}
}

func (t *assetGraphTool) IsAvailable() bool {
	return t.service != nil
}

func (t *assetGraphTool) Handle(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if !t.IsAvailable() {
		return "", fmt.Errorf("asset graph is not available")
	}

	logger := logrus.WithContext(ctx).WithFields(enrichLogrusFields(t.flowID, nil, nil, logrus.Fields{
		"tool": name,
		"args": string(args),
	}))

	switch name {
	case AssetObserveToolName:
		return t.handleObserve(ctx, logger, args)
	case ExplorationRecordToolName:
		return t.handleExplorationRecord(ctx, logger, args)
	case GraphQueryToolName:
		return t.handleQuery(ctx, logger, args)
	}

	return "", fmt.Errorf("unknown asset graph tool %q", name)
}

func (t *assetGraphTool) handleObserve(ctx context.Context, logger *logrus.Entry, args json.RawMessage) (string, error) {
	var action AssetObserveAction
	if err := json.Unmarshal(args, &action); err != nil {
		logger.WithError(err).Error("failed to unmarshal asset observe action arguments")
		return "", fmt.Errorf("failed to unmarshal %s action arguments: %w", AssetObserveToolName, err)
	}

	if len(action.Assets) == 0 && len(action.Relations) == 0 {
		return "", fmt.Errorf("nothing to observe: pass at least one asset or relation")
	}

	observations := make([]assets.Observation, 0, len(action.Assets))
	for _, a := range action.Assets {
		observations = append(observations, observationOf(a))
	}
	relations := make([]assets.AssetRelation, 0, len(action.Relations))
	for _, r := range action.Relations {
		relations = append(relations, assets.AssetRelation{
			Source:   observationOf(r.Source),
			Target:   observationOf(r.Target),
			Relation: r.Relation,
		})
	}

	result, err := t.service.Observe(ctx, action.Company, "", observations, relations)
	if err != nil {
		logger.WithError(err).Warn("asset observation was rejected")
		return "", fmt.Errorf("asset_observe: %w", err)
	}
	t.publishCoverage(ctx, logger)

	var b strings.Builder
	created := 0
	for _, a := range result.Assets {
		if a.Created {
			created++
		}
	}
	fmt.Fprintf(&b, "Company %q (id %d): %d assets (%d new), %d relations.\n",
		result.Company.Name, result.Company.ID, len(result.Assets), created, len(result.Edges))

	values := assetValues(result.Assets)
	if len(result.Assets) > 0 {
		b.WriteString("Assets:\n")
		for _, a := range result.Assets {
			suffix := ""
			if a.Created {
				suffix = " (new)"
			}
			fmt.Fprintf(&b, "- [%s] %s%s\n", a.Asset.Type, a.Asset.Value, suffix)
		}
	}
	if len(result.Edges) > 0 {
		b.WriteString("Relations:\n")
		for _, e := range result.Edges {
			fmt.Fprintf(&b, "- %s\n", formatAssetEdge(e, values))
		}
	}

	return strings.TrimRight(b.String(), "\n"), nil
}

func (t *assetGraphTool) handleExplorationRecord(ctx context.Context, logger *logrus.Entry, args json.RawMessage) (string, error) {
	var action ExplorationRecordAction
	if err := json.Unmarshal(args, &action); err != nil {
		logger.WithError(err).Error("failed to unmarshal exploration record action arguments")
		return "", fmt.Errorf("failed to unmarshal %s action arguments: %w", ExplorationRecordToolName, err)
	}

	switch action.Action {
	case ExplorationActionRecordNode:
		return t.recordNode(ctx, logger, action)
	case ExplorationActionUpdateStatus:
		return t.updateNodeStatus(ctx, logger, action)
	case ExplorationActionLinkNodes:
		return t.linkNodes(ctx, logger, action)
	}

	return "", fmt.Errorf("unsupported action %q: use record_node, update_status or link_nodes", action.Action)
}

func (t *assetGraphTool) recordNode(ctx context.Context, logger *logrus.Entry, action ExplorationRecordAction) (string, error) {
	if strings.TrimSpace(action.Title) == "" {
		return "", fmt.Errorf("title is required to record a node")
	}

	anchors := make([]assets.AnchorRef, 0, len(action.Anchors))
	for _, a := range action.Anchors {
		anchors = append(anchors, assets.AnchorRef{Company: a.Company, Type: a.Type, Value: a.Value})
	}

	node, err := t.service.RecordNode(ctx, t.flowID, action.Kind, action.Title, action.Description, action.Status, anchors)
	if err != nil {
		logger.WithError(err).Warn("exploration node was rejected")
		return "", fmt.Errorf("exploration_record: %w", err)
	}
	t.publishCoverage(ctx, logger)

	var b strings.Builder
	fmt.Fprintf(&b, "Node %d %q [%s, %s] is in the graph", node.ID, node.Title, node.Kind, node.Status)
	if len(action.Anchors) > 0 {
		fmt.Fprintf(&b, ", anchored to %d asset(s)", len(action.Anchors))
	}
	b.WriteString(".\n")

	return b.String(), nil
}

func (t *assetGraphTool) updateNodeStatus(ctx context.Context, logger *logrus.Entry, action ExplorationRecordAction) (string, error) {
	if action.NodeID == nil {
		return "", fmt.Errorf("node_id is required to update a node status")
	}
	if action.Status == "" {
		return "", fmt.Errorf("status is required to update a node status")
	}

	node, err := t.service.SetNodeStatus(ctx, t.flowID, action.NodeID.Int64(), action.Status, action.Description)
	if err != nil {
		logger.WithError(err).Warn("node status update was rejected")
		return "", fmt.Errorf("exploration_record: %w", err)
	}
	t.publishCoverage(ctx, logger)

	return fmt.Sprintf("Node %d %q is now '%s'.\n", node.ID, node.Title, node.Status), nil
}

func (t *assetGraphTool) linkNodes(ctx context.Context, logger *logrus.Entry, action ExplorationRecordAction) (string, error) {
	if action.FromNodeID == nil || action.ToNodeID == nil {
		return "", fmt.Errorf("from_node_id and to_node_id are required to link nodes")
	}
	if action.Relation == "" {
		return "", fmt.Errorf("relation is required to link nodes")
	}

	edge, err := t.service.RecordEdge(ctx, t.flowID, action.FromNodeID.Int64(), action.ToNodeID.Int64(), action.Relation)
	if err != nil {
		logger.WithError(err).Warn("exploration edge was rejected")
		return "", fmt.Errorf("exploration_record: %w", err)
	}
	t.publishCoverage(ctx, logger)

	return fmt.Sprintf("Node %d -[%s]-> node %d recorded.\n", edge.FromNodeID, edge.Relation, edge.ToNodeID), nil
}

// publishCoverage is best effort: the change is already committed, and a subscriber
// that misses the event loses only immediacy, so a failure here is logged, not returned.
func (t *assetGraphTool) publishCoverage(ctx context.Context, logger *logrus.Entry) {
	if t.publisher == nil {
		return
	}

	coverage, err := t.service.Coverage(ctx, t.flowID)
	if err != nil {
		logger.WithError(err).Warn("failed to read coverage for the exploration subscribers")
		return
	}

	t.publisher.ExplorationUpdated(ctx, coverage)
}

func (t *assetGraphTool) handleQuery(ctx context.Context, logger *logrus.Entry, args json.RawMessage) (string, error) {
	var action GraphQueryAction
	if err := json.Unmarshal(args, &action); err != nil {
		logger.WithError(err).Error("failed to unmarshal graph query action arguments")
		return "", fmt.Errorf("failed to unmarshal %s action arguments: %w", GraphQueryToolName, err)
	}

	switch action.View {
	case "coverage":
		return t.queryCoverage(ctx)
	case "companies":
		return t.queryCompanies(ctx)
	case "exploration":
		return t.queryExploration(ctx)
	case "assets":
		return t.queryAssets(ctx, action)
	}

	return "", fmt.Errorf("unsupported view %q: use coverage, companies, exploration or assets", action.View)
}

func (t *assetGraphTool) queryCoverage(ctx context.Context) (string, error) {
	coverage, err := t.service.Coverage(ctx, t.flowID)
	if err != nil {
		return "", fmt.Errorf("graph_query: %w", err)
	}

	return fmt.Sprintf("Flow %d coverage: %d of %d exploration nodes resolved (%.0f%%); %d of %d company assets anchored (%.0f%%).\n",
		t.flowID,
		coverage.CompletedNodes, coverage.TotalNodes, coverage.NodePercent(),
		coverage.AnchoredAssets, coverage.TotalAssets, coverage.AssetPercent()), nil
}

func (t *assetGraphTool) queryCompanies(ctx context.Context) (string, error) {
	companies, err := t.service.Companies(ctx)
	if err != nil {
		return "", fmt.Errorf("graph_query: %w", err)
	}
	if len(companies) == 0 {
		return "No companies are in the asset graph yet; record assets with " + AssetObserveToolName + " first.\n", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Companies (%d):\n", len(companies))
	for _, c := range companies {
		fmt.Fprintf(&b, "- id %d %q — %d assets", c.ID, c.Name, c.AssetCount)
		if notes := strings.TrimSpace(c.Notes); notes != "" {
			fmt.Fprintf(&b, " (notes: %s)", excerpt(notes, 80))
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func (t *assetGraphTool) queryExploration(ctx context.Context) (string, error) {
	graph, err := t.service.FlowGraph(ctx, t.flowID)
	if err != nil {
		return "", fmt.Errorf("graph_query: %w", err)
	}
	if len(graph.Nodes) == 0 {
		return fmt.Sprintf("The exploration graph of flow %d is empty; record nodes with %s as work progresses.\n", t.flowID, ExplorationRecordToolName), nil
	}

	anchors := make(map[int64][]string)
	for _, a := range graph.Anchors {
		anchors[a.NodeID] = append(anchors[a.NodeID], a.Value)
	}
	titles := make(map[int64]string, len(graph.Nodes))
	for _, n := range graph.Nodes {
		titles[n.ID] = n.Title
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Exploration graph of flow %d: %d nodes, %d links.\n", t.flowID, len(graph.Nodes), len(graph.Edges))
	b.WriteString("Nodes:\n")
	for _, n := range graph.Nodes {
		fmt.Fprintf(&b, "- %d [%s/%s] %s", n.ID, n.Kind, n.Status, excerpt(n.Title, 120))
		if len(anchors[n.ID]) > 0 {
			fmt.Fprintf(&b, " — anchored to: %s", strings.Join(anchors[n.ID], ", "))
		}
		b.WriteString("\n")
		if description := strings.TrimSpace(n.Description); description != "" {
			fmt.Fprintf(&b, "  %s\n", excerpt(description, 200))
		}
	}
	if len(graph.Edges) > 0 {
		b.WriteString("Links:\n")
		for _, e := range graph.Edges {
			fmt.Fprintf(&b, "- %s -[%s]-> %s\n",
				nodeRef(e.FromNodeID, titles), e.Relation, nodeRef(e.ToNodeID, titles))
		}
	}

	return b.String(), nil
}

func (t *assetGraphTool) queryAssets(ctx context.Context, action GraphQueryAction) (string, error) {
	if action.CompanyID == nil {
		return "", fmt.Errorf("company_id is required for the assets view; list companies first")
	}

	graph, err := t.service.AssetGraph(ctx, action.CompanyID.Int64())
	if err != nil {
		return "", fmt.Errorf("graph_query: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Company %q (id %d): %d assets, %d relations.\n", graph.Company.Name, graph.Company.ID, len(graph.Assets), len(graph.Edges))

	values := make(map[int64]string, len(graph.Assets))
	if len(graph.Assets) > 0 {
		b.WriteString("Assets:\n")
		for _, a := range graph.Assets {
			values[a.ID] = a.Value
			fmt.Fprintf(&b, "- [%s] %s\n", a.Type, a.Value)
		}
	}
	if len(graph.Edges) > 0 {
		b.WriteString("Relations:\n")
		for _, e := range graph.Edges {
			fmt.Fprintf(&b, "- %s\n", formatAssetEdge(e, values))
		}
	}

	return b.String(), nil
}

func observationOf(a AssetObservation) assets.Observation {
	observation := assets.Observation{Type: a.Type, Value: a.Value}
	if len(a.Properties) > 0 {
		properties := make(map[string]any, len(a.Properties))
		for k, v := range a.Properties {
			properties[k] = v
		}
		observation.Properties = properties
	}
	return observation
}

func assetValues(observed []assets.ObservedAsset) map[int64]string {
	values := make(map[int64]string, len(observed))
	for _, a := range observed {
		values[a.Asset.ID] = a.Asset.Value
	}
	return values
}

func formatAssetEdge(edge database.AssetEdge, values map[int64]string) string {
	return fmt.Sprintf("%s -[%s]-> %s", assetRef(edge.SrcAssetID, values), edge.Relation, assetRef(edge.DstAssetID, values))
}

func assetRef(id int64, values map[int64]string) string {
	if value, ok := values[id]; ok {
		return value
	}
	return fmt.Sprintf("asset #%d", id)
}

func nodeRef(id int64, titles map[int64]string) string {
	if title, ok := titles[id]; ok {
		return fmt.Sprintf("%d %q", id, excerpt(title, 60))
	}
	return fmt.Sprintf("node %d", id)
}

func excerpt(s string, limit int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= limit {
		return s
	}
	return strings.TrimRight(s[:limit], " ") + "…"
}
