# WebShell Bridge

## Overview

WebShell Bridge enables IoM to operate through webshells (JSP/PHP/ASPX) by establishing a communication channel via suo5 HTTP tunnels. The architecture has three clean layers:

- **Product layer**: Server sees a `CustomPipeline(type="webshell")`. Operators interact via `webshell new/start/stop/delete` commands. No knowledge of rem/suo5/proxyclient required.
- **Implementation layer**: Bridge binary runs on the operator machine, managing transport (rem + proxyclient + suo5), session lifecycle, and task forwarding.
- **Transport layer**: The webshell only handles initial DLL loading and raw HTTP body send/receive. It never parses protocol bytes.

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

  ┌─ transport adapter ──────────────────────────────────────┐
  │  rem (internal, not exposed as product concept)          │
  │  proxyclient/suo5 (HTTP full-duplex tunnel)              │
  └──────────────────────────────────────────────────────────┘

  ┌─ spite/session adapter ──────────────────────────────────┐
  │  SpiteStream ↔ rem channel protocol translation          │
  │  Session registration, checkin, task routing              │
  └──────────────────────────────────────────────────────────┘


Target Web Server Process
─────────────────────────
  WebShell (JSP/PHP/ASPX)
    - Initial bridge DLL loading (reflective/memory)
    - HTTP body send/receive
    - Pass raw bytes to bridge, no parsing

  Bridge Runtime DLL (in web server process memory)
    ┌─ transport adapter ─────────────────────────────────┐
    │  rem server on 127.0.0.1:<port>                     │
    │  Bridge binary connects as rem client via suo5      │
    └─────────────────────────────────────────────────────┘

    ┌─ spite/session adapter ─────────────────────────────┐
    │  Receives Spite over rem channel                    │
    │  Routes to module runtime by spite.Name             │
    └─────────────────────────────────────────────────────┘

    ┌─ malefic module runtime ────────────────────────────┐
    │  exec / bof / execute_pe / upload / download / ...  │
    │  All malefic modules available                      │
    └─────────────────────────────────────────────────────┘
```

## Data Flow

```
Client exec("whoami")
  → Server (SpiteStream)
    → Bridge binary (session adapter)
      → [rem channel through suo5 HTTP tunnel]
        → Bridge Runtime DLL (module runtime)
          → exec("whoami") → "root"
        → Spite response over rem channel
      → [suo5 HTTP tunnel]
    → Bridge binary → SpiteStream.Send(response)
  → Server → Client displays "root"
```

## Usage

### 1. Run bridge binary

```bash
webshell-bridge \
  --auth listener.auth \
  --suo5 suo5://target.com/suo5.jsp \
  --listener my-listener \
  --pipeline webshell_my-listener \
  --dll-addr 127.0.0.1:13338
```

The `--dll-addr` flag tells the bridge binary which address to connect to through the suo5 tunnel (default: `127.0.0.1:13338`). This must match the DLL's compiled `DEFAULT_ADDR` in `malefic-bridge-dll/src/lib.rs` and the webshell's status probe port (`BRIDGE_DLL_PORT` in PHP, port constant in ASPX/JSP). Changing the port requires updating all three locations and recompiling the DLL.

At startup the bridge registers the listener, opens `JobStream`, and waits for pipeline start/stop/sync control. It does **not** auto-register or auto-start the `CustomPipeline`.

### 2. Register and start the pipeline from Client/TUI

```
webshell new --listener my-listener
```

This creates `CustomPipeline(type="webshell")` and sends the pipeline start control to the already running bridge.

### 3. Deploy suo5 webshell + bridge DLL on target

Deploy the suo5 webshell (JSP/PHP/ASPX) to the target web server. The webshell loads the bridge DLL into the web server process memory. The bridge DLL starts a rem server on `127.0.0.1:13338` (or the port matching `--dll-addr`).

If the DLL is not loaded when the pipeline starts, the bridge keeps retrying `connectDLL` with exponential backoff until the rem server becomes reachable or the retry budget is exhausted.

### 4. Interact

```
use <session-id>
exec whoami
upload /local/file /remote/path
download /remote/file
```

## Rem Channel Protocol

The bridge binary communicates with the bridge DLL using the rem wire protocol over a TCP connection tunneled through suo5.

### Wire Format

Each message: `[1 byte msg_type][4 bytes LE length][protobuf payload]`

Uses `cio.WriteMsg`/`cio.ReadMsg` from `github.com/chainreactors/rem/protocol/cio`.

### Session Lifecycle

```
1. Bridge dials DLL:   transport.Dial("tcp", dllAddr)  [through suo5]
2. Login handshake:    Login{Agent: id, Mod: "bridge"} → Ack{Status: 1}
3. DLL sends:          Packet{ID: 0, Data: Marshal(Register{SysInfo, Modules})}
4. Bridge registers session with server using real SysInfo/Modules
5. Task exchange:      Packet{ID: taskID, Data: Marshal(Spite)} ↔ bidirectional
```

### DLL Requirements

The bridge DLL (malefic create branch) must:
1. Start a rem-compatible TCP listener on the configured port
2. Accept Login, respond with Ack
3. Send a handshake Packet{ID: 0} containing serialized `implantpb.Register`
4. For each received Packet, unmarshal the Spite, execute the module, and reply with a Packet containing the response Spite

## Key Files

| Purpose | Path |
|---------|------|
| Bridge binary | `server/cmd/webshell-bridge/` |
| Rem channel | `server/cmd/webshell-bridge/channel.go` |
| Client commands | `client/command/pipeline/webshell.go` |
| CustomPipeline (server) | `server/listener/custom.go` |
| proxyclient/suo5 | `github.com/chainreactors/proxyclient/suo5` |
