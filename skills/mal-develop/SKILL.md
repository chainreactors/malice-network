---
name: mal-develop
description: >
  MAL (Malice Scripting Language) plugin development guide. Helps users write Lua plugins for IoM,
  covering plugin structure, command registration, BOF invocation, resource management, event callbacks,
  testing, debugging, and publishing workflows.
  Trigger conditions: when users want to write MAL plugins, extend IoM commands, write Lua scripts,
  integrate BOFs, develop custom modules, or ask questions like "how to write a mal plugin",
  "how to add a new command to IoM", or "what Lua APIs are available".
---

# MAL Plugin Development Guide

MAL is the Lua 5.1 plugin system for IoM. It extends the client with Lua scripts. Each plugin can register new commands, invoke BOFs, execute implant modules, and listen for events.

## Plugin Structure at a Glance

```
my-plugin/
├── mal.yaml             # Plugin manifest (required)
├── main.lua             # Entry script (required)
├── modules/             # Lua modules (optional, used via require)
│   └── utils.lua
└── resources/           # Resource files (optional, BOFs, DLLs, etc.)
    └── bof/
        ├── tool.x64.o
        └── tool.x86.o
```

### mal.yaml

```yaml
name: my-plugin
type: lua
author: your-name
version: 1.0.0
entry: main.lua          # Entry file
lib: false               # true = library-only plugin (does not register commands)
depend_modules: []       # Required implant modules
depend_armory: []        # Required armory resources
```

See [reference/plugin-structure.md](reference/plugin-structure.md) for details.

## Quick Example: Registering a Command

```lua
-- main.lua
local function run_hello(arg_name, cmd)
    print("Hello, " .. (arg_name or "world"))
end

local cmd = command("hello", run_hello, "Say hello", "")
opsec("hello", 10.0)
help("hello", "Usage: hello [name]")
```

## High-Frequency API Quick Reference

Sorted by usage frequency, these are the most commonly used functions when developing MAL plugins:

| Function | Purpose | Frequency |
|----------|---------|-----------|
| `command(name, fn, short, ttp)` | Register a command | Highest |
| `active()` | Get the current session | Very high |
| `script_resource(path)` | Get a plugin resource path | Very high |
| `opsec(name, score)` | Set OPSEC score | High |
| `bof(session, path, args, output)` | Execute a BOF | High |
| `bof_pack(format, ...)` | Pack BOF arguments | High |
| `bexecute_assembly(session, path, args)` | Execute .NET assembly | Medium |
| `help(name, text)` | Set help text | Medium |
| `new_sacrifice(ppid, block, etw, amsi, argue)` | Sacrifice process config | Medium |

### Parameter Conventions

```lua
local function handler(arg_target, flag_port, cmdline, args, cmd)
    -- arg_target  -> positional argument    flag_port -> --port flag
    -- cmdline     -> command line           args      -> argument array
    -- cmd         -> cobra.Command object
end
```

### BOF Argument Format

```lua
bof_pack("Ziz", wide_string, integer, ansi_string)
-- z=ANSI string  Z=wide string  i=int32  s=int16  b=binary
```

See [reference/api-reference.md](reference/api-reference.md) for the full API reference.

## Development Workflow

```
 Create          Write          Load          Verify         Debug          Publish
┌─────┐      ┌─────┐      ┌─────┐      ┌─────┐      ┌─────┐      ┌─────┐
│mkdir│─────→│ lua │─────→│load │─────→│test │─────→│ fix │─────→│push │
│yaml │      │code │      │     │      │     │      │     │      │     │
└─────┘      └─────┘      └─────┘      └─────┘      └──┬──┘      └─────┘
                                                        │
                                                  ┌─────┘
                                                  ↓ loop
                                               ┌─────┐
                                               │write │
                                               └─────┘
```

### 1. Create
```bash
mkdir -p my-plugin/resources/bof
# Write mal.yaml
```

### 2. Write
```bash
# Write main.lua, starting with the simplest command
# Refer to patterns in reference/examples.md
```

### 3. Load and Test
```
mal load /path/to/my-plugin
```

### 4. Verify
```
search_commands("my-command")       # Confirm command registration succeeded
my-command --help                   # Confirm help text is correct
my-command <test-args>              # Execute for real (requires a session)
```

### 5. Debug (on failure)
```
# Check logs
# print() in Lua outputs directly to the terminal
# After modifications, reload:
mal remove my-plugin
mal load /path/to/my-plugin
```

### 6. Publish
```
mal install /path/to/my-plugin.tar.gz    # Local install
# Or submit to https://github.com/chainreactors/mal-community
```

See [reference/testing.md](reference/testing.md) for detailed testing and verification methods.

## Reference Documentation

| Topic | Reference File |
|-------|---------------|
| Full API Reference | [reference/api-reference.md](reference/api-reference.md) |
| Plugin Structure Details | [reference/plugin-structure.md](reference/plugin-structure.md) |
| Practical Examples | [reference/examples.md](reference/examples.md) |
| Testing, Verification & Debugging | [reference/testing.md](reference/testing.md) |

### External Documentation

| Resource | Link |
|----------|------|
| MAL Quick Start | https://chainreactors.github.io/wiki/IoM/manual/mal/quickstart/ |
| IoM Wiki | https://chainreactors.github.io/wiki/IoM/ |
| Community Plugin Repository | https://github.com/chainreactors/mal-community |
| Implant Repository | https://github.com/chainreactors/malefic |
| Community Plugin Source | `helper/intl/community/modules/` (best learning reference) |
