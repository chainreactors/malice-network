# Configuration Details

## Client Configuration Directory

```
~/.config/malice/
├── malice.yaml          # Main client configuration
├── configs/             # .auth files (auto-migrated here after login)
├── log/                 # Log files
├── resources/           # Resource files (scripts, templates, etc.)
└── temp/                # Temporary files
```

## .auth File

The `.auth` file is the credential for connecting to the server. It contains mTLS certificates and the server address in YAML format.

- Naming: `<operator>_<host>.auth` (client) or `<listener>.auth` (listener)
- Automatically migrated to `~/.config/malice/configs/` after login
- If a file with the same name already exists, a `.backup` copy is created automatically

```bash
iom login /path/to/server.auth    # Import and log in
```

## Client Configuration (malice.yaml)

```yaml
# MCP server configuration
mcp:
  enable: true
  address: "127.0.0.1:5005"

# LocalRPC configuration
localrpc:
  enable: false
  address: "127.0.0.1:15004"

# Logging configuration
log:
  max_size: 10  # MB
```

These can also be overridden via startup parameters:
- `--mcp 127.0.0.1:5005` overrides the MCP address
- `--rpc 127.0.0.1:15004` overrides the RPC address

## Server Configuration (server/config.yaml)

```yaml
server:
  grpc:
    host: "0.0.0.0"
    port: 5004

listeners:
  - name: "default-tcp"
    auth: "listener.auth"
    tcp:
      host: "0.0.0.0"
      port: 5001
      tls: true
  - name: "default-http"
    auth: "listener.auth"
    http:
      host: "0.0.0.0"
      port: 8080
      tls: true

encryption:
  type: "aes"          # aes or xor
  key: "maliceofinternal"

audit:
  level: 1             # Audit log level

build:
  target: "x86_64-pc-windows-gnu"
```

### Key Field Descriptions

| Field | Description |
|-------|-------------|
| `server.grpc.port` | gRPC listening port; the client connects to this port |
| `listeners[].auth` | Path to the listener's auth file |
| `listeners[].tcp/http` | Pipeline type and network configuration |
| `encryption.type` | Implant communication encryption method |
| `encryption.key` | Encryption key (change the default value for deployment) |
| `audit.level` | Audit log verbosity (higher numbers mean more detail) |

## AI Integration Configuration

Configure the AI agent via the `config ai` command:

```
config ai --enable                           # Enable AI
config ai --api-key <key>                    # Set API key
config ai --model <model>                    # Set model
config ai --endpoint <url>                   # Set API endpoint
```

Supported providers: OpenAI, OpenRouter, Deepseek, Groq, Moonshot.
