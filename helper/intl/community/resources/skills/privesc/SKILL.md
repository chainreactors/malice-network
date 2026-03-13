---
name: privesc
description: Enumerate privilege escalation vectors on the target
---
Enumerate potential privilege escalation vectors on the target system. This is a DISCOVERY task — do NOT exploit anything, only report findings.

## Linux checks

1. **Sudo**: `sudo -l` (list allowed commands), sudoers misconfigurations
2. **SUID/SGID**: find SUID/SGID binaries, cross-reference with GTFOBins
3. **Capabilities**: files with dangerous capabilities (cap_setuid, cap_dac_override, etc.)
4. **Writable paths**: world-writable directories in PATH, writable /etc/passwd or /etc/shadow
5. **Cron jobs**: cron entries running as root with writable scripts/paths
6. **Kernel**: kernel version, check for known exploits (Dirty Pipe, etc.)
7. **Services**: services running as root with writable configs
8. **Docker/LXD**: current user in docker/lxd group

## Windows checks

1. **Token privileges**: whoami /priv — look for SeImpersonate, SeDebug, SeBackup, SeRestore
2. **Unquoted service paths**: services with spaces in path and no quotes
3. **Writable service binaries**: service executables writable by current user
4. **AlwaysInstallElevated**: registry check for MSI privilege escalation
5. **Stored credentials**: cmdkey /list, credential manager
6. **Scheduled tasks**: tasks running as SYSTEM with writable actions
7. **UAC level**: registry check for ConsentPromptBehaviorAdmin

## Output

Rate each finding: 🔴 HIGH / 🟡 MEDIUM / 🟢 LOW exploitability. Provide a final ranked summary.

$ARGUMENTS
