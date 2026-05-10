# Component Inventory — Plugin

## Overview

The Figma Plugin UI is intentionally minimal — a single Svelte component handles everything.

## `App.svelte`

- **Path**: `plugin/src/ui/App.svelte`
- **Size**: ~13 KB (includes inline styles)
- **Framework**: Svelte 5

### Responsibilities

1. **WebSocket management**: Connects to Go server at `ws://{host}:{port}/ws`, auto-reconnects on disconnect (1.5s delay).
2. **Server config**: Inline settings panel for host/port, persisted via `figma.clientStorage` (through plugin sandbox).
3. **Request tracking**: Maintains `activeRequests` set of pending `requestId`s, shows "AI is working…" spinner banner.
4. **Status display**: Shows current file name, page name, and selection count (from `plugin-status` messages).
5. **Message relay**: Bridges WebSocket ↔ postMessage (forwards commands from server to sandbox, results back to server).

### UI Layout

```
┌──────────────────────────────┐
│  File      MyDesign.fig      │
│  Page      Home              │
│  Selection 3 node(s)         │
├──────────────────────────────┤
│  🔄 AI is working…           │  ← shown when activeRequests > 0
├──────────────────────────────┤
│  127.0.0.1:1994  🟢 Connected│
│  impeterwayne Bug  Suggest   │
└──────────────────────────────┘
```

### Visual Design

- Dark theme (`#1e1e1e` background, `#e0e0e0` text).
- System font stack (`-apple-system, BlinkMacSystemFont, "Segoe UI"`).
- Color-coded connection badge (green/red pill).
- Animated spinner (`@keyframes spin` for loading state).
- Compact layout: 320×230 px viewport.
