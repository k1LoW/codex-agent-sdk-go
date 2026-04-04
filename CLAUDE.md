# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

- `make test` — run all tests with race detection and coverage
- `make lint` — run golangci-lint and gostyle
- `make ci` — install dev deps + run tests (used in CI)
- `go test -v -run TestSessionSendRequest ./...` — run a single test
- `make test-integration` — integration tests requiring `codex` CLI
- `make credits` — regenerate CREDITS file (gocredits)

## Architecture

Unofficial Go SDK for [OpenAI Codex](https://github.com/openai/codex). Communicates with `codex app-server --listen stdio://` via bidirectional JSON-RPC over stdin/stdout. Architectural reference: [claude-agent-sdk-go](https://github.com/k1LoW/claude-agent-sdk-go).

### Protocol Flow

```
Client/Query → session → Transport (subprocess: codex app-server --listen stdio://)
                  ↕ JSON-RPC (no "jsonrpc":"2.0" field)
  sendRequest(method, params) → {id, result} or {id, error}
  readLoop receives:
    {id, result/error}       → route to pendingRequests[id] (or buffer if early)
    {id, method, params}     → server request → dispatch to approval callback
    {method, params}         → notification → parse as Event → eventCh
```

### Key Layers

- **codex.go** / **client.go** — Two public APIs: `Query()` (one-shot) and `Client` (multi-turn). Both follow: initialize → thread/start → turn/start → consume events.
- **session.go** — Core bidirectional JSON-RPC handler. Routes responses to pending requests (with buffering for race conditions), dispatches server requests to approval callbacks, and forwards notifications as Events.
- **jsonrpc.go** — Classify/marshal helpers. Codex protocol omits the `"jsonrpc": "2.0"` field.
- **types.go** — Sealed interfaces `Event` (notifications) and `ThreadItem` (item union) with concrete types. `Thread`, `Turn`, `UserInput`.
- **parser.go** — `parseEvent()` switches on notification method, `parseThreadItem()` switches on item `type` field.
- **approvals.go** — Typed callback signatures for server-initiated requests (command approval, file change approval, tool calls, user input, MCP elicitation).
- **options.go** — Three-level functional options: `Option` (instance), `ThreadOption` (thread/start), `TurnOption` (turn/start), plus `QueryOption` wrapper.
- **subprocess.go** — `subprocessTransport` spawns `codex app-server --listen stdio://`, finds the `codex` binary via PATH or common locations.

### Conventions

- Package name: `codex`. Zero external dependencies (stdlib only). Go 1.25+.
- Sealed interfaces use unexported marker methods (`eventMethod()`, `itemType()`).
- Tests use `mockTransport` (channel-based) defined in `session_test.go`.
- Version is in `version/version.go`, auto-updated by tagpr on release.
- All inline lint suppression comments (`//nolint:`, `//nostyle:`) must include a reason.
