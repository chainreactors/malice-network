# Mock Implant E2E

## Overview

This document records the mock implant mechanism used to exercise the real task path without requiring a compiled implant binary.

The goal is to cover a deeper layer than recorder-based command tests:

- real gRPC + mTLS
- real `ListenerRPC/SpiteStream`
- real task creation on the server
- real task wait APIs observing the returned implant response

This is not a packet-level transport emulator. It intentionally starts at the server-facing listener RPC boundary.

## Why This Layer Exists

Recorder-backed command tests are still the fastest way to catch:

- flag parsing errors
- argument validation gaps
- wrong RPC method selection
- malformed protobuf assembly

They do not prove that a task can actually traverse:

1. client/server RPC entry
2. server task creation
3. request delivery over the listener stream
4. implant response delivery back into the task runtime
5. `WaitTaskFinish` and later E2E wait behavior

The mock implant closes that gap.

The intended defect-detection chain is:

1. command conformance tests catch flag parsing and protobuf assembly bugs
2. mock implant E2E catches task/runtime/listener-stream bugs on the real server path
3. real implant E2E catches command -> RPC -> implant behavior drift

The point of this stack is to expose problems early, not to hide them with a
forgiving mock.

If a failure is caused by the implant implementation itself, keep the test
signal, document the mismatch, and fix the implant separately. Do not weaken
the server or mock harness just to make an implant bug disappear.

## Scope

The mock implant currently simulates:

- a listener-role client that authenticates with real mTLS
- an implant session registering through `ListenerRPC.Register`
- a live bidirectional `ListenerRPC.SpiteStream`
- request capture by module name
- scripted responses returned as real `SpiteResponse` messages
- active response streaming from the handler, so one request can emit multiple delayed callbacks
- reusable scenario state for:
  - filesystem paths and file contents
  - environment variables
  - process and netstat inventory
  - drive inventory
  - registry keys and values
  - service inventory and service status transitions
  - scheduled task inventory and enable/run transitions
  - module/addon inventory

It does not simulate:

- raw TCP/HTTP listener packet formats
- encryption/parser compatibility at the network listener boundary
- a full implant runtime or OS behavior

That tradeoff is deliberate. The task/runtime bugs found so far have been above the packet layer.

## Harness Design

The reusable harness lives in:

- `server/testsupport/mock_implant.go`

It works with `server/testsupport/controlplane.go`:

- seeds a real runtime pipeline into the listener registry
- creates a real listener-role mTLS identity
- opens `ListenerRPC.SpiteStream` with `pipeline_id` metadata
- registers a session with `ListenerRPC.Register`
- immediately performs the first `ListenerRPC.Checkin`, so the session is in a
  post-register ready state before task assertions begin
- keeps an optional periodic checkin loop enabled by default to simulate the
  normal beacon cadence more closely
- receives `SpiteRequest` values from the server
- dispatches them to per-module scripted handlers
- sends `SpiteResponse` values back over the same stream

Recent changes intentionally moved the mock closer to the real implant runtime
at the process level:

- mock `SessionID` now follows the real `raw id -> session id` derivation model
- startup is modeled as `register -> first checkin -> task-ready session`
- periodic checkins can be paused per test with `PauseAutoCheckins()` when an
  edge case needs a forced stale/dead window
- realtime `exec` now uses the same visible-output-then-terminal-marker shape as
  the real implant

This is the current priority order:

1. simulate register/checkin/task flow correctly
2. simulate task wait/progress/finish/recovery correctly
3. only then add richer per-module behavior

Closer to real does not mean more permissive. The mock should mirror the real
implant's normal request/response shape, but it should not silently normalize:

- wrong command argument order
- wrong protobuf field mapping
- missing state transitions
- server-side assumptions that only pass against a fake happy path

When the real implant later disagrees with the mock, update the documentation
first, then decide whether the mock or the implant is wrong.

## Current E2E Guards

The current mock-implant E2E regression tests are:

- `server/mock_implant_task_e2e_test.go`
- `server/mock_implant_common_rpc_e2e_test.go`
- `server/mock_implant_state_e2e_test.go`
- `server/mock_implant_lifecycle_edge_e2e_test.go`

They now prove that:

