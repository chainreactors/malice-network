---
name: iom
description: >
  Complete user guide for the IoM (Implant-over-Middleware) C2 framework. Covers
  architecture concepts, basic operations, startup parameters, authentication
  configuration, troubleshooting, documentation resources, and community feedback.
  Trigger conditions: use this skill when the user asks about how IoM works,
  command usage, architecture concepts, configuration methods, how to troubleshoot
  issues, where to find documentation, or how to file an issue.
  Should also trigger for questions like "how do I connect", "what is a session",
  "I got an error", or "is there documentation".
---

# IoM User Guide

IoM (Implant-over-Middleware) is a modular C2 framework built on a three-layer architecture.

## Architecture Overview

```
┌──────────────────────────────┐
│     Client (CLI/TUI)         │  User interaction layer
│  Cobra commands + Bubble Tea │
└──────────────┬───────────────┘
               │ gRPC + mTLS
               ▼
┌──────────────────────────────┐
│     Server                   │  Control hub
│  Session/Task/Event mgmt     │
│  Listener/Pipeline dispatch  │
└──────────────┬───────────────┘
               │ ListenerRPC
               ▼
┌──────────────────────────────┐
│  Listener → Pipeline         │  Communication middleware
│  TCP / HTTP / Bind / Custom  │
└──────────────┬───────────────┘
               │ Network
               ▼
┌──────────────────────────────┐
│     Implant (malefic)        │  Target-side agent
│  Modular capability manifest │
└──────────────────────────────┘
```

## Core Concepts

See [reference/concepts.md](reference/concepts.md) for detailed explanations.

| Concept | Summary |
|---------|---------|
| **Session** | An active implant connection containing target system info, a task queue, and module capabilities |
| **Listener** | A network service that accepts implant connections and manages multiple pipelines |
| **Pipeline** | A transport channel under a listener, responsible for encryption, protocol parsing, and frame handling |
| **Task** | An execution unit sent to the implant, with status tracking and timeout control |
| **Module** | A capability category declared by the implant, determining which commands are available |
| **Event** | A real-time notification (new session, task completion, session offline, etc.) |

## Quick Start

### Starting the Client

```bash
iom login server.auth            # Log in using a .auth file
```

Client startup parameters:

| Parameter | Description |
|-----------|-------------|
| `--mcp <addr>` | Enable MCP server (e.g., `127.0.0.1:5005`) |
| `--rpc <addr>` | Enable LocalRPC gRPC server |
| `--daemon` | Daemon mode — run in background without interactive terminal |
| `--tui` | TUI multiplexing mode (split-pane terminal) |
| `--quiet` | Quiet mode — suppress startup event output |

### Authentication and Configuration

The `.auth` file contains mTLS certificates and the server address. It serves as the credential for connecting to the server.

Configuration directory structure (see [reference/config.md](reference/config.md) for details):

```
~/.config/malice/
├── malice.yaml          # Client configuration (MCP, RPC, logging, etc.)
├── configs/             # .auth file storage (auto-migrated after login)
├── log/                 # Log files
├── resources/           # Resource files
└── temp/                # Temporary files
```

## Basic Operations

### Session Management

```
session                          # List active sessions
session --all                    # List all sessions (including offline)
use <session_id>                 # Enter a session context (supports ID prefix matching)
background                       # Exit the current session and return to the main menu
```

### Common Commands Within a Session

```
sysinfo                          # System information
whoami                           # Current user
privs                            # Current privileges
ps                               # Process list
ls / cd / pwd / cat              # File system operations
upload <local> <remote>          # Upload a file
download <remote>                # Download a file
shell <command>                  # Execute a shell command
```

### Infrastructure Management

```
listener                         # List listeners
pipeline list                    # List pipelines
pipeline tcp --name <n> --host <h> --port <p>   # Create a TCP pipeline
```

### Task Management

```
tasks                            # List tasks
tasks --task-id <id>             # View task details
tasks cancel --task-id <id>      # Cancel a task
```

### Module Management

```
modules list                     # View available modules
modules refresh                  # Refresh modules from the implant
modules load <addon>             # Load an extension module
```

## Command Categories

IoM commands are organized by function. See [reference/commands.md](reference/commands.md) for the full reference.

| Category | Description | Examples |
|----------|-------------|----------|
| Basic | Info, heartbeat, binding | `sysinfo`, `sleep`, `suicide` |
| Execution | Run programs and code | `shell`, `execute_exe`, `execute_assembly`, `bof` |
| File System | Browse and manipulate files | `ls`, `cd`, `cat`, `rm`, `mkdir`, `cp`, `mv` |
| File Transfer | Upload and download | `upload`, `download` |
| System | System operations | `ps`, `kill`, `env`, `netstat`, `whoami` |
| Privileges | Privilege management | `privs`, `getsystem`, `runas`, `rev2self` |
| Network | Proxying and forwarding | `proxy`, `forward`, `reverse` |
| Enumeration | Environment discovery | `enum av`, `enum software`, `enum dc` |

