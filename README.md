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

Warning:
If you bind the server to a non-loopback address such as `0.0.0.0`, it becomes reachable from the network and the current server does not provide authentication.

### 2. Configure your AI client

#### Claude Code

Add the server with `@latest` if you want the latest published npm version at install time:

```bash
claude mcp add figma-mcp-android -- npx -y @impeterwayne/figma-mcp-android@latest
```

Or pin to whatever version you prefer:

```bash
claude mcp add figma-mcp-android -- npx -y @impeterwayne/figma-mcp-android
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

If your client supports environment variables or custom command wrappers, keep the process local unless you fully trust the network you expose it to.

### 3. Load the Figma plugin in Figma Desktop

This server depends on the local plugin bridge.

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

### Server

This repo's server is written in Go and lives under `cmd/` and `internal/`.

Typical development requirements:

- Go `1.26.1+`
- Node `>=18`
- Figma Desktop
- Bun for plugin work

Key pieces:

- `cmd/figma-mcp-android/main.go` starts the stdio MCP server
- `internal/` contains bridge, tool registration, exports, and Android drawable conversion logic
- `plugin/` contains the Figma plugin
- `npm/` contains the npm launcher package metadata

### Plugin

The Figma plugin is loaded locally from this repo via `plugin/manifest.json`. For plugin development, use Bun and the assets under `plugin/`.

## Publishing checklist

Use this as a practical release checklist for npm and GitHub distribution.

- Confirm `README.md` matches the actual tool surface and setup flow
- Confirm package name is `@impeterwayne/figma-mcp-android`
- Confirm the MCP server key in examples is `figma-mcp-android`
- Verify the default bridge address is `127.0.0.1:1994`
- Re-check the warning about non-loopback bindings and no authentication
- Verify plugin import instructions still point to `plugin/manifest.json`
- Confirm npm package contents include the launcher and runtime assets you intend to ship
- Validate client config examples in Claude Code, `.mcp.json`, VS Code, and Cursor
- Tag and publish only when the npm package is actually ready

## Contributing

Issues and pull requests are welcome.

If you contribute:

- keep README claims aligned with the current code
- avoid adding docs that promise unreleased artifacts or unsupported tool counts
- test the plugin bridge flow with Figma Desktop when changing server or plugin behavior
- document any new tool clearly, including whether it is read-only, export-related, or writes local files

## License

MIT. See [LICENSE](LICENSE).