- `Sleep` creates a real task
- the server sends the expected request to the implant stream
- the mock implant can respond later
- `WaitTaskFinish` blocks until that real streamed response arrives
- realtime `Execute` can emit multiple callbacks through the same stream
- `WaitTaskContent` can observe callback `0` and callback `1` on the real task path
- the finished streaming task is exposed back to the caller as a true finished state
- common query RPCs preserve request parameters and return realistic state:
  - `Info`
  - `Ping`
  - `Pwd`
  - `Ls`
  - `Cat`
  - `Ps`
  - `Netstat`
  - `Env`
  - `Whoami`
- inventory RPCs return consistent session-side state:
  - `EnumDrivers`
  - `ServiceList`
  - `ServiceQuery`
  - `TaskSchdList`
  - `TaskSchdQuery`
  - `RegListKey`
  - `RegListValue`
  - `RegQuery`
  - `ListModule`
  - `ListAddon`
  - `ListTasks`
  - `QueryTask`
- mutation and lifecycle RPCs actually mutate the mock implant state across follow-up requests:
  - `SetEnv` / `UnsetEnv`
  - `RegAdd` / `RegDelete`
  - `Mkdir`
  - `Cd`
  - `Touch`
  - `Cp`
  - `Mv`
  - `Rm`
  - `ServiceCreate` / `ServiceStart` / `ServiceStop` / `ServiceDelete`
  - `TaskSchdCreate` / `TaskSchdStart` / `TaskSchdStop` / `TaskSchdRun` / `TaskSchdDelete`
  - `LoadModule`
  - `LoadAddon`
  - `ExecuteAddon`
- control and execution RPCs preserve transport semantics and runtime side effects:
  - `Keepalive`
  - `Switch`
  - `Clear`
  - non-realtime `Execute`
- system-action RPCs preserve assembly and return type expectations:
  - `Curl`
  - `WmiQuery`
  - `WmiExecute`
  - `Runas`
  - `Privs`
  - `GetSystem`
  - `Kill`
  - `Bypass`
  - `Rev2Self`
- session state transitions are validated across runtime memory and DB persistence:
  - register-time sysinfo/workdir initialization
  - post-register checkin updates the session into a task-ready state
  - `Sleep` timer update
  - `Cd` working-directory update
  - `Info` sysinfo refresh and normalization
- task state transitions are validated across gRPC return values, runtime memory, DB rows, and task-content APIs:
  - single-response tasks: created -> pending -> finished -> closed
  - streaming tasks: `Total=-1` pending -> visible callback progress -> empty terminal end marker -> finish normalization -> recovery from persisted state
  - `GetTasks`, `GetTaskContent`, `GetAllTaskContent`, and `WaitTaskFinish` all reflect the same state progression
- session/task lifecycle edge behavior is validated through the real listener stream:
  - a dead sweep does not remove a session that still owns unfinished tasks
  - a late single-response callback after dead marking still finishes the task
  - a late streaming callback after dead marking still advances and finishes the task
  - an actually idle dead session is removed from runtime memory
  - a real implant `Checkin` can recover that removed session into runtime memory again
  - late response activity and recovered checkins both restore DB/runtime alive state

This is the current minimum end-to-end guard for the task path without needing a real implant executable.

## Regressions Found With This Layer

The mock implant E2E layer exposed two runtime bugs that lower-level tests had not exercised through the real listener stream:

- `WaitTaskContent` rejected streaming tasks because it treated `task.Total == -1` as a normal upper bound and considered `need=0` already out of range.
- finished streaming tasks still looked unfinished because runtime and DB protobuf state only considered `Cur == Total`, while streaming tasks stayed at `Total = -1`.

Running the suite also exposed an unrelated but important build blocker:

- the new bridge-agent/LLM work in `client/command/agent` and `server/rpc` was ahead of the current `external/IoM-go` proto definitions and broke the default build.

That bridge-agent code is now gated behind a build tag until the proto/RPC definitions actually exist, so task/runtime tests stay runnable in the default suite.

The expanded common-RPC suite exposed another concrete integration bug:

- `Runas` reached `server/rpc.Runas`, but `external/IoM-go/types.BuildSpite` did not know how to encode `RunAsRequest`, so the request failed before it ever hit the listener stream with `unknown spite body`.

That has now been fixed by adding `RunAsRequest -> Spite_RunasRequest` mapping in `external/IoM-go/types/build.go`.

The state-oriented suite exposed two more server runtime issues:

