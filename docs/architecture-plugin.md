# Architecture — Plugin (Figma Extension)

## Overview

The plugin is the Figma-side runtime for `figma-mcp-android`. Its job is simple:

- accept commands from the local Go server,
- execute those commands through the Figma Plugin API, and
- send structured results back to the server.

Because Figma splits plugins into two isolated contexts, this project also splits its responsibilities in two:

- the **plugin sandbox** (`dist/code.js`) can call `figma.*` APIs but cannot own the network bridge used here,
- the **UI iframe** (`dist/index.html`) can open the WebSocket connection to the Go server but cannot call the Figma API directly.

The architecture is therefore a small bridge: **WebSocket in the UI, Figma API in the sandbox, `postMessage` between them**.

For the full cross-process picture, see [Integration Architecture](./integration-architecture.md). For message shapes, see [API Contracts — Plugin](./api-contracts-plugin.md).

## What the plugin contains

The plugin is a separate TypeScript app under `plugin/` with two build targets:

| Output | Source | Purpose |
|---|---|---|
| `dist/index.html` | `src/ui/` | Svelte UI iframe that manages the WebSocket connection and lightweight status/settings UI |
| `dist/code.js` | `src/main.ts` | Figma sandbox entry point that dispatches requests to read/write handlers |

These outputs are wired into `plugin/manifest.json`:

```json
{ "main": "dist/code.js", "ui": "dist/index.html" }
```

## Technology stack

- **Language**: TypeScript
- **UI framework**: Svelte 5
- **Build tool**: Vite 6
- **UI packaging**: `vite-plugin-singlefile` to inline the UI into one HTML file
- **Plugin typings**: `@figma/plugin-typings`
- **Test runner**: Bun (`bun test`)

## Build architecture

The plugin uses two Vite configs because the two Figma contexts have different runtime needs.

### 1. UI build

`plugin/vite.config.ts` builds the UI from `plugin/src/ui/` into `plugin/dist/index.html`.

Key details:

- Svelte is enabled through `@sveltejs/vite-plugin-svelte`.
- `vite-plugin-singlefile` inlines scripts and styles into one file.
- `cssCodeSplit: false` and `inlineDynamicImports: true` help keep the UI self-contained.
- `emptyOutDir: true` clears `dist/` before this build runs.

This matters because the plugin UI is loaded as a single embedded document rather than as a normal multi-file web app.

### 2. Sandbox build

`plugin/vite.config.main.ts` builds `plugin/src/main.ts` into `plugin/dist/code.js`.

Key details:

- output format is **IIFE**,
- file name is fixed to `code.js`,
- `minify: false` keeps the sandbox bundle easier to inspect,
- `emptyOutDir: false` preserves the UI artifact already written to `dist/`.

### Build commands

Defined in `plugin/package.json`:

- `bun run build` or `npm run build` → build both outputs once
- `bun run dev` or `npm run dev` → watch both outputs in parallel
- `bun test` → run plugin tests

## Runtime boundaries

Figma plugins in this project run across two contexts with a strict boundary:

### Plugin sandbox (`src/main.ts`)

The sandbox is the only place that can use the Figma Plugin API. In this project it is responsible for:

- calling `figma.showUI(__html__, { width: 320, height: 230 })`,
- publishing lightweight document status to the UI,
- loading and saving WebSocket configuration through `figma.clientStorage`,
- dispatching incoming tool requests to read and write handler chains,
- catching request errors and returning structured error payloads.

The sandbox also listens to:

- `selectionchange`
- `currentpagechange`

and forwards updated file/page/selection information to the UI.

### UI iframe (`src/ui/App.svelte`)

The UI is intentionally small. It is not where Figma operations happen. Its responsibilities are:

- opening a WebSocket to `ws://{host}:{port}/ws`,
- forwarding server messages to the sandbox with `parent.postMessage`,
- forwarding sandbox responses back to the WebSocket,
- showing connection state,
- showing file name, page name, and selection count,
- tracking in-flight requests so the user sees when work is running,
- letting the user edit the server host and port.

The saved server address defaults to `127.0.0.1:1994`.

## Request flow

The core runtime path looks like this:

