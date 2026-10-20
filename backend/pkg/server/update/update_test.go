package update

import (
	"context"
	"testing"
	"time"

	"github.com/kwkkd/arcana/pkg/config"
	"github.com/kwkkd/arcana/pkg/version"
)

func TestUpdate_DisabledStatus_NamesTheRunningBuild(t *testing.T) {
	status := DisabledStatus(&config.Config{UpdateStrategy: "stable"})

	if status.State != StateDisabled {
		t.Errorf("state = %q, want %q", status.State, StateDisabled)
	}
	if status.Current != version.GetBinaryVersion() {
		t.Errorf("current = %q, want the running build %q", status.Current, version.GetBinaryVersion())
	}
	if status.Strategy != "stable" {
		t.Errorf("strategy = %q, want %q", status.Strategy, "stable")
	}
}

func TestUpdate_EffectiveStrategy_FallsBackToTheSharedDefault(t *testing.T) {
	for _, tt := range []struct {
		name       string
		configured string
		cfg        *config.Config
		want       string
	}{
		{"the preview channel", "preview", &config.Config{UpdateStrategy: "preview"}, "preview"},
		{"the stable channel", "stable", &config.Config{UpdateStrategy: "stable"}, "stable"},
		{"the nightly channel", "nightly", &config.Config{UpdateStrategy: "nightly"}, "nightly"},
		{"a value padded the way a .env file pads it", "  Stable  ", &config.Config{UpdateStrategy: "  Stable  "}, "stable"},
		{"a channel that does not exist", "beta", &config.Config{UpdateStrategy: "beta"}, "preview"},
		{"nothing configured", "", &config.Config{}, "preview"},
		{"no configuration at all", "", nil, "preview"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveStrategy(tt.cfg); got != tt.want {
				t.Errorf("configured %q read as %q, want %q", tt.configured, got, tt.want)
			}
		})
	}
}

func TestUpdate_New_ReportsTheDisabledStatus(t *testing.T) {
	service, err := New(&config.Config{UpdateStrategy: "stable"}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	status := service.Status()
	if status.State != StateDisabled {
		t.Errorf("state = %q, want %q", status.State, StateDisabled)
	}
	if status.Strategy != "stable" {
		t.Errorf("strategy = %q, want %q", status.Strategy, "stable")
	}
}

func TestUpdate_Run_ReturnsWhenTheContextEnds(t *testing.T) {
	service, err := New(&config.Config{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Run returned while its context was live")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}
