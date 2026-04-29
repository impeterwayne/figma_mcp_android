# Project Overview: figma-mcp-go

## Executive Summary

**figma-mcp-go** is a full-featured Model Context Protocol (MCP) server for Figma. It enables AI models (LLMs) to read, write, modify, and export Figma documents — with zero API token required and no rate limits. The server communicates with Figma through a companion plugin that runs inside the Figma Desktop or Web app.

The project is authored by [@vkhanhqui](https://github.com/vkhanhqui) and licensed under MIT.

## How It Works

```
┌─────────────┐  stdio/JSON-RPC   ┌──────────────┐  WebSocket   ┌──────────────────┐
│  AI / LLM   │ ◀──────────────▶  │  Go MCP      │ ◀──────────▶ │  Figma Plugin    │
│  (Claude,   │                   │  Server      │   (JSON)     │  (Svelte UI +    │
│   Cursor…)  │                   │              │              │   Figma API)     │
└─────────────┘                   └──────────────┘              └──────────────────┘
```

1. The LLM connects to the Go server via MCP (JSON-RPC over stdio).
2. The Go server translates tool calls into JSON commands and sends them over WebSocket.
3. The Figma Plugin receives commands, executes them against the Figma Plugin API, and returns results.

## Repository Structure

- **Type**: Multi-part monorepo (3 parts)
- **Parts**:
  - `root` — Go backend: MCP server, WebSocket bridge, leader/follower election.
  - `plugin` — Figma Plugin: Svelte 5 UI + TypeScript handlers that execute Figma API calls.
  - `npm` — NPM CLI wrapper: cross-platform binary distribution.

## Tech Stack Summary

| Part   | Language / Framework         | Build Tool     | Test Runner       |
|--------|------------------------------|----------------|-------------------|
| Root   | Go 1.26.1                   | `go build`     | `go test`         |
| Plugin | TypeScript, Svelte 5        | Vite 6 (dual)  | Bun               |
| NPM    | Node.js (CommonJS)          | —              | —                 |

## Key Dependencies

| Dependency                  | Purpose                                |
|-----------------------------|----------------------------------------|
| `mark3labs/mcp-go v0.46.0`  | MCP server framework                   |
| `coder/websocket v1.8.14`   | WebSocket server for plugin bridge     |
| `pdfcpu/pdfcpu v0.11.1`     | Multi-page PDF export                  |
| `svelte ^5`                 | Plugin UI framework                    |
| `vite ^6`                   | Plugin bundler                         |
| `vite-plugin-singlefile ^2` | Inlines UI into single HTML file       |

## Key Documents

- [Architecture — Root](./architecture-root.md)
- [Architecture — Plugin](./architecture-plugin.md)
- [Architecture — NPM](./architecture-npm.md)
- [Source Tree Analysis](./source-tree-analysis.md)
- [Integration Architecture](./integration-architecture.md)
- [API Contracts — Root](./api-contracts-root.md)
- [API Contracts — Plugin](./api-contracts-plugin.md)
- [Data Models — Root](./data-models-root.md)
- [Development Guide](./development-guide.md)
