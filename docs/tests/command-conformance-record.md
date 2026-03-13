# Command Conformance Record

## Overview

This document records the test expansion work for command-layer conformance under `client/command`.

The primary goal of this effort was not to increase coverage numbers mechanically. The goal was to make the tests reliably expose command-layer defects, especially:

- Cobra argument parsing mistakes
- flag-to-request mapping mistakes
- missing validation before transport
- wrong default and fallback behavior
- wrong RPC method selection
- wrong protobuf field or value assembly
- swallowed control-plane RPC failures

This work intentionally treats most `server/rpc` handlers as thin transport adapters. Those handlers still keep regression tests, but they are not the main confidence layer for client commands whose real risk lives in command parsing and request assembly.

## Testing Strategy

### Why Command-First

For both implant and control-plane command paths, the most failure-prone logic is usually not in server-side forwarding. It is in the client command layer:

- flags are optional when they should be required
- aliases and shorthand values are normalized incorrectly
- path and registry formatting changes silently
- user input is accepted but assembled into the wrong protobuf shape
- the command calls the wrong RPC even though the transport itself works

Testing only direct helper functions or only server-side RPC forwarding would miss these failures.

### Chosen Shape

The adopted shape is a command-first conformance layer:

1. execute the real Cobra command path through the `implant` root
2. keep the backend deterministic with a recorder RPC
3. assert both transport intent and transport payload
4. fail fast when invalid input should never reach transport

This keeps the tests fast enough for default `go test ./...` while staying close to real operator behavior.

### Why Server RPC Was Kept Thin

Many RPC handlers in this area mainly forward a request to another layer. Those handlers still need regression protection, but duplicating the same assertions there produces little extra signal.

The command layer is where the user-facing contract is defined. That is where the stronger tests now live.

## Implementation Process

The work was implemented in the following order:

1. inventory the implant-related commands and existing tests
2. identify which paths were only covered by ad hoc fake RPC tests
3. identify command-layer defects that the new tests should expose
4. build shared harnesses under `client/command/testsupport`
5. migrate existing command tests onto the shared harnesses
6. add missing suites for uncovered command families
7. fix production issues exposed by the conformance cases
8. validate with package tests and full repository checks

## Shared Harness Design

The reusable harnesses live in:

- `client/command/testsupport/harness.go`
- `client/command/testsupport/recorder.go`

### Harness Responsibilities

The harness layer now provides:

- a temporary client runtime directory
- a real `core.Console`
- execution through the real Cobra command roots
- an implant-seeded harness for `implant` commands
- a client-root harness for non-implant control-plane commands
- optional pipeline and session fixtures

### Recorder Responsibilities

The recorder RPC captures:

- the RPC method name
- outgoing metadata such as `session_id` and `callee`
- the exact protobuf request object
- `SessionEvent` calls triggered by `session.Console(...)`

It also supports responder hooks for cases that need command flow control, such as:

- `WaitTaskFinish`
- `GetSession`
- `GetBasic`
- `GetListeners`
- `ListJobs`
- `GetLicenseInfo`
- `GetContexts`
- certificate and ACME control-plane RPCs
- default task-producing RPCs

### Why This Matters For Future E2E

The case shape is command-path driven instead of helper-function driven. That was intentional.

When a real implant E2E layer is added later, the same command cases can be reused with a different backend:

- today: recorder backend
- future: live implant backend

That means the current tests are not throwaway mocks. They are the fast layer of a future multi-layer test stack.

## Coverage Added

The current command conformance layer covers the following command families.

### Basic Commands

- `sleep`
- `keepalive`
- `suicide`
- `ping`
- `wait`
- `polling`
- `init`
- `recover`
- `switch`
- session prefix matching helper

### Service Commands

- `service list`
- `service create`
- `service start`
- `service stop`
- `service query`
- `service delete`

### Registry Commands

- `reg query`
- `reg add`
- `reg delete`
- `reg list_key`
- `reg list_value`

Registry type coverage includes:

- `REG_SZ`
- `REG_BINARY`
- `REG_DWORD`
- `REG_QWORD`

### Scheduled Task Commands

- `taskschd list`
- `taskschd create`
- `taskschd start`
- `taskschd stop`
- `taskschd delete`
- `taskschd query`
- `taskschd run`

Trigger alias coverage includes:

- `daily`
- `weekly`
- `monthly`
- `atlogon`
- `startup`

### System Commands

- `whoami`
- `kill`
- `ps`
- `env`
- `env set`
- `env unset`
- `netstat`
- `sysinfo`
- `bypass`
- `wmi_query`
- `wmi_execute`

### Control-Plane Commands

