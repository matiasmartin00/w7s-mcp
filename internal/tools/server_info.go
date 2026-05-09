package tools

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var startTime = time.Now()

func RegisterServerInfo(s *server.MCPServer) {
	tool := mcp.NewTool("server_info",
		mcp.WithDescription("Return runtime metadata about this MCP server"),
	)

	s.AddTool(tool, serverInfoHandler)
}

func serverInfoHandler(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info := fmt.Sprintf("name=w7s-mcp version=0.1.0 uptime=%s", time.Since(startTime).Round(time.Second))
	slog.Info("server_info called", "info", info)
	return mcp.NewToolResultText(info), nil
}
