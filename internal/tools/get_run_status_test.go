package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

func TestGetRunStatus_ByRunID(t *testing.T) {
	s := newServerWithStore(t, tools.RegisterStartRun, tools.RegisterGetRunStatus)
	client := startClient(t, s)

	// Start a run first.
	startResult, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_id": loaderTestdataPath("valid.yaml"),
				"task":        "do something important",
			},
		},
	})
	if err != nil {
		t.Fatalf("start_run call: %v", err)
	}
	if startResult.IsError {
		t.Fatalf("start_run error: %v", startResult.Content)
	}

	var startResp domain.StartRunResponse
	if err := json.Unmarshal([]byte(startResult.Content[0].(mcp.TextContent).Text), &startResp); err != nil {
		t.Fatalf("parse start_run response: %v", err)
	}

	// Get run status by run_id.
	statusResult, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_run_status",
			Arguments: map[string]any{
				"run_id": startResp.RunID,
			},
		},
	})
	if err != nil {
		t.Fatalf("get_run_status call: %v", err)
	}
	if statusResult.IsError {
		t.Fatalf("get_run_status error: %v", statusResult.Content)
	}

	var resp domain.GetRunStatusResponse
	if err := json.Unmarshal([]byte(statusResult.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatalf("parse get_run_status response: %v", err)
	}

	if resp.RunID != startResp.RunID {
		t.Errorf("expected run_id %s, got %s", startResp.RunID, resp.RunID)
	}
	if resp.Status != domain.RunStatusRunning {
		t.Errorf("expected status 'running', got %s", resp.Status)
	}
	if len(resp.Steps) == 0 {
		t.Error("expected non-empty steps")
	}
	if resp.Variables["task"] != "do something important" {
		t.Errorf("expected task variable 'do something important', got %q", resp.Variables["task"])
	}
}

func TestGetRunStatus_ByWorkflowID(t *testing.T) {
	s := newServerWithStore(t, tools.RegisterStartRun, tools.RegisterGetRunStatus)
	client := startClient(t, s)

	wfPath := loaderTestdataPath("valid.yaml")

	// Start a run.
	startResult, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_id": wfPath,
				"task":        "workflow id lookup",
			},
		},
	})
	if err != nil {
		t.Fatalf("start_run call: %v", err)
	}
	if startResult.IsError {
		t.Fatalf("start_run error: %v", startResult.Content)
	}

	// Get run status by workflow_id only.
	statusResult, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_run_status",
			Arguments: map[string]any{
				"workflow_id": wfPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("get_run_status call: %v", err)
	}
	if statusResult.IsError {
		t.Fatalf("get_run_status error: %v", statusResult.Content)
	}

	var resp domain.GetRunStatusResponse
	if err := json.Unmarshal([]byte(statusResult.Content[0].(mcp.TextContent).Text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.WorkflowID != wfPath {
		t.Errorf("expected workflow_id %q, got %q", wfPath, resp.WorkflowID)
	}
	if resp.Status != domain.RunStatusRunning {
		t.Errorf("expected status running, got %s", resp.Status)
	}
}

func TestGetRunStatus_UnknownRunID(t *testing.T) {
	s := newServerWithStore(t, tools.RegisterGetRunStatus)
	client := startClient(t, s)

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_run_status",
			Arguments: map[string]any{
				"run_id": "00000000-0000-0000-0000-000000000000",
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for unknown run_id")
	}
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "run not found") {
		t.Errorf("expected 'run not found' in error, got: %s", text)
	}
}

func TestGetRunStatus_MissingBothIDs(t *testing.T) {
	s := newServerWithStore(t, tools.RegisterGetRunStatus)
	client := startClient(t, s)

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_run_status",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when both run_id and workflow_id are missing")
	}
}
