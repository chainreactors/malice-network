# Task Runtime Regression Record

## Overview

This document records the regressions found while checking the client/server task path, especially:

- task wait behavior
- task progress signalling
- task command parameter handling
- client/server task metadata consistency

The focus here is not just task-related commands. It is the runtime contract around a task:

- a task is created
- progress is emitted
- wait APIs observe progress and completion correctly
- client-side task helpers do not send malformed or inconsistent requests

## Areas Checked

The inspection focused on:

- `server/internal/core/task.go`
- `server/rpc/rpc-task.go`
- `client/command/tasks/*`
- command-path task coverage under `client/command`

## Regressions Found

### WaitTaskContent Could Not Observe Task Progress

`WaitTaskContent` was waiting on `task.DoneCh`, but `Task.Done(...)` never signalled that channel.

Practical effect:

- progress could arrive
- task cache could already contain the new spite
- `WaitTaskContent` would still block

### WaitTaskContent Returned The Wrong Result On Close

When `task.DoneCh` was closed, `WaitTaskContent` immediately returned `Task content not found` instead of re-checking the in-memory or disk-backed task content first.

Practical effect:

- a caller could miss valid content that already existed by the time the task closed

### WaitTaskContent Ignored Caller Cancellation

`WaitTaskContent` did not watch the RPC request context.

Practical effect:

- if the caller timed out or canceled the request, the server-side wait could keep hanging

### WaitTaskFinish Ignored Caller Cancellation

`WaitTaskFinish` only waited on the task context and ignored the RPC request context.

Practical effect:

- a client-side timeout or cancellation would not reliably stop the server-side wait

### WaitTaskContent Had An Index Validation Gap

The boundary check allowed `need == total`, even though valid task content indexes are `0 .. total-1`.

Practical effect:

- an invalid content index could slip past validation and fall into a wait path that could never succeed

### WaitTaskContent Rejected Streaming Tasks

Streaming task handlers use `Total = -1` until the stream is finished.

`WaitTaskContent` treated that value like a normal upper bound and rejected `need = 0` immediately because `0 >= -1`.

Practical effect:

- realtime task output could already be flowing from the implant
- the caller still got `ErrTaskIndexExceed` for the first chunk
- mock-implant E2E could not use `WaitTaskContent` against a real streaming task

### Finished Streaming Tasks Still Looked Unfinished

The runtime and DB finish checks only used `Cur == Total`.

Streaming tasks keep `Total = -1` while they are active, and the finish path never normalized that value or treated `FinishedAt` as authoritative completion state.

Practical effect:

- `WaitTaskFinish` returned a task protobuf that still reported `Finished=false`
- polling and runtime inspection could treat an already finished streaming task as active
- recovered streaming tasks with a recorded finish time could be rebuilt as open tasks again

### fetch_task Accepted Invalid IDs And Still Called RPC

`fetchTaskByID(...)` logged a parse error for an invalid task id but still continued and called `GetAllTaskContent`.

Practical effect:

- malformed user input was not rejected at the command layer
- the client could send a bogus `task_id=0` request

### tasks --all Did Not Actually Request Full History

The `tasks --all` flag existed, but the command always used the default `UpdateTasks(...)` path and never passed `All=true` to the server.

Practical effect:

- operators could ask for full task history and still receive only the default task set

### Task Commands Used Global Context Instead Of Session Context

`tasks` and `fetch_task` used `con.Context()` instead of `session.Context()`.

Practical effect:

- outgoing metadata such as `session_id` and `callee` was missing
- task-related command behavior diverged from the rest of the implant command path

### Incremental Task Progress Was Persisted As Full Completion

`db.UpdateTask(...)` wrote `task.Total` into the `cur` column instead of `task.Cur`.

Practical effect:

- a multi-stage task looked finished in the database after its first callback
- `tasks --all`, task recovery, and any DB-backed inspection path could observe a fake-complete state
- task recovery after reconnect could rebuild the wrong runtime state

### Recovered Tasks Were Missing Runtime Wiring

`RecoverSession(...)` rebuilt task protobuf fields but did not restore the runtime-only links needed by the task state machine.

Missing pieces:

- `task.Session`
- `task.DoneCh`

Practical effect:

- recovered tasks were not structurally equivalent to live tasks
- wait and cleanup paths could behave differently after reconnect/recovery
- any runtime path that expected a fully wired task object could panic or silently stop observing progress

### GetOrRecover Detached Task Context From Session Context

