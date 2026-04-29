# Data Models — Root (Backend)

## Overview

This document covers the primary data structures used by the Go backend for MCP tool execution, WebSocket communication, and multi-instance coordination.

## Core Types (`internal/types.go`)

### `BridgeRequest`

Sent from the Go server to the Figma Plugin over WebSocket.

```go
type BridgeRequest struct {
    Type      string                 `json:"type"`       // Tool/command name (e.g. "create_rectangle")
    RequestID string                 `json:"requestId"`  // Correlation ID for matching responses
    NodeIDs   []string               `json:"nodeIds,omitempty"`
    Params    map[string]interface{} `json:"params,omitempty"`
}
```

### `BridgeResponse`

Received from the Figma Plugin over WebSocket.

```go
type BridgeResponse struct {
    Type      string      `json:"type"`
    RequestID string      `json:"requestId"`
    Data      interface{} `json:"data,omitempty"`      // Successful payload
    Error     string      `json:"error,omitempty"`     // Present if the command failed
    Progress  int         `json:"progress,omitempty"`  // 1–100 for long-running operations
    Message   string      `json:"message,omitempty"`   // Human-readable progress description
}
```

**Note**: When `Progress > 0`, the response is a progress update — the bridge extends the timeout but does not resolve the pending request.

### `RPCRequest`

Wire format for **Follower → Leader** HTTP calls to `/rpc`.

```go
type RPCRequest struct {
    Tool    string                 `json:"tool"`
    NodeIDs []string               `json:"nodeIds,omitempty"`
    Params  map[string]interface{} `json:"params,omitempty"`
}
```

### `RPCResponse`

Returned by the Leader's `/rpc` endpoint.

```go
type RPCResponse struct {
    Data  interface{} `json:"data,omitempty"`
    Error string      `json:"error,omitempty"`
}
```

### `Role`

Represents the current role of a server process in the multi-instance system.

```go
type Role int

const (
    RoleUnknown  Role = 0
    RoleLeader   Role = 1
    RoleFollower Role = 2
)
```

## Internal Types (`internal/tools.go`)

### `saveItem`

Represents a node to capture and write to disk in `save_screenshots`.

```go
type saveItem struct {
    NodeID     string  `json:"nodeId"`
    OutputPath string  `json:"outputPath"`
    Format     string  `json:"format,omitempty"`   // PNG, SVG, JPG, PDF
    Scale      float64 `json:"scale,omitempty"`
}
```

### `screenshotExport`

Represents an exported screenshot returned by the plugin.

```go
type screenshotExport struct {
    NodeID   string  `json:"nodeId"`
    NodeName string  `json:"nodeName"`
    Base64   string  `json:"base64"`
    Width    float64 `json:"width"`
    Height   float64 `json:"height"`
}
```

### `saveResult`

Result of a single `save_screenshots` item, returned in the aggregated response.

```go
type saveResult struct {
    Index        int     `json:"index"`
    NodeID       string  `json:"nodeId"`
    NodeName     string  `json:"nodeName,omitempty"`
    OutputPath   string  `json:"outputPath"`
    Format       string  `json:"format,omitempty"`
    Width        float64 `json:"width,omitempty"`
    Height       float64 `json:"height,omitempty"`
    BytesWritten int     `json:"bytesWritten,omitempty"`
    Success      bool    `json:"success"`
    Error        string  `json:"error,omitempty"`
}
```

## Bridge Internals (`internal/bridge.go`)

### `pendingEntry`

Tracks an in-flight request waiting for a WebSocket response.

```go
type pendingEntry struct {
    ch    chan BridgeResponse
    timer *time.Timer
    once  sync.Once   // Guards against concurrent timeout + response
}
```

### `Bridge`

Manages the single WebSocket connection to the Figma Plugin:

- `conn` — Current WebSocket connection (replaced on reconnect).
- `pending` — Map of `requestId → pendingEntry` for correlating responses.
- `counter` — Atomic counter for generating `req-HHMMSS-N` format IDs.
- `wmu` — Write mutex (the websocket library does not support concurrent writes).

## Data Flow

```
LLM → MCP (JSON-RPC/stdio) → Node.Send()
    → Leader.Bridge.Send()  [if Leader]
    → Follower.Send()       [if Follower] → HTTP POST /rpc → Leader.Bridge.Send()
    → WebSocket → Plugin → Figma API → Response → WebSocket → Bridge.readLoop() → channel → result
```
