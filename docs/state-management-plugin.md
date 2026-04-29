# State Management — Plugin

## Overview

The plugin uses Svelte 5 reactive variables for UI state. There is no external state management library — all state lives in `App.svelte`.

## State Variables

| Variable | Type | Purpose |
|----------|------|---------|
| `connected` | `boolean` | WebSocket connection status |
| `fileName` | `string` | Current Figma file name (from `plugin-status`) |
| `pageName` | `string` | Current page name (from `plugin-status`) |
| `selectionCount` | `number` | Currently selected node count |
| `activeRequests` | `Set<string>` | Pending request IDs (drives loading state) |
| `isWorking` | `boolean` (derived) | `activeRequests.size > 0` — shows spinner |
| `serverHost` | `string` | WebSocket host (default `"127.0.0.1"`) |
| `serverPort` | `string` | WebSocket port (default `"1994"`) |
| `showSettings` | `boolean` | Whether settings panel is open |
| `editHost` / `editPort` | `string` | Temporary values during settings editing |
| `socket` | `WebSocket \| null` | Current WebSocket instance |
| `reconnectTimer` | `ReturnType<typeof setTimeout> \| null` | Pending reconnect timer |
| `configLoaded` | `boolean` | Whether config has been loaded from storage |

## Data Flow

### Config Loading (on mount)

```
App mounts → postMessage("get_ws_config") → plugin sandbox
    → figma.clientStorage.getAsync("ws_config")
    → postMessage("ws_config", { host, port }) → App
    → sets serverHost/serverPort → connect()
```

A 500ms fallback timer ensures connection even if the sandbox doesn't respond (e.g., during hot-reload).

### WebSocket Lifecycle

```
connect() → new WebSocket(ws://{host}:{port}/ws)
    → onopen  → connected = true, postMessage("ui-ready")
    → onclose → connected = false, schedule reconnect (1.5s)
    → onerror → connected = false
    → onmessage → add requestId to activeRequests
                 → forward payload to sandbox via postMessage("server-request")
```

### Request Tracking

```
WebSocket message arrives → activeRequests.add(requestId)
    → isWorking = true → spinner shown

Sandbox returns response → handleMessage receives {requestId, ...}
    → if type !== "progress_update": activeRequests.delete(requestId)
    → socket.send(JSON.stringify(response)) → back to Go server
    → isWorking updates reactively
```

### Persistence

| Key | Storage | Purpose |
|-----|---------|---------|
| `ws_config` | `figma.clientStorage` | `{ host, port }` — WebSocket server address |

`localStorage` is unavailable in Figma's `data:` URL sandbox, so all persistence goes through `figma.clientStorage` via postMessage round-trips to the plugin sandbox.
