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

## Requirements

- Node.js `>=18`
- Figma Desktop
- Go `1.26.1+` for server development
- Bun for plugin development

## Installation and setup

### 1. Configure your AI client

#### Project `opencode.json`

```json
{
    "$schema": "https://opencode.ai/config.json",
    "mcp": {
        "figma-mcp-android": {
            "type": "local",
            "command": [
                "npx",
                "-y",
                "@impeterwayne/figma-mcp-android@latest"
            ],
            "enabled": true
        }
    }
}
```

#### VS Code / Cursor / Antigravity style JSON

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

### 2. Load the Figma plugin in Figma Desktop

This server depends on the local plugin bridge.

1. Download the latest `figma-plugin.zip` from the [releases](https://github.com/impeterwayne/figma-mcp-android/releases) page.
2. Extract the downloaded ZIP file to a local directory on your machine.
3. Open Figma Desktop.
4. Go to **Plugins** -> **Development** -> **Import plugin from manifest...**.
5. Select the `manifest.json` file inside the extracted directory.
6. Run the plugin in the file you want to inspect.
7. Start your MCP client and connect through the configured `figma-mcp-android` server.

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

## License

MIT. See [LICENSE](LICENSE).
