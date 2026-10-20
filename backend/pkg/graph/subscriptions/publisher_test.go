package subscriptions

import (
	"context"
	"testing"
	"time"

	"github.com/kwkkd/arcana/pkg/assets"
	"github.com/kwkkd/arcana/pkg/graph/model"
)

// Subtests are keyed by method.
func TestPublisher_FlowFile_StampsTheFlowID(t *testing.T) {
	for _, tc := range []struct {
		name    string
		publish func(FlowPublisher, context.Context, *model.FlowFile)
		receive func(FlowSubscriber, context.Context) (<-chan *model.FlowFile, error)
	}{
		{
			"an added file carries the flow id",
			func(p FlowPublisher, ctx context.Context, f *model.FlowFile) { p.FlowFileAdded(ctx, f) },
			func(s FlowSubscriber, ctx context.Context) (<-chan *model.FlowFile, error) {
				return s.FlowFileAdded(ctx)
			},
		},
		{
			"an updated file carries the flow id",
			func(p FlowPublisher, ctx context.Context, f *model.FlowFile) { p.FlowFileUpdated(ctx, f) },
			func(s FlowSubscriber, ctx context.Context) (<-chan *model.FlowFile, error) {
				return s.FlowFileUpdated(ctx)
			},
		},
		{
			"a deleted file carries the flow id",
			func(p FlowPublisher, ctx context.Context, f *model.FlowFile) { p.FlowFileDeleted(ctx, f) },
			func(s FlowSubscriber, ctx context.Context) (<-chan *model.FlowFile, error) {
				return s.FlowFileDeleted(ctx)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			ctrl := NewSubscriptionsController()

			frames, err := tc.receive(ctrl.NewFlowSubscriber(1, 42), ctx)
			if err != nil {
				t.Fatalf("subscribe: %v", err)
			}

			tc.publish(ctrl.NewFlowPublisher(1, 42), ctx, &model.FlowFile{ID: "d41d8cd98f00b204e9800998ecf8427e"})

			select {
			case got := <-frames:
				if got.FlowID != 42 {
					t.Errorf("flowId = %d, want 42", got.FlowID)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("no frame arrived")
			}
		})
	}
}

func TestPublisher_ExplorationUpdated_CarriesTheConvertedCoverage(t *testing.T) {
	ctx := t.Context()
	ctrl := NewSubscriptionsController()

	frames, err := ctrl.NewFlowSubscriber(1, 42).ExplorationUpdated(ctx)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	ctrl.NewFlowPublisher(1, 42).ExplorationUpdated(ctx, assets.Coverage{
		TotalNodes:     7,
		CompletedNodes: 3,
		AnchoredAssets: 12,
		TotalAssets:    20,
	})

	select {
	case got := <-frames:
		if got.TotalNodes != 7 || got.CompletedNodes != 3 || got.AnchoredAssets != 12 || got.TotalAssets != 20 {
			t.Errorf("coverage = %+v, want {7 3 12 20}", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no frame arrived")
	}
}