```text
Go server
  -> WebSocket message to UI iframe
  -> UI forwards { type: "server-request", payload } to sandbox
  -> sandbox routes to read/write handlers
  -> sandbox returns result via figma.ui.postMessage(...)
  -> UI forwards JSON response back over WebSocket
```

That split is the main architectural constraint of the plugin.

## Sandbox entry point and dispatch

`plugin/src/main.ts` is intentionally small. It mostly coordinates other modules.

Its message handling currently supports these UI-originated events:

| Message | Purpose |
|---|---|
| `ui-ready` | tell the sandbox the UI is ready and request fresh status |
| `get_ws_config` | load persisted host/port from `figma.clientStorage` |
| `save_ws_config` | persist updated host/port |
| `server-request` | execute a tool request from the Go server |

Request execution uses a two-stage chain:

```text
handleRequest(request)
  -> handleReadRequest(request)
  -> handleWriteRequest(request)
  -> error if no handler recognizes the request type
```

If a handler throws, `main.ts` catches the error and returns a structured error response using the same `type` and `requestId`.

## Module map

### Read side

`plugin/src/read-handlers.ts` aggregates read-only operations across three modules:

- `read-document.ts` — document structure, node lookup, search, scanning, metadata-style reads
- `read-styles.ts` — local styles, variables, fonts, components, annotations, related style data
- `read-export.ts` — screenshot and export-oriented reads

### Write side

`plugin/src/write-handlers.ts` aggregates mutating operations across these modules:

- `write-create.ts` — create nodes
- `write-modify.ts` — change existing nodes
- `write-styles.ts` — create and update styles, apply effects, and bind variables to supported node fields
- `write-variables.ts` — variable and token operations
- `write-components.ts` — component-related changes
- `write-prototype.ts` — reaction and prototype updates
- `write-page.ts` — page-level operations

### Shared support modules

- `serializers.ts` — converts Figma objects into clean JSON-friendly structures for tool responses
- `write-helpers.ts` — shared write utilities used by multiple write modules

`serializers.ts` is especially important because raw Figma objects are not suitable for direct JSON transport. This module normalizes them into the shapes the bridge can safely return.

## UI behavior that matters architecturally

Although the UI is small, a few behaviors are important to understand when changing the system:

### Connection lifecycle

- The UI requests saved config from the sandbox on mount.
- If the sandbox does not answer quickly, the UI falls back after 500 ms and tries the default address.
- On disconnect, it schedules reconnect after 1.5 seconds.
- If settings change, any pending reconnect is canceled and a new connection starts immediately.

### Request tracking

`App.svelte` keeps a `Set<string>` of active request IDs.

- When a WebSocket message with `requestId` arrives, the request is marked active.
- When the sandbox sends back a non-`progress_update` response, the request is removed.
- The UI shows an **“AI is working…”** banner while at least one request is active.

This is not business logic, but it is part of the operator experience and explains why the UI needs to understand request IDs.

### Config persistence

The UI cannot rely on browser `localStorage` in this environment, so host/port settings are stored through the sandbox using `figma.clientStorage` under the `ws_config` key.

## Manifest-level constraints

`plugin/manifest.json` currently declares:

- `documentAccess: "dynamic-page"`
- `editorType: ["figma", "dev"]`
- `capabilities: ["inspect"]`
- `networkAccess.allowedDomains: ["*"]`

The open network allowlist is not because the plugin talks broadly to the internet. It exists because the host and port are user-configurable, so the UI must be able to connect to custom server addresses.

## Testing shape

The plugin has focused TypeScript tests around the modules that carry most of the behavior, including:

- serializers,
- read styles,
- write create,
- write modify,
- write effects,
- write components,
- write page,
- write prototype,
- shared write helpers.

Run them from `plugin/` with:

```bash
bun test
```

## Practical reading guide

If you are new to the plugin code, read it in this order:

1. `plugin/src/main.ts` — understand the boundary and dispatch flow
2. `plugin/src/ui/App.svelte` — understand the WebSocket bridge and operator-facing behavior
3. `plugin/src/read-handlers.ts` and `plugin/src/write-handlers.ts` — see how requests are routed
4. the specific handler module you want to change
5. `plugin/src/serializers.ts` if your change affects returned data shapes

That order mirrors the runtime path and is usually the fastest way to understand a change safely.
