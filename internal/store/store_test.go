package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
	"github.com/matiasmartin00/w7s-mcp/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestCreateAndGetRun(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-1",
		WorkflowID: "workflow-1",
		Task:       "test task",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}

	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	got, err := st.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}

	if got.ID != run.ID {
		t.Errorf("id: want %s, got %s", run.ID, got.ID)
	}
	if got.WorkflowID != run.WorkflowID {
		t.Errorf("workflow_id: want %s, got %s", run.WorkflowID, got.WorkflowID)
	}
	if got.Task != run.Task {
		t.Errorf("task: want %s, got %s", run.Task, got.Task)
	}
	if got.Status != run.Status {
		t.Errorf("status: want %s, got %s", run.Status, got.Status)
	}
	if got.CreatedAt != run.CreatedAt {
		t.Errorf("created_at: want %d, got %d", run.CreatedAt, got.CreatedAt)
	}
}

func TestCreateStepAndGetByRun(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-2",
		WorkflowID: "workflow-2",
		Task:       "multi step task",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	steps := []domain.Step{
		{ID: "run-2:step-a", RunID: "run-2", StepID: "step-a", Status: domain.StepStatusPending, Attempt: 0, CreatedAt: time.Now().UnixMilli()},
		{ID: "run-2:step-b", RunID: "run-2", StepID: "step-b", Status: domain.StepStatusPending, Attempt: 0, CreatedAt: time.Now().UnixMilli()},
		{ID: "run-2:step-c", RunID: "run-2", StepID: "step-c", Status: domain.StepStatusPending, Attempt: 0, CreatedAt: time.Now().UnixMilli()},
	}
	for _, s := range steps {
		if err := st.CreateStep(ctx, s); err != nil {
			t.Fatalf("create step %s: %v", s.ID, err)
		}
	}

	got, err := st.GetStepsByRun(ctx, "run-2")
	if err != nil {
		t.Fatalf("get steps by run: %v", err)
	}

	if len(got) != len(steps) {
		t.Fatalf("want %d steps, got %d", len(steps), len(got))
	}

	for i, s := range steps {
		if got[i].StepID != s.StepID {
			t.Errorf("step[%d] id: want %s, got %s", i, s.StepID, got[i].StepID)
		}
	}
}

func TestSetVariableAndGetByRun(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-3",
		WorkflowID: "workflow-3",
		Task:       "variable test",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Upsert the same key twice — second value should win.
	if err := st.SetVariable(ctx, domain.Variable{RunID: "run-3", Key: "foo", Value: "first"}); err != nil {
		t.Fatalf("set variable first: %v", err)
	}
	if err := st.SetVariable(ctx, domain.Variable{RunID: "run-3", Key: "foo", Value: "second"}); err != nil {
		t.Fatalf("set variable second: %v", err)
	}
	if err := st.SetVariable(ctx, domain.Variable{RunID: "run-3", Key: "bar", Value: "baz"}); err != nil {
		t.Fatalf("set variable bar: %v", err)
	}

	vars, err := st.GetVariablesByRun(ctx, "run-3")
	if err != nil {
		t.Fatalf("get variables by run: %v", err)
	}

	if vars["foo"] != "second" {
		t.Errorf("foo: want 'second', got %q", vars["foo"])
	}
	if vars["bar"] != "baz" {
		t.Errorf("bar: want 'baz', got %q", vars["bar"])
	}
}

func TestGetLatestActiveRunByWorkflow(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	now := time.Now().UnixMilli()

	run1 := domain.Run{
		ID:         "run-older",
		WorkflowID: "workflow-common",
		Task:       "old task",
		Status:     domain.RunStatusRunning,
		CreatedAt:  now - 1000,
	}
	run2 := domain.Run{
		ID:         "run-newer",
		WorkflowID: "workflow-common",
		Task:       "new task",
		Status:     domain.RunStatusRunning,
		CreatedAt:  now,
	}

	for _, r := range []domain.Run{run1, run2} {
		if err := st.CreateRun(ctx, r); err != nil {
			t.Fatalf("create run %s: %v", r.ID, err)
		}
	}

	got, err := st.GetLatestActiveRunByWorkflow(ctx, "workflow-common")
	if err != nil {
		t.Fatalf("get latest active run: %v", err)
	}

	if got.ID != run2.ID {
		t.Errorf("want newest run %s, got %s", run2.ID, got.ID)
	}
}

func TestGetRun_NotFound(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	_, err := st.GetRun(ctx, "nonexistent-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want domain.ErrNotFound, got %v", err)
	}
}

