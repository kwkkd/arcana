// Package update is the update-check surface of the server.
//
// Updates are applied by pulling the container images under their configured
// tags, so no check runs against a remote service and nothing here replaces the
// process's own binary. What remains is the shape the version badge in the UI is
// fed through: the build this process runs and a disabled state.
package update

import (
	"context"
	"strings"
	"time"

	"github.com/kwkkd/arcana/pkg/config"
	"github.com/kwkkd/arcana/pkg/database"
	"github.com/kwkkd/arcana/pkg/version"
)

// State is the vocabulary the GraphQL schema mirrors. Only StateDisabled is
// ever produced here; the rest stay so a Status from anywhere maps onto the
// schema without a second definition of what a state is.
type State string

const (
	// StateDisabled says update checks do not run.
	StateDisabled State = "disabled"
	// StatePending says no exchange has completed since this process started.
	StatePending State = "pending"
	// StateUnreachable says every exchange so far failed before an answer arrived.
	StateUnreachable State = "unreachable"
	// StateUnknown says nothing could be established about this build.
	StateUnknown State = "unknown"
	// StateUpToDate says this build is the newest offered on its channel.
	StateUpToDate State = "up_to_date"
	// StateUpdateAvailable says a newer version is offered; Status.Latest names it.
	StateUpdateAvailable State = "update_available"
)

// Status is what this installation knows about updates to itself.
type Status struct {
	// Current is the build this process runs.
	Current string
	// State is the verdict of the most recent answer, or why there is none.
	State State
	// Latest is the newest version the service offered, when it named one.
	Latest string
	// Strategy is the channel updates would be computed under.
	Strategy string
	// CheckedAt is when the answer behind State arrived; zero until one has.
	CheckedAt time.Time
	// FailedAt is when the most recent exchange failed; zero when the last one
	// produced an answer.
	FailedAt time.Time
}

// defaultStrategy is the channel name shared with the composer and the
// installer: the environment default for UPDATE_STRATEGY.
const defaultStrategy = "preview"

// DisabledStatus describes an installation whose checks never run. It still
// names the build, which is the half of the badge that does not depend on any
// service.
func DisabledStatus(cfg *config.Config) Status {
	return Status{
		Current:  version.GetBinaryVersion(),
		State:    StateDisabled,
		Strategy: effectiveStrategy(cfg),
	}
}

// effectiveStrategy is the channel the badge reports: the configured one when
// it is a channel name, the shared default otherwise. A typo reads as the
// default rather than as itself, so the badge never names a channel that does
// not exist.
func effectiveStrategy(cfg *config.Config) string {
	if cfg == nil {
		return defaultStrategy
	}
	strategy := strings.ToLower(strings.TrimSpace(cfg.UpdateStrategy))
	switch strategy {
	case "preview", "stable", "nightly":
		return strategy
	default:
		return defaultStrategy
	}
}

// Service carries the status the rest of the server reads.
type Service struct {
	status Status
}

// New builds the service. It cannot fail: the surface exists whether or not
// anything is ever checked, and the badge must name the running build either
// way.
func New(cfg *config.Config, _ *database.Queries) (*Service, error) {
	return &Service{status: DisabledStatus(cfg)}, nil
}

// Status is a copy of what is currently known. Safe to call from any goroutine,
// and cheap enough to answer on every page load.
func (s *Service) Status() Status {
	return s.status
}

// Run keeps the service alive until the context ends. There is nothing to check
// while checks are disabled.
func (s *Service) Run(ctx context.Context) {
	<-ctx.Done()
}
