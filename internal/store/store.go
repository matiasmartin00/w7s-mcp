// Package store provides a SQLite-backed data access layer for workflow runs,
// steps, and variables.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // register the sqlite3 driver

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
)

// Store is the data access layer for runs, steps, and variables.
type Store struct {
	db *sql.DB
}

// Open opens or creates the SQLite database at the given path and runs migrations.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates all required tables if they do not exist.
func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    task TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS steps (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    status TEXT NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    output TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (run_id) REFERENCES runs(id)
);

CREATE TABLE IF NOT EXISTS variables (
    run_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (run_id, key),
    FOREIGN KEY (run_id) REFERENCES runs(id)
);
`
	_, err := db.Exec(schema)
	return err
}

// CreateRun persists a new Run record.
func (s *Store) CreateRun(ctx context.Context, run domain.Run) error {
	const q = `INSERT INTO runs (id, workflow_id, task, status, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, run.ID, run.WorkflowID, run.Task, string(run.Status), run.CreatedAt)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}
	return nil
}

// CreateStep persists a new Step record.
func (s *Store) CreateStep(ctx context.Context, step domain.Step) error {
	const q = `INSERT INTO steps (id, run_id, step_id, status, attempt, output, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, step.ID, step.RunID, step.StepID, string(step.Status), step.Attempt, step.Output, step.CreatedAt)
	if err != nil {
		return fmt.Errorf("create step: %w", err)
	}
	return nil
}

// SetVariable upserts a variable (INSERT OR REPLACE).
func (s *Store) SetVariable(ctx context.Context, v domain.Variable) error {
	const q = `INSERT OR REPLACE INTO variables (run_id, key, value) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, v.RunID, v.Key, v.Value)
	if err != nil {
		return fmt.Errorf("set variable: %w", err)
	}
	return nil
}

// GetRun returns the Run for the given ID. Returns domain.ErrNotFound if absent.
func (s *Store) GetRun(ctx context.Context, runID string) (domain.Run, error) {
	const q = `SELECT id, workflow_id, task, status, created_at FROM runs WHERE id = ?`
	row := s.db.QueryRowContext(ctx, q, runID)

	var run domain.Run
	var status string
	err := row.Scan(&run.ID, &run.WorkflowID, &run.Task, &status, &run.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Run{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Run{}, fmt.Errorf("get run: %w", err)
	}
	run.Status = domain.RunStatus(status)
	return run, nil
}

// GetStep returns the Step for the given run_id + step_id. Returns domain.ErrNotFound if absent.
func (s *Store) GetStep(ctx context.Context, runID, stepID string) (domain.Step, error) {
	const q = `SELECT id, run_id, step_id, status, attempt, output, created_at FROM steps WHERE run_id = ? AND step_id = ?`
	row := s.db.QueryRowContext(ctx, q, runID, stepID)

	var step domain.Step
	var status string
	err := row.Scan(&step.ID, &step.RunID, &step.StepID, &status, &step.Attempt, &step.Output, &step.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Step{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Step{}, fmt.Errorf("get step: %w", err)
	}
	step.Status = domain.StepStatus(status)
	return step, nil
}

// UpdateRunStatus updates the status of the run with the given ID.
func (s *Store) UpdateRunStatus(ctx context.Context, runID string, status domain.RunStatus) error {
	const q = `UPDATE runs SET status = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, q, string(status), runID)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	return nil
}

// UpdateStepStatus updates the status, attempt count, and output of a step.
func (s *Store) UpdateStepStatus(ctx context.Context, runID, stepID string, status domain.StepStatus, attempt int, output *string) error {
	const q = `UPDATE steps SET status = ?, attempt = ?, output = ? WHERE run_id = ? AND step_id = ?`
	_, err := s.db.ExecContext(ctx, q, string(status), attempt, output, runID, stepID)
	if err != nil {
		return fmt.Errorf("update step status: %w", err)
	}
	return nil
}

// ResetStepsToPending resets the step at fromStepID and all steps that come after it
// in stepOrder to pending status, clearing output and resetting attempt to 0.
func (s *Store) ResetStepsToPending(ctx context.Context, runID string, fromStepID string, stepOrder []string) error {
	startIdx := -1
	for i, id := range stepOrder {
		if id == fromStepID {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		return fmt.Errorf("reset steps to pending: step %q not found in stepOrder", fromStepID)
	}

	const q = `UPDATE steps SET status = ?, attempt = 0, output = NULL WHERE run_id = ? AND step_id = ?`
	for _, id := range stepOrder[startIdx:] {
		if _, err := s.db.ExecContext(ctx, q, string(domain.StepStatusPending), runID, id); err != nil {
			return fmt.Errorf("reset step %q to pending: %w", id, err)
		}
	}
	return nil
}

// GetStepsByRun returns all steps for a run, in insertion order.
func (s *Store) GetStepsByRun(ctx context.Context, runID string) ([]domain.Step, error) {
	const q = `SELECT id, run_id, step_id, status, attempt, output, created_at FROM steps WHERE run_id = ? ORDER BY rowid`
	rows, err := s.db.QueryContext(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("get steps by run: %w", err)
	}
	defer rows.Close()

	var steps []domain.Step
	for rows.Next() {
		var step domain.Step
		var status string
		if err := rows.Scan(&step.ID, &step.RunID, &step.StepID, &status, &step.Attempt, &step.Output, &step.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		step.Status = domain.StepStatus(status)
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate steps: %w", err)
	}
	return steps, nil
}

// GetVariablesByRun returns all variables for a run as a map.
func (s *Store) GetVariablesByRun(ctx context.Context, runID string) (map[string]string, error) {
	const q = `SELECT key, value FROM variables WHERE run_id = ?`
	rows, err := s.db.QueryContext(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("get variables by run: %w", err)
	}
	defer rows.Close()

	vars := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan variable: %w", err)
		}
		vars[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variables: %w", err)
	}
	return vars, nil
}

// GetLatestActiveRunByWorkflow returns the most recent run for the given workflow_id
// with status 'running'. Returns domain.ErrNotFound if none.
func (s *Store) GetLatestActiveRunByWorkflow(ctx context.Context, workflowID string) (domain.Run, error) {
	const q = `SELECT id, workflow_id, task, status, created_at FROM runs WHERE workflow_id = ? AND status = 'running' ORDER BY created_at DESC LIMIT 1`
	row := s.db.QueryRowContext(ctx, q, workflowID)

	var run domain.Run
	var status string
	err := row.Scan(&run.ID, &run.WorkflowID, &run.Task, &status, &run.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Run{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Run{}, fmt.Errorf("get latest active run by workflow: %w", err)
	}
	run.Status = domain.RunStatus(status)
	return run, nil
}
