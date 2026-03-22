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

  ┌─ Bootstrap (HTTP POST) ──────────────────────────────────┐
  │  Body envelope: [1B stage][4B sid LE][1B tok_len][tok][…] │
  │  Optional XOR obfuscation (key = sha256(token)[:16])     │
  │  OPSEC: no User-Agent, Content-Type mimics form POST     │
  └──────────────────────────────────────────────────────────┘

  ┌─ Data channel (suo5 full-duplex) ────────────────────────┐
  │  proxyclient/suo5 → net.Conn                             │
  │  TLV frames: [0xd1][4B sid][4B len][spite bytes][0xd2]   │
  │  Bidirectional streaming over persistent connection       │
  └──────────────────────────────────────────────────────────┘

  ┌─ Forward integration ────────────────────────────────────┐
  │  SpiteStream ↔ TLV frame translation                     │
  │  Session registration, checkin, task routing              │
  └──────────────────────────────────────────────────────────┘


Target Web Server Process
─────────────────────────
  WebShell (JSP/PHP/ASPX)
    - Bridge DLL loading (ReflectiveLoader)
    - Export resolution (bridge_init, bridge_process)
    - TLV frames → call bridge_process() → return TLV response
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
      → writeFrame: TLV pack → suo5 conn
        → WebShell (calls bridge_process via function pointer)
          → DLL module runtime → exec("whoami") → "root"
        → TLV response via suo5 conn
      → readLoop: TLV unpack → Forwarders.Send(SpiteResponse)
    → Server → Client displays "root"
```

## Usage

### 1. Deploy webshell

Deploy the suo5 webshell (JSP/PHP/ASPX) to the target web server.

### 2. Register and start the pipeline

```
webshell new --listener my-listener --suo5 suo5://target/bridge.jsp --token SECRET --dll /path/to/bridge.dll
```

The optional `--token` must match the `STAGE_TOKEN` constant in the webshell if set. Use the suo5 URL scheme (`suo5://` or `suo5s://` for HTTPS).

The `--dll` flag enables auto-loading: when the pipeline starts, the bridge automatically delivers the DLL to the webshell if it is not already loaded.

### 3. Interact

```
use <session-id>
exec whoami
upload /local/file /remote/path
download /remote/file
```

## Protocol

### Bootstrap Envelope

Bootstrap requests use HTTP POST with `Content-Type: application/x-www-form-urlencoded` and no `User-Agent` header. The envelope is sent in plaintext; authentication is via the token field in the envelope header.

**Envelope format (before optional XOR):** `[1B stage][4B sessionID LE][1B token_len][token bytes][payload...]`

| Stage byte | Name | Payload | Response |
|-----------|------|---------|----------|
| `0x01` | load | Raw DLL bytes | `OK:memory` or error string |
| `0x02` | status | (empty) | JSON `{"ready":true,...}` or legacy `LOADED`/`NOT_LOADED` |
| `0x03` | init | (empty) | `[4B sessionID LE][Register protobuf]` |
| `0x06` | deps | `[1B dep_name_len][dep_name][file bytes]` | `OK:<path>` or error string |

Token validation uses HMAC-SHA256 for secrets longer than 32 characters (rotates every 30s with +/-30s tolerance). Short secrets use static comparison.

### Data Channel TLV Frame

After bootstrap, a persistent suo5 connection carries bidirectional TLV frames:

```
[1B 0xd1][4B sessionID LE][4B payload_len LE][payload bytes][1B 0xd2]
```

- `0xd1` / `0xd2` are start/end delimiters matching malefic wire format
- Payload is serialized `Spites` protobuf
- Maximum frame size: 10 MiB
- Future: payload will be encrypted (outer streaming encryption layer)

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
| User-Agent | Empty (Go default stripped) |
| Content-Type | `application/x-www-form-urlencoded` (common POST type) |
| Bootstrap body | Plaintext envelope (token included for auth) |
| Data channel | TLV-framed, ready for encryption layer |
| Ports opened | None on target |
| Disk artifacts | None (DLL is memory-only) |

## Key Files

| Purpose | Path |
|---------|------|
| WebShell pipeline | `server/listener/webshell.go` |
| Pipeline tests | `server/listener/webshell_test.go` |
| Client commands | `client/command/pipeline/webshell.go` |
| Webshell (ASPX) | `suo5-webshell/bridge.aspx` |
| Webshell (PHP) | `suo5-webshell/bridge.php` |
| Webshell (JSP) | `suo5-webshell/bridge.jsp` |
