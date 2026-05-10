package prompts

import "github.com/mark3labs/mcp-go/server"

// RegisterAll registers all MCP prompts on the server.
func RegisterAll(s *server.MCPServer) {
	addReadDesignStrategy(s)
	addReactionToConnectorStrategy(s)
	addStyleAuditStrategy(s)
}
