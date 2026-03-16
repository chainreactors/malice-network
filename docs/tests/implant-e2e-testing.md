# Implant E2E Testing

This document describes the current real-implant integration test path for the
Go teamserver repository.

It replaces the earlier generic Rust-module guide. The current test target is
the Malice server/listener/session/task stack in this repository, not the
standalone Rust module workspace.

## Goals

The real-implant suite exists to validate the parts that mock-only coverage
cannot prove:

- the teamserver can start a real implant-facing listener socket
- a patched `malefic.exe` can register against that listener
- task delivery reaches a real implant runtime
- real task callbacks drive the server-side task/session state machine
- dead-session and late-response recovery still work with an actual implant

Mock implant tests remain the main regression suite for command parameter
parsing, request assembly, and broad RPC matrix coverage. Real implant tests are
the narrow but high-signal verification layer on top.

The real suite reuses the same task/session assertions introduced by the
mock-based state suites. The difference is that the transport, registration,
checkin cadence, and late callback behavior now come from a real
`malefic.exe` process instead of an in-memory responder.

## Current Coverage

The `realimplant` suite currently covers these server-side behaviors through a
real `malefic.exe` process:

- `sleep`
- `keepalive`
- `pwd`
- `ls`
- `sysinfo`
- `run`
- `exec` realtime streaming
- task progress transition `0/-1 -> 1/-1 -> 2/2`
- dead-session mark while a task is still pending
- late task response reborning a dead session
- database/runtime consistency for session alive state and task finish state

It also now covers the client command closure for the same real transport path:

- `implant --use <sid> --wait sleep 7 --jitter 0.15`
- `implant --use <sid> --wait keepalive enable|disable`
- `implant --use <sid> --wait sysinfo`
- `implant --use <sid> --wait pwd`
- `implant --use <sid> --wait ls <workdir>`
- `implant --use <sid> --wait run cmd.exe /c echo ...`
- `implant --use <sid> --wait mkdir/cd/pwd/touch/cp/cat/mv/ls/rm`
- `implant --use <sid> --wait upload|download`
- `implant --use <sid> --wait env`, `env set`, `env unset`, `whoami`, `ps`, `netstat`, `enum_drivers`
- `implant --use <sid> --wait kill`, `bypass`
- `implant --use <sid> --wait reg ...` on `HKCU`
- `implant --use <sid> --wait service list|query`
- `implant --use <sid> --wait taskschd list|query`
- `implant --use <sid> --wait wmi_query|wmi_execute`
- `implant --use <sid> --wait privs`
- `implant --use <sid> --wait runas` invalid-credential diagnostic path
- `implant --use <sid> --wait getsystem` diagnostic path

This is intentionally smaller than the `mockimplant` matrix.

The rule is:

- mock tests cover breadth
- real implant tests cover transport reality and state-machine truth

## Current Findings

Keep confirmed implant-side defects in:

- [implant-bugs.md](/D:/Programing/go/chainreactors/malice-network/docs/tests/implant-bugs.md)

One separate server-side bootstrap issue is still visible in real runs:

- the first listener-side `Checkin` can race ahead of `Register`, producing a
  transient `record not found` warning during session bootstrap

## Privilege Split

Default real-implant regression should stay non-admin. The current non-admin
path includes:

- basic control: `sleep`, `keepalive`, `sysinfo`
- filesystem: `mkdir`, `cd`, `pwd`, `touch`, `cp`, `cat`, `mv`, `ls`, `rm`
- system inventory: `env`, `setenv`, `unsetenv`, `whoami`, `ps`, `netstat`, `enum_drivers`
- Windows management without elevation: `reg` on `HKCU`, `service list|query`, `taskschd list|query`, `wmi_query`, `wmi_execute`

Admin-required scenarios are tracked separately and should not block the
default non-admin pass:

- `taskschd create|run|delete`
- future `service create|start|stop|delete`
- registry writes under privileged hives such as `HKLM`
- fully successful token/elevation flows that require real credentials or an
  elevated implant

