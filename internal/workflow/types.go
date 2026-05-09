// Package workflow defines the Go types for a w7s workflow definition.
package workflow

// Workflow is the top-level structure for a workflow YAML document.
type Workflow struct {
	ID          string  `yaml:"id" json:"id"`
	Name        string  `yaml:"name" json:"name"`
	Version     string  `yaml:"version" json:"version"`
	Description string  `yaml:"description,omitempty" json:"description,omitempty"`
	Agents      []Agent `yaml:"agents" json:"agents"`
	Steps       []Step  `yaml:"steps" json:"steps"`
}

// Agent describes an agent used in the workflow.
type Agent struct {
	ID        string         `yaml:"id" json:"id"`
	Name      string         `yaml:"name" json:"name"`
	Workspace AgentWorkspace `yaml:"workspace,omitempty" json:"workspace,omitempty"`
}

// AgentWorkspace holds file mappings for an agent's working context.
type AgentWorkspace struct {
	Files map[string]string `yaml:"files,omitempty" json:"files,omitempty"`
}

// Step is a single execution unit in a workflow.
type Step struct {
	ID      string            `yaml:"id" json:"id"`
	Agent   string            `yaml:"agent" json:"agent"`
	Input   string            `yaml:"input" json:"input"`
	Expects string            `yaml:"expects,omitempty" json:"expects,omitempty"`
	Extract map[string]string `yaml:"extract,omitempty" json:"extract,omitempty"`
	OnFail  *OnFail           `yaml:"on_fail,omitempty" json:"on_fail,omitempty"`
}

// OnFail describes retry and escalation behaviour when a step fails.
type OnFail struct {
	RetryStep   string     `yaml:"retry_step,omitempty" json:"retry_step,omitempty"`
	FeedbackVar string     `yaml:"feedback_var,omitempty" json:"feedback_var,omitempty"`
	MaxRetries  int        `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
	OnExhausted *Exhausted `yaml:"on_exhausted,omitempty" json:"on_exhausted,omitempty"`
}

// Exhausted describes what to do when all retries are exhausted.
type Exhausted struct {
	EscalateTo string `yaml:"escalate_to" json:"escalate_to"`
}