## MAL Plugins

IoM extends its command capabilities through MAL (a Lua plugin system). The built-in `community` plugin package provides 90+ commands covering enumeration, privilege escalation, credential harvesting, lateral movement, persistence, and more.

```
mal list                             # List installed plugins
mal install <name>                   # Install from the community repository
mal remove <name>                    # Uninstall
search_commands("keyword")           # Search for plugin-provided commands
```

See [reference/mal-usage.md](reference/mal-usage.md) for MAL usage details.
See [reference/builtin-plugins.md](reference/builtin-plugins.md) for the full built-in command list.

## MCP Tool Interface

The IoM client provides an MCP server that exposes the following tools:

| Tool | Purpose |
|------|---------|
| `search_commands` | Fuzzy-search commands by name/description, returns lightweight summaries |
| `execute_command` | Execute any client command, automatically waits for the task result |
| `execute_lua` | Execute a Lua script (with access to all IoM internal APIs) |
| `get_history` | Retrieve output from historical tasks |

**Recommended workflow (progressive discovery)**:
1. `search_commands("keyword")` — search for related commands
2. `execute_command("<cmd> --help")` — view specific usage
3. `execute_command("<cmd> <args>")` — execute the command

### Lua Script Execution

Use `execute_lua` to run Lua scripts directly for automation and batch operations:

```lua
-- Simple example
execute_lua('return string.format("2+2=%d", 2+2)')

-- Use IoM APIs within a session context
execute_lua('local s = active(); return s.Os.Hostname', session_id="abc123")
```

See [reference/lua-scripting.md](reference/lua-scripting.md) for details.

## Troubleshooting

See [reference/troubleshooting.md](reference/troubleshooting.md) for common issues, covering:
- Connection failures (auth files, certificates, network)
- Session anomalies (offline, heartbeat timeout, unavailable modules)
- Task issues (stuck, timed out, no output)
- Debugging methods (log directory, audit log level)

## Documentation and Community

| Resource | Link |
|----------|------|
| Official Wiki | https://chainreactors.github.io/wiki/IoM/ |
| Quick Start | https://chainreactors.github.io/wiki/IoM/quickstart/ |
| Implant Manual | https://chainreactors.github.io/wiki/IoM/manual/implant/ |
| MAL Scripting | https://chainreactors.github.io/wiki/IoM/manual/mal/quickstart/ |
| Roadmap | https://chainreactors.github.io/wiki/IoM/roadmap/ |
| GitHub (Server) | https://github.com/chainreactors/malice-network |
| GitHub (Implant) | https://github.com/chainreactors/malefic |

### Reporting Issues

Submit feedback via GitHub Issues:

```bash
# Create an issue using gh cli
gh issue create --repo chainreactors/malice-network \
  --title "Brief description of the issue" \
  --body "## Environment
OS:
IoM version:
## Steps to Reproduce
1.
2.
## Expected Behavior

## Actual Behavior

## Logs
(Paste relevant logs from ~/.config/malice/log/)"
```

When filing an issue, include: OS, IoM version, reproduction steps, expected vs. actual behavior, and relevant logs.

## Related Skills

IoM capabilities are split across multiple skills by responsibility. Consult the appropriate skill for your scenario:

| Skill | Responsibility | Use When |
|-------|---------------|----------|
| **iom** (this skill) | Core concepts, usage guide, configuration, troubleshooting | "How do I connect", "How to use commands", "I got an error" |
| **iom-pentest** | Penetration testing execution | "Privilege escalation", "Lateral movement", "Credentials", "Persistence" |
| **iom-opsec** | Operational security methodology and case studies | "Is this safe", "Will it be detected", "Risk assessment" |
| **mal-develop** | MAL plugin development | "Write a plugin", "Extend commands", "Lua API" |

## Reference Documents

| Topic | Reference File |
|-------|---------------|
| Core concept details | [reference/concepts.md](reference/concepts.md) |
| Command quick reference | [reference/commands.md](reference/commands.md) |
| MAL plugin usage | [reference/mal-usage.md](reference/mal-usage.md) |
| Built-in plugin command list | [reference/builtin-plugins.md](reference/builtin-plugins.md) |
| Lua script execution | [reference/lua-scripting.md](reference/lua-scripting.md) |
| Troubleshooting | [reference/troubleshooting.md](reference/troubleshooting.md) |
| Configuration details | [reference/config.md](reference/config.md) |
