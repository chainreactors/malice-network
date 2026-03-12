# Malice Network

## Project Overview

**Three-layer architecture:**
- `client/` — CLI/TUI client (Cobra + Bubble Tea), command tree under `client/command/`
- `server/` — gRPC/mTLS server, core logic in `server/internal/`, RPC handlers in `server/rpc/`
- `helper/` — shared utilities (crypto, encoders, config, file operations)

**External dependencies (local replacements):**
- `external/IoM-go` — Proto definitions + gRPC client (submodule)
- `external/tui` — UI component library (submodule)
- `external/console`, `external/readline`, `external/mcp-go`, `external/gonut`

> Changes under `external/` are treated as dependency work, not routine app edits. Run `go mod tidy` after any modification.

## Build & Test

Mirror the CI pipeline (`.github/workflows/ci.yaml`):

```bash
go mod tidy                              # required after dependency changes
go vet ./...                             # static analysis
go test ./... -count=1 -timeout 300s     # full test suite
CGO_ENABLED=0 go build ./...             # compile verification
```

**Local development:**
```bash
go run ./server    # start server (uses server/config.yaml)
go run ./client    # start client
```

**Pre-commit checklist:**
1. `go vet` passes with no warnings
2. `go test` all pass
3. `go build` succeeds
4. For proto changes: confirm `external/IoM-go` submodule is synced

## Language & Encoding

- **Encoding**: UTF-8 without BOM for all files
- **Code & comments**: English only (variable names, function names, comments, commit messages)
- **Docs**: All files under `docs/` written in English

## Code Conventions

### File Organization

- New Cobra commands go in `client/command/<area>/`, one feature per file
- RPC handlers go in `server/rpc/`, split by service
- Utilities go in `helper/`, organized by functional subdirectory
- Test files live alongside implementation as `*_test.go`, use table-driven tests
- `server/internal/core/` manages Session/Task/Job concurrent state — mind locks and race conditions when editing

## Proto / gRPC Conventions

Proto files are located at `external/IoM-go/generate/proto/`:

| Proto file | Purpose |
|-----------|---------|
| `client/clientpb/client.proto` | Client message definitions |
| `client/rootpb/root.proto` | Admin service definitions |
| `implant/implantpb/implant.proto` | Implant protocol |
| `services/clientrpc/service.proto` | Client RPC service |
| `services/listenerrpc/service.proto` | Listener RPC service |

**Rules:**
- Make proto changes inside the `external/IoM-go` submodule
- Never manually edit generated Go code
- After changes, update the submodule reference and run `go mod tidy`

## docs/ Conventions

`docs/` is the shared development knowledge base. Every new feature must include documentation here.

**Directory structure:**
```
docs/
├── architecture.md           # overall architecture
├── getting-started.md        # quick start guide
├── client/                   # client-side topics
│   ├── commands.md           # command system overview
│   └── <feature>.md          # specific features
├── server/                   # server-side topics
│   ├── listeners.md          # listener details
│   ├── build.md              # build pipeline
│   └── <feature>.md          # specific features
├── protocol/                 # protocol & communication
│   └── <topic>.md
└── development/              # development guides
    ├── contributing.md       # contribution guide
    └── <topic>.md
```

**Documentation requirements:**
- Every new feature PR must include a corresponding `docs/<module>/<feature>.md`
- Each doc must cover: overview, usage, configuration, and examples
- Architecture-level changes must update `docs/architecture.md`
- File names use kebab-case, e.g. `tcp-listener.md`

## Security

- **Never commit**: real secrets, tokens, certificates, or environment-specific config
- **Sensitive paths**: `server/config.yaml` (keep defaults only), `helper/intl/`
- **Binary files**: changes under `client/assets/` or `server/assets/` require documented provenance
- **external/ submodules**: verify upstream state before making changes to avoid pulling in unreleased breaking changes

## Key Paths

| Purpose | Path |
|---------|------|
| Client entry | `client/cmd/cli/` |
| Server entry | `server/cmd/server/` |
| Command tree | `client/command/` |
| RPC handlers | `server/rpc/` |
| Core runtime | `server/internal/core/` |
| Config management | `server/internal/configs/` |
| Listeners | `server/listener/` |
| Build pipeline | `server/build/` |
| Database | `server/internal/db/` |
| Proto definitions | `external/IoM-go/generate/proto/` |
| CI workflow | `.github/workflows/ci.yaml` |
| Release config | `.goreleaser.yml` |
| Default config | `server/config.yaml` |
