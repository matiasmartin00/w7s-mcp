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

// registerAllFour registers start_run, get_next_step, complete_step, and fail_step.
func registerAllFour(s *mcpgoserver.MCPServer, st *store.Store) {
	tools.RegisterStartRun(s, st)
	tools.RegisterGetNextStep(s, st)
	tools.RegisterCompleteStep(s, st)
	tools.RegisterFailStep(s, st)
}

// callFailStep is a test helper that calls fail_step and returns the response.
func callFailStep(t *testing.T, client interface {
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}, runID, stepID, reason string) (*mcp.CallToolResult, domain.FailStepResponse) {
	t.Helper()
	args := map[string]any{
		"run_id":  runID,
		"step_id": stepID,
	}
	if reason != "" {
		args["reason"] = reason
	}
	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "fail_step",
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("fail_step call failed: %v", err)
	}
	var resp domain.FailStepResponse
	if !result.IsError {
		text := result.Content[0].(mcp.TextContent).Text
		if err := json.Unmarshal([]byte(text), &resp); err != nil {
			t.Fatalf("parse fail_step response: %v", err)
		}
	}
	return result, resp
}

// setupRetryRun starts a run with the retry_workflow and puts the verify step in running state.
func setupRetryRun(t *testing.T, st *store.Store, client interface {
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}, attempt int) string {
	t.Helper()
	runID := startRunAndGetID(t, client, loaderTestdataPath("retry_workflow.yaml"), "build it")

	// Mark implement step as done.
	if err := st.UpdateStepStatus(context.Background(), runID, "implement", domain.StepStatusDone, 1, nil); err != nil {
		t.Fatalf("set implement done: %v", err)
	}
	// Set verify step as running at the given attempt.
	if err := st.UpdateStepStatus(context.Background(), runID, "verify", domain.StepStatusRunning, attempt, nil); err != nil {
		t.Fatalf("set verify running: %v", err)
	}
	return runID
}

func TestFailStep_RetryPath(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllFour(srv, st)
	})
	client := startClient(t, s)

	runID := setupRetryRun(t, st, client, 1)

	result, resp := callFailStep(t, client, runID, "verify", "output did not match expectations")

	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	if resp.Status != "retry" {
		t.Errorf("expected status 'retry', got %q", resp.Status)
	}
	if resp.StepID != "verify" {
		t.Errorf("expected step_id 'verify', got %q", resp.StepID)
	}
	if resp.RetryStep != "implement" {
		t.Errorf("expected retry_step 'implement', got %q", resp.RetryStep)
	}
	if resp.AttemptsUsed != 1 {
		t.Errorf("expected attempts_used=1, got %d", resp.AttemptsUsed)
	}
	if resp.AttemptsRemaining != 2 {
		t.Errorf("expected attempts_remaining=2, got %d", resp.AttemptsRemaining)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestFailStep_EscalationPath_RetriesExhausted(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllFour(srv, st)
	})
	client := startClient(t, s)

	// Set verify at attempt 3 (= max_retries), so it should escalate.
	runID := setupRetryRun(t, st, client, 3)

	result, resp := callFailStep(t, client, runID, "verify", "still broken")

	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	if resp.Status != "escalated" {
		t.Errorf("expected status 'escalated', got %q", resp.Status)
	}
	if resp.StepID != "verify" {
		t.Errorf("expected step_id 'verify', got %q", resp.StepID)
	}
	if resp.EscalateTo != "human" {
		t.Errorf("expected escalate_to 'human', got %q", resp.EscalateTo)
	}
	if resp.Attempts != 3 {
		t.Errorf("expected attempts=3, got %d", resp.Attempts)
	}

	// Run should now be escalated.
	run, err := st.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status != domain.RunStatusEscalated {
		t.Errorf("expected run status 'escalated', got %q", run.Status)
	}
}

func TestFailStep_IntermediateStepsResetToPending(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllFour(srv, st)
	})
	client := startClient(t, s)

	runID := setupRetryRun(t, st, client, 1)

	_, resp := callFailStep(t, client, runID, "verify", "reason")

	if resp.Status != "retry" {
		t.Fatalf("expected retry status, got %q", resp.Status)
	}

	// implement step should now be pending (reset by retry logic).
	implementStep, err := st.GetStep(context.Background(), runID, "implement")
	if err != nil {
		t.Fatalf("get implement step: %v", err)
	}
	if implementStep.Status != domain.StepStatusPending {
		t.Errorf("expected implement step to be pending after retry, got %q", implementStep.Status)
	}
	// verify step should also be pending.
	verifyStep, err := st.GetStep(context.Background(), runID, "verify")
	if err != nil {
		t.Fatalf("get verify step: %v", err)
	}
	if verifyStep.Status != domain.StepStatusPending {
		t.Errorf("expected verify step to be pending after retry, got %q", verifyStep.Status)
	}
}

func TestFailStep_NoOnFail_Escalates(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterFailStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "some task")
	// Mark step1 as running.
	if err := st.UpdateStepStatus(context.Background(), runID, "step1", domain.StepStatusRunning, 1, nil); err != nil {
		t.Fatalf("set step running: %v", err)
	}

	result, resp := callFailStep(t, client, runID, "step1", "")

	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	if resp.Status != "escalated" {
		t.Errorf("expected status 'escalated', got %q", resp.Status)
	}

	// Verify run is escalated.
	run, err := st.GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status != domain.RunStatusEscalated {
		t.Errorf("expected run status 'escalated', got %q", run.Status)
	}
}

func TestFailStep_FeedbackVarStoredOnRetry(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		registerAllFour(srv, st)
	})
	client := startClient(t, s)

	runID := setupRetryRun(t, st, client, 1)
	reason := "the output was missing required fields"

	_, resp := callFailStep(t, client, runID, "verify", reason)

	if resp.Status != "retry" {
		t.Fatalf("expected retry, got %q", resp.Status)
	}

	// feedback_var should be stored as "feedback" variable.
	vars, err := st.GetVariablesByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("get variables: %v", err)
	}
	if vars["feedback"] != reason {
		t.Errorf("expected feedback variable %q, got %q", reason, vars["feedback"])
	}
}

func TestFailStep_RunNotFound(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterFailStep))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "fail_step",
			Arguments: map[string]any{
				"run_id":  "nonexistent-run",
				"step_id": "step1",
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

func TestFailStep_StepNotFound(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterFailStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "task")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "fail_step",
			Arguments: map[string]any{
				"run_id":  runID,
				"step_id": "nonexistent-step",
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

func TestFailStep_MissingArguments(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterFailStep))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "fail_step",
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
