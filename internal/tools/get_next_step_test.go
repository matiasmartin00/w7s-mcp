package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
	"github.com/matiasmartin00/w7s-mcp/internal/tools"
)

func registerBoth(s *mcpgoserver.MCPServer, st *store.Store) {
	tools.RegisterStartRun(s, st)
	tools.RegisterGetNextStep(s, st)
}

// startRunAndGetID calls start_run and returns the run_id.
func startRunAndGetID(t *testing.T, client interface {
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}, workflowPath, task string) string {
	t.Helper()
	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "start_run",
			Arguments: map[string]any{
				"workflow_id": workflowPath,
				"task":        task,
			},
		},
	})
	if err != nil {
		t.Fatalf("start_run call failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("start_run returned error: %v", result.Content)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.StartRunResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse start_run response: %v", err)
	}
	if resp.RunID == "" {
		t.Fatal("start_run returned empty run_id")
	}
	return resp.RunID
}

func TestGetNextStep_FirstStep(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerBoth))
	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "my task")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.Status != "next_step" {
		t.Errorf("expected status 'next_step', got %q", resp.Status)
	}
	if resp.StepID != "step1" {
		t.Errorf("expected step_id 'step1', got %q", resp.StepID)
	}
	if resp.Attempt != 1 {
		t.Errorf("expected attempt 1, got %d", resp.Attempt)
	}
	if resp.Agent == nil {
		t.Fatal("expected agent info, got nil")
	}
	if resp.Agent.ID != "worker" {
		t.Errorf("expected agent id 'worker', got %q", resp.Agent.ID)
	}
	if resp.Expects == "" {
		t.Error("expected non-empty expects")
	}
	if resp.Instruction == "" {
		t.Error("expected non-empty instruction")
	}
}

func TestGetNextStep_AttemptIncrementsOnRepeat(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerBoth))
	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "my task")

	// Call get_next_step twice — step stays running, attempt should increment.
	callGetNextStep := func() domain.GetNextStepResponse {
		result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "get_next_step",
				Arguments: map[string]any{"run_id": runID},
			},
		})
		if err != nil {
			t.Fatalf("call tool: %v", err)
		}
		text := result.Content[0].(mcp.TextContent).Text
		var resp domain.GetNextStepResponse
		if err := json.Unmarshal([]byte(text), &resp); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		return resp
	}

	r1 := callGetNextStep()
	r2 := callGetNextStep()

	if r1.Attempt != 1 {
		t.Errorf("first call: expected attempt 1, got %d", r1.Attempt)
	}
	if r2.Attempt != 2 {
		t.Errorf("second call: expected attempt 2, got %d", r2.Attempt)
	}
}

func TestGetNextStep_PromptInterpolation(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerBoth))
	runID := startRunAndGetID(t, client, loaderTestdataPath("multi_step.yaml"), "build feature X")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if !strings.Contains(resp.Prompt, "build feature X") {
		t.Errorf("expected interpolated task in prompt, got: %s", resp.Prompt)
	}
	if !strings.Contains(resp.Prompt, runID) {
		t.Errorf("expected interpolated run_id in prompt, got: %s", resp.Prompt)
	}
}

func TestGetNextStep_MissingInterpolationVariableReturnsError(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerBoth))
	runID := startRunAndGetID(t, client, loaderTestdataPath("interpolation_missing_var.yaml"), "build feature X")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error, got success: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "failed to interpolate step input") {
		t.Fatalf("expected interpolation error message, got: %s", text)
	}
	if !strings.Contains(text, `missing variable "unknown_var"`) {
		t.Fatalf("expected missing variable name in error message, got: %s", text)
	}
}

func TestGetNextStep_WorkflowDoneWhenNoStepsRemain(t *testing.T) {
	// We need to mark the single step as done via the store directly,
	// then call get_next_step to trigger the "done" path.
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterGetNextStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "task")

	// Mark the step as done so no pending steps remain.
	if err := st.UpdateStepStatus(context.Background(), runID, "step1", domain.StepStatusDone, 1, nil); err != nil {
		t.Fatalf("update step status: %v", err)
	}

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.Status != "done" {
		t.Errorf("expected status 'done', got %q", resp.Status)
	}
	if !strings.Contains(resp.Message, "All steps completed") {
		t.Errorf("expected 'All steps completed' in message, got %q", resp.Message)
	}
}

func TestGetNextStep_InactiveRunEscalated(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterGetNextStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "task")

	// Mark run as escalated.
	if err := st.UpdateRunStatus(context.Background(), runID, domain.RunStatusEscalated); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.Status != "escalated" {
		t.Errorf("expected status 'escalated', got %q", resp.Status)
	}
}

func TestGetNextStep_InactiveRunFailed(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterGetNextStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "task")

	// Mark run as failed.
	if err := st.UpdateRunStatus(context.Background(), runID, domain.RunStatusFailed); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
}

func TestGetNextStep_RunNotFound(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterGetNextStep))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": "nonexistent-run-id"},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for nonexistent run, got success")
	}
}

func TestGetNextStep_MissingRunID(t *testing.T) {
	client := startClient(t, newServerWithStore(t, tools.RegisterGetNextStep))

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing run_id, got success")
	}
}

