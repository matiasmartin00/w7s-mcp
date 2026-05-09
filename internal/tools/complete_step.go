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
)

// RegisterCompleteStep registers the complete_step tool with the given MCP server.
func RegisterCompleteStep(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("complete_step",
		mcp.WithDescription("Marks a step as done, persists its output, and extracts variables."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID returned by start_run"),
		),
		mcp.WithString("step_id",
			mcp.Required(),
			mcp.Description("Step ID to complete"),
		),
		mcp.WithString("output",
			mcp.Required(),
			mcp.Description("Full output produced by the step"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return completeStepHandler(ctx, req, st)
	})
}

func completeStepHandler(ctx context.Context, req mcp.CallToolRequest, st *store.Store) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	output, err := req.RequireString("output")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	input := domain.CompleteStepRequest{RunID: runID, StepID: stepID, Output: output}
	if err := input.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Load the run to ensure it exists and get the workflow ID.
	run, err := st.GetRun(ctx, runID)
	if err != nil {
		if err == domain.ErrNotFound {
			return mcp.NewToolResultError(fmt.Sprintf("run %q not found", runID)), nil
		}
		slog.Error("complete_step failed to get run", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get run: %s", err)), nil
	}

	// Load the step to ensure it exists and get the current attempt.
	step, err := st.GetStep(ctx, runID, stepID)
	if err != nil {
		if err == domain.ErrNotFound {
			return mcp.NewToolResultError(fmt.Sprintf("step %q not found in run %q", stepID, runID)), nil
		}
		slog.Error("complete_step failed to get step", "run_id", runID, "step_id", stepID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get step: %s", err)), nil
	}

	// Update step to done with the full output.
	if err := st.UpdateStepStatus(ctx, runID, stepID, domain.StepStatusDone, step.Attempt, &output); err != nil {
		slog.Error("complete_step failed to update step status", "run_id", runID, "step_id", stepID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to update step status: %s", err)), nil
	}

	// Load workflow definition to get extraction patterns for this step.
	cn := clientNameFromContext(ctx)
	wf, err := loader.LoadByID(cn, run.WorkflowID)
	if err != nil {
		slog.Error("complete_step failed to load workflow", "workflow_id", run.WorkflowID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workflow: %s", err)), nil
	}

	// Find the step definition.
	var extractPatterns map[string]string
	for i := range wf.Steps {
		if wf.Steps[i].ID == stepID {
			extractPatterns = wf.Steps[i].Extract
			break
		}
	}

	// Extract variables using first capture group semantics. Missing matches are silently skipped.
	extracted := parser.Extract(output, extractPatterns)

	// Persist extracted variables.
	for k, v := range extracted {
		if err := st.SetVariable(ctx, domain.Variable{RunID: runID, Key: k, Value: v}); err != nil {
			slog.Error("complete_step failed to persist variable", "run_id", runID, "key", k, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to persist variable %q: %s", k, err)), nil
		}
	}

	slog.Info("complete_step completed step", "run_id", runID, "step_id", stepID, "variables_extracted", len(extracted))

	out := domain.CompleteStepResponse{
		Status:             "step_done",
		StepID:             stepID,
		VariablesExtracted: extracted,
		Message:            fmt.Sprintf("Step '%s' completed. Call get_next_step to continue.", stepID),
	}
	outBytes, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(outBytes)), nil
}
