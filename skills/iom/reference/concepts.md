# IoM Core Concepts

## Session

A Session represents an active connection between an implant and the C2 server.

**Key attributes:**
- `SessionId` — unique identifier (MD5 hash), supports prefix matching
- System info — OS, architecture, hostname, username, process info
- Privilege flag — `*` indicates admin/elevated privileges
- Module list — capabilities declared by the implant
- Task queue — pending and completed tasks
- Heartbeat — periodic check-in; marked offline on timeout

**Lifecycle:** Register → Initialize → Active → Offline/Cleanup

**Commands:**
```
session                  # List active sessions
session --all            # Include offline sessions
use <id_prefix>          # Enter a session context (prefix matching)
background               # Exit session and return to main menu
note --session <id> "note"  # Add a note
```

Session data is persisted to the database and automatically restored after a server restart.

## Listener

A Listener is a network service that accepts implant connections. Each listener can manage multiple pipelines.

**Types:**
- **TCP** — direct TCP connection
- **HTTP/HTTPS** — HTTP-based transport
- **Bind** — reverse bind mode
- **REM** — Remote Execution Manager

**Commands:**
```
listener                 # List all listeners
```

Listeners communicate with the server via the `ListenerRPC` gRPC service.

## Pipeline

A Pipeline is a concrete transport channel under a listener, responsible for:
- Encryption (TLS, custom encryption)
- Protocol parsing and frame handling
- Carrying the actual implant connections

**Types:**
- **TCP Pipeline** — raw TCP, supports TLS/mTLS
- **HTTP Pipeline** — HTTP(S) transport
- **Bind Pipeline** — reverse connection
- **Custom Pipeline** — external process integration (LLM agent, MCP server, etc.)
- **Website** — static file serving / payload delivery page
- **REM Pipeline** — remote execution

**Commands:**
```
pipeline list                                    # List all pipelines
pipeline tcp --name my_tcp --host 0.0.0.0 --port 5555   # Create a TCP pipeline
pipeline start --name my_tcp                     # Start a pipeline
pipeline stop --name my_tcp                      # Stop a pipeline
```

## Task

A Task is an execution unit sent to the implant. Every command execution creates a task.

**Key attributes:**
- `TaskId` — unique identifier
- Type — the corresponding command name
- Status — Created → Running → Finished / Cancelled
- Progress — cur/total
- Timeout — deadline control
- Context — working directory, environment variables

**Commands:**
```
tasks                            # List tasks
tasks --task-id <id>             # View details
tasks cancel --task-id <id>      # Cancel a running task
```

When executing commands via the MCP `execute_command` tool, the system automatically waits for the task to complete and returns the result.

## Module

A Module is a capability category declared by the implant. Different implant builds can include different module combinations.

**Common modules:**
- `exec` — command execution
- `ls`, `pwd`, `cd`, `cat` — file system
- `upload`, `download` — file transfer
- `execute_exe`, `execute_dll`, `execute_bof` — advanced execution
- `bridge_agent` — built-in agent loop

Modules determine which client commands are available for the current session. If the implant does not support a given module, the corresponding commands will not appear.

**Commands:**
```
modules list                     # View modules available in the current session
modules refresh                  # Re-fetch module declarations from the implant
modules load <addon>             # Load an extension module
```

## Event

An Event is a real-time notification pushed by the server to all connected clients.

**Event types:**
- `EventSession` — session online, offline, or updated
- `EventTask` — task status change (started, completed, cancelled)
- `EventPipeline` — pipeline status change

Clients receive events via the `Events()` RPC stream, which is used to update the UI in real time.

## Communication Flow

```
Client ──(MaliceRPC/gRPC+mTLS)──→ Server
                                     │
                                     │ ListenerRPC
                                     ▼
                                  Listener
                                     │
                                     │ Pipeline (TCP/HTTP/...)
                                     ▼
                                  Implant
```

1. **Client → Server**: The client sends command requests via MaliceRPC
2. **Server → Implant**: The server wraps commands as SpiteRequests and sends them through the pipeline
3. **Implant → Server**: The implant executes the command and returns a SpiteResponse
4. **Server → Client**: The server wraps the result as task output and pushes it to the client
