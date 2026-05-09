package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/loader"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
)

// RegisterStartRun registers the start_run tool with the given MCP server.
// The store is used to persist the new run, its steps, and the task variable.
func RegisterStartRun(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("start_run",
		mcp.WithDescription("Load and validate a workflow, then start a run"),
		mcp.WithString("workflow_id",
			mcp.Required(),
			mcp.Description("ID or absolute path of the YAML workflow file"),
		),
		mcp.WithString("task",
			mcp.Required(),
			mcp.Description("Task description to pass to the workflow"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return startRunHandler(ctx, req, st)
	})
}

func startRunHandler(ctx context.Context, req mcp.CallToolRequest, st *store.Store) (*mcp.CallToolResult, error) {
	workflowID, err := req.RequireString("workflow_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	task, err := req.RequireString("task")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	input := domain.StartRunRequest{WorkflowID: workflowID, Task: task}
	if err := input.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// clientName is read from the session in ctx — safe for concurrent clients,
	// each request carries its own session with its own clientInfo.
	cn := clientNameFromContext(ctx)
	wf, err := loader.LoadByID(cn, workflowID)
	if err != nil {
		slog.Error("start_run failed to load workflow", "workflow_id", workflowID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to load workflow: %s", err)), nil
	}

	runID := domain.NewRunID()
	now := time.Now().UnixMilli()

	// Persist the run.
	run := domain.Run{
		ID:         runID,
		WorkflowID: workflowID,
		Task:       task,
		Status:     domain.RunStatusRunning,
		CreatedAt:  now,
	}
	if err := st.CreateRun(ctx, run); err != nil {
		slog.Error("start_run failed to persist run", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to persist run: %s", err)), nil
	}

	// Persist each step.
	stepIDs := make([]string, len(wf.Steps))
	for i, step := range wf.Steps {
		stepIDs[i] = step.ID
		s := domain.Step{
			ID:        runID + ":" + step.ID,
			RunID:     runID,
			StepID:    step.ID,
			Status:    domain.StepStatusPending,
			Attempt:   0,
			CreatedAt: now,
		}
		if err := st.CreateStep(ctx, s); err != nil {
			slog.Error("start_run failed to persist step", "run_id", runID, "step_id", step.ID, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to persist step: %s", err)), nil
		}
	}

	// Persist the task variable.
	if err := st.SetVariable(ctx, domain.Variable{RunID: runID, Key: "task", Value: task}); err != nil {
		slog.Error("start_run failed to persist task variable", "run_id", runID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to persist task variable: %s", err)), nil
	}

	slog.Info("start_run loaded workflow", "workflow", wf.Name, "steps", stepIDs, "task", task, "run_id", runID)

	out := domain.StartRunResponse{
		RunID:    runID,
		Workflow: wf.Name,
		Steps:    stepIDs,
		Message:  fmt.Sprintf("Run started. Call get_next_step with run_id: %s", runID),
	}
	outBytes, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(outBytes)), nil
}

// clientNameFromContext extracts the MCP client name from the session stored in ctx.
// Returns an empty string if the session does not implement SessionWithClientInfo,
// which causes the loader to fall back to the default workflow directories.
func clientNameFromContext(ctx context.Context) string {
	session := server.ClientSessionFromContext(ctx)
	if session == nil {
		return ""
	}
	ci, ok := session.(server.SessionWithClientInfo)
	if !ok {
		return ""
	}
	return ci.GetClientInfo().Name
}
