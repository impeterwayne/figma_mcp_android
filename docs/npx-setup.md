# MCP Client Setup via NPX

This guide explains how to configure various AI clients to use the `figma-mcp-android` MCP server via `npx`.

The server is distributed via NPM under the package `@impeterwayne/figma-mcp-android`. Using `npx` allows clients to automatically download and run the appropriate binary for your platform.

## Claude Desktop

To use this server with Claude Desktop, you need to edit your Claude Desktop configuration file.

1. Open your configuration file depending on your OS:
   - **MacOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

2. Add the following to your `mcpServers` object:

```json
{
  "mcpServers": {
    "figma-mcp-android": {
      "command": "npx",
      "args": [
        "-y",
        "@impeterwayne/figma-mcp-android"
      ]
    }
  }
}
```

3. Restart Claude Desktop for the changes to take effect.

## Cursor

To add the server in Cursor:

1. Open **Cursor Settings**.
2. Navigate to **Features** > **MCP Servers**.
3. Click **+ Add New MCP Server**.
4. Configure with the following settings:
   - **Type**: `command`
   - **Name**: `figma-mcp-android`
   - **Command**: `npx -y @impeterwayne/figma-mcp-android`
5. Click **Save** and ensure the status shows as green/connected.

## Windsurf

To add the server in Windsurf:

1. Open your `~/.codeium/windsurf/mcp_config.json` file.
2. Add the configuration:

```json
{
  "mcpServers": {
    "figma-mcp-android": {
      "command": "npx",
      "args": [
        "-y",
        "@impeterwayne/figma-mcp-android"
      ]
    }
  }
}
```

3. Restart Windsurf.

## Opencode

To add the server in Opencode:

1. Open your `opencode.json` configuration file in your project root (or global config at `~/.config/opencode/opencode.json`).
2. Add the following configuration:

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

3. Restart or reload Opencode.
## Local Development & Testing

If you want to run the server standalone from your terminal to verify it starts correctly, you can run:

```bash
npx -y @impeterwayne/figma-mcp-android
```

You should see MCP JSON-RPC protocol messages or a startup confirmation (depending on how the server logs initialized state) on standard output/error.
