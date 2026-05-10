package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/loader"
	"github.com/matiasmartin00/w7s-mcp/internal/parser"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
	"github.com/matiasmartin00/w7s-mcp/internal/workflow"
)

// RegisterGetNextStep registers the get_next_step tool with the given MCP server.
func RegisterGetNextStep(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("get_next_step",
		mcp.WithDescription("Returns the next step to execute with the interpolated prompt. Marks the step as running."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID returned by start_run"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getNextStepHandler(ctx, req, st)
	})
}

func getNextStepHandler(ctx context.Context, req mcp.CallToolRequest, st *store.Store) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	input := domain.GetNextStepRequest{RunID: runID}
	if err := input.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Load the run.
	run, err := st.GetRun(ctx, runID)
	if err != nil {
		if err == domain.ErrNotFound {
			return mcp.NewToolResultError(fmt.Sprintf("run %q not found", runID)), nil
		}
		slog.Error("get_next_step failed to get run", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get run: %s", err)), nil
	}

	// If run is not active, return inactive status.
	if run.Status != domain.RunStatusRunning {
		out := domain.GetNextStepResponse{
			Status:  string(run.Status),
			Message: fmt.Sprintf("Run is not active (status: %s)", run.Status),
		}
		outBytes, _ := json.Marshal(out)
		return mcp.NewToolResultText(string(outBytes)), nil
	}

	// Load all steps for the run (in insertion order).
	steps, err := st.GetStepsByRun(ctx, runID)
	if err != nil {
		slog.Error("get_next_step failed to get steps", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get steps: %s", err)), nil
	}

	// Select first pending or running step (in insertion order).
	var nextStep *domain.Step
	for i := range steps {
		s := &steps[i]
		if s.Status == domain.StepStatusPending || s.Status == domain.StepStatusRunning {
			nextStep = s
			break
		}
	}

	// No pending/running steps — workflow is complete.
	if nextStep == nil {
		if err := st.UpdateRunStatus(ctx, runID, domain.RunStatusDone); err != nil {
			slog.Error("get_next_step failed to mark run done", "run_id", runID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to update run status: %s", err)), nil
		}
		out := domain.GetNextStepResponse{
			Status:  "done",
			Message: "All steps completed. Workflow finished.",
		}
		outBytes, _ := json.Marshal(out)
		return mcp.NewToolResultText(string(outBytes)), nil
	}

	// Mark step as running and increment attempt.
	newAttempt := nextStep.Attempt + 1
	if err := st.UpdateStepStatus(ctx, runID, nextStep.StepID, domain.StepStatusRunning, newAttempt, nextStep.Output); err != nil {
		slog.Error("get_next_step failed to mark step running", "run_id", runID, "step_id", nextStep.StepID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to update step status: %s", err)), nil
	}

	// Load workflow definition to get step metadata (agent, input template, expects).
	cn := clientNameFromContext(ctx)
	wf, err := loader.LoadByID(cn, run.WorkflowID)
	if err != nil {
		slog.Error("get_next_step failed to load workflow", "workflow_id", run.WorkflowID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workflow: %s", err)), nil
	}

	wfStep, agent, err := findStepAndAgent(wf, nextStep.StepID)
	if err != nil {
		slog.Error("get_next_step: step definition not found in workflow", "step_id", nextStep.StepID, "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Load variables for interpolation.
	vars, err := st.GetVariablesByRun(ctx, runID)
	if err != nil {
		slog.Error("get_next_step failed to get variables", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get variables: %s", err)), nil
	}
	// Built-in variables available in all step prompts.
	vars["run_id"] = run.ID
	vars["task"] = run.Task

	prompt, err := parser.InterpolateStrict(wfStep.Input, vars)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to interpolate step input: %s", err)), nil
	}

	requiredOutputs, err := collectRequiredOutputs(ctx, st, runID, wf, wfStep)
	if err != nil {
		slog.Error("get_next_step failed to resolve required outputs", "run_id", runID, "step_id", nextStep.StepID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to resolve required outputs: %s", err)), nil
	}

	expects := wfStep.Expects
	if expects == "" {
		expects = `STATUS:\s*done`
	}

	agentInfo := &domain.AgentInfo{
		ID:    agent.ID,
		Name:  agent.Name,
		Files: agent.Workspace.Files,
	}

	instruction := fmt.Sprintf(
		"Execute the agent '%s' with the prompt above. Evaluate its output against the 'expects' pattern. If it matches, call complete_step. If it does not match or the step failed, call fail_step.",
		agent.Name,
	)

	slog.Info("get_next_step dispatching step", "run_id", runID, "step_id", nextStep.StepID, "attempt", newAttempt)

	out := domain.GetNextStepResponse{
		Status:          "next_step",
		StepID:          nextStep.StepID,
		Agent:           agentInfo,
		Prompt:          prompt,
		Attempt:         newAttempt,
		Expects:         expects,
		RequiredOutputs: requiredOutputs,
		Instruction:     instruction,
	}
	outBytes, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(outBytes)), nil
}

func collectRequiredOutputs(ctx context.Context, st *store.Store, runID string, wf *workflow.Workflow, wfStep *workflow.Step) (map[string]*string, error) {
	if len(wfStep.RequiresOutput) == 0 {
		return nil, nil
	}

	stepOrder := make(map[string]int, len(wf.Steps))
	for i, step := range wf.Steps {
		stepOrder[step.ID] = i
	}

	currentIdx, ok := stepOrder[wfStep.ID]
	if !ok {
		return nil, fmt.Errorf("step %q not found in workflow definition", wfStep.ID)
	}

	requiredOutputs := make(map[string]*string, len(wfStep.RequiresOutput))
	for _, requiredStepID := range wfStep.RequiresOutput {
		requiredIdx, exists := stepOrder[requiredStepID]
		if !exists {
			return nil, fmt.Errorf("required step %q (from step %q) not found in workflow definition", requiredStepID, wfStep.ID)
		}
		if requiredIdx >= currentIdx {
			return nil, fmt.Errorf("required step %q must appear before step %q", requiredStepID, wfStep.ID)
		}

		requiredStep, err := st.GetStep(ctx, runID, requiredStepID)
		if err != nil {
			if err == domain.ErrNotFound {
				return nil, fmt.Errorf("required step %q not found in run %q", requiredStepID, runID)
			}
			return nil, fmt.Errorf("get required step %q: %w", requiredStepID, err)
		}

		requiredOutputs[requiredStepID] = requiredStep.Output
	}

	return requiredOutputs, nil
}

// findStepAndAgent locates the workflow step definition and its agent by stepID.
func findStepAndAgent(wf *workflow.Workflow, stepID string) (*workflow.Step, *workflow.Agent, error) {
	var wfStep *workflow.Step
	for i := range wf.Steps {
		if wf.Steps[i].ID == stepID {
			wfStep = &wf.Steps[i]
			break
		}
	}
	if wfStep == nil {
		return nil, nil, fmt.Errorf("step %q not found in workflow definition", stepID)
	}

	var agent *workflow.Agent
	for i := range wf.Agents {
		if wf.Agents[i].ID == wfStep.Agent {
			agent = &wf.Agents[i]
			break
		}
	}
	if agent == nil {
		return nil, nil, fmt.Errorf("agent %q (referenced by step %q) not found in workflow definition", wfStep.Agent, stepID)
	}

	return wfStep, agent, nil
}
