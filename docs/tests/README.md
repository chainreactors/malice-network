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

## How To Use This Directory

Use these records when:

- extending an existing test harness
- deciding whether a new command should be covered at the command layer or integration layer
- understanding why a regression guard exists
- checking which bugs were already found and fixed

Use `docs/development/testing.md` for the current test entrypoints and CI-facing commands.
