# Figma MCP Android

A Go stdio MCP server that connects AI clients to a local Figma Desktop plugin over WebSocket, with no Figma API token required. It focuses on read-heavy design inspection, lightweight exports, and includes an Android SVG to VectorDrawable conversion helper runtime for Android workflows.

This repo contains two cooperating pieces:

- an MCP server for tools exposed over stdio
- a Figma Desktop plugin that talks to the server on `ws://127.0.0.1:1994` by default

## Highlights

- Read-focused bridge from AI clients to live Figma documents
- No Figma REST API key required
- Local plugin connection over WebSocket
- 21 currently exposed tools for document inspection, metadata, search, exports, screenshots, and Android asset conversion
- Design token export support
- Screenshot export helpers, including direct file saves
- Android VectorDrawable conversion via `convert_svg_to_android_drawable`
- Go server with a bundled npm launcher package for MCP clients

## Why this exists

Most Figma integrations rely on remote APIs, API tokens, or rate-limited access patterns. This project takes a different route. It bridges an MCP client to a local Figma plugin running inside Figma Desktop, so the model can inspect the document you already have open.

That makes it useful when you want to:

- inspect pages, nodes, styles, variables, components, and annotations from an AI client
- export screenshots or tokens without wiring up a separate API integration
- convert exported or generated SVG into Android VectorDrawable XML as part of the same workflow
- keep the connection local to your machine by default

## Requirements

- Node.js `>=18`
- Figma Desktop
- Go `1.26.1+` for server development
- Bun for plugin development

## Installation and setup

### 1. Start the MCP server with npx

Use the npm package name below in your MCP client config:

```bash
npx -y @impeterwayne/figma-mcp-android
```

The server key to use in MCP client configs is:

```text
figma-mcp-android
```

By default, the server listens on:

```text
127.0.0.1:1994
```

Warning: if you bind the server to a non-loopback address such as `0.0.0.0`, it becomes reachable from the network and the current server does not provide authentication.

### 2. Configure your AI client

#### Claude Code

```bash
claude mcp add figma-mcp-android -- npx -y @impeterwayne/figma-mcp-android@latest
```

#### Project `.mcp.json`

```json
{
  "mcpServers": {
    "figma-mcp-android": {
      "command": "npx",
      "args": ["-y", "@impeterwayne/figma-mcp-android"]
    }
  }
}
```

#### VS Code / Cursor style JSON

```json
{
  "mcpServers": {
    "figma-mcp-android": {
      "command": "npx",
      "args": ["-y", "@impeterwayne/figma-mcp-android@latest"]
    }
  }
}
```

### 3. Load the Figma plugin in Figma Desktop

1. Open Figma Desktop.
2. Go to Plugins, Development, Import plugin from manifest.
3. Select `plugin/manifest.json` from this repo.
4. Run the plugin in the file you want to inspect.
5. Start your MCP client and connect through the configured `figma-mcp-android` server.

The plugin manifest in this repo points to the built plugin assets under `plugin/dist/`, and the bridge defaults to `ws://127.0.0.1:1994`.

## Available tools

The current server exposes 21 tools.

### Document and node inspection

- `get_document`
- `get_pages`
- `get_metadata`
- `get_selection`
- `get_node`
- `get_nodes_info`
- `get_design_context`
- `search_nodes`
- `scan_text_nodes`
- `scan_nodes_by_types`
- `get_reactions`
- `get_viewport`
- `get_fonts`

### Styles, variables, components, annotations, tokens

- `get_styles`
- `get_variable_defs`
- `get_local_components`
- `get_annotations`
- `export_tokens`

### Exports and Android helper

- `get_screenshot`
- `save_screenshots`
- `convert_svg_to_android_drawable`

### Notes on scope

This project is primarily a read-focused Figma bridge. It is built for inspection, context gathering, screenshots, token export, and Android drawable conversion. It should not be described as full write access, and it does not expose 73 tools.

## Development

The server is written in Go and lives under `cmd/` and `internal/`. The npm launcher package lives under `npm/`.

Useful commands from the repository root:

```bash
make build-npm
make validate-npm-pack
make test-go
```

## Publishing

Build and validate the npm package before publishing:

```bash
make validate-npm-pack
cd npm
npm publish --access public
```

Because this is a scoped public package, the first publish usually needs `--access public`.

## License

MIT. See the repository `LICENSE` file.
