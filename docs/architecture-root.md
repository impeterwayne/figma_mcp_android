# Architecture — Root Go App

## Executive Summary

The Root Go App is the MCP server that bridges AI/LLM tool calls to the Figma Plugin. It exposes 70+ tools via the Model Context Protocol (JSON-RPC over stdio) and maintains a WebSocket connection to the Figma Plugin for executing commands.

## Technology Stack

- **Language**: Go 1.26.1
- **MCP Framework**: `mark3labs/mcp-go v0.46.0`
- **WebSocket**: `coder/websocket v1.8.14`
- **PDF Export**: `pdfcpu/pdfcpu v0.11.1`
- **Architecture Pattern**: Leader/Follower MCP server with election-based role assignment

## Key Architectural Components

### Entry Point (`cmd/figma-mcp-android/main.go`)

The binary accepts `--ip` (default `127.0.0.1`) and `--port` (default `1994`) flags. On startup it:

1. Creates a `Node` (role-based request router).
2. Creates an `Election` and starts it (determines Leader vs Follower role).
3. Registers all MCP tools and prompts on the server.
4. Serves MCP over stdio.

### Node (`internal/node.go`)

The central request router. It dynamically dispatches MCP tool calls based on the current role:

- **Leader**: Routes to the local `Bridge` (direct WebSocket to Plugin).
- **Follower**: Routes to the Leader via HTTP `/rpc` proxy.

Also normalizes Figma node IDs (hyphen → colon format) that LLMs sometimes produce.

### Leader (`internal/leader.go`)

Owns the WebSocket bridge and exposes three HTTP endpoints:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/ws`    | GET    | WebSocket upgrade for the Figma Plugin |
| `/ping`  | GET    | Health check (used by Followers and Election) |
| `/rpc`   | POST   | JSON RPC proxy for Follower tool calls |

### Follower (`internal/follower.go`)

Proxies MCP tool calls to the Leader via HTTP POST to `/rpc`. Includes a 35-second timeout (intentionally longer than the 30-second bridge timeout, so the Leader times out first).

### Election (`internal/election.go`)

Determines and maintains the server role:

1. **Startup**: Tries to bind the port (become Leader). If the port is taken, checks if a healthy Leader exists — if so, becomes Follower.
2. **Monitoring**: Runs a background loop with 3–5 second jitter. Followers ping the Leader; if the Leader is unresponsive, the Follower attempts a takeover.

### Bridge (`internal/bridge.go`)

Manages the single WebSocket connection to the Figma Plugin:

- Accepts one plugin connection at a time (new connections replace old ones).
- Matches responses to pending requests via `requestId` correlation.
- Read limit set to 100 MB (Figma documents can be large).
- Default timeout: 30 seconds (60 seconds for `get_document`).
- Supports progress updates from the plugin that extend the timeout.

### Tools

Tools are split by read/write responsibility:

| File | Category |
|------|----------|
| `tools_read_document.go` | Document & node inspection, search, scan |
| `tools_read_styles.go`   | Styles, variables, components, fonts, annotations |
| `tools_read_export.go`   | Screenshots, PDF export |
| `tools_write_create.go`  | Node creation (frame, rectangle, ellipse, text, image) |
| `tools_write_modify.go`  | Node modification (fills, strokes, transforms, etc.) |
| `tools_write_styles.go`  | Style CRUD (paint, text, effect, grid) |
| `tools_write_variables.go` | Variable/token management |
| `tools_write_components.go` | Component creation, swapping, detaching |
| `tools_write_prototype.go` | Prototype reactions |
| `tools_write_page.go`    | Page management |

### Prompts (`internal/prompts/`)

Registers 12 MCP prompt strategies for LLM-assisted workflows:

- `read_design_strategy` — Reading/inspecting designs
- `design_strategy` — Creating designs
- `text_replacement_strategy` — Text find/replace
- `annotation_conversion_strategy` — Converting annotations
- `swap_overrides_instances` — Swapping component instances
- `reaction_to_connector_strategy` — Prototype reaction conversion
- `style_audit_strategy` — Auditing styles
- `bulk_rename_strategy` — Batch renaming
- `design_token_generation_strategy` — Generating design tokens
- `generate_color_palette` — Color palette generation
- `generate_type_scale` — Typography scale generation
- `generate_component_variants` — Component variant generation

### Schema (`internal/schema.go`)

Contains MCP tool definitions (names, descriptions, JSON schemas for parameters). At ~30 KB, this is the largest source file and defines the full tool surface area.

## Testing Strategy

- Go's built-in testing framework with `*_test.go` files alongside source.
- Comprehensive test coverage for bridge, election, node, tools, and schema.
- Run via `go test ./...` or `make test-go`.
