// Package domain defines the canonical runtime entities and status enumerations
// for w7s-mcp workflow executions.
package domain

import (
	"crypto/rand"
	"fmt"
)

// RunStatus represents the lifecycle state of a workflow run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusDone      RunStatus = "done"
	RunStatusEscalated RunStatus = "escalated"
	RunStatusFailed    RunStatus = "failed"
)

// StepStatus represents the lifecycle state of a single step execution.
type StepStatus string

const (
	StepStatusPending  StepStatus = "pending"
	StepStatusRunning  StepStatus = "running"
	StepStatusDone     StepStatus = "done"
	StepStatusFailed   StepStatus = "failed"
)

// Run represents a workflow execution instance.
type Run struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	Task       string    `json:"task"`
	Status     RunStatus `json:"status"`
	CreatedAt  int64     `json:"created_at"` // unix ms
}

// Step represents a single step execution within a run.
type Step struct {
	ID        string     `json:"id"`       // "{run_id}:{step_id}"
	RunID     string     `json:"run_id"`
	StepID    string     `json:"step_id"`
	Status    StepStatus `json:"status"`
	Attempt   int        `json:"attempt"`
	Output    *string    `json:"output,omitempty"`
	CreatedAt int64      `json:"created_at"`
}

// Variable represents a key-value pair extracted from a step output.
type Variable struct {
	RunID string `json:"run_id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewRunID generates a new UUID v4 string using crypto/rand.
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
func NewRunID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Set version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}
