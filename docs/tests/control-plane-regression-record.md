# Control Plane Regression Record

## Overview

This document records the concrete regressions found while expanding test coverage for the client command layer, server RPC layer, and control-plane integration path.

The current focus areas are:

- `pipeline`
- `website`
- `sessions`
- `context`
- client/server state reconciliation

The goal of this record is to keep bug discovery, fixes, and regression coverage traceable in one place.

## Coverage Scope

The current regression coverage now includes:

- command conformance tests under `client/command/...`
- client/server integration tests with real gRPC and mTLS
- control-plane harness tests for listener job flows
- persistence and round-trip tests under `server/internal/db/...`
- CI automation through `.github/workflows/ci.yaml`

## Regressions Found And Fixed

### Website And Web Content

- Website event rendering could panic when `pipeline.Tls == nil`.
- Client state reconciliation did not handle `website_start` and `website_stop`.
- Client state reconciliation did not handle `web_content_add`, `web_content_remove`, or `web_content_add_artifact`.
- Initial client sync did not load persisted websites.
- Initial client sync did not load persisted website contents.
- `website stop --listener` ignored the explicit listener selector.
- `StartWebsite` did not resolve the website by listener safely and did not roll back cleanly on listener startup failure.
- `StopWebsite` disabled database/runtime state before listener stop succeeded.
- `DeleteWebsite` removed persisted state before listener stop succeeded.
- Explicit `ContentType` values were overwritten by extension-derived MIME detection.
- Website content updates could return stale metadata after an overwrite.
- Removing website content deleted the database row but left the stored file behind.

### Pipeline And REM

- `StartPipeline` did not consistently wait for listener control success before finalizing state.
- `StopPipeline` disabled and removed runtime state before listener stop success was confirmed.
- `DeletePipeline` removed database/runtime state before listener stop success was confirmed.
- `StartRem` had the same missing control-status handling and rollback problem.
- `StopRem` changed state before listener stop was confirmed.
- `DeleteRem` deleted persisted state before listener stop was confirmed.
- Client state reconciliation did not handle `rem_start` and `rem_stop`, so REM runtime changes were invisible in the client cache.
- Listener pipeline listing rendered REM pipelines as `bind`.
- `bind` pipeline creation without a name could register an empty-name pipeline.
- `rem new` without a name could register an empty-name pipeline.
- `http --error-page` sent the file path string instead of the file content payload expected by the runtime.

### Sessions And Context

- `sessions note` swallowed no-session and RPC failure paths.
- `sessions group` returned an incomplete error path when no session was selected.
- Removing a missing session could report success instead of not found.
- `GetContextsByTask` ignored the type filter.
- `context list` could panic when a context had no associated session.
- `AddDownload` allowed missing session/task state and failed too late.
- Download, credential, media, port, screenshot, upload, and keylogger listing paths were not robust when `Session == nil`.
- Screenshot listing used the wrong identifier field in output.
- `observe` without explicit arguments failed to fall back to the active interactive session.
- Short session identifiers could trigger slicing panics in prefix matching and session listing.

### Output And CLI Consistency

- `ListRemCmd` bypassed the console logger and wrote directly to standard output, making output capture and behavior inconsistent with other commands.

## Regression Guards

The fixes above are now protected by a mix of:

- tagged integration tests for `listener`, `pipeline`, `website`, `sessions`, `context`, and `server`
- command-level unit tests for session and command parsing edge cases
- database persistence tests for website content lifecycle
- protobuf/model round-trip tests for TCP, HTTP, Bind, REM, and Website pipeline structures

## Verification Commands

The validated commands for the current record are:

```bash
go test ./... -count=1 -timeout 300s
go test -tags=integration ./server ./client/command/listener ./client/command/pipeline ./client/command/website ./client/command/sessions ./client/command/context -count=1 -timeout 300s
go vet ./...
CGO_ENABLED=0 go build ./...
```

## CI Coverage

The GitHub Actions workflow now runs:

- unit and default package tests in the `unit` job
- tagged client/server integration suites in the `integration` job

This is the current automated baseline for control-plane regression prevention.
