---
name: portscan
description: Scan network hosts and ports using built-in OS tools
---
Perform a network port scan using ONLY built-in OS tools (no nmap, masscan, or external tools).

## Target

- Host(s): $0
- Ports: $1

If ports not specified, scan common ports: 21,22,23,25,53,80,110,135,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,6379,8080,8443,8888,9200,27017

## Methods (auto-select by OS)

**Linux/macOS**:
- `/dev/tcp` bash built-in: `echo >/dev/tcp/HOST/PORT`
- `curl --connect-timeout 1 HOST:PORT`
- `nc -z -w1 HOST PORT` (if available)

**Windows**:
- PowerShell: `Test-NetConnection -ComputerName HOST -Port PORT -InformationLevel Quiet`
- .NET socket: `(New-Object Net.Sockets.TcpClient).ConnectAsync(HOST, PORT).Wait(1000)`

## Rules

- Timeout per port: 1 second max
- Run scans concurrently where possible (background jobs, xargs -P)
- For port ranges, scan in batches to avoid overwhelming the target
- Output format: `HOST:PORT  OPEN/CLOSED  [service guess]`
- Provide a final summary of all open ports

$ARGUMENTS
