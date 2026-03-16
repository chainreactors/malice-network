# Implant Bug Record

This document tracks defects confirmed on the real implant side during
`command -> rpc -> implant` E2E testing.

It intentionally excludes:

- server runtime bugs
- mock implant mismatches
- expected non-admin failures that already return correct diagnostics

Use this file as the handoff record for implant-side fixes.

## Latest Verification

### 2026-03-15 addon/task real closed-loop rerun

- Coverage:
  - `list_task`, `query_task`, `cancel_task`
  - `list_addon`, `load_addon`, `execute_addon`
- Result:
  - no new implant-side defect confirmed in this rerun
  - the issues exposed in this round were on the teamserver/client test chain,
    not in the real implant
  - focused real regression command:
    `go test ./client/command -tags realimplant -run "TestRealImplantCommand(TaskControlE2E|AddonModulesE2E)$" -count=1 -timeout 600s`

## Open Issues

### `switch` still re-registers on the original pipeline

- Status: open
- First confirmed: 2026-03-15
- Reconfirmed: 2026-03-16
- Detection path:
  - real E2E: [client/command/real_implant_command_e2e_test.go#L676](/D:/Programing/go/chainreactors/malice-network/client/command/real_implant_command_e2e_test.go#L676)
- Symptom:
  - after Go-side `Switch{targets, action, key}` sync, the switch task now
    completes, but the session still does not move to the selected pipeline
  - once the original pipeline is stopped, the implant re-registers on the
    original pipeline instead of the requested target pipeline
- Latest observed failure:
  - focused rerun on 2026-03-16 failed with
    `switch did not migrate session to secondary pipeline`
  - server log showed:
    `session <id> re-register`
  - server log showed:
    `<id> re-registered at <primary-pipeline>`
- Implant-side evidence:
  - new Rust implant flow stores `pending_switch` in
    [stub.rs](/D:/Programing/rust/implant/malefic-crates/stub/src/stub.rs)
    and applies it later in
    [beacon.rs](/D:/Programing/rust/implant/malefic/src/beacon.rs)
  - beacon mode exits the session loop as soon as
    `should_reconnect_for_switch()` becomes true in
    [session_loop.rs](/D:/Programing/rust/implant/malefic/src/session_loop.rs)
  - the teamserver/client side has already been synced to the new
    `Switch{targets, action, key}` schema, and the real task now finishes
  - despite that, the implant still reconnects back to the original pipeline
- Likely cause:
  - the implant applies the pending switch but the reconnect path still selects
    the original target set instead of the replaced target
- Expected fix:
  - make the post-switch reconnect use the replaced target set as the next
    active transport target
  - rerun `TestRealImplantCommandSwitchModuleE2E`

### `rev2self` is not exposed in the real module list

- Status: open
- First confirmed: 2026-03-15
- Detection path:
  - real E2E: [client/command/real_implant_command_e2e_test.go#L994](/D:/Programing/go/chainreactors/malice-network/client/command/real_implant_command_e2e_test.go#L994)
- Symptom:
  - the real session reports modules such as `runas`, `privs`, and `getsystem`,
    but does not report `rev2self`
  - the token suite fails during module discovery before the command can be
    executed
- Implant-side evidence:
  - feature is declared in [Cargo.toml](/D:/Programing/rust/implant/malefic-modules/Cargo.toml)
  - implementation exists in [token.rs](/D:/Programing/rust/implant/malefic-modules/src/sys/token.rs)
  - registration is missing from [lib.rs](/D:/Programing/rust/implant/malefic-modules/src/lib.rs)
- Likely cause:
  - `sys::token::Rev2Self` is implemented but not added to the module registry
- Expected fix:
  - register `rev2self` in the implant module map
  - rebuild the implant
  - rerun `TestRealImplantCommandTokenModulesE2E`

## Fixed And Reverified

### Scheduled-task path semantics drift

- Status: fixed in implant and reverified
- Detection path:
  - real `taskschd list|query` E2E in [client/command/real_implant_command_e2e_test.go#L1160](/D:/Programing/go/chainreactors/malice-network/client/command/real_implant_command_e2e_test.go#L1160)
- Previous symptom:
  - scheduled-task list/query behavior had path semantics that did not match the
    teamserver command/query expectation
- Current state:
  - the non-admin real suite now passes `taskschd list|query`
  - the privileged lifecycle suite also has a stable regression entrypoint for
    `taskschd create|run|delete`

## Confirmed Non-Bugs

These paths are covered by real implant E2E and currently behave as expected.
They should not be logged as implant defects unless behavior changes.

### `runas` invalid credentials

- Real behavior:
  - returns a task-level diagnostic for bad credentials
- Latest observed diagnostic:
  - `用户名或密码不正确。 (0x8007052E)`
- Coverage:
  - [client/command/real_implant_command_e2e_test.go#L924](/D:/Programing/go/chainreactors/malice-network/client/command/real_implant_command_e2e_test.go#L924)

### `getsystem` from a non-elevated implant

- Real behavior:
  - returns a task-level diagnostic instead of hanging or silently succeeding
- Latest observed diagnostic:
  - `拒绝访问。 (0x80070005)`
- Coverage:
  - [client/command/real_implant_command_e2e_test.go#L974](/D:/Programing/go/chainreactors/malice-network/client/command/real_implant_command_e2e_test.go#L974)

## Retest Command

Use this focused real-implant command when validating implant-side fixes:

```powershell
$env:MALICE_REAL_IMPLANT_RUN = "1"
go test ./client/command -tags realimplant -run TestRealImplantCommandTokenModulesE2E -count=1
```
