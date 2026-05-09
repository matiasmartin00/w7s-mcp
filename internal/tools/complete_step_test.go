package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

// registerAllThree registers start_run, get_next_step, and complete_step.
func registerAllThree(s *mcpgoserver.MCPServer, st *store.Store) {
	tools.RegisterStartRun(s, st)
	tools.RegisterGetNextStep(s, st)
	tools.RegisterCompleteStep(s, st)
}

// callCompleteStep is a test helper that calls complete_step and returns the response.
func callCompleteStep(t *testing.T, client interface {
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}, runID, stepID, output string) (*mcp.CallToolResult, domain.CompleteStepResponse) {
	t.Helper()
	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "complete_step",
			Arguments: map[string]any{
				"run_id":  runID,
				"step_id": stepID,
				"output":  output,
			},
		},
	})
	if err != nil {
		t.Fatalf("complete_step call failed: %v", err)
	}
	var resp domain.CompleteStepResponse
	if !result.IsError {
		text := result.Content[0].(mcp.TextContent).Text
		if err := json.Unmarshal([]byte(text), &resp); err != nil {
			t.Fatalf("parse complete_step response: %v", err)
		}
	}
	return result, resp
}

func TestCompleteStep_StepTransitionsToDone(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllThree(srv, st)
	})
	client := startClient(t, s)

	// Start a run with multi_step workflow (has extract patterns).
	runID := startRunAndGetID(t, client, loaderTestdataPath("multi_step.yaml"), "build feature X")

	// Mark the first step as running so complete_step is valid.
	if err := st.UpdateStepStatus(context.Background(), runID, "explore", domain.StepStatusRunning, 1, nil); err != nil {
		t.Fatalf("set step running: %v", err)
	}

	output := "STATUS: done\nSCOPE: authentication module"
	result, resp := callCompleteStep(t, client, runID, "explore", output)

	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	if resp.Status != "step_done" {
		t.Errorf("expected status 'step_done', got %q", resp.Status)
	}
	if resp.StepID != "explore" {
		t.Errorf("expected step_id 'explore', got %q", resp.StepID)
	}

	// Verify step is actually done in store.
	step, err := st.GetStep(context.Background(), runID, "explore")
	if err != nil {
		t.Fatalf("get step: %v", err)
	}
	if step.Status != domain.StepStatusDone {
		t.Errorf("expected step status 'done', got %q", step.Status)
	}
	if step.Output == nil || *step.Output != output {
		t.Errorf("expected step output to be persisted, got %v", step.Output)
	}
}

func TestCompleteStep_VariablesExtracted(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllThree(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("multi_step.yaml"), "build feature X")

	if err := st.UpdateStepStatus(context.Background(), runID, "explore", domain.StepStatusRunning, 1, nil); err != nil {
		t.Fatalf("set step running: %v", err)
	}

	output := "STATUS: done\nSCOPE: authentication module"
	_, resp := callCompleteStep(t, client, runID, "explore", output)

	// Check extracted variables in response.
	if resp.VariablesExtracted == nil {
		t.Fatal("expected variables_extracted map, got nil")
	}
	if val, ok := resp.VariablesExtracted["scope"]; !ok {
		t.Error("expected 'scope' in variables_extracted")
	} else if val != "authentication module" {
		t.Errorf("expected scope='authentication module', got %q", val)
	}

	// Check variables persisted in store.
	vars, err := st.GetVariablesByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("get variables: %v", err)
	}
	if vars["scope"] != "authentication module" {
		t.Errorf("expected scope variable persisted, got %q", vars["scope"])
	}
}

func TestCompleteStep_MissingMatchDoesNotFail(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllThree(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("multi_step.yaml"), "build feature X")

	if err := st.UpdateStepStatus(context.Background(), runID, "explore", domain.StepStatusRunning, 1, nil); err != nil {
		t.Fatalf("set step running: %v", err)
	}

	// Output does NOT contain SCOPE — extraction should silently skip.
	output := "STATUS: done\nNo scope information here."
	result, resp := callCompleteStep(t, client, runID, "explore", output)

	if result.IsError {
		t.Fatalf("expected success even with missing match, got error: %v", result.Content)
	}
	if resp.Status != "step_done" {
		t.Errorf("expected status 'step_done', got %q", resp.Status)
	}
	// 'scope' should NOT be in extracted variables.
	if _, ok := resp.VariablesExtracted["scope"]; ok {
		t.Error("expected 'scope' to be absent when pattern did not match")
	}

	// Verify step is still done.
	step, err := st.GetStep(context.Background(), runID, "explore")
	if err != nil {
		t.Fatalf("get step: %v", err)
	}
	if step.Status != domain.StepStatusDone {
		t.Errorf("expected step status 'done', got %q", step.Status)
	}
}

func TestCompleteStep_ResponseHasMessage(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerAllThree))
	runID := startRunAndGetID(t, client, loaderTestdataPath("multi_step.yaml"), "task")

	// Use store directly to mark step as running.
	// We need the store from newServerWithStore — use a separate setup.
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllThree(srv, st)
	})
	c2 := startClient(t, s)
	runID = startRunAndGetID(t, c2, loaderTestdataPath("multi_step.yaml"), "task")

	if err := st.UpdateStepStatus(context.Background(), runID, "explore", domain.StepStatusRunning, 1, nil); err != nil {
		t.Fatalf("set step running: %v", err)
	}

	_, resp := callCompleteStep(t, c2, runID, "explore", "STATUS: done")
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}

	_ = client // suppress unused
}

func TestCompleteStep_RunNotFound(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterCompleteStep))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "complete_step",
			Arguments: map[string]any{
				"run_id":  "nonexistent-run",
				"step_id": "step1",
				"output":  "some output",
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent run, got success")
	}
}

func TestCompleteStep_StepNotFound(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterCompleteStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "task")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "complete_step",
			Arguments: map[string]any{
				"run_id":  runID,
				"step_id": "nonexistent-step",
				"output":  "some output",
			},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent step, got success")
	}
}

func TestCompleteStep_MissingArguments(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterCompleteStep))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "complete_step",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing arguments, got success")
	}
}
