# Core Testing Roadmap

## Overview

This document defines the ongoing engineering plan for test coverage in `malice-network`.

The goal is not to chase a raw coverage percentage. The goal is to keep the highest-risk components and operator-visible paths under stable, layered regression guards.

The roadmap uses:

- risk-first prioritization
- a fixed core component manifest
- a repeatable inventory command that refreshes recommendations
- explicit CI lanes instead of ad hoc local-only suites

The machine-readable source of truth lives in `docs/development/core-testing-manifest.json`.

## Stable Baseline

The baseline must stay green before any new coverage expansion is considered complete:

```bash
go mod tidy
go vet ./...
go test ./... -count=1 -timeout 300s
CGO_ENABLED=0 go build ./...
go test -race ./server/internal/core -count=1 -timeout 300s
go test -tags=mockimplant ./server -count=1 -timeout 300s
go test -tags=integration ./server ./client/command/listener ./client/command/pipeline ./client/command/website ./client/command/sessions ./client/command/context -count=1 -timeout 300s
```

If the default baseline fails, fix that first. Do not stack new testing work on top of a broken default suite.

## Core Layers

The repository currently uses four primary layers:

- `unit`: deterministic package-level tests for parsing, validation, helpers, and side-effect boundaries
- `command_conformance`: real Cobra command execution with recorder-backed RPC assertions
- `integration`: real client/server control-plane tests with gRPC and mTLS
- `mockimplant`: listener-facing implant transport tests with the mock implant harness

These layers map to core chains as follows:

| Chain | Preferred Layers | Intent |
| --- | --- | --- |
| command parsing | `command_conformance` | Catch CLI parsing, validation, and protobuf assembly regressions |
| control plane | `integration`, `unit` | Catch client/server orchestration and state reconciliation regressions |
| implant transport | `mockimplant`, `unit` | Catch parser, stream, checkin, and task transport regressions |
| build and certificate | `unit` | Keep certificate, build wrapper, and artifact logic deterministic |
| task output | `unit` | Keep task context parsing and formatting stable |

## Phases

### Phase 0

- Keep the default `go test ./...` baseline green.
- Run the inventory command and review the current report before adding new tests.
- Do not change CI gates and test shape in the same patch unless the current baseline already passes.

### Phase 1

Prioritize missing or thin tests for Tier-1 service boundaries:

- `server/internal/parser`
- `server/internal/certutils`
- `server/root`
- `server/internal/mutant`
- `server/internal/saas`
- `helper/utils/output`

The expected outcome is deterministic package coverage for the highest-risk helper and boundary packages.

### Phase 2

Expand command conformance coverage for the remaining Tier-1 command families:

- `client/command/agent`
- `client/command/pivot`
- `client/command/pipe`
- `client/command/mutant`

Each command family should have:

- at least one happy-path case
- at least one validation failure that produces zero RPC calls
- at least one transport failure assertion
- direct protobuf field assertions for the main request shape

### Phase 3

Add deeper chain coverage where unit tests alone are not enough:

- parser or transport edge cases through the mock implant harness
- command-to-server flows that need tagged integration coverage
- build and certificate flows that need end-to-end file or config round-trips

### Phase 4

Use the inventory report to keep the roadmap current:

- update the manifest when the architecture changes
- rerun the inventory command after major test additions
- review the top gap list during test-related PRs
- promote the next Tier-1 gaps into the active sprint plan

## Refresh Workflow

Run the inventory command from the repository root:

```bash
go run ./scripts/testinventory -output dist/testing
```

The command writes:

- `dist/testing/core-testing-report.json`
- `dist/testing/core-testing-report.md`

Use that report to refresh priorities:

1. Review Tier-1 components with `missing` or `needs_attention` status.
2. Review chain-level missing layers.
3. Review the top gap list for broad package-level blind spots.
4. Pick the next smallest change set that upgrades one Tier-1 component or one core chain.

## Acceptance Criteria

The roadmap is being followed correctly when:

- every Tier-1 component in the manifest maps to at least one active test layer
- new command families land with `command_conformance` coverage first
- new transport and listener behavior lands with either `unit` or `mockimplant` guards
- CI keeps the inventory command runnable and the baseline suites green
- regression records under `docs/tests/` are updated when a coverage expansion finds real defects
