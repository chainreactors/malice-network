# Testing

## Overview

The repository now uses three test layers:

- Unit tests: default `go test ./...`
- Core race tests: `go test -race ./server/internal/core -count=1 -timeout 300s`
- Integration tests: explicit `integration` build tag
- Stress tests: reserved for future `stress`-tagged suites

PR CI runs unit tests, the targeted core race suite, and the client/server integration suite. Stress tests are intentionally out of scope for the current pipeline.

## Local Commands

Run the default CI-equivalent checks:

```bash
go mod tidy
go vet ./...
go test ./... -count=1 -timeout 300s
CGO_ENABLED=0 go build ./...
```

Run the client/server integration suite:

```bash
go test -tags=integration ./server ./client/command/listener ./client/command/pipeline ./client/command/website ./client/command/sessions ./client/command/context -count=1 -timeout 300s
```

Run the core race guard for concurrent state/session regressions:

```bash
go test -race ./server/internal/core -count=1 -timeout 300s
```

Run the mock implant task E2E guard:

```bash
go test -tags=mockimplant ./server -run MockImplant -count=1 -timeout 300s
```

Run the workflow locally with `act`:

```bash
act pull_request -W .github/workflows/ci.yaml
```

## Test Layout

- `client/core`: client-side state handling
- `client/command`: command-first conformance coverage for implant-facing CLI commands
- `server/rpc`: control-plane routing, authorization matching, and listener/pipeline resolution
- `helper/intl`: Lua bundle validation and embedded resource loading
- `server`: client/server integration entrypoint
- `server/testsupport`: reusable mTLS/gRPC harness for integration tests and mock implant E2E coverage

## Notes

- Integration tests use a real gRPC server, real mTLS certificates, and a lightweight fake listener control loop. This keeps authentication and state-sync behavior realistic without requiring implants or external processes.
- `server/internal/core` now has dedicated guards around task recovery, cache trimming, listener/job runtime state, secure rotation counters, and db-only session recovery through the real listener `Checkin` path.
- The mock implant harness adds a deeper task-path layer at `ListenerRPC/SpiteStream`. It is documented in `docs/tests/mock-implant-e2e.md`.
- Command conformance tests are documented in `docs/development/command-testing.md`.
- Detailed test records live under `docs/tests/`.
- Control-plane regression findings are tracked in `docs/tests/control-plane-regression-record.md`.
- `helper/intl` tests depend on the community Lua/resource bundle. When that bundle is not present in the checkout, the suite skips explicitly instead of failing nondeterministically.
- Local coverage collection on some Windows environments can be blocked by antivirus when Go writes instrumented temporary files. Coverage is useful for analysis, but it is not the sole CI gate.
