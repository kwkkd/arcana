# Processor Implementation Summary

## Overview
Processor package implements the operational engine for ARCANA installer operations per [processor.md](processor.md). Core lifecycle flows, file integrity logic, Docker/Compose orchestration, and Bubble Tea integration are implemented. An update is a compose pull followed by a compose recreate.

## Implementation Notes

### Architecture Decisions
- **Interface-based design**: Internal interfaces per operation type (`fileSystemOperations`, `dockerOperations`, `composeOperations`) enable separation of concerns and testability
- **Two-track execution**: Docker API SDK for worker environment; Compose stacks via console commands with live output streaming
- **OperationOption pattern**: Functional options applied to an internal `operationState` support force mode, embedded terminal integration and the reset-password value via `WithForce`, `WithTerminal` and `WithPasswordValue`
- **State machine logic**: `ApplyChanges` implements four-phase stack management (Observability → Langfuse → Graphiti → ARCANA) with integrity validation; the wizard performs a pre-phase interactive integrity check with Y/N decision for force mode
- **Single-responsibility operations**: business logic delegates to compose layer; strict purge of images is implemented as `purgeImagesStack` alongside other compose operations
- **One lock for everything**: every entry point on `Processor` takes the processor mutex, so an operation and the checker facts it refreshes cannot race

### Key Features Implemented
1. **File System Operations** (`fs.go`):
   - Ensure/verify stack file integrity with force mode support
   - Handle embedded directory trees (observability) and compose files
   - YAML validation and automatic file recovery
   - Support deployment modes (embedded/external/disabled) for applicable stacks
   - Excluded files policy for integrity verification (`filesToExcludeFromVerification`): presence ensured, content changes tolerated — service configs (`observability/otel/config.yml`, `observability/grafana/config/grafana.ini`), user-editable presets (`example.custom.provider.yml`, `example.ollama.provider.yml`, `example.bedrock.provider.yml`, `neo4j/conf/neo4j.conf`, `neo4j/conf/apoc.conf`, `graphiti/{custom,gemini,litellm,openai}.yaml`) and both Jaeger storage plugin binaries

2. **Docker Operations** (`docker.go`):
   - Worker and default image management with progress reporting
   - Worker container lifecycle management (removal, purging)
   - Support for custom Docker configuration via environment variables

3. **Compose Operations** (`compose.go`):
   - Stack lifecycle management with dependency ordering
   - Rolling updates with health checks
   - Live output streaming to TUI callbacks
   - Environment variable injection for compose commands
    - `purgeStack` (down -v) and `purgeImagesStack` (down --rmi all -v) placed together for clarity

4. **Update Operations** (`logic.go`, `compose.go`):
   - `updateStacks` is a pull followed by a recreate: `downloadStack` (`docker compose pull`) fetches the stack's images, `updateStack` (`docker compose up -d`) recreates the containers from them, and a `Gather*Info` call then refreshes that stack's facts for the menus
   - `ProductStackWorker` pulls the worker image instead (`pullWorkerImage`) and refreshes the worker facts; `update` announces the operation once and recurses for `compose` and `all` over every stack the update applies to

5. **Remove/Purge Operations**:
   - Soft removal (preserve data) vs purge (complete cleanup); strict image purge via compose in `purgeImagesStack`
   - Proper cleanup ordering and external/existing deployment handling
   - `removeInstaller` removes nothing by design: it resolves the running binary's path and reports that an installer cannot delete the binary it is executing, so the installer stack step succeeds without touching the file

### Critical Implementation Details
- **Four-phase execution**: Observability → Langfuse → Graphiti → ARCANA with state validation after each phase
- **Force mode behavior**: Aggressive file overwriting and state correction when explicitly requested
- **File integrity logic**: `ensureStackIntegrity` for missing files, `verifyStackIntegrity` for existing files; modified files are explicitly skipped and logged when `force=false`; excluded files are ensured to exist but not overwritten when modified
- **State consistency**: `Gather*Info` calls after each phase validate operation success
- **Error isolation**: Phase failures don't affect other stacks, partial state preserved
- **Compose environment tweaks**: `COMPOSE_IGNORE_ORPHANS=1`, `PYTHONUNBUFFERED=1`; ANSI disabled on narrow terminals via `COMPOSE_ANSI=never`
- **Ordering inside a stack update**: `downloadStack` first, then `updateStack`, then the `Gather*Info` refresh. The recreate goes second because an image that has moved upstream is only fetched by the explicit pull, and the refresh goes last so the menus are labelled from what is actually running
- **The Jaeger plugin is exempt from integrity verification**: both plugin paths are in `filesToExcludeFromVerification`, because the embedded copy is only the source for a first install and an operator-replaced plugin must survive a forced files update. Without the exemption it reads as "modified" and a stack update, which runs with force, copies the embedded version back over it; factory reset restores the embedded copy along with everything else

### Operation Interfaces
The three internal interfaces in `processor.go` are how the business logic in `logic.go` reaches anything external. A method takes the `*operationState` when it produces output, because that is what carries the terminal panel and the message chain; the two that produce a value instead of output — `checkStackIntegrity` and `determineComposeFile` — do not:

```go
type composeOperations interface {
    startStack(ctx context.Context, stack ProductStack, state *operationState) error
    stopStack(ctx context.Context, stack ProductStack, state *operationState) error
    restartStack(ctx context.Context, stack ProductStack, state *operationState) error
    updateStack(ctx context.Context, stack ProductStack, state *operationState) error
    downloadStack(ctx context.Context, stack ProductStack, state *operationState) error
    removeStack(ctx context.Context, stack ProductStack, state *operationState) error
    purgeStack(ctx context.Context, stack ProductStack, state *operationState) error
    purgeImagesStack(ctx context.Context, stack ProductStack, state *operationState) error
    performStackCommand(ctx context.Context, stack ProductStack, state *operationState, args ...string) error
    determineComposeFile(stack ProductStack) (string, error)
}
```