- `Cd` completed successfully but did not update the server-side session `Workdir`, so later session state was stale even though the implant had changed directory.
- newly created runtime tasks had no `CreatedAt` or `Deadline`, which made them appear timed out immediately in the task protobuf view.

These are now fixed in:

- `server/rpc/rpc-filesystem.go`
- `server/internal/core/session.go`

The lifecycle edge suite exposed another task/session state bug:

- inactive-session sweeping removed runtime sessions unconditionally, even when unfinished tasks were still waiting on implant callbacks. That canceled the parent session context, canceled task contexts, and caused late implant responses to be dropped because `ListenerRPC.SpiteStream` could no longer find the session.

This is now fixed by splitting dead marking from runtime removal:

- `server/internal/core/session.go`
  - dead sessions with unfinished tasks are kept in memory
  - idle dead sessions are still removed
- `server/rpc/rpc-listener.go`
  - late implant responses refresh `LastCheckin`, persist the session, and clear the dead marker
- `server/rpc/rpc-implant.go`
  - checkins for retained dead sessions now also clear the dead marker and publish reborn state correctly

The closer-to-real streaming shape exposed another task-runtime bug:

- runtime cache used zero-based callback indexes, but on-disk `TaskLog` content
  was persisted with one-based indexes after `task.Done()`
- that let `WaitTaskContent(need=1)` incorrectly return the first callback as
  soon as disk fallback was consulted, before callback index `1` had really
  arrived

This is now fixed in:

- `server/internal/core/session.go`
  - persisted task-content indexes now match the in-memory zero-based callback
    indexes
- `server/rpc/rpc_task_wait_test.go`
  - added a regression test to ensure disk fallback waits for the correct next
    callback index

The broader command -> RPC -> implant effort also exposed real-implant issues
that should stay documented instead of being hidden behind mock behavior:

- scheduled-task behavior previously had real implant mismatches around task
  scheduler lifecycle/path semantics and required implant-side fixes
- these should be treated as implant defects, not reasons to relax the server
  or mock expectations

## Scenario Library

The reusable scenario library lives in:

- `server/testsupport/mock_scenarios.go`

It intentionally keeps mutable state so a test can validate real follow-up behavior instead of single-call smoke output.

Examples:

- `Cd` changes the working directory seen by the next `Pwd`
- `Cp` and `Mv` affect the next `Ls` / `Cat`
- `RegAdd` and `RegDelete` affect the next `RegQuery`
- `ServiceStart` affects the next `ServiceQuery`
- `TaskSchdRun` affects the next `TaskSchdQuery`
- `LoadModule` and `LoadAddon` affect the next session inventory refresh

This is the key property that makes the mock implant useful as a bridge toward real implant E2E:

- command/request assembly is still verified at the server boundary
- the same task transport path is exercised
- follow-up queries can detect state drift or missing side effects

## Running The Suite

These tests are behind the `mockimplant` build tag.

Typical entrypoints:

- `go test -tags mockimplant ./server -run "MockImplant" -count=1`
- `go test -tags mockimplant ./server/... -count=1 -timeout 300s`

## How This Fits With Existing Layers

- `client/command/testsupport`: still the main command conformance layer
- `server/rpc` and `server/internal/core`: still the main runtime regression layer
- `server/testsupport/mock_implant.go`: new server-facing E2E layer for task transport realism
- `server/testsupport/mock_scenarios.go`: reusable realistic implant-state layer for multi-step RPC scenarios
- `client/command/real_implant_command_e2e_test.go`: real command -> RPC -> implant closure for final confirmation

The intended progression is:

1. recorder-backed command tests catch parsing/assembly bugs fast
2. runtime tests catch task-state and wait bugs directly
3. mock implant E2E proves the task can survive the real listener stream boundary
4. real implant E2E confirms the same command path against a real binary

Failure handling rule:

- server/mock issue: fix it here and add coverage
- command assembly issue: fix the command test and command code
- implant issue: record it in docs and hand it back to implant development

## Current Limitation

The mock implant now covers both single-response completion and multi-callback task progress, plus multi-step mutable scenarios, but it is still intentionally scoped to the server-facing stream boundary.

The next useful expansions are:

- cancellation behavior
- broader reconnect/recovery flows with a live mock implant after transport interruption
- one command-path integration test that drives the mock implant from `client/command`
- more streaming-style modules beyond `Execute`
- duplicate/late-extra callback handling after a task is already finished
