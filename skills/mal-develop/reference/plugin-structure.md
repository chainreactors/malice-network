# MAL Plugin Structure in Detail

## Directory Structure

```
my-plugin/
├── mal.yaml             # Plugin manifest (required)
├── main.lua             # Entry script (required, specified by the entry field in mal.yaml)
├── modules/             # Lua modules (optional)
│   ├── utils.lua        # Referenced via require("modules.utils")
│   └── scan.lua
└── resources/           # Resource files (optional)
    ├── bof/
    │   ├── tool.x64.o   # x64 BOF
    │   └── tool.x86.o   # x86 BOF
    ├── lib/
    │   └── helper.x64.dll
    └── common/
        └── config.json
```

## mal.yaml Field Reference

```yaml
name: my-plugin          # Unique plugin identifier (required)
type: lua                # Plugin type, currently only "lua" is supported (required)
author: your-name        # Author (required)
version: 1.0.0           # Semantic version (required)
entry: main.lua          # Entry file path (required)
lib: false               # Whether this is a library-only plugin (optional, default false)
depend_modules: []       # Required implant modules (optional)
depend_armory: []        # Required armory resources (optional)
```

### Library Mode

Plugins with `lib: true` do not register commands. They only provide Lua modules that other plugins can `require`. During installation, resource files are copied to the global `~/.malice/resources/` directory.

### Dependency Declarations

```yaml
depend_modules:
  - execute_exe          # Requires the implant to support the execute_exe module
  - execute_bof          # Requires BOF execution capability

depend_armory:
  - nanodump             # Requires the nanodump resource from the armory
```

## Entry File (main.lua)

The entry file is executed when the plugin is loaded. It typically does two things:
1. Import sub-modules
2. Register commands

```lua
-- Import sub-modules
require("modules.utils")
require("modules.scan")

-- Simple command registered directly in the entry file
local function run_info(cmd)
    print("Plugin version: 1.0.0")
end

command("my-info", run_info, "Show plugin info", "")
```

### Module References

Lua files under `modules/` are imported via `require`. The path is automatically added to `package.path`.

```lua
-- modules/utils.lua
local M = {}

function M.format_output(data)
    return string.format("[*] %s", data)
end

return M

-- main.lua
local utils = require("modules.utils")
print(utils.format_output("hello"))
```

## Plugin Levels

IoM supports three plugin levels. Higher levels can override commands with the same name from lower levels:

| Level | Priority | Location | Description |
|-------|----------|----------|-------------|
| custom | Highest | `helper/intl/custom/` or external | User-defined |
| professional | Medium | `helper/intl/professional/` | Professional edition features |
| community | Lowest | `helper/intl/community/` | Community contributions |

## Embedded Plugins vs External Plugins

**Embedded plugins**: Compiled into the binary (under `helper/intl/`), loaded via Go's `embed` package.

**External plugins**: Installed to `~/.malice/mals/`, loaded dynamically at runtime.

```bash
# Install an external plugin
mal install /path/to/plugin.tar.gz

# Load an installed plugin
mal load my-plugin

# List all plugins
mal list

# Uninstall
mal remove my-plugin
```

## Resource File Conventions

### BOF File Naming

```
resources/bof/<name>/<name>.<arch>.o
```

Use `find_resource()` to automatically locate by architecture:

```lua
local session = active()
local path = find_resource(session, "bof/tool/tool", "o")
-- x64 -> resources/bof/tool/tool.x64.o
```

### Common Helper Function

A commonly used BOF path helper in community plugins:

```lua
local function bof_path(bof_name, arch)
    return "bof/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
```

## Plugin Lifecycle

```
Install (mal install)
    ↓ Extract to ~/.malice/mals/<name>/
Load (mal load / auto-load)
    ↓ Parse mal.yaml
    ↓ Create Lua VM Pool (10 VMs)
    ↓ Register Go -> Lua functions
    ↓ Execute entry script -> register commands
    ↓ Register event callbacks
Run
    ↓ User invokes command -> acquire VM from pool -> execute handler -> return VM
Unload (mal remove)
    ↓ Destroy VM Pool -> remove commands -> delete directory
```

## Community Repository Publishing

Default repository: https://github.com/chainreactors/mal-community

```bash
# Refresh plugin index
mal refresh

# Install from repository
mal install <plugin-name>

# Update all plugins
mal update --all
```
