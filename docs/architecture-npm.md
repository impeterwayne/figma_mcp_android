# Architecture — NPM (CLI Wrapper)

## Executive Summary

The NPM package (`@vkhanhqui/figma-mcp-go`) provides cross-platform distribution of the pre-compiled Go binary. It is the recommended way for end users to install and run the MCP server.

## Package Structure

```
npm/
├── package.json      # NPM manifest (@vkhanhqui/figma-mcp-go)
└── bin/
    ├── run.js        # Platform detection + binary launcher
    ├── darwin-amd64/  # macOS Intel binary
    ├── darwin-arm64/  # macOS Apple Silicon binary
    ├── linux-amd64/   # Linux x64 binary
    ├── linux-arm64/   # Linux ARM64 binary
    ├── windows-amd64/ # Windows x64 binary
    └── windows-arm64/ # Windows ARM64 binary
```

## How `run.js` Works

1. **Platform detection**: Maps Node.js `process.platform` and `process.arch` to Go's `GOOS`/`GOARCH`:

   | Node.js `platform` | Go `GOOS`  | Node.js `arch` | Go `GOARCH` |
   |---------------------|-----------|----------------|-------------|
   | `darwin`            | `darwin`  | `x64`          | `amd64`     |
   | `linux`             | `linux`   | `arm64`        | `arm64`     |
   | `win32`             | `windows` |                |             |

2. **Binary resolution**: Constructs path as `bin/{goos}-{goarch}/figma-mcp-go(.exe)`.

3. **Error handling**: Exits with a clear message if platform is unsupported or binary is missing.

4. **Execution**: Uses `spawnSync` with `stdio: 'inherit'` to map stdin/stdout/stderr directly to the terminal. This is critical because MCP communicates via JSON-RPC over stdio — any buffering or piping would break the protocol.

5. **Argument forwarding**: Passes `process.argv.slice(2)` to the binary, allowing `--ip` and `--port` flags to be forwarded.

## NPM Metadata

- **Package name**: `@vkhanhqui/figma-mcp-go`
- **Binary name**: `figma-mcp-go` (registered as a global `bin`)
- **Engines**: Node.js >= 18.0.0
- **License**: MIT
