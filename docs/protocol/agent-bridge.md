# Agent Bridge Protocol

## Overview

Agent Bridge enables implants with a built-in agent loop (`bridge_agent` module) to execute natural-language tasks autonomously. The implant runs its own agent locally, while the server proxies LLM API calls on its behalf.

This mechanism coexists with the legacy poison/tapping pipeline (which hijacks an external LLM provider session). The two are completely independent; the client dispatches to the correct backend based on which modules the session has loaded.

## Architecture

```
Client                     Server                      Implant
  |                          |                            |
  |-- BridgeAgentChat() ---->|                            |
  |   (text, model,         |-- Spite(BridgeAgentReq) -->|
  |    provider, api_key,    |                            |
  |    endpoint, max_turns)  |                            |
  |                          |   agent loop running...    |
  |                          |                            |
  |                          |<-- Spite(BridgeLlmReq) ---|
  |                          |   (raw OpenAI JSON body)   |
  |                          |                            |
  |                          |-- POST /chat/completions ->| LLM API
  |                          |<-- JSON response ----------|
  |                          |                            |
  |                          |-- Spite(BridgeLlmResp) -->|
  |                          |   {"payload": <raw>}       |
  |                          |                            |
  |                          |   ... repeat per turn ...  |
  |                          |                            |
  |                          |<-- Spite(BridgeAgentResp)-|
  |<-- Task Done ------------|   (text, tool_calls, etc.) |
```

## Proto Messages

All messages are defined in `implant/implantpb/implant.proto` as Spite body variants:

| Field Number | Message | Direction | Description |
|---|---|---|---|
| 164 | `BridgeAgentRequest` | server -> implant | Initial task with text, model, config |
| 165 | `BridgeAgentResponse` | implant -> server | Final result with text, tool calls |
| 166 | `BridgeLlmRequest` | implant -> server | Raw OpenAI-format request body |
| 167 | `BridgeLlmResponse` | server -> implant | Wrapped API response `{"payload": ...}` |

Legacy `llm_event = 160` is preserved for the poison/tapping pipeline.

## RPC

```protobuf
rpc BridgeAgentChat(implantpb.BridgeAgentRequest) returns (clientpb.Task);
```

The handler uses `StreamGenericHandler` for bidirectional streaming (same pattern as `handlePtyStart`). It spawns a `runTaskHandler` goroutine that:

1. Reads from the implant channel (`out`)
2. If `BridgeLlmRequest`: calls `llm.CallProvider()`, sends response back via `in.Send()`
3. If `BridgeAgentResponse`: calls `HandlerSpite()` + `Finish()`, returns

## LLM Provider

`server/internal/llm/provider.go` handles LLM API proxying with a three-level config resolution:

1. **Request parameters** (from client's `config ai` settings, passed via `BridgeAgentRequest`)
2. **Environment variables** (`BRIDGE_<PROVIDER>_BASE_URL`, `BRIDGE_API_KEY`, etc.)
3. **Provider presets** (built-in base URLs for openai, openrouter, deepseek, groq, moonshot)

## Client Commands

### `chat`

```
chat [message]
chat -m gpt-4o "list all files"
chat -p deepseek "scan the network"
```

Sends a message to the implant's self-agent. Reads LLM config from `config ai` automatically. Flags `--model`/`--provider` override the config values.

Requires the `bridge_agent` module on the session.

### `skill` (dual dispatch)

```
skill <name> [arguments...]
skill list
```

Loads a `SKILL.md` file and dispatches based on session capabilities:
- Session has `bridge_agent` module -> `BridgeAgentChat`
- Otherwise -> `Poison` (legacy pipeline)

The `depend` annotation is `bridge_agent,poison`, meaning the command is visible when either module is present.

## Configuration

No separate configuration is needed. The `chat` command reads from the existing `config ai` settings:

```
config ai --provider openai --api-key sk-xxx --endpoint https://api.openai.com/v1
```

These values are passed through `BridgeAgentRequest` to the server, which uses them to proxy LLM calls. If the request fields are empty, the server falls back to environment variables and provider presets.

## Module Detection

The client checks `session.Modules` for the `bridge_agent` capability:

```go
func hasModule(sess *client.Session, name string) bool
```

This determines which backend to use for `skill` dispatch and whether `chat` is available in the command tree.