func TestGetNextStep_AgentFilesIncluded(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerBoth))
	runID := startRunAndGetID(t, client, loaderTestdataPath("multi_step.yaml"), "task")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.Agent == nil {
		t.Fatal("expected agent info")
	}
	if len(resp.Agent.Files) == 0 {
		t.Error("expected agent files to be populated")
	}
	if _, ok := resp.Agent.Files["AGENTS.md"]; !ok {
		t.Errorf("expected AGENTS.md in agent files, got %v", resp.Agent.Files)
	}
}

func TestGetNextStep_NoRequiresOutput(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerBoth))
	runID := startRunAndGetID(t, client, loaderTestdataPath("valid.yaml"), "my task")

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp.RequiredOutputs != nil {
		t.Errorf("expected required_outputs to be nil, got %#v", resp.RequiredOutputs)
	}
}

func TestGetNextStep_OneRequiredOutput(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerAllThree))
	runID := startRunAndGetID(t, client, loaderTestdataPath("requires_output.yaml"), "task")

	first := getNextStepForTest(t, client, runID)
	if first.StepID != "explore" {
		t.Fatalf("expected first step explore, got %q", first.StepID)
	}

	exploreOutput := "STATUS: done\nExplore output"
	result, _ := callCompleteStep(t, client, runID, "explore", exploreOutput)
	if result.IsError {
		t.Fatalf("complete_step returned error: %v", result.Content)
	}

	second := getNextStepForTest(t, client, runID)
	if second.StepID != "implement" {
		t.Fatalf("expected step implement, got %q", second.StepID)
	}
	if second.RequiredOutputs == nil {
		t.Fatal("expected required_outputs map, got nil")
	}
	required, ok := second.RequiredOutputs["explore"]
	if !ok {
		t.Fatalf("expected required_outputs to contain explore, got %#v", second.RequiredOutputs)
	}
	if required == nil || *required != exploreOutput {
		t.Fatalf("expected required output %q, got %#v", exploreOutput, required)
	}
}

func TestGetNextStep_MultipleRequiredOutputs(t *testing.T) {
	client := startClient(t, newServerWithStore(t, registerAllThree))
	runID := startRunAndGetID(t, client, loaderTestdataPath("requires_output.yaml"), "task")

	step := getNextStepForTest(t, client, runID)
	if step.StepID != "explore" {
		t.Fatalf("expected step explore, got %q", step.StepID)
	}
	exploreOutput := "STATUS: done\nExplore output"
	result, _ := callCompleteStep(t, client, runID, "explore", exploreOutput)
	if result.IsError {
		t.Fatalf("complete_step explore returned error: %v", result.Content)
	}

	step = getNextStepForTest(t, client, runID)
	if step.StepID != "implement" {
		t.Fatalf("expected step implement, got %q", step.StepID)
	}
	implementOutput := "STATUS: done\nImplement output"
	result, _ = callCompleteStep(t, client, runID, "implement", implementOutput)
	if result.IsError {
		t.Fatalf("complete_step implement returned error: %v", result.Content)
	}

	step = getNextStepForTest(t, client, runID)
	if step.StepID != "verify" {
		t.Fatalf("expected step verify, got %q", step.StepID)
	}
	if len(step.RequiredOutputs) != 2 {
		t.Fatalf("expected 2 required outputs, got %#v", step.RequiredOutputs)
	}
	exploreRequired := step.RequiredOutputs["explore"]
	if exploreRequired == nil || *exploreRequired != exploreOutput {
		t.Fatalf("expected explore output %q, got %#v", exploreOutput, exploreRequired)
	}
	implementRequired := step.RequiredOutputs["implement"]
	if implementRequired == nil || *implementRequired != implementOutput {
		t.Fatalf("expected implement output %q, got %#v", implementOutput, implementRequired)
	}
}

func TestGetNextStep_RequiredStepWithoutOutput(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := newServerWith(func(srv *mcpgoserver.MCPServer) {
		tools.RegisterStartRun(srv, st)
		tools.RegisterGetNextStep(srv, st)
	})
	client := startClient(t, s)

	runID := startRunAndGetID(t, client, loaderTestdataPath("requires_output.yaml"), "task")

	// Simulate completed required step but with no output yet.
	if err := st.UpdateStepStatus(context.Background(), runID, "explore", domain.StepStatusDone, 1, nil); err != nil {
		t.Fatalf("update step status: %v", err)
	}

	resp := getNextStepForTest(t, client, runID)
	if resp.StepID != "implement" {
		t.Fatalf("expected step implement, got %q", resp.StepID)
	}
	required, ok := resp.RequiredOutputs["explore"]
	if !ok {
		t.Fatalf("expected required_outputs to contain explore, got %#v", resp.RequiredOutputs)
	}
	if required != nil {
		t.Fatalf("expected nil output for explore, got %#v", required)
	}
}

func getNextStepForTest(t *testing.T, client interface {
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}, runID string) domain.GetNextStepResponse {
	t.Helper()

	result, err := client.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_next_step",
			Arguments: map[string]any{"run_id": runID},
		},
	})
	if err != nil {
		t.Fatalf("call get_next_step: %v", err)
	}
	if result.IsError {
		t.Fatalf("get_next_step returned error: %v", result.Content)
	}

	text := result.Content[0].(mcp.TextContent).Text
	var resp domain.GetNextStepResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse get_next_step response: %v", err)
	}
	return resp
}
