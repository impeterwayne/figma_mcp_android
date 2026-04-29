package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newTestServer returns an MCPServer with all tools + prompts registered
// against an Unknown-role Node (no real Figma connection).
func newTestServer(t *testing.T) (*server.MCPServer, *Node) {
	t.Helper()
	s := server.NewMCPServer("test", "0.0.1")
	node := NewNode("127.0.0.1", 19940, "test")
	RegisterTools(s, node)
	RegisterPrompts(s)
	return s, node
}

// callTool dispatches a tool call through the server's full HandleMessage path.
// With an Unknown node, every call succeeds at the MCP level but returns
// an IsError=true tool result (no Figma connection).
func callTool(t *testing.T, s *server.MCPServer, name string, args map[string]any) {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	msg := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`,
		name, argsJSON,
	)
	resp := s.HandleMessage(context.Background(), []byte(msg))
	if resp == nil {
		t.Errorf("HandleMessage returned nil for tool %q", name)
	}
}

// ── Registration smoke tests ──────────────────────────────────────────────────

func TestRegisterTools_Smoke(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1")
	RegisterTools(s, NewNode("127.0.0.1", 19940, "test"))
}

func TestRegisterPrompts_Smoke(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.1")
	RegisterPrompts(s)
}

// ── makeHandler ───────────────────────────────────────────────────────────────

func TestMakeHandler_UnknownNode(t *testing.T) {
	node := NewNode("127.0.0.1", 19940, "test")
	handler := makeHandler(node, "get_document", nil, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true when node has no Figma connection")
	}
}

// ── Read – no-param tools (all use makeHandler) ───────────────────────────────

func TestHandlers_NoParamReadTools(t *testing.T) {
	s, _ := newTestServer(t)
	noParamTools := []string{
		"get_document", "get_pages", "get_metadata", "get_selection",
		"get_viewport", "get_fonts", "get_styles", "get_variable_defs",
		"get_local_components", "get_annotations",
	}
	for _, name := range noParamTools {
		callTool(t, s, name, nil)
	}
}

// ── Read – param tools ────────────────────────────────────────────────────────

func TestHandlers_GetNode(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_node", map[string]any{"nodeId": "1:1"})
}

func TestHandlers_GetNodesInfo(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_nodes_info", map[string]any{"nodeIds": []string{"1:1", "2:2"}})
}

func TestHandlers_GetDesignContext(t *testing.T) {
	s, _ := newTestServer(t)
	// with all optional params
	callTool(t, s, "get_design_context", map[string]any{
		"depth": float64(2), "detail": "compact", "dedupe_components": true,
	})
	// with no params (defaults)
	callTool(t, s, "get_design_context", nil)
	// depth = 0 should be ignored (not passed through)
	callTool(t, s, "get_design_context", map[string]any{"depth": float64(0)})
}

func TestHandlers_SearchNodes(t *testing.T) {
	s, _ := newTestServer(t)
	// all optional params present
	callTool(t, s, "search_nodes", map[string]any{
		"query":  "button",
		"nodeId": "1:1",
		"types":  []any{"TEXT", "FRAME"},
		"limit":  float64(25),
	})
	// minimal (query only)
	callTool(t, s, "search_nodes", map[string]any{"query": "icon"})
}

func TestHandlers_ScanTextNodes(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "scan_text_nodes", map[string]any{"nodeId": "1:1"})
}

func TestHandlers_ScanNodesByTypes(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "scan_nodes_by_types", map[string]any{
		"nodeId": "1:1",
		"types":  []any{"FRAME", "COMPONENT"},
	})
}

func TestHandlers_GetReactions(t *testing.T) {
	s, _ := newTestServer(t)
	callTool(t, s, "get_reactions", map[string]any{"nodeId": "1:1"})
}

// ── Read – export tools ───────────────────────────────────────────────────────

func TestHandlers_GetScreenshot(t *testing.T) {
	s, _ := newTestServer(t)
	// with format + scale
	callTool(t, s, "get_screenshot", map[string]any{
		"nodeIds": []any{"1:1"},
		"format":  "PNG",
		"scale":   float64(2),
	})
	// no params (exports current selection)
	callTool(t, s, "get_screenshot", nil)
}

// TestHandlers_SaveScreenshots exercises executeSaveScreenshots + saveScreenshotItem.
// With Unknown node, node.Send fails → error captured in result JSON (no panic).
func TestHandlers_SaveScreenshots(t *testing.T) {
	s, _ := newTestServer(t)

	// single item – reaches saveScreenshotItem → node.Send fails → error result
	callTool(t, s, "save_screenshots", map[string]any{
		"items": []any{
			map[string]any{"nodeId": "1:1", "outputPath": "out/screen.png"},
		},
	})

	// multiple items with default format + scale
	callTool(t, s, "save_screenshots", map[string]any{
		"format": "SVG",
		"scale":  float64(1),
		"items": []any{
			map[string]any{"nodeId": "1:1", "outputPath": "out/a.svg"},
			map[string]any{"nodeId": "2:2", "outputPath": "out/b.svg", "format": "PNG"},
		},
	})

	// item with explicit per-item format + scale
	callTool(t, s, "save_screenshots", map[string]any{
		"items": []any{
			map[string]any{"nodeId": "3:3", "outputPath": "out/c.jpg", "format": "JPG", "scale": float64(2)},
		},
	})
}

