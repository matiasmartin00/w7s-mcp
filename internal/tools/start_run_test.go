package tools_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

func TestStartRun_ValidWorkflow(t *testing.T) {
	client := startClient(t, newServerWith(tools.RegisterStartRun))

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
	client := startClient(t, newServerWith(tools.RegisterStartRun))

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
	client := startClient(t, newServerWith(tools.RegisterStartRun))

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
