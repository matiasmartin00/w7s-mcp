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
	"github.com/matiasmartin00/w7s-mcp/internal/store"
)

// RegisterFailStep registers the fail_step tool with the given MCP server.
func RegisterFailStep(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("fail_step",
		mcp.WithDescription("Marks a step as failed. Applies retry logic or escalates the run based on on_fail configuration."),
		mcp.WithString("run_id",
			mcp.Required(),
			mcp.Description("Run ID returned by start_run"),
		),
		mcp.WithString("step_id",
			mcp.Required(),
			mcp.Description("Step ID that failed"),
		),
		mcp.WithString("reason",
			mcp.Description("Optional reason for failure; stored in feedback_var if configured"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return failStepHandler(ctx, req, st)
	})
}

func failStepHandler(ctx context.Context, req mcp.CallToolRequest, st *store.Store) (*mcp.CallToolResult, error) {
	runID, err := req.RequireString("run_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	stepID, err := req.RequireString("step_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var reason string
	if args, ok := req.Params.Arguments.(map[string]any); ok {
		reason, _ = args["reason"].(string)
	}

	input := domain.FailStepRequest{RunID: runID, StepID: stepID, Reason: reason}
	if err := input.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Load the run to ensure it exists and get the workflow ID.
	run, err := st.GetRun(ctx, runID)
	if err != nil {
		if err == domain.ErrNotFound {
			return mcp.NewToolResultError(fmt.Sprintf("run %q not found", runID)), nil
		}
		slog.Error("fail_step failed to get run", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get run: %s", err)), nil
	}

	// Load the step to ensure it exists and get the current attempt.
	step, err := st.GetStep(ctx, runID, stepID)
	if err != nil {
		if err == domain.ErrNotFound {
			return mcp.NewToolResultError(fmt.Sprintf("step %q not found in run %q", stepID, runID)), nil
		}
		slog.Error("fail_step failed to get step", "run_id", runID, "step_id", stepID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get step: %s", err)), nil
	}

	// Load workflow definition to get on_fail config and step order.
	cn := clientNameFromContext(ctx)
	wf, err := loader.LoadByID(cn, run.WorkflowID)
	if err != nil {
		slog.Error("fail_step failed to load workflow", "workflow_id", run.WorkflowID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workflow: %s", err)), nil
	}

	// Find on_fail config for this step.
	var retryStepID string
	var feedbackVar string
	var maxRetries int
	var escalateTo string
	hasOnFail := false

	for i := range wf.Steps {
		if wf.Steps[i].ID == stepID {
			if wf.Steps[i].OnFail != nil {
				hasOnFail = true
				retryStepID = wf.Steps[i].OnFail.RetryStep
				feedbackVar = wf.Steps[i].OnFail.FeedbackVar
				maxRetries = wf.Steps[i].OnFail.MaxRetries
				if wf.Steps[i].OnFail.OnExhausted != nil {
					escalateTo = wf.Steps[i].OnFail.OnExhausted.EscalateTo
				}
			}
			break
		}
	}

	// Build step order from workflow definition.
	stepOrder := make([]string, len(wf.Steps))
	for i, s := range wf.Steps {
		stepOrder[i] = s.ID
	}

	// Current attempt count for this step.
	attempt := step.Attempt

	// Determine retry or escalate.
	shouldRetry := hasOnFail && retryStepID != "" && maxRetries > 0 && attempt < maxRetries

	if !shouldRetry {
		// Escalate: mark step as failed, update run to escalated.
		if err := st.UpdateStepStatus(ctx, runID, stepID, domain.StepStatusFailed, attempt, step.Output); err != nil {
			slog.Error("fail_step failed to update step status", "run_id", runID, "step_id", stepID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to update step status: %s", err)), nil
		}
		if err := st.UpdateRunStatus(ctx, runID, domain.RunStatusEscalated); err != nil {
			slog.Error("fail_step failed to update run status", "run_id", runID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to update run status: %s", err)), nil
		}

		if escalateTo == "" {
			escalateTo = "human"
		}

		slog.Info("fail_step escalated run", "run_id", runID, "step_id", stepID, "attempts", attempt)

		out := domain.FailStepResponse{
			Status:     "escalated",
			StepID:     stepID,
			Attempts:   attempt,
			EscalateTo: escalateTo,
			Message:    fmt.Sprintf("Step '%s' failed after %d attempt(s). Escalating to %s.", stepID, attempt, escalateTo),
		}
		outBytes, _ := json.Marshal(out)
		return mcp.NewToolResultText(string(outBytes)), nil
	}

	// Retry path.

	// Store reason in feedback_var if configured.
	if feedbackVar != "" && reason != "" {
		if err := st.SetVariable(ctx, domain.Variable{RunID: runID, Key: feedbackVar, Value: reason}); err != nil {
			slog.Error("fail_step failed to persist feedback_var", "run_id", runID, "key", feedbackVar, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to persist feedback_var %q: %s", feedbackVar, err)), nil
		}
	}

	// Reset retry_step and all steps between retry_step and the failed step to pending.
	if err := st.ResetStepsToPending(ctx, runID, retryStepID, stepOrder); err != nil {
		slog.Error("fail_step failed to reset steps to pending", "run_id", runID, "retry_step", retryStepID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to reset steps to pending: %s", err)), nil
	}

	attemptsUsed := attempt
	attemptsRemaining := maxRetries - attempt

	slog.Info("fail_step retrying", "run_id", runID, "step_id", stepID, "retry_step", retryStepID, "attempts_used", attemptsUsed, "attempts_remaining", attemptsRemaining)

	out := domain.FailStepResponse{
		Status:            "retry",
		StepID:            stepID,
		RetryStep:         retryStepID,
		AttemptsUsed:      attemptsUsed,
		AttemptsRemaining: attemptsRemaining,
		Message:           fmt.Sprintf("Step '%s' failed. Retrying from '%s'. Call get_next_step to continue.", stepID, retryStepID),
	}
	outBytes, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(outBytes)), nil
}
