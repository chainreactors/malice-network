# MAL Plugin Testing, Verification, and Debugging

## Verification Loop: From Loading to Confirming Runability

### Step 1: Load the Plugin

```
mal load /path/to/my-plugin
```

On success, there is no error output. On failure, Lua compilation or loading errors are displayed.

### Step 2: Confirm Command Registration

```
search_commands("my-command")
```

If your command appears in the search results, registration was successful. If not:
- Check whether `command()` is actually reached in the entry file
- Check whether `entry` in `mal.yaml` points to the correct file
- Check whether `lib: true` is set by mistake (library plugins do not register commands)

### Step 3: Verify Help Text

```
my-command --help
```

Confirm:
- Short description is correct
- Flags are registered
- Help/Example text is correct
- OPSEC score is displayed (if `opsec()` was used)

### Step 4: Test Without a Session (Client Commands)

Commands that do not involve implant operations can be tested directly:

```
my-command --flag value
```

### Step 5: Test With a Session (Implant Commands)

Requires an active session:

```
use <session_id>              # Enter session context
my-command --flag value       # Execute the command
tasks                         # Check task status
```

### Step 6: Iterative Modification

After modifying Lua code, you must reload:

```
mal remove my-plugin
mal load /path/to/my-plugin
```

Then repeat verification from Step 2.

## Common Errors and Troubleshooting

### Lua Syntax Errors

**Symptom**: Compilation error during `mal load`

**Lua syntax notes** (differences from other languages):

| Construct | Lua | Not |
|-----------|-----|-----|
| Not equal | `~=` | `!=` |
| String concatenation | `..` | `+` |
| Null value | `nil` | `null` |
| Array start index | Starts at 1 | Not 0 |
| Booleans | `true/false` | Lowercase |
| Comments | `--` / `--[[ ]]` | `//` / `/* */` |
| Logical operators | `and`, `or`, `not` | `&&`, `||`, `!` |

### Command Not Appearing

| Cause | Troubleshooting |
|-------|----------------|
| Lua errors before `command()` | Add `print("reached")` before `command()` |
| Command name conflict | Try a different command name |
| `mal.yaml` entry error | Confirm the `entry` field points to the correct file |
| `lib: true` | Library plugins do not register commands; change to `false` |

### BOF Execution Crash

| Cause | Troubleshooting |
|-------|----------------|
| Format string mismatch | Verify the argument types and count expected by the BOF source |
| Architecture mismatch | x64 BOF cannot run on an x86 session |
| Character encoding error | Use `z` for ANSI, `Z` for Unicode |
| Resource file not found | Confirm the path and filename under `resources/` are correct |

### active() Returns nil

No session context is active. First enter a session with `use <session_id>`.

### VM Pool Exhaustion

**Symptom**: Command execution hangs; logs show `"VM pool is full, waiting for available VM..."`

**Cause**: Each plugin has a concurrency pool of 10 Lua VMs. If more than 10 commands are executing concurrently, new requests will wait.

**Solution**: Check whether a command handler is stuck in an infinite loop or blocking for too long.

## Debugging Tips

### Print Debugging

`print()` outputs directly to the client terminal:

```lua
local function handler(args, cmd)
    print("[DEBUG] args: " .. #args)
    for i, v in ipairs(args) do
        print("[DEBUG]   " .. i .. ": " .. tostring(v))
    end
    print("[DEBUG] flag: " .. cmd:Flags():GetString("opt"))
end
```

### Incremental Development

Start with the simplest command and add features step by step, verifying at each stage:

```
Step 1: Empty command        -> mal load -> search_commands to verify
Step 2: Add print output     -> mal remove + load -> execute to verify
Step 3: Add flags            -> verify --help and flag reading
Step 4: Add BOF execution    -> verify in a session
Step 5: Add help/opsec       -> verify --help output
Step 6: Add error handling   -> test edge cases
```

### Logging

Client log directory: `~/.config/malice/log/`

### Study Community Code

The best way to learn is by reading community plugin source code:

```
helper/intl/community/modules/
├── lib.lua             # Helper functions (bof_pack, read, has_clr_version)
├── common.lua          # Common commands (screenshot, curl, nanodump, mimikatz)
├── enum.lua            # Enumeration commands (minimal BOF pattern)
├── elevate.lua         # Privilege escalation (multi-mode execution, CLR detection, shellcode loading)
├── move.lua            # Lateral movement (credential handling, Kerberos tickets)
├── persistence.lua     # Persistence (payload source abstraction, registry/service/scheduled tasks)
├── token.lua           # Token operations (enum validation pattern)
├── base.lua            # Module loading (load_module, tab completion)
├── clipboard.lua       # Clipboard (minimal no-argument BOF pattern)
├── net_user.lua        # Network user operations
├── exclusion.lua       # Exclusion lists
├── route.lua           # Route operations
└── rem.lua             # REM related
```

Each module demonstrates a different development pattern. Refer to them as needed:
- **Minimal BOF** -> `clipboard.lua`, `enum.lua`
- **Complex flag handling** -> `common.lua` (curl, nanodump)
- **Multi-mode execution** -> `elevate.lua` (EfsPotato)
- **Credential passing** -> `move.lua` (psexec, wmi)
- **Payload management** -> `persistence.lua`
- **Module loading** -> `base.lua`

## Go Test Framework

If you want to merge a plugin into the community repository, it must pass the Go test framework.

Test files are located in `helper/intl/`:

| File | What It Verifies |
|------|-----------------|
| `mal_compile_test.go` | All .lua files have correct syntax (compilable) |
| `mal_command_test.go` | Command registration is complete (count, TTP, description, OPSEC) |
| `mal_test_harness_test.go` | Command handlers execute without crashing (mock environment) |

### How the Test Harness Works

The test harness provides a mock Lua VM with stub versions of all APIs registered (`active()` returns a mock session, `bof()` records call arguments but does not execute).

Verification items:
- All `command()` calls register successfully
- Every command has a short description
- Non-utility commands have TTP annotations
- BOF pack format strings are valid (contain only `z Z i s b`)
- Resource path references are reasonable

### Running Tests

```bash
# Compilation check
go test ./helper/intl/ -run TestMalCompile -v

# Command registration check
go test ./helper/intl/ -run TestMalCommand -v

# Handler execution check
go test ./helper/intl/ -run TestMalHarness -v
```

## Pre-Release Checklist

Confirm before publishing:

- [ ] `mal.yaml` fields are complete (name, type, author, version, entry)
- [ ] All `command()` calls have a short description
- [ ] Commands involving ATT&CK have TTP annotations
- [ ] All commands have `opsec()` scores
- [ ] Key commands have detailed `help()` text
- [ ] Resource file paths are correct (x64/x86 suffixes)
- [ ] Core functionality has been verified on a real session
- [ ] Invalid input does not crash the handler
- [ ] `mal load` + `mal remove` + `mal load` cycle works correctly
