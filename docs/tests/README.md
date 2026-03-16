# Test Records

## Overview

This directory stores concrete testing records for major coverage expansions.

These documents are not API references. They capture:

- what was tested
- why the test shape was chosen
- which regressions were discovered
- which fixes were made
- what remains intentionally out of scope

## Records

- `command-conformance-record.md`: implant command parsing and protobuf assembly coverage
- `control-plane-regression-record.md`: client/server control-plane regressions and integration coverage
- `module-management-regression-record.md`: addon, module load, and build-triggered module compilation regressions plus shared command harness extensions
- `mock-implant-e2e.md`: server-facing mock implant transport, reusable scenario library, real task/request streaming `WaitTask*` E2E, and dead/reborn lifecycle edge coverage
- `implant-bugs.md`: implant-side defects confirmed by real command -> RPC -> implant E2E, separated from server/mock issues
- `task-runtime-regression-record.md`: task wait semantics, streaming task finish state, recovery/runtime wiring, dead-sweep/task-cancel regressions, and task command regressions
- `implant-e2e-testing.md`: real implant module E2E testing guide — compilation, proto round-trip, bridge transport simulation, response normalization, and reusable patterns for any `malefic-3rd` module

## How To Use This Directory

Use these records when:

- extending an existing test harness
- deciding whether a new command should be covered at the command layer or integration layer
- understanding why a regression guard exists
- checking which bugs were already found and fixed

Use `docs/development/testing.md` for the current test entrypoints and CI-facing commands.
