# Development Guide

## Prerequisites

- **Go 1.26.1+** — Building the backend.
- **Node.js >= 18.0.0** — NPM wrapper packaging.
- **Bun** — TypeScript execution and plugin test runner.
- **Figma Desktop App** — Required to run the plugin locally.

## Installation & Setup

### 1. Go Backend

```bash
go mod download
make build-go
# or: go build -o bin/figma-mcp-go ./cmd/figma-mcp-go
```

### 2. Figma Plugin

```bash
cd plugin
bun install
bun run build
```

To load in Figma: **Plugins → Development → Import plugin from manifest…** → select `plugin/manifest.json`.

### 3. NPM Wrapper

```bash
cd npm
npm install
```

## Development Commands

| Command | Directory | Description |
|---------|-----------|-------------|
| `make build` | root | Builds Go backend + plugin |
| `make test` | root | Runs all tests (Go + TS) |
| `make coverage` | root | Tests with coverage for both |
| `bun run dev` | plugin | Live dev — two concurrent Vite watchers for UI + sandbox |
| `bun test` | plugin | Plugin unit tests (Bun) |
| `go test ./... -v` | root | Verbose Go tests |

## Running the Server

```bash
./bin/figma-mcp-go                          # default localhost:1994
./bin/figma-mcp-go --ip 127.0.0.1 --port 3000  # custom
```

## Architecture Notes for Contributors

- **Dual Vite configs**: `vite.config.ts` builds UI (Svelte → `index.html`), `vite.config.main.ts` builds sandbox (`main.ts` → `code.js` IIFE).
- **Handler chains**: Go and TypeScript sides mirror the same structure (read-document, write-modify, etc.).
- **Version injection**: Set at build time via `-ldflags "-X main.version=..."`. Defaults to `"dev"`.
