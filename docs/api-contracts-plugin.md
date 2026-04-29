# API Contracts — Plugin (Extension)

## Overview

This document details the internal messaging contracts used by the Figma Plugin, covering the three communication boundaries: WebSocket, postMessage (UI ↔ Sandbox), and Figma clientStorage.

## WebSocket Protocol (UI ↔ Go Server)

The Plugin UI iframe connects to `ws://{host}:{port}/ws`.

### Inbound Messages (Go Server → Plugin UI)

JSON objects conforming to `BridgeRequest`:

```json
{
  "type": "create_rectangle",
  "requestId": "req-143022-5",
  "nodeIds": ["4029:12345"],
  "params": { "width": 200, "height": 100 }
}
```

- `type` — The tool/command name.
- `requestId` — Unique ID for correlating the response.
- `nodeIds` — Target Figma node IDs (optional).
- `params` — Tool-specific parameters (optional).

### Outbound Messages (Plugin UI → Go Server)

#### Success Response

```json
{
  "type": "create_rectangle",
  "requestId": "req-143022-5",
  "data": { "id": "123:456", "name": "Rectangle 1" }
}
```

#### Error Response

```json
{
  "type": "create_rectangle",
  "requestId": "req-143022-5",
  "error": "Node not found"
}
```

#### Progress Update

```json
{
  "type": "progress_update",
  "requestId": "req-143022-5",
  "progress": 45,
  "message": "Processing 45 of 100 nodes"
}
```

Progress updates extend the server-side timeout without resolving the request.

## postMessage Protocol (UI iframe ↔ Plugin Sandbox)

All messages are wrapped in `{ pluginMessage: <payload> }` per Figma plugin conventions.

### UI → Sandbox

| Message Type | Payload | Purpose |
|-------------|---------|---------|
| `ui-ready` | `{ type: "ui-ready" }` | Signal WebSocket connection established |
| `get_ws_config` | `{ type: "get_ws_config" }` | Request stored host/port config |
| `save_ws_config` | `{ type: "save_ws_config", host, port }` | Persist updated host/port |
| `server-request` | `{ type: "server-request", payload: BridgeRequest }` | Forward command for Figma API execution |

### Sandbox → UI

| Message Type | Payload | Purpose |
|-------------|---------|---------|
| `ws_config` | `{ type: "ws_config", host, port }` | Return stored host/port |
| `plugin-status` | `{ type: "plugin-status", payload: { fileName, pageName, selectionCount } }` | Document context update |
| Response | `{ type, requestId, data?, error? }` | Command result to forward to WebSocket |

## Storage Contract (figma.clientStorage)

| Key | Type | Purpose |
|-----|------|---------|
| `ws_config` | `{ host: string, port: string }` | WebSocket server address (default: `127.0.0.1:1994`) |

`figma.clientStorage` is used instead of `localStorage` because the UI runs in a `data:` URL sandbox where `localStorage` is unavailable.

## Handler Dispatch Chain

The plugin sandbox routes requests through a handler chain:

```
handleRequest(request)
  → handleReadRequest(request)
    → handleReadDocumentRequest(request)    // get_document, get_node, search_nodes, etc.
    → handleReadStyleRequest(request)       // get_styles, get_variables, get_fonts, etc.
    → handleReadExportRequest(request)      // get_screenshot
  → handleWriteRequest(request)
    → handleWriteCreateRequest(request)     // create_frame, create_rectangle, etc.
    → handleWriteModifyRequest(request)     // set_fills, move_nodes, etc.
    → handleWriteStyleRequest(request)      // create_paint_style, etc.
    → handleWriteVariableRequest(request)   // create_variable, etc.
    → handleWriteComponentRequest(request)  // create_component, swap_component, etc.
    → handleWritePrototypeRequest(request)  // set_reactions, etc.
    → handleWritePageRequest(request)       // add_page, delete_page, etc.
  → Error("Unknown request type")
```

Each handler returns a result object or `null` to pass to the next handler.
