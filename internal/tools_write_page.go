package internal

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerWritePageTools(s *server.MCPServer, node *Node) {
	s.AddTool(mcp.NewTool("add_page",
		mcp.WithDescription("Add a new page to the Figma document."),
		mcp.WithString("name", mcp.Description("Name for the new page (default 'Page')")),
		mcp.WithNumber("index", mcp.Description("Position index to insert the page (0 = first). Defaults to last position.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if name, ok := req.GetArguments()["name"].(string); ok && name != "" {
			params["name"] = name
		}
		if idx, ok := req.GetArguments()["index"].(float64); ok {
			params["index"] = idx
		}
		resp, err := node.Send(ctx, "add_page", nil, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("delete_page",
		mcp.WithDescription("Delete a page from the Figma document. Cannot delete the only remaining page."),
		mcp.WithString("pageId", mcp.Description("Page node ID in colon format e.g. '0:2'")),
		mcp.WithString("pageName", mcp.Description("Exact page name to delete (alternative to pageId)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if id, ok := req.GetArguments()["pageId"].(string); ok && id != "" {
			params["pageId"] = id
		}
		if name, ok := req.GetArguments()["pageName"].(string); ok && name != "" {
			params["pageName"] = name
		}
		resp, err := node.Send(ctx, "delete_page", nil, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("rename_page",
		mcp.WithDescription("Rename an existing page in the Figma document."),
		mcp.WithString("pageId", mcp.Description("Page node ID in colon format e.g. '0:2'")),
		mcp.WithString("pageName", mcp.Description("Current page name to find (alternative to pageId)")),
		mcp.WithString("newName",
			mcp.Required(),
			mcp.Description("New name for the page"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if id, ok := req.GetArguments()["pageId"].(string); ok && id != "" {
			params["pageId"] = id
		}
		if name, ok := req.GetArguments()["pageName"].(string); ok && name != "" {
			params["pageName"] = name
		}
		if newName, ok := req.GetArguments()["newName"].(string); ok {
			params["newName"] = newName
		}
		resp, err := node.Send(ctx, "rename_page", nil, params)
		return renderResponse(resp, err)
	})
}