- `version`
- `broadcast`
- `license`
- `pivot`
- `listener`
- `job`
- `cert`
- `cert self_signed`
- `cert update`
- `cert download`
- `cert acme_config`

## Assertion Model

Each command case focuses on one operator-visible contract:

- the command path and argv are real
- exactly the expected RPC is called
- the protobuf request fields match the parsed user input
- metadata such as `session_id` and `callee` is preserved
- task-producing commands emit a session task event
- invalid input causes zero transport calls

This model is stricter than checking only that a helper function returns a request, because it verifies the full CLI path.

## Problems Found

The following defects were identified while expanding the tests.

### Fixed Before Or During This Expansion

- `service start` used the wrong module type before the earlier regression fix. It now sends `ModuleServiceStart` instead of `ModuleServiceCreate`.
- `basic wait` dereferenced the `WaitTaskFinish` result without checking `err` first. A failing RPC could produce a nil dereference path.
- `sys wmi_execute` assumed every `--params` item contained `=` and indexed `kv[1]` unconditionally. Malformed input could panic.
- `service create` documented `--name` and `--path` as required but did not enforce them.
- `taskschd create` documented `--name` and `--path` as required but did not enforce them.
- `cert` list swallowed `GetAllCertificates` failures and reported success on transport failure.
- `cert download` swallowed `DownloadCertificate` failures and reported success on transport failure.
- `cert update` dropped `cert` and `key` payloads whenever `--ca-cert` was omitted.
- `cert update` accepted a partial key pair instead of rejecting `--cert` or `--key` alone.
- `cert delete`, `cert update`, and `cert download` accepted a missing certificate name and could issue malformed requests.
- `version` swallowed `GetBasic` failures because the command path did not return errors.
- `broadcast` swallowed `Broadcast` and `Notify` failures because the command path logged and returned success.

### Product Lessons From These Defects

- Help text alone is not validation.
- Thin helper wrappers can still hide crash paths.
- Parsing bugs are often more dangerous than transport bugs because they are user-controlled.
- A command that "works" in the happy path may still be unsafe when malformed input is accepted.

## Fixes Applied

The following production changes were made and kept under regression tests:

- `client/command/service/start.go`
  - send `ModuleServiceStart`
- `client/command/basic/wait.go`
  - return RPC errors before accessing `content.Task`
  - reject empty responses safely
- `client/command/sys/wmi.go`
  - validate `--params` as `key=value`
- `client/command/service/commands.go`
  - mark `service create --name` and `--path` as required
- `client/command/taskschd/commands.go`
  - mark `taskschd create --name` and `--path` as required
- `client/command/cert/commands.go`
  - require a certificate name for `delete`, `update`, and `download`
- `client/command/cert/cert.go`
  - propagate certificate list and download transport errors
  - process `cert` and `key` without requiring `ca-cert`
  - reject partial key-pair updates
- `client/command/generic/commands.go`
  - convert `version` and `broadcast` to error-returning command paths
- `client/command/generic/version.go`
  - return `GetBasic` failures to Cobra instead of logging and succeeding
- `client/command/generic/broadcast.go`
  - return `Broadcast` and `Notify` failures to Cobra instead of logging and succeeding

## Why These Tests Are Effective

This layer is effective at exposing the classes of bugs that matter most here:

- wrong command subpath
- wrong Cobra parsing behavior
- wrong request type
- wrong enum mapping
- wrong path normalization
- missing pre-transport validation

It is intentionally less concerned with:

- rendering details of formatted output tables
- remote implant runtime behavior
- end-to-end server and listener orchestration

Those concerns belong to other layers.

## Relationship To Other Test Layers

The current stack is:

- command conformance tests in default `go test ./...`
- thin server RPC regression tests
- tagged client/server integration tests with real gRPC and mTLS
- future live-implant E2E layer

This separation keeps the suite fast while still making failures local and diagnosable.

## Remaining Limits

The current conformance layer still uses a recorder backend, so it does not prove:

- that a real implant executes the request correctly
- that transport framing matches an implant runtime perfectly
- that asynchronous multi-event behaviors match production timing

That is acceptable for the current stage. The important point is that the cases are now structured so they can be promoted to live E2E later.

## Verification

The current record was validated with:

```bash
go test ./... -count=1 -timeout 300s
go vet ./...
CGO_ENABLED=0 go build ./...
```

## Follow-Up Guidance

When adding a new implant-facing command:

1. add a command conformance case first
2. assert the exact RPC method and protobuf payload
3. add at least one invalid-input case that must produce zero transport calls
4. only add server-side duplication if the RPC handler contains real logic instead of forwarding

This keeps the main confidence layer aligned with where defects are most likely to appear.
