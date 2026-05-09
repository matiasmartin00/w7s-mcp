package tools_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

func TestServerInfo_ReturnsMetadata(t *testing.T) {
	client := startClient(t, newServerWith(tools.RegisterServerInfo))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "server_info"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "w7s-mcp") {
		t.Errorf("expected 'w7s-mcp' in output, got: %s", text)
	}
	if !strings.Contains(text, "uptime") {
		t.Errorf("expected 'uptime' in output, got: %s", text)
	}
}
