# Project Documentation Index

## Project Overview

- **Project**: figma-mcp-android
- **Type**: Multi-part monorepo (3 parts)
- **Language**: Go + TypeScript
- **Architecture**: MCP Server (Leader/Follower) + Figma Plugin + NPM CLI wrapper

## Quick Reference

### Root Go App
- **Role**: MCP server, WebSocket bridge, leader/follower election
- **Tech**: Go 1.26.1, mcp-go, coder/websocket, pdfcpu
- **Root**: repository root

### Plugin
- **Role**: Figma extension — executes Figma API commands
- **Tech**: Svelte 5, Vite 6, TypeScript, Bun
- **Root**: `plugin/`

### NPM Wrapper
- **Role**: Cross-platform binary distribution
- **Tech**: Node.js (CommonJS)
- **Root**: `npm/`

## Documentation

- [Project Overview](./project-overview.md) — Executive summary, architecture diagram, tech stack
- [Architecture — Root](./architecture-root.md) — Go server internals, Leader/Follower/Election, Bridge, tools, prompts
- [Architecture — Plugin](./architecture-plugin.md) — Dual build, handler chain, serializers, WebSocket-postMessage bridge
- [Architecture — NPM](./architecture-npm.md) — Platform detection, binary launcher
- [Source Tree Analysis](./source-tree-analysis.md) — Full directory tree, entry points, file sizes
- [Integration Architecture](./integration-architecture.md) — End-to-end communication, security model
- [API Contracts — Root](./api-contracts-root.md) — Complete MCP tool catalog (70+ tools)
- [API Contracts — Plugin](./api-contracts-plugin.md) — WebSocket protocol, postMessage contracts, handler dispatch
- [Data Models — Root](./data-models-root.md) — Go type definitions, data flow
- [Component Inventory — Plugin](./component-inventory-plugin.md) — App.svelte breakdown
- [State Management — Plugin](./state-management-plugin.md) — Reactive state, data flow diagrams
- [UI Components — Plugin](./ui-components-plugin.md) — Build pipeline, styling
- [Development Guide](./development-guide.md) — Setup, commands, contributor notes

## Getting Started

Refer to the [Development Guide](./development-guide.md) for setup and run instructions.
