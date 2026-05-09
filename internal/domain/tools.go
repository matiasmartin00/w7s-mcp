package domain

// ---------------------------------------------------------------------------
// StartRun
// ---------------------------------------------------------------------------

// StartRunRequest is the typed input for the start_run MCP tool.
type StartRunRequest struct {
	WorkflowID string `json:"workflow_id"`
	Task       string `json:"task"`
}

// Validate checks that all required fields are present.
func (r StartRunRequest) Validate() error {
	if r.WorkflowID == "" {
		return ValidationError{Field: "workflow_id", Code: ErrCodeRequired, Message: "workflow_id is required"}
	}
	if r.Task == "" {
		return ValidationError{Field: "task", Code: ErrCodeRequired, Message: "task is required"}
	}
	return nil
}

// StartRunResponse is the typed output for the start_run MCP tool.
type StartRunResponse struct {
	RunID    string   `json:"run_id"`
	Workflow string   `json:"workflow"`
	Steps    []string `json:"steps"`
	Message  string   `json:"message"`
}

// ---------------------------------------------------------------------------
// GetNextStep
// ---------------------------------------------------------------------------

// GetNextStepRequest is the typed input for the get_next_step MCP tool.
type GetNextStepRequest struct {
	RunID string `json:"run_id"`
}

// Validate checks that all required fields are present.
func (r GetNextStepRequest) Validate() error {
	if r.RunID == "" {
		return ValidationError{Field: "run_id", Code: ErrCodeRequired, Message: "run_id is required"}
	}
	return nil
}

// AgentInfo contains metadata about the agent assigned to a step.
type AgentInfo struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Files map[string]string `json:"files"`
}

// GetNextStepResponse is the typed output for the get_next_step MCP tool.
// Status is one of: "next_step", "done", "escalated".
type GetNextStepResponse struct {
	Status          string             `json:"status"`
	StepID          string             `json:"step_id,omitempty"`
	Agent           *AgentInfo         `json:"agent,omitempty"`
	Prompt          string             `json:"prompt,omitempty"`
	Attempt         int                `json:"attempt,omitempty"`
	Expects         string             `json:"expects,omitempty"`
	RequiredOutputs map[string]*string `json:"required_outputs,omitempty"`
	Instruction     string             `json:"instruction,omitempty"`
	Message         string             `json:"message,omitempty"`
}

// ---------------------------------------------------------------------------
// CompleteStep
// ---------------------------------------------------------------------------

// CompleteStepRequest is the typed input for the complete_step MCP tool.
type CompleteStepRequest struct {
	RunID  string `json:"run_id"`
	StepID string `json:"step_id"`
	Output string `json:"output"`
}

// Validate checks that all required fields are present.
func (r CompleteStepRequest) Validate() error {
	if r.RunID == "" {
		return ValidationError{Field: "run_id", Code: ErrCodeRequired, Message: "run_id is required"}
	}
	if r.StepID == "" {
		return ValidationError{Field: "step_id", Code: ErrCodeRequired, Message: "step_id is required"}
	}
	if r.Output == "" {
		return ValidationError{Field: "output", Code: ErrCodeRequired, Message: "output is required"}
	}
	return nil
}

// CompleteStepResponse is the typed output for the complete_step MCP tool.
type CompleteStepResponse struct {
	Status             string            `json:"status"` // "step_done"
	StepID             string            `json:"step_id"`
	VariablesExtracted map[string]string `json:"variables_extracted"`
	Message            string            `json:"message"`
}

// ---------------------------------------------------------------------------
// FailStep
// ---------------------------------------------------------------------------

// FailStepRequest is the typed input for the fail_step MCP tool.
type FailStepRequest struct {
	RunID  string `json:"run_id"`
	StepID string `json:"step_id"`
	Reason string `json:"reason,omitempty"`
}

// Validate checks that all required fields are present.
func (r FailStepRequest) Validate() error {
	if r.RunID == "" {
		return ValidationError{Field: "run_id", Code: ErrCodeRequired, Message: "run_id is required"}
	}
	if r.StepID == "" {
		return ValidationError{Field: "step_id", Code: ErrCodeRequired, Message: "step_id is required"}
	}
	return nil
}

// FailStepResponse is the typed output for the fail_step MCP tool.
// Status is one of: "retry", "escalated".
type FailStepResponse struct {
	Status            string `json:"status"`
	StepID            string `json:"step_id"`
	RetryStep         string `json:"retry_step,omitempty"`
	AttemptsUsed      int    `json:"attempts_used,omitempty"`
	AttemptsRemaining int    `json:"attempts_remaining,omitempty"`
	Attempts          int    `json:"attempts,omitempty"`
	EscalateTo        string `json:"escalate_to,omitempty"`
	Message           string `json:"message"`
}

// ---------------------------------------------------------------------------
// GetRunStatus
// ---------------------------------------------------------------------------

// GetRunStatusRequest is the typed input for the get_run_status MCP tool.
// At least one of RunID or WorkflowID must be provided.
type GetRunStatusRequest struct {
	RunID      string `json:"run_id,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
}

// Validate checks that at least one identifier is provided.
func (r GetRunStatusRequest) Validate() error {
	if r.RunID == "" && r.WorkflowID == "" {
		return ValidationError{
			Field:   "run_id",
			Code:    ErrCodeRequired,
			Message: "at least one of run_id or workflow_id is required",
		}
	}
	return nil
}

// StepSummary is a lightweight representation of a step for status responses.
type StepSummary struct {
	StepID  string     `json:"step_id"`
	Status  StepStatus `json:"status"`
	Attempt int        `json:"attempt"`
}

// GetRunStatusResponse is the typed output for the get_run_status MCP tool.
type GetRunStatusResponse struct {
	RunID      string            `json:"run_id"`
	WorkflowID string            `json:"workflow_id"`
	Task       string            `json:"task"`
	Status     RunStatus         `json:"status"`
	Steps      []StepSummary     `json:"steps"`
	Variables  map[string]string `json:"variables"`
}
