---
name: recon
description: Enumerate target system info, users, network, and processes
---
Perform reconnaissance on the target system. Collect ALL of the following and output in a structured summary:

1. **OS & Host**: OS version, architecture, hostname, domain, kernel version
2. **Current User**: username, UID/SID, group memberships, privileges/sudo access
3. **Users & Groups**: all local users, recently logged-in users, admin/root group members
4. **Network**: interfaces, IP addresses, routing table, DNS servers, active connections (ESTABLISHED), listening ports
5. **Processes**: running processes with PID, user, command line — highlight security tools (AV/EDR/HIPS)
6. **Environment**: PATH, interesting environment variables (proxy, credentials, tokens)

Rules:
- Auto-detect OS (Linux/macOS/Windows) and use appropriate commands
- Run each command individually, do NOT chain with `&&` — if one fails, continue with the rest
- Do NOT install any packages or modify the system
- Output a final structured summary with all findings

$ARGUMENTS