### Streaming Output Into the Terminal Panel
There are two producers of panel content, and they differ in whether the output is a whole view or a single line.

- `runCommand` hands the `*exec.Cmd` to `state.terminal.Execute`, waits for both the process and the terminal, and then sends `terminal.View()` as a **full** replacement (`isPartial=false`). Compose redraws its own progress in place, so the panel has to be the terminal's rendering rather than an append-only log.
- `appendLog` writes one line through `terminal.Append` and again sends the whole `View()`. The installer-stack steps report this way — factory reset, the removal notice, the password reset — as does the compose wrapper for the enter/exit lines around each command.

Both end at `operationState.sendOutput`, which appends a `ProcessorOutputMsg` to `state.msgs` under `state.mx`, stamped with its own 1-based position. Nothing is pushed to Bubble Tea — `HandleMsg` polls: handed a message numbered *n*, it returns `msgs[n]` (the next one) when it has arrived, and otherwise a `ProcessorWaitMsg` carrying the same *n* after 100 ms, which brings it back. The operation itself runs in a goroutine started by `wrapCommand`, which returns the first `ProcessorWaitMsg` as soon as the operation finishes or after 500 ms, whichever comes first. `ProcessorCompletionMsg` ends the chain by returning no command, and a `ProcessorWaitMsg` carrying an error stops the polling too. Walking an accumulated slice rather than draining a channel is what keeps the chain ordered and replayable when the screen is slower than the operation.

When no terminal is supplied, `newOperationState` builds a headless one (80×24, `WithNoPty`), so an operation invoked outside the wizard still produces the same message chain.

### Testing Strategy
Comprehensive tests include:
- Mock implementations for external dependencies (state, checker, files)
- Unit tests for file system integrity operations (ensure/verify/cleanup, excluded files policy, YAML validation)
- Validation tests for operation applicability
- Factory reset, lifecycle, and ordering behavior at logic level

### Integration Points
- **State management**: Integrates with `state.State` for configuration and environment variables
- **System assessment**: Uses `checker.CheckResult` for current system state analysis
- **File handling**: Integrates with `files.Files` for embedded content extraction
- **TUI integration**: Bubble Tea integration via `ProcessorModel` with message polling; wizard performs pre-phase integrity scan (Enter → scan; Y/N → overwrite decision; Ctrl+C → cancel integrity stage)

## Files Created/Modified

### Core Implementation
- `processor.go` - Processor interface, options, operation interfaces, and synchronous operations entry points
- `model.go` - Bubble Tea `ProcessorModel` with `HandleMsg` polling
- `logic.go` - Business logic (ApplyChanges, lifecycle operations, factory reset)
- `state.go` - `operationState`, message types and the `sendOutput`/`sendCompletion` chain
- `fs.go` - File system operations and integrity verification
- `docker.go` - Docker API/CLI operations and worker image/volumes management
- `compose.go` - Docker Compose stack lifecycle management (including `purgeImagesStack`)
- `locale.go` - Operation message strings

### Testing
- `fixtures_test.go` - Mocks for interfaces with call tracking
- `logic_test.go` - Business logic tests (state machine and sequencing)
- `fs_test.go` - File system operations tests (including excluded files policy, the Jaeger plugin among them)
- `compose_test.go`, `pg_test.go`, `state_test.go`, `model_test.go`

## Status
✅ **COMPLETE** - Processor functionality implemented and tested
- Lifecycle, file integrity, Docker/Compose orchestration are production-ready
- Bubble Tea integration via `ProcessorModel` is complete
- Update is a compose pull (`downloadStack`) followed by a recreate (`updateStack`) and a facts refresh
- `removeInstaller` reports why the running binary is left in place instead of deleting it; the installer stack step succeeds without touching the file

## Current Architecture

### ProcessorModel Integration
The processor integrates with Bubble Tea through `ProcessorModel` that wraps operations as `tea.Cmd` and provides a polling handler:

```go
// ProcessorModel provides tea.Cmd wrappers for all operations and a polling handler
type ProcessorModel interface {
    ApplyChanges(ctx context.Context, opts ...OperationOption) tea.Cmd
    CheckFiles(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    FactoryReset(ctx context.Context, opts ...OperationOption) tea.Cmd
    Install(ctx context.Context, opts ...OperationOption) tea.Cmd
    Update(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    Download(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    Remove(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    Purge(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    Start(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    Stop(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    Restart(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    ResetPassword(ctx context.Context, stack ProductStack, opts ...OperationOption) tea.Cmd
    HandleMsg(msg tea.Msg) tea.Cmd
}
```

### Message Types
Processor operations communicate via messages:
- `ProcessorStartedMsg` - operation started
- `ProcessorOutputMsg` - command output (partial or full)
- `ProcessorFilesCheckMsg` - file statuses computed during `CheckFiles`
- `ProcessorCompletionMsg` - operation completed
- `ProcessorWaitMsg` - polling tick

### Terminal Integration
Operations support real-time terminal output through `WithTerminal(term terminal.Terminal)`; ANSI is auto-disabled for narrow terminals (< 56 columns) and on Windows. Compose commands inherit current env plus `COMPOSE_IGNORE_ORPHANS=1` and `PYTHONUNBUFFERED=1`. Without the option an operation still runs, against a headless 80×24 terminal built by `newOperationState`. See [Streaming Output Into the Terminal Panel](#streaming-output-into-the-terminal-panel) for how a line becomes a message.