## Real Runtime Differences From Mock

The real implant exposed several behaviors that the mock harness did not model:

- registration is not enough for the first task to be reliable; the suite waits
  for the first post-register checkin before issuing the first RPC task
- realtime `exec` may emit multiple stdout chunks before completion
- the final realtime `exec` callback can be an empty terminal marker with
  `end=true`, so the suite validates aggregate content with `GetAllTaskContent`
  instead of assuming the final chunk contains the last visible output
- repeated real test runs can reuse the same raw/session identifier, so process
  global transport and RPC stream state must be reset between harness instances

These are real protocol/runtime facts, not test-only workarounds.

## Architecture

The real test path is:

1. Start the in-process gRPC control plane with `ControlPlaneHarness`.
2. Start a real in-process listener via `server/listener.NewListener`.
3. Register and start a real TCP pipeline over admin RPC.
4. Generate an `implant.yaml` from the started pipeline.
5. Patch the local Rust `malefic.exe` template with `malefic-mutant tool patch-config`.
6. Spawn the patched implant process.
7. Wait for the real session to register.
8. Run the same style of task/session assertions used by the mock state tests.

This matters because the old harness only seeded pipeline metadata in memory and
DB. It did not open a real implant-facing socket, so a real implant had nothing
to connect to.

## Files

Main implementation files:

- [server/testsupport/real_implant.go](/D:/Programing/go/chainreactors/malice-network/server/testsupport/real_implant.go)
- [server/testsupport/runtime_inspect.go](/D:/Programing/go/chainreactors/malice-network/server/testsupport/runtime_inspect.go)
- [server/real_implant_e2e_test.go](/D:/Programing/go/chainreactors/malice-network/server/real_implant_e2e_test.go)
- [client/command/real_implant_command_e2e_test.go](/D:/Programing/go/chainreactors/malice-network/client/command/real_implant_command_e2e_test.go)

The existing mock state suites that the real tests were derived from:

- [server/mock_implant_state_e2e_test.go](/D:/Programing/go/chainreactors/malice-network/server/mock_implant_state_e2e_test.go)
- [server/mock_implant_lifecycle_edge_e2e_test.go](/D:/Programing/go/chainreactors/malice-network/server/mock_implant_lifecycle_edge_e2e_test.go)

## Prerequisites

The real suite expects a local Rust implant workspace and debug binaries. By
default it uses:

- workspace: `D:\Programing\rust\implant`
- template: `D:\Programing\rust\implant\target\debug\malefic.exe`
- mutant: `D:\Programing\rust\implant\target\debug\malefic-mutant.exe`

The suite is guarded twice:

- build tag: `realimplant`
- env gate: `MALICE_REAL_IMPLANT_RUN=1`

If the env gate is not set, the tests skip cleanly.

## Environment Variables

Optional overrides:

- `MALICE_REAL_IMPLANT_RUN=1`
- `MALICE_REAL_IMPLANT_WORKSPACE`
- `MALICE_REAL_IMPLANT_BIN`
- `MALICE_REAL_IMPLANT_MUTANT`

Examples:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
$env:MALICE_REAL_IMPLANT_BIN = "D:\Programing\rust\implant\target\debug\malefic.exe"
$env:MALICE_REAL_IMPLANT_MUTANT = "D:\Programing\rust\implant\target\debug\malefic-mutant.exe"
```

## Running

Run only the real implant suite:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
go test ./server -tags realimplant -run TestRealImplant -count=1 -timeout 300s
```

Run the client command closure against the same real implant path:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
go test ./client/command -tags realimplant -run TestRealImplantCommand -count=1 -timeout 300s
```

Run only the default non-admin command suites:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
go test ./client/command -tags realimplant -run "TestRealImplantCommand(BasicModulesE2E|FilesystemModulesE2E|SystemInventoryModulesE2E|WindowsManagementModulesE2E)$" -count=1 -timeout 300s
```

