# Troubleshooting

## Connection Issues

### Invalid Auth File
**Symptoms**: Login fails with "no auth config found" or a certificate error

**Diagnosis**:
1. Confirm the .auth file exists and is correctly formatted (YAML containing mTLS certificates and server address)
2. Verify the server address is reachable: `ping <server_ip>` or `telnet <server_ip> <port>`
3. Confirm the .auth file matches the server (certificates signed by the same CA)
4. Imported auth files are stored in `~/.config/malice/configs/` — check whether the file exists there

**Common causes**:
- Copied an auth file from another operator, but the server has regenerated its certificates
- The server address in the auth file is incorrect (IP or port changed)
- A firewall is blocking the gRPC port (default 5004)

### Connection Timeout
**Symptoms**: The client hangs for a long time after startup

**Diagnosis**:
1. Confirm the server is running
2. Check whether the gRPC port is open: default 5004
3. Check for proxies interfering with the gRPC connection
4. Review client logs: `~/.config/malice/log/`

## Session Issues

### Session Offline
**Symptoms**: The session list shows an offline status

**Possible causes**:
- The implant process was killed
- Network connection was interrupted
- The implant's sleep interval is too long (waiting for the next check-in)
- The pipeline was stopped

**Diagnosis**:
1. Run `session --all` to check the last heartbeat time
2. Verify the corresponding pipeline is running: `pipeline list`
3. Confirm target network connectivity
4. If the sleep interval is long, wait patiently for the next check-in

### Command Unavailable (Missing Module)
**Symptoms**: After entering a command, it reports as nonexistent, or certain commands are missing from the session

**Cause**: The implant was built without the corresponding module. Commands are dynamically shown based on module availability.

**Resolution**:
1. Run `modules list` to view modules supported by the current session
2. Run `modules load <addon>` to attempt dynamic loading of the missing module
3. If the module cannot be loaded dynamically, rebuild the implant with that module included

## Task Issues

### Task Stuck / No Output
**Symptoms**: No result is returned for a long time after executing a command

**Diagnosis**:
1. Run `tasks` to check task status (Running / Finished / Cancelled)
2. If the status is Running:
   - The implant may be performing a time-consuming operation
   - Network latency may be preventing the result from being transmitted
   - The implant's sleep interval may be causing a delay
3. Run `tasks cancel --task-id <id>` to cancel the stuck task
4. Re-execute the command

### Task Timeout
**Symptoms**: The task is automatically marked as timed out

**Resolution**: For operations known to be time-consuming, adjust the task deadline. Large file uploads/downloads, complex scans, and similar operations may require more time.

## Build and Development Issues

### go mod tidy Failure
**Symptoms**: `go mod tidy` reports dependency conflicts

**Resolution**:
1. Confirm submodules under `external/` are properly initialized: `git submodule update --init`
2. Check that the replace directives in `go.mod` point to the correct local paths
3. If you modified `external/IoM-go`, commit the submodule changes first

### Proto Regeneration
After modifying `.proto` files, regenerate the Go code:

```bash
cd external/IoM-go/generate
go generate
```

Then return to the project root and run `go mod tidy`.

## Debugging Methods

### Logs
- Client log directory: `~/.config/malice/log/`
- Server audit log level is configured in `server/config.yaml` (the `audit` field)
- Logs use colored output — levels: Debug (gray), Info (cyan), Warn (yellow), Error (red)

### MCP / RPC Debugging
- Start with `--mcp 127.0.0.1:5005` to enable the MCP server for interaction via an MCP client
- Start with `--rpc 127.0.0.1:15004` to enable LocalRPC for debugging via a gRPC client

### Verifying Component Status
```
listener                 # Check whether listeners are running
pipeline list            # Check whether pipelines are started
session                  # Check session status
tasks                    # Check task status
```

## How to File an Issue

File issues on GitHub: https://github.com/chainreactors/malice-network/issues

Required information:
1. **OS and architecture** — what systems the client and server are running on
2. **IoM version** — `iom --version` or commit hash
3. **Steps to reproduce** — how to reproduce the issue from scratch
4. **Expected behavior** — what you expected to happen
5. **Actual behavior** — what actually happened
6. **Logs** — relevant log excerpts from `~/.config/malice/log/`

Using gh cli:
```bash
gh issue create --repo chainreactors/malice-network --title "Issue title"
```
