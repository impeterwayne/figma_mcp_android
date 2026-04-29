# Integration Architecture

## Overview

This document details the communication pathways and boundaries between the three parts of the figma-mcp-go project.

## End-to-End Architecture

```
┌─────────────┐                ┌─────────────────────────────────────────────────┐
│             │  stdio         │  Go Process                                     │
│   AI / LLM  │ ◀═══════════▶ │  ┌──────┐   ┌──────────┐   ┌────────────────┐  │
│   (Claude,  │  JSON-RPC     │  │ Node │──▶│ Leader   │──▶│ Bridge         │  │
│   Cursor…)  │  (MCP)        │  │      │   │ /ws      │   │ WebSocket conn │  │
│             │               │  │      │   │ /ping    │   │ pending map    │  │
└─────────────┘               │  │      │   │ /rpc     │   └───────┬────────┘  │
                              │  │      │   └──────────┘           │            │
                              │  │      │   ┌──────────┐           │            │
                              │  │      │──▶│ Follower │──HTTP──▶ /rpc         │
                              │  └──────┘   └──────────┘                       │
                              └────────────────────────────────┬───────────────┘
                                                               │ WebSocket
                              ┌────────────────────────────────┼───────────────┐
                              │  Figma Plugin                  │               │
                              │  ┌─────────────────┐    ┌──────┴──────┐        │
                              │  │  Plugin Sandbox  │◀──│  UI iframe  │        │
                              │  │  (code.js)       │──▶│  (Svelte)   │        │
                              │  │  main.ts         │   │  App.svelte │        │
                              │  │  handlers/*.ts   │   └─────────────┘        │
                              │  │  Figma API calls │                          │
                              │  └─────────────────┘                           │
                              └────────────────────────────────────────────────┘
```

## Communication Layers

### Layer 1: LLM ↔ Go Server (MCP over stdio)

- The Go server uses the `mcp-go` framework to expose tools via JSON-RPC over stdin/stdout.
- When started via the NPM wrapper, `spawnSync` with `stdio: 'inherit'` ensures the LLM host process directly owns the pipes.
- The server registers 70+ tools and 12 prompt strategies.

### Layer 2: Go Server ↔ Figma Plugin (WebSocket)

- The Leader starts an HTTP server on `{ip}:{port}` (default `127.0.0.1:1994`).
- The Figma Plugin's Svelte UI iframe connects via `ws://{host}:{port}/ws`.
- Messages are JSON-encoded `BridgeRequest` / `BridgeResponse` objects.
- Each request carries a unique `requestId` for async correlation.
- Read limit is 100 MB. Default timeout is 30s (60s for `get_document`).
- Progress updates (`Progress > 0`) extend the timeout without resolving.

### Layer 3: Plugin UI ↔ Plugin Sandbox (postMessage)

Figma plugins have a two-context architecture:

- **Sandbox** (code.js): Has access to `figma.*` API, no network access.
- **UI iframe** (index.html): Has network access (WebSocket), no Figma API access.

Communication uses `parent.postMessage` / `figma.ui.postMessage`:

| Direction | Message Type | Purpose |
|-----------|-------------|---------|
| UI → Sandbox | `ui-ready` | Signal WebSocket connected |
| UI → Sandbox | `get_ws_config` | Request stored host/port |
| UI → Sandbox | `save_ws_config` | Persist host/port to clientStorage |
| UI → Sandbox | `server-request` | Forward WebSocket command for execution |
| Sandbox → UI | `ws_config` | Return stored host/port |
| Sandbox → UI | `plugin-status` | File name, page name, selection count |
| Sandbox → UI | `{requestId, ...}` | Command result (forwarded to WebSocket) |

### Layer 4: Multi-Instance (Leader/Follower)

Multiple Go server instances can run simultaneously. The Election system ensures only one Leader owns the WebSocket connection:

1. First instance binds the port → becomes **Leader**.
2. Subsequent instances detect the port is taken, ping the Leader → become **Followers**.
3. Followers proxy all tool calls to the Leader via HTTP POST to `/rpc`.
4. If the Leader dies, a Follower detects it (3–5s jitter health check) and takes over.

This means multiple AI agents can share a single Figma Plugin connection.

## Security and Sandboxing

- **Localhost-only by default**: The Go server binds to `127.0.0.1`. Binding to `0.0.0.0` logs a warning about network exposure with no authentication.
- **Figma sandbox**: The plugin core (`code.js`) cannot make network requests. All network I/O is done by the UI iframe, which connects only to the localhost Go server.
- **Network access**: Plugin manifest allows all domains (`"allowedDomains": ["*"]`) because the host/port are user-configurable.
- **File write safety**: `save_screenshots` enforces that output paths resolve inside the current working directory (path traversal protection).
- **No API token**: The server operates entirely through the Figma Plugin API, requiring no Figma REST API token.

## Port Assignment

- Default: `1994` (configurable via `--port` flag on the Go binary).
- The Plugin UI also allows changing host/port in its settings panel.
- Port must match between the Go server and the Plugin.
