package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
)

// RegisterGetRunStatus registers the get_run_status tool with the given MCP server.
func RegisterGetRunStatus(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("get_run_status",
		mcp.WithDescription("Returns the full state of a run: steps and accumulated variables"),
		mcp.WithString("run_id",
			mcp.Description("Run ID (optional if workflow_id provided)"),
		),
		mcp.WithString("workflow_id",
			mcp.Description("Workflow ID to find the most recent active run (optional if run_id provided)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return getRunStatusHandler(ctx, req, st)
	})
}

func getRunStatusHandler(ctx context.Context, req mcp.CallToolRequest, st *store.Store) (*mcp.CallToolResult, error) {
	runID := mcp.ParseString(req, "run_id", "")
	workflowID := mcp.ParseString(req, "workflow_id", "")

	input := domain.GetRunStatusRequest{RunID: runID, WorkflowID: workflowID}
	if err := input.Validate(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var run domain.Run
	var err error

	if runID != "" {
		run, err = st.GetRun(ctx, runID)
	} else {
		run, err = st.GetLatestActiveRunByWorkflow(ctx, workflowID)
	}

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("run not found: %s", func() string {
				if runID != "" {
					return runID
				}
				return workflowID
			}())), nil
		}
		slog.Error("get_run_status failed to get run", "run_id", runID, "workflow_id", workflowID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get run: %s", err)), nil
	}

	steps, err := st.GetStepsByRun(ctx, run.ID)
	if err != nil {
		slog.Error("get_run_status failed to get steps", "run_id", run.ID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get steps: %s", err)), nil
	}

	variables, err := st.GetVariablesByRun(ctx, run.ID)
	if err != nil {
		slog.Error("get_run_status failed to get variables", "run_id", run.ID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get variables: %s", err)), nil
	}

	stepSummaries := make([]domain.StepSummary, len(steps))
	for i, s := range steps {
		stepSummaries[i] = domain.StepSummary{
			StepID:  s.StepID,
			Status:  s.Status,
			Attempt: s.Attempt,
		}
	}

	out := domain.GetRunStatusResponse{
		RunID:      run.ID,
		WorkflowID: run.WorkflowID,
		Task:       run.Task,
		Status:     run.Status,
		Steps:      stepSummaries,
		Variables:  variables,
	}
	outBytes, _ := json.Marshal(out)
	return mcp.NewToolResultText(string(outBytes)), nil
}
