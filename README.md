# Figma MCP Android

***English** · [Tiếng Việt](README.vi.md)*

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

#### Claude Code / Claude Desktop / VS Code / Cursor / Antigravity

Same server entry for all of them; only the file differs:

- Claude Code — `.mcp.json` in your project root
- Claude Desktop — `claude_desktop_config.json`
- Cursor — `.cursor/mcp.json`
- VS Code — `.vscode/mcp.json` (VS Code names the top-level key `servers`, not `mcpServers`)
- Antigravity — MCP settings panel (MCP Store -> **View raw config**), backed by `~/.gemini/config/mcp_config.json`, or `.agents/mcp_config.json` for a single workspace

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

#### Codex CLI

`~/.codex/config.toml`:

```toml
[mcp_servers.figma-mcp-android]
command = "npx"
args = ["-y", "@impeterwayne/figma-mcp-android@latest"]
```

#### OpenCode
`opencode.json`
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

### 2. Load the Figma plugin in Figma Desktop

This server depends on the local plugin bridge.

1. Download the latest `figma-plugin.zip` from the [releases](https://github.com/impeterwayne/figma_mcp_android/releases) page.
2. Extract the downloaded ZIP file to a local directory on your machine.
3. Open Figma Desktop.
4. Go to **Plugins** -> **Development** -> **Import plugin from manifest...**.
5. Select the `manifest.json` file inside the extracted directory.
6. Run the plugin in the file you want to inspect.
7. Start your MCP client and connect through the configured `figma-mcp-android` server.

## Available tools

21 tools — all read-only against your Figma document, plus one local file converter.

### Document and node inspection

| Tool | What it does |
|------|--------------|
| `get_design_context` | Depth-limited, token-efficient tree of the selection or page. **Start here** when exploring a large file. |
| `get_document` | Full node tree of the *current page* (not the whole file). Recursive and potentially very large. |
| `get_pages` | Lists every page with ID and name — the lightweight alternative to `get_document`. |
| `get_metadata` | File name, page list, and current page. |
| `get_selection` | Nodes currently selected in Figma; empty array when nothing is selected. |
| `get_node` | One node by ID. IDs use colon format (`4029:12345`), never hyphens. |
| `get_nodes_info` | Several nodes by ID in one round-trip — prefer over repeated `get_node` calls. |
| `search_nodes` | Finds nodes by name substring and/or type within a subtree. |
| `scan_nodes_by_types` | Finds every node of the given types in a subtree, regardless of name. |
| `scan_text_nodes` | Returns the copy from all TEXT nodes in a subtree. |
| `get_reactions` | Prototype reactions on a node — trigger plus actions. |
| `get_viewport` | Scroll center, zoom level, and visible bounds. |
| `get_fonts` | Fonts used on the current page, sorted by usage frequency. |

### Styles, variables, components, annotations, tokens

| Tool | What it does |
|------|--------------|
| `get_styles` | Local paint, text, effect, and grid styles with IDs and properties. |
| `get_variable_defs` | Local variable definitions — collections, modes, values (Figma's design tokens). |
| `get_local_components` | Components defined in the current file. |
| `get_annotations` | Dev-mode annotations, document-wide or scoped to a node. |
| `export_tokens` | Exports variables and paint styles as JSON or CSS custom properties. |

### Exports and Android helper

| Tool | What it does |
|------|--------------|
| `get_screenshot` | Exports nodes as base64 image data held in memory. |
| `save_screenshots` | Exports nodes straight to disk, returning path, size, and dimensions — no base64 in the response. |
| `convert_svg_to_android_drawable` | Converts SVG files on disk into Android VectorDrawable XML, one asset or a batch per call. |

## License

MIT. See [LICENSE](LICENSE).
