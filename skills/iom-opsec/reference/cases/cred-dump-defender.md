# Credential Harvesting in a Defender Environment

## Environment

| Item | Value |
|------|-------|
| OS | Windows 10 21H2 x64 |
| AV/EDR | Windows Defender (MsMpEng.exe) |
| Privileges | High (Admin, UAC bypassed) |
| Domain | CONTOSO.LOCAL |
| Patch level | 2024-01 |

## Objective

Obtain local and domain credentials (hashes + plaintext)

## Attempts

### Attempt 1: hashdump — Success

```
hashdump
```

Result: Success — local SAM hashes obtained
Analysis: hashdump does not touch LSASS; Defender did not detect it. OPSEC 9.0.

### Attempt 2: logonpasswords — Blocked

```
logonpasswords
```

Result: Blocked by Defender real-time protection
Analysis: logonpasswords reads LSASS memory directly; Defender has dedicated protection rules for LSASS access. OPSEC 5.9, as expected.

### Attempt 3: nanodump --fork --spoof-callstack — Success

```
nanodump --fork --spoof-callstack
```

Result: Success — LSASS dump obtained
Analysis: Fork mode creates a copy of the LSASS process rather than reading it directly; spoof-callstack conceals the call stack. Defender did not detect it.

### Attempt 4: credman — Success

```
credman
```

Result: Success — credentials saved in Credential Manager obtained
Analysis: Reads the current user's Credential Manager; no special privileges required. OPSEC 9.0.

## Final Solution

1. `hashdump` to obtain local hashes (OPSEC 9.0)
2. `credman` to obtain saved credentials (OPSEC 9.0)
3. `nanodump --fork --spoof-callstack` to obtain domain credentials (OPSEC 8.0)

No need to use logonpasswords or mimikatz.

## Lessons Learned

- logonpasswords is invariably blocked under Defender — skip it entirely
- The nanodump fork + spoof-callstack combination is the best option for obtaining LSASS credentials under Defender
- Prefer methods that do not touch LSASS (hashdump, credman, klist)

## Tags

`Defender` `Credential Harvesting` `OPSEC-8.0+` `Success`
