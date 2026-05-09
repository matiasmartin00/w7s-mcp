package tools_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/store"
	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

// newServerWithStore creates a minimal MCPServer with an ephemeral SQLite store
// and one or more tool registration functions applied.
func newServerWithStore(t *testing.T, registers ...func(*mcpgoserver.MCPServer, *store.Store)) *mcpgoserver.MCPServer {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	s := mcpgoserver.NewMCPServer("test", "0.0.0", mcpgoserver.WithToolCapabilities(true))
	for _, reg := range registers {
		reg(s, st)
	}
	return s
}

func TestStartRun_ValidWorkflow(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterStartRun))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_id": loaderTestdataPath("valid.yaml"),
				"task":        "Implement the feature",
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "Test Workflow") {
		t.Errorf("expected workflow name 'Test Workflow' in response, got: %s", text)
	}
	if !strings.Contains(text, "run_id") {
		t.Errorf("expected 'run_id' in response, got: %s", text)
	}
}

func TestStartRun_InvalidWorkflowID(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterStartRun))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_id": "/nonexistent/path/workflow.yaml",
				"task":        "some task",
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Errorf("expected error for missing workflow, got success: %v", result.Content)
	}
}

func TestStartRun_MissingArguments(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterStartRun))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "start_run",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Errorf("expected error when arguments are missing, got success: %v", result.Content)
	}
}