func TestGetStep(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-gs",
		WorkflowID: "wf-gs",
		Task:       "get step test",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	step := domain.Step{
		ID:        "run-gs:step-x",
		RunID:     "run-gs",
		StepID:    "step-x",
		Status:    domain.StepStatusPending,
		Attempt:   0,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := st.CreateStep(ctx, step); err != nil {
		t.Fatalf("create step: %v", err)
	}

	got, err := st.GetStep(ctx, "run-gs", "step-x")
	if err != nil {
		t.Fatalf("get step: %v", err)
	}
	if got.StepID != step.StepID {
		t.Errorf("step_id: want %s, got %s", step.StepID, got.StepID)
	}
	if got.RunID != step.RunID {
		t.Errorf("run_id: want %s, got %s", step.RunID, got.RunID)
	}

	// not found
	_, err = st.GetStep(ctx, "run-gs", "nonexistent")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("want domain.ErrNotFound, got %v", err)
	}
}

func TestUpdateRunStatus(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-urs",
		WorkflowID: "wf-urs",
		Task:       "update run status test",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := st.UpdateRunStatus(ctx, "run-urs", domain.RunStatusDone); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	got, err := st.GetRun(ctx, "run-urs")
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if got.Status != domain.RunStatusDone {
		t.Errorf("status: want %s, got %s", domain.RunStatusDone, got.Status)
	}
}

func TestUpdateStepStatus(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-uss",
		WorkflowID: "wf-uss",
		Task:       "update step status test",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	step := domain.Step{
		ID:        "run-uss:step-y",
		RunID:     "run-uss",
		StepID:    "step-y",
		Status:    domain.StepStatusPending,
		Attempt:   0,
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := st.CreateStep(ctx, step); err != nil {
		t.Fatalf("create step: %v", err)
	}

	out := "result output"
	if err := st.UpdateStepStatus(ctx, "run-uss", "step-y", domain.StepStatusDone, 2, &out); err != nil {
		t.Fatalf("update step status: %v", err)
	}

	steps, err := st.GetStepsByRun(ctx, "run-uss")
	if err != nil {
		t.Fatalf("get steps by run: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("want 1 step, got %d", len(steps))
	}
	got := steps[0]
	if got.Status != domain.StepStatusDone {
		t.Errorf("status: want %s, got %s", domain.StepStatusDone, got.Status)
	}
	if got.Attempt != 2 {
		t.Errorf("attempt: want 2, got %d", got.Attempt)
	}
	if got.Output == nil || *got.Output != out {
		t.Errorf("output: want %q, got %v", out, got.Output)
	}
}

func TestResetStepsToPending(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	run := domain.Run{
		ID:         "run-rsp",
		WorkflowID: "wf-rsp",
		Task:       "reset steps test",
		Status:     domain.RunStatusRunning,
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := st.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	out := "done output"
	stepsToCreate := []domain.Step{
		{ID: "run-rsp:step-a", RunID: "run-rsp", StepID: "step-a", Status: domain.StepStatusDone, Attempt: 1, Output: &out, CreatedAt: time.Now().UnixMilli()},
		{ID: "run-rsp:step-b", RunID: "run-rsp", StepID: "step-b", Status: domain.StepStatusDone, Attempt: 1, Output: &out, CreatedAt: time.Now().UnixMilli()},
		{ID: "run-rsp:step-c", RunID: "run-rsp", StepID: "step-c", Status: domain.StepStatusDone, Attempt: 1, Output: &out, CreatedAt: time.Now().UnixMilli()},
	}
	for _, s := range stepsToCreate {
		if err := st.CreateStep(ctx, s); err != nil {
			t.Fatalf("create step %s: %v", s.StepID, err)
		}
	}

	stepOrder := []string{"step-a", "step-b", "step-c"}
	if err := st.ResetStepsToPending(ctx, "run-rsp", "step-b", stepOrder); err != nil {
		t.Fatalf("reset steps to pending: %v", err)
	}

	steps, err := st.GetStepsByRun(ctx, "run-rsp")
	if err != nil {
		t.Fatalf("get steps by run: %v", err)
	}

	byID := make(map[string]domain.Step)
	for _, s := range steps {
		byID[s.StepID] = s
	}

	// step-a should remain done
	if byID["step-a"].Status != domain.StepStatusDone {
		t.Errorf("step-a: want done, got %s", byID["step-a"].Status)
	}

	// step-b and step-c should be reset to pending
	for _, id := range []string{"step-b", "step-c"} {
		s := byID[id]
		if s.Status != domain.StepStatusPending {
			t.Errorf("%s: want pending, got %s", id, s.Status)
		}
		if s.Attempt != 0 {
			t.Errorf("%s: want attempt 0, got %d", id, s.Attempt)
		}
		if s.Output != nil {
			t.Errorf("%s: want nil output, got %v", id, s.Output)
		}
	}
}
