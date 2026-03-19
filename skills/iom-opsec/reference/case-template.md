# OPSEC Case Template

Each case records the full context of a real operation, serving as a reference for similar future scenarios.

## Case File Naming

Format: `<operation-type>-<security-product>.md`

Examples:
- `cred-dump-defender.md` — credential harvesting in a Defender environment
- `uac-bypass-crowdstrike.md` — UAC bypass in a CrowdStrike environment
- `lateral-wmi-no-av.md` — WMI lateral movement with no AV present

## Case Structure

```markdown
# [Short Title]

## Environment

| Item | Value |
|------|-------|
| OS | Windows 10 21H2 x64 |
| AV/EDR | [Security product name and version] |
| Privileges | [Medium/High/SYSTEM] |
| Domain | [WORKGROUP/domain name] |
| Patch level | [Last KB date] |

## Objective

[What operation needs to be accomplished]

## Attempts

### Attempt 1: [Technique Name] — [Success/Failure/Blocked]

Command:
```
[Actual command executed]
```

Result: [Success/Failure/Detected]
Analysis: [Why it succeeded or failed]

### Attempt 2: [Technique Name] — [Success/Failure/Blocked]

...

## Final Solution

[The approach that ultimately succeeded, or the conclusion if all approaches failed]

## Lessons Learned

- [Key finding 1]
- [Key finding 2]
- [Recommendations for subsequent operations]

## Tags

`[Security Product]` `[Operation Type]` `[OPSEC Score]` `[Success/Failure]`
```

## Case Accumulation

After each operation conducted through the iom-opsec skill, if the result has reference value (especially failed cases), it should be recorded as a new case and stored in the `reference/cases/` directory.

Characteristics of a valuable case:
- A technique was blocked by a specific AV (can be skipped directly in the future)
- A detection blind spot was discovered in a specific AV
- An unconventional but effective operation path was found
- A recovery strategy after a failed operation