Run the privileged command suite explicitly:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
go test ./client/command -tags realimplant -run TestRealImplantCommandWindowsPrivilegedModulesE2E -count=1 -timeout 300s
```

Run a single case:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
go test ./server -tags realimplant -run TestRealImplantDeadSweepKeepsPendingStreamingTaskAlive -count=1 -timeout 300s
```

## Design Choices

### Real listener instead of seeded pipeline

The critical change is that the real suite starts an actual listener process in
the test runtime and then starts a real TCP pipeline through RPC.

Without that, real implant tests are fake: the session/task logic may run, but
the implant transport layer is never exercised.

### TCP + AES only

The first real suite uses a plain TCP pipeline with AES payload encryption:

- no TLS
- no secure mode
- no HTTP camouflage

This is deliberate. The first goal is reliable task/session state validation.
TLS, HTTP, and secure-mode coverage can be added after the plain transport path
is stable.

### Keepalive before edge-case lifecycle checks

The dead-session streaming test enables `keepalive` before forcing the session
stale. That suppresses normal heartbeat timing enough to make the edge case
deterministic:

- the session is marked dead
- the pending task keeps the runtime session resident
- the late task response revives the session

If the test relied on the normal 1-second heartbeat loop, spontaneous checkins
could mask the bug.

### Process-global isolation between real tests

Running the real cases one by one was not enough. When the combined suite ran in
the same Go process, stale entries in the transport and RPC globals could route
the second test through the first test's stream state.

The harness now resets these transient structures for every isolated real test
control plane:

- `core.Connections`
- `core.Forwarders`
- `core.ListenerSessions`
- `rpc.pipelinesCh`
- `rpc.ptyStreamingSessions`

Without this reset, the suite can pass individually and still fail when the two
real tests run back-to-back.

## Pitfalls And Lessons

The real suite exposed a set of recurring failure modes. These are the practical
rules that came out of that work.

### Do not treat registration as task-ready

A real session existing in runtime memory is not yet enough to issue the first
task safely.

The stable sequence is:

1. implant registers
2. server persists and exposes the session
3. implant performs the first normal checkin
4. only then issue the first RPC task

If the test sends the first task immediately after `Register`, the first task
can race the real beacon loop and fail intermittently.

### Realtime output and task completion are not the same thing

Visible output is only part of the realtime `exec` contract.

The real implant may:

- emit multiple visible stdout callbacks
- end with a final empty callback
- mark only that final callback as `end=true`

The test strategy that proved stable is:

- use `WaitTaskContent` for intermediate progress checks
- use `GetAllTaskContent` to validate that expected visible output appeared
- use `WaitTaskFinish` to validate the terminal marker and finished task state

Do not assume the last visible output chunk is also the finishing callback.

### Listener teardown order matters

Real implant teardown was initially flaky because stopping the pipeline alone
did not fully release listener-side gRPC state.

The cleanup that proved reliable is:

1. stop the implant process
2. stop the pipeline through RPC
3. close the in-process listener explicitly
4. stop the control-plane gRPC server

Without the explicit listener close, `GracefulStop` could remain blocked on open
streams.

### Always run the combined suite, not only single tests

The real suite initially passed case-by-case and still failed as a group.

That failure turned out to be test pollution from process-global state, not a
task-state bug in the individual case itself. For real implant coverage, a
single passing test is necessary but not sufficient.

The minimum validation loop is:

- run the single case while developing it
- run the full `TestRealImplant` suite before considering the test stable

### Keep edge cases deterministic by suppressing normal heartbeats

For dead-session and late-response scenarios, normal checkins are noise.

The most reliable pattern was:

- enable `keepalive`
- force the session stale
- sweep inactive sessions
- wait for the pending task callback to revive the session

This removes dependence on the normal heartbeat cadence and makes the
dead/reborn transition reproducible.

