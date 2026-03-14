# Module Management Regression Record

## Overview

This document records the regression work around module management on the client side, especially the linked paths between:

- `build modules`
- `load_module --modules/--3rd`
- `load_module --artifact/--path`
- `load_addon`
- `execute_addon`

The goal was not to add happy-path-only tests. The goal was to make the command layer reliably expose:

- malformed input that should never reach transport
- build/load linkage bugs
- wrong profile assembly for module compilation
- swallowed failures on build-triggered load paths
- false-positive task or session events
- harness gaps that could hide real defects behind nil interface promotion

## Areas Checked

The inspection focused on:

- `client/command/addon/load.go`
- `client/command/addon/execute.go`
- `client/command/modules/load.go`
- `client/command/build/build.go`
- `client/command/build/build-module.go`
- `client/command/testsupport/recorder.go`

## Regressions Found

### Recorder Harness Could Panic On Build-Oriented RPC Calls

`RecorderRPC` embedded generated gRPC interfaces, but it did not implement several module/build methods explicitly.

Practical effect:

- tests could compile
- the harness could still hit promoted nil interface methods at runtime
- command regressions could be masked by harness panics instead of producing actionable failures

### `CheckSource(...)` Could Dereference A Nil Build Config

The shared build helper accepted a `nil` config and then accessed `buildConfig.Source`.

Practical effect:

- module auto-build paths could panic before transport
- tests for error handling around source detection were not reliable

### `build modules` Had A Missing-Target Crash Path

`ModulesCmd(...)` used the parsed build config before checking `parseBasicConfig(...)` errors.

Practical effect:

- direct command execution could nil-deref on invalid input
- the failure mode depended on whether Cobra intercepted the missing flag first

### Module Auto-Build Ignored Requested Module Selection

`load_module --modules ...` and `load_module --3rd ...` built through `SyncBuild(...)`, but the selected module list was not forwarded into `MaleficConfig`.

Practical effect:

- the operator could request one module set
- the build request could still use defaults instead of the requested selection

### Third-Party Module Builds Leaked Default Built-In Modules

The generated module profile started from defaults and enabled `3rd_modules`, but it did not clear default built-in modules first.

Practical effect:

- `--3rd rem` could still compile with default built-in modules such as `full`
- the produced artifact no longer matched the requested operator intent

### Module Auto-Build Did Not Enforce Library Output

The build-triggered load path did not validate or normalize output type for module builds.

Practical effect:

- a module build path could request or inherit an output type that was not valid for module loading
- the build/load contract was looser than the dedicated `build modules` command

### Build Failures Were Swallowed In The Auto-Build Load Path

The module build-and-load path used a goroutine around synchronous work.

Practical effect:

- `SyncBuild(...)` failures could be logged but not returned to the caller
- tests and callers could observe apparent success while the load never happened

### `execute_addon` Could Emit A False Session Event On RPC Failure

The command emitted `session.Console(...)` before checking the RPC error.

Practical effect:

- task history and session event streams could show a task that never actually existed
- operators could get false-positive success traces

### `load_module` Accepted Multiple Input Sources And Silently Chose One

The command accepted combinations such as `--artifact` with `--modules` or `--path`, then used branch priority instead of rejecting the input.

Practical effect:

- malformed operator input could still reach transport
- debugging became harder because one source silently shadowed another

## Fixes Applied

The following production changes were made:

- `client/command/testsupport/recorder.go`
  - added explicit recorder implementations for build, module, addon, artifact, and cleanup RPCs
  - added responder hooks for `Artifact` and `BuildConfig` flows
  - kept metadata and task event recording on these paths
- `client/command/build/build.go`
  - `CheckSource(...)` now handles `nil` build configs safely
- `client/command/build/build-module.go`
  - `ModulesCmd(...)` now returns `parseBasicConfig(...)` errors before dereferencing the config
  - module selector validation now happens before transport
  - `BuildModuleMaleficConfig(...)` now normalizes module lists and clears default module leakage before applying `--3rd`
- `client/command/modules/load.go`
  - module auto-build now uses a real build config for source detection
  - module selection is forwarded into `MaleficConfig`
  - build output is validated and forced to `lib`
  - build-triggered load is synchronous so `SyncBuild(...)` failures propagate
  - input sources are now validated as mutually exclusive
- `client/command/addon/execute.go`
  - session events are emitted only after `ExecuteAddon(...)` succeeds

## Regression Coverage Added

The following command and helper tests were added or expanded:

- `client/command/build/modules_command_test.go`
  - direct `ModulesCmd(...)` missing-target error path
  - command-layer rejection of mutually exclusive `--modules` and `--3rd`
  - forwarding of built-in module selection into `MaleficConfig`
  - forwarding of third-party module selection into `MaleficConfig`
  - regression guard that `--3rd` does not leak default built-in modules
- `client/command/modules/modules_test.go`
  - `load_module --path`
  - `load_module --artifact`
  - `load_module --modules`
  - `load_module --3rd`
  - `SyncBuild(...)` failure propagation
  - rejection of conflicting module selectors
  - rejection of multiple input sources
  - rejection of missing input sources
- `client/command/addon/addon_test.go`
  - module inference from addon file extension
  - explicit module override on addon load
  - execution requires the addon to be present in session state
  - forwarding of sacrifice and execution arguments
  - default command-layer process and arch behavior
  - no session event on RPC failure

## Why This Layer Matters

These paths are easy to break because they mix:

- CLI parsing
- local file handling
- build-time profile generation
- RPC selection
- task event side effects

A thin helper-only test would miss the integration points between those steps. The current harness keeps those steps connected while still running inside default unit-test speed.

## Verification

This record was validated with:

```bash
go test ./client/command/build ./client/command/modules ./client/command/addon -count=1
go test ./client/command/... -count=1
```
