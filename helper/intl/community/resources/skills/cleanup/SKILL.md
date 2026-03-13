---
name: cleanup
description: Remove artifacts and traces from the target system
---
Clean up operational artifacts and traces from the target system.

## Cleanup targets

1. **Shell history**: clear current session history, remove entries from history files related to our operations
2. **Log entries**: recent entries in auth.log, syslog, wtmp/btmp, Windows Event Logs related to our activity
3. **Temp files**: remove any temporary files created during operations in /tmp, %TEMP%, working directories
4. **Persistence artifacts**: remove any persistence mechanisms we installed (specify which: $ARGUMENTS)
5. **Timestamps**: restore file modification times if we modified any configs (use touch -r with reference file)
6. **Network traces**: clear ARP cache, DNS cache, recent connections from logs

## Rules

- List all artifacts you intend to remove BEFORE removing them — wait for confirmation
- Show the exact commands that will be executed
- For log files, only remove specific entries — do NOT truncate or delete entire log files
- Record what was cleaned for the operator's reference
- If $ARGUMENTS is empty, only perform discovery (list what WOULD be cleaned) without actually cleaning

$ARGUMENTS