### Keep the first real transport simple

TCP + AES only was the correct first step.

Trying to validate:

- real implant process
- real listener socket
- TLS setup
- secure mode
- HTTP camouflage

all at once would have hidden the actual failure source. The useful order is to
prove plain transport and state-machine behavior first, then layer additional
transport features later.

### Prefer absolute paths and existing fixture files in filesystem E2E

The first filesystem command suite used `shell` redirection to create a source
file inside the implant. That added avoidable `cmd.exe` quoting noise and
produced a false negative before `cp` ever ran.

The stable pattern is:

- use absolute paths
- create empty files with `touch`
- copy an existing real text file such as the generated implant YAML

This keeps the failure signal on the filesystem module under test instead of on
shell escaping.

### Preserve implant stdout and stderr in failures

When a real implant exits early, the binary's own output is often the only fast
way to distinguish:

- config schema mismatch
- binary/module mismatch
- local security interference
- transport startup failure

Every real test harness should keep process stdout/stderr attached to the
failure path.

## Authoring Checklist

When adding a new real implant case, keep this checklist:

- start a real listener and a real started pipeline, not only seeded metadata
- wait for register and then for the first post-register checkin
- prefer harmless read-only modules first
- for streaming tasks, validate progress and terminal marker separately
- force deterministic timing for lifecycle edge cases instead of relying on
  ambient heartbeats
- close implant, pipeline, and listener explicitly during cleanup
- run the single test and then the combined `TestRealImplant` suite

## What Real Tests Should Cover

Use real implant tests for:

- session registration truth
- listener/pipeline transport truth
- Cobra command -> RPC -> implant closure
- task callback timing
- wait/task completion behavior
- dead/reborn lifecycle transitions
- runtime vs DB state consistency

Do not use real implant tests as the main place for:

- full RPC breadth
- exhaustive parameter assembly
- rare error permutations
- command parser corner cases

Those stay in `mockimplant` because they are faster, broader, and easier to
debug.

## Extending Coverage

Recommended next additions, in order:

1. `info`
2. `ls`
3. `ping`
4. HTTP pipeline variant
5. TLS TCP pipeline variant
6. idle-dead-session removal and later heartbeat reborn

Additions should stay conservative:

- use harmless commands
- prefer read-only modules first
- only add mutation RPC coverage when the expected host-side effect is stable on
  the CI/local environment

## Troubleshooting

### Test skipped

Most common cause:

```text
set MALICE_REAL_IMPLANT_RUN=1 to enable real implant integration tests
```

Set the env var and rerun.

### Patch step failed

Check:

- `malefic-mutant.exe` exists
- `malefic.exe` exists
- the generated `implant.yaml` is valid for the current Rust implant version

The suite shells out to:

```powershell
malefic-mutant.exe tool patch-config -f malefic.exe --from-implant <temp-yaml> -o <temp-exe>
```

### Implant exits before registering

The test fixture includes captured stdout/stderr from the implant process in the
failure message.

Typical causes:

- template binary and runtime config schema are from mismatched Rust revisions
- selected template was built without the required modules
- pipeline port was unavailable
- local security software killed the implant process immediately

### Session revives too early in lifecycle tests

That usually means the test path still allowed normal heartbeats to race with
the forced dead sweep. The current suite handles this by enabling `keepalive`
before the delayed `exec` task.

### Combined suite fails while single tests pass

That points to leaked in-process runtime state, not necessarily a protocol bug.

Check that the harness is resetting the transient transport/RPC maps listed
above. The failure mode is usually that the second test's task traffic is still
associated with the first test's pipeline stream.

## Relationship To Mock Tests

The mock suite is still the authoritative coverage for command breadth:

- command parameter parsing
- request body assembly
- mock scenario state mutation
- large RPC matrix

The real suite is intentionally smaller and should remain so. Its value is not
volume. Its value is that when it fails, the transport or lifecycle behavior is
actually broken.