`Tasks.GetOrRecover(...)` rebuilt task context from `context.Background()` instead of `sess.Ctx`.

Practical effect:

- on-demand recovered tasks no longer followed the owning session lifecycle
- session shutdown or removal did not cancel these recovered task contexts
- wait/cleanup paths could outlive the session they belonged to

### Dead Sweep Removed Sessions That Still Owned Pending Tasks

Inactive-session sweeping removed the runtime session unconditionally.

Practical effect:

- `sessions.Remove(...)` canceled the parent session context
- pending task contexts derived from that session context were canceled too
- `WaitTaskFinish` / `WaitTaskContent` could fail before the implant replied
- a late implant callback hit `core.Sessions.Get(...)` inside `ListenerRPC.SpiteStream`, could not find the session, and was dropped
- DB state still showed the task/session relationship, but the live runtime path had already been torn down

## Fixes Applied

The following production changes were made:

- `server/internal/core/task.go`
  - `Task.Done(...)` now signals `DoneCh`
  - recovered unfinished tasks no longer start with a closed wait channel
  - streaming task finish now seals `Total` to the observed callback count
  - `Finished()` now also honors `FinishedAt`
- `server/internal/core/session.go`
  - session recovery now restores runtime task wiring (`Session`, `DoneCh`, closed state)
  - recovered tasks now use `Finished()` instead of raw `Cur == Total`
  - inactive session sweeping now keeps dead sessions in memory while unfinished tasks still exist
  - dead/reborn runtime state is now tracked explicitly so dead events are not re-published on every sweep
- `server/rpc/rpc-task.go`
  - `WaitTaskContent` now:
    - validates indexes correctly
    - skips upper-bound rejection for streaming tasks (`Total < 0`)
    - re-checks cache/disk after signals
    - respects caller cancellation
  - `WaitTaskFinish` now respects caller cancellation
- `server/rpc/rpc-listener.go`
  - late implant callbacks now refresh session activity and revive retained dead sessions before task delivery
- `server/rpc/rpc-implant.go`
  - checkins for retained dead sessions now also publish a correct reborn transition
- `server/internal/db/session_helper.go`
  - incremental task progress now persists `Cur` instead of overwriting DB state with `Total`
  - runtime task updates now persist `Total` as well, so streaming tasks can be normalized on finish
- `server/internal/db/models/task.go`
  - DB task protobuf conversion now treats recorded finish time as authoritative finished state
- `client/command/tasks/tasks.go`
  - `tasks --all` now sends `All=true`
  - `fetch_task` now rejects invalid ids before transport
  - task commands now use `session.Context()`

## Regression Coverage Added

The following tests were added:

- `server/rpc/rpc_task_wait_test.go`
  - task progress unblocks `WaitTaskContent`
  - `WaitTaskContent` rejects `need == total`
  - `WaitTaskContent` respects caller timeout/cancel
  - `WaitTaskFinish` respects caller timeout/cancel
- `server/rpc/generic_runtime_test.go`
  - multi-stage task callbacks persist incremental DB progress instead of fake completion
- `server/rpc/rpc_task_recovery_test.go`
  - `RecoverSession(...)` restores runtime task wiring for recovered tasks
  - `GetOrRecover(...)` binds recovered task contexts to the owning session lifecycle
- `server/mock_implant_task_e2e_test.go`
  - `Sleep` proves single-response task completion over the real listener stream
  - realtime `Execute` proves multi-callback `WaitTaskContent` and final streaming task state
- `server/mock_implant_lifecycle_edge_e2e_test.go`
  - dead sweep keeps a pending single-response task alive until the delayed callback arrives
  - dead sweep keeps a streaming task alive after partial output and allows the final callback to finish it
  - idle dead sessions are removed from runtime memory and recovered again by a real implant `Checkin`
- `client/command/tasks/tasks_test.go`
  - `tasks --all` sends `All=true`
  - `fetch_task` rejects invalid ids before transport
  - `fetch_task` sends the expected task lookup request
- `server/internal/core/session_test.go`
  - `SweepInactive` keeps sessions with unfinished tasks
  - `SweepInactive` removes truly idle dead sessions

## Verification

This record was validated with:

```bash
go test -tags mockimplant ./server -run "MockImplant" -count=1 -timeout 300s
go test -tags mockimplant ./server/... -count=1 -timeout 300s
go test ./server/rpc ./client/command/tasks -count=1 -timeout 300s
go test ./... -count=1 -timeout 300s
go vet ./server/internal/core ./server/rpc ./client/command/tasks ./client/command/testsupport
```
