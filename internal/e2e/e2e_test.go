// Package e2e_test contains end-to-end tests for the w7s-mcp MCP server.
// Tests are black-box: they spin up a full server with all tools registered
// and interact exclusively through MCP tool calls via an in-process client.
package e2e_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	mcpgoclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

// e2eSetup holds the client and store for an e2e test.
type e2eSetup struct {
	client *mcpgoclient.Client
	st     *store.Store
}

// newE2ESetup creates an in-memory store, registers all tools, and starts an in-process client.
func newE2ESetup(t *testing.T) *e2eSetup {
	t.Helper()

	st, err := store.Open(filepath.Join(t.TempDir(), "e2e.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := mcpgoserver.NewMCPServer("e2e-test", "0.0.0", mcpgoserver.WithToolCapabilities(true))
	tools.RegisterStartRun(s, st)
	tools.RegisterGetNextStep(s, st)
	tools.RegisterCompleteStep(s, st)
	tools.RegisterFailStep(s, st)
	tools.RegisterGetRunStatus(s, st)

	client, err := mcpgoclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("start client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	if _, err := client.Initialize(context.Background(), mcp.InitializeRequest{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	return &e2eSetup{client: client, st: st}
}

// fixtureAbsPath returns the absolute path to a file in internal/e2e/testdata.
func fixtureAbsPath(filename string) string {
	_, self, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(self), "testdata", filename)
}

// loaderFixturePath returns the absolute path to a file in internal/loader/testdata.
func loaderFixturePath(filename string) string {
	_, self, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(self), "..", "loader", "testdata", filename)
}

// callTool is a generic helper to call an MCP tool and return the raw result.
func callTool(t *testing.T, client *mcpgoclient.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

// textOf extracts the text from the first content item.
func textOf(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	return result.Content[0].(mcp.TextContent).Text
}

// mustParseJSON parses JSON text into dst or fatals.
func mustParseJSON(t *testing.T, text string, dst any) {
	t.Helper()
	if err := json.Unmarshal([]byte(text), dst); err != nil {
		t.Fatalf("parse JSON: %v\ntext: %s", err, text)
	}
}

// startRun calls start_run and returns the run_id.
func startRun(t *testing.T, client *mcpgoclient.Client, workflowPath, task string) string {
	t.Helper()
	result := callTool(t, client, "start_run", map[string]any{
		"workflow_id": workflowPath,
		"task":        task,
	})
	if result.IsError {
		t.Fatalf("start_run error: %v", textOf(t, result))
	}
	var resp domain.StartRunResponse
	mustParseJSON(t, textOf(t, result), &resp)
	if resp.RunID == "" {
		t.Fatal("start_run returned empty run_id")
	}
	return resp.RunID
}

// getNextStep calls get_next_step and returns the parsed response.
func getNextStep(t *testing.T, client *mcpgoclient.Client, runID string) domain.GetNextStepResponse {
	t.Helper()
	result := callTool(t, client, "get_next_step", map[string]any{"run_id": runID})
	if result.IsError {
		t.Fatalf("get_next_step error: %v", textOf(t, result))
	}
	var resp domain.GetNextStepResponse
	mustParseJSON(t, textOf(t, result), &resp)
	return resp
}

// completeStep calls complete_step and returns the parsed response.
func completeStep(t *testing.T, client *mcpgoclient.Client, runID, stepID, output string) domain.CompleteStepResponse {
	t.Helper()
	result := callTool(t, client, "complete_step", map[string]any{
		"run_id":  runID,
		"step_id": stepID,
		"output":  output,
	})
	if result.IsError {
		t.Fatalf("complete_step(%s) error: %v", stepID, textOf(t, result))
	}
	var resp domain.CompleteStepResponse
	mustParseJSON(t, textOf(t, result), &resp)
	return resp
}

// failStep calls fail_step and returns the parsed response.
func failStep(t *testing.T, client *mcpgoclient.Client, runID, stepID, reason string) domain.FailStepResponse {
	t.Helper()
	result := callTool(t, client, "fail_step", map[string]any{
		"run_id":  runID,
		"step_id": stepID,
		"reason":  reason,
	})
	if result.IsError {
		t.Fatalf("fail_step(%s) error: %v", stepID, textOf(t, result))
	}
	var resp domain.FailStepResponse
	mustParseJSON(t, textOf(t, result), &resp)
	return resp
}

// TestE2E_HappyPath tests the full explore → implement → verify → done flow.
func TestE2E_HappyPath(t *testing.T) {
	e := newE2ESetup(t)
	runID := startRun(t, e.client, fixtureAbsPath("feature-dev.yaml"), "implement dark mode")

	// Step 1: explore
	step := getNextStep(t, e.client, runID)
	if step.Status != "next_step" {
		t.Fatalf("expected next_step, got %q", step.Status)
	}
	if step.StepID != "explore" {
		t.Fatalf("expected step 'explore', got %q", step.StepID)
	}
	if step.Agent == nil || step.Agent.ID != "explorer" {
		t.Errorf("expected agent 'explorer', got %v", step.Agent)
	}

	exploreOut := "STATUS: done\nSCOPE: ui-components\nFILES: src/theme.ts"
	completeResp := completeStep(t, e.client, runID, "explore", exploreOut)
	if completeResp.Status != "step_done" {
		t.Errorf("explore: expected step_done, got %q", completeResp.Status)
	}
	if completeResp.VariablesExtracted["scope"] != "ui-components" {
		t.Errorf("explore: expected scope='ui-components', got %q", completeResp.VariablesExtracted["scope"])
	}
	if completeResp.VariablesExtracted["files"] != "src/theme.ts" {
		t.Errorf("explore: expected files='src/theme.ts', got %q", completeResp.VariablesExtracted["files"])
	}

	// Step 2: implement — verify prompt interpolation includes extracted vars
	step = getNextStep(t, e.client, runID)
	if step.Status != "next_step" {
		t.Fatalf("expected next_step (implement), got %q", step.Status)
	}
	if step.StepID != "implement" {
		t.Fatalf("expected step 'implement', got %q", step.StepID)
	}
	if !strings.Contains(step.Prompt, "ui-components") {
		t.Errorf("implement prompt should contain 'ui-components', got: %s", step.Prompt)
	}
	if !strings.Contains(step.Prompt, "src/theme.ts") {
		t.Errorf("implement prompt should contain 'src/theme.ts', got: %s", step.Prompt)
	}

	implOut := "STATUS: done\nCHANGES: added dark mode toggle"
	completeStep(t, e.client, runID, "implement", implOut)

	// Step 3: verify
	step = getNextStep(t, e.client, runID)
	if step.Status != "next_step" {
		t.Fatalf("expected next_step (verify), got %q", step.Status)
	}
	if step.StepID != "verify" {
		t.Fatalf("expected step 'verify', got %q", step.StepID)
	}
	if !strings.Contains(step.Prompt, "added dark mode toggle") {
		t.Errorf("verify prompt should contain changes, got: %s", step.Prompt)
	}

	completeStep(t, e.client, runID, "verify", "STATUS: done\nAll checks passed.")

	// Done
	step = getNextStep(t, e.client, runID)
	if step.Status != "done" {
		t.Errorf("expected status 'done' after all steps, got %q", step.Status)
	}
}

// TestE2E_RetryPath tests verify fail → rollback to implement → retry → done.
func TestE2E_RetryPath(t *testing.T) {
	e := newE2ESetup(t)
	runID := startRun(t, e.client, fixtureAbsPath("feature-dev.yaml"), "implement caching")

	// Complete explore
	getNextStep(t, e.client, runID)
	completeStep(t, e.client, runID, "explore", "STATUS: done\nSCOPE: cache-layer\nFILES: src/cache.ts")

	// Complete implement (first time)
	getNextStep(t, e.client, runID)
	completeStep(t, e.client, runID, "implement", "STATUS: done\nCHANGES: added LRU cache")

	// Fail verify — should retry to implement
	step := getNextStep(t, e.client, runID)
	if step.StepID != "verify" {
		t.Fatalf("expected verify step, got %q", step.StepID)
	}

	failResp := failStep(t, e.client, runID, "verify", "cache invalidation is broken")
	if failResp.Status != "retry" {
		t.Fatalf("expected retry status, got %q", failResp.Status)
	}
	if failResp.RetryStep != "implement" {
		t.Errorf("expected retry_step='implement', got %q", failResp.RetryStep)
	}

	// Should roll back to implement
	step = getNextStep(t, e.client, runID)
	if step.Status != "next_step" {
		t.Fatalf("expected next_step after retry, got %q", step.Status)
	}
	if step.StepID != "implement" {
		t.Fatalf("expected implement step after retry, got %q", step.StepID)
	}

	// Complete implement (second time) and verify
	completeStep(t, e.client, runID, "implement", "STATUS: done\nCHANGES: fixed cache invalidation")
	step = getNextStep(t, e.client, runID)
	if step.StepID != "verify" {
		t.Fatalf("expected verify step, got %q", step.StepID)
	}
	completeStep(t, e.client, runID, "verify", "STATUS: done\nAll good.")

	// Done
	step = getNextStep(t, e.client, runID)
	if step.Status != "done" {
		t.Errorf("expected done after recovery, got %q", step.Status)
	}
}

// TestE2E_EscalationPath tests that max retries being exhausted triggers escalation.
// The retry counter accumulates per get_next_step call (attempts) not per retry cycle.
// retry_workflow.yaml has max_retries=3 for verify. We build up 3 attempts by calling
// get_next_step 3 times on the same verify step (without resetting), then fail it.
func TestE2E_EscalationPath(t *testing.T) {
	e := newE2ESetup(t)
	// Use retry_workflow.yaml: verify has max_retries=3
	runID := startRun(t, e.client, loaderFixturePath("retry_workflow.yaml"), "build auth")

	// Complete implement
	getNextStep(t, e.client, runID)
	completeStep(t, e.client, runID, "implement", "STATUS: done\nimplemented auth")
	if err := e.st.SetVariable(context.Background(), domain.Variable{
		RunID: runID,
		Key:   "feedback",
		Value: "seed feedback for first verify attempt",
	}); err != nil {
		t.Fatalf("seed feedback variable: %v", err)
	}

	// Call get_next_step 3 times on verify to accumulate attempts to 3
	// Each call increments the attempt counter on the running step.
	var step domain.GetNextStepResponse
	for i := 1; i <= 3; i++ {
		step = getNextStep(t, e.client, runID)
		if step.StepID != "verify" {
			t.Fatalf("call %d: expected verify step, got %q", i, step.StepID)
		}
		if step.Attempt != i {
			t.Fatalf("call %d: expected attempt=%d, got %d", i, i, step.Attempt)
		}
	}

	// Now fail verify — attempt=3 == max_retries=3, so should escalate
	resp := failStep(t, e.client, runID, "verify", "persistently broken")
	if resp.Status != "escalated" {
		t.Fatalf("expected escalated after max retries, got %q", resp.Status)
	}
	if resp.EscalateTo != "human" {
		t.Errorf("expected escalate_to='human', got %q", resp.EscalateTo)
	}
	if resp.Attempts != 3 {
		t.Errorf("expected attempts=3, got %d", resp.Attempts)
	}

	// get_next_step should return escalated status
	step = getNextStep(t, e.client, runID)
	if step.Status != "escalated" {
		t.Errorf("expected run status 'escalated', got %q", step.Status)
	}
}

// TestE2E_InvalidWorkflow tests start_run with a nonexistent workflow_id.
func TestE2E_InvalidWorkflow(t *testing.T) {
	e := newE2ESetup(t)
	result := callTool(t, e.client, "start_run", map[string]any{
		"workflow_id": "/nonexistent/path/to/workflow.yaml",
		"task":        "some task",
	})
	if !result.IsError {
		t.Errorf("expected error for nonexistent workflow, got success: %v", textOf(t, result))
	}
}

// TestE2E_InvalidToolInput tests get_next_step with an empty run_id.
func TestE2E_InvalidToolInput(t *testing.T) {
	e := newE2ESetup(t)
	result := callTool(t, e.client, "get_next_step", map[string]any{
		"run_id": "",
	})
	if !result.IsError {
		t.Errorf("expected error for empty run_id, got success: %v", textOf(t, result))
	}
}
