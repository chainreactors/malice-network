# Lua Script Execution

The IoM client supports executing Lua scripts directly via the MCP tool `execute_lua`, enabling automation and batch processing.

## MCP Tool: execute_lua

```
execute_lua(script, session_id?)
```

- `script` (required) — Lua 5.1 script code
- `session_id` (optional) — switch to the specified session before execution

The script runs in the client's Lua VM and has access to all IoM internal APIs.

## Basic Usage

### Simple Calculations and String Operations

```lua
-- The return value is output as the tool result
return "hello from lua"

-- String formatting
return string.format("2 + 2 = %d", 2 + 2)

-- Multiple return values (one per line)
return "line1", "line2", "line3"
```

### Scripts Without a Return Value

```lua
-- print() outputs to the client terminal but is not returned as an MCP result
print("this goes to terminal")

-- When there is no return value, MCP returns "Script executed successfully (no return value)"
local x = 1 + 1
```

## Available Standard Libraries

| Library | Description | Examples |
|---------|-------------|----------|
| `string` | String manipulation | `string.format()`, `string.find()`, `string.gsub()` |
| `table` | Table operations | `table.insert()`, `table.concat()`, `table.sort()` |
| `math` | Math operations | `math.floor()`, `math.random()` |
| `os` | System operations | `os.time()`, `os.date()` |
| `io` | File I/O | `io.open()`, `file:read()` |

## Using IoM APIs

Within a session context, Lua scripts can call IoM internal APIs:

```lua
-- Get current session information
local session = active()
return string.format("Host: %s, Arch: %s, User: %s",
    session.Os.Hostname, session.Os.Arch, session.Os.Username)
```

```lua
-- Batch operation example: format a process list
local session = active()
local ps_result = ps(session)
-- Process the result...
```

```lua
-- Timestamps and random strings
return string.format("Time: %s, Random: %s", timestamp(), random_string(8))
```

## Use Cases

### Automated Batch Operations

Lua scripts can combine multiple commands into a single automated workflow, which is more efficient than executing commands one by one manually.

### Data Processing

Filter, format, and aggregate data returned by commands.

### Custom Logic

Dynamically determine execution paths based on environment conditions (OS, architecture, privileges).

## Error Handling

- **Syntax errors** — MCP returns an error message including the line number and error description
- **Runtime errors** — `error("message")` is caught and returned to the MCP caller
- **Script timeout** — Long-running scripts may be blocked due to VM pool contention

## Notes

- Scripts execute in the client's Lua VM pool (5 shared VMs); concurrent executions may queue
- `print()` outputs to the client terminal and is not returned as the MCP tool result
- Use `return` to send results back to the MCP caller
- APIs involving implant operations (e.g., `bof()`, `execute_module()`) require entering a session context first
