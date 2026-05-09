// Package workflow defines the Go types for a w7s workflow definition.
package workflow

// Workflow is the top-level structure for a workflow YAML document.
type Workflow struct {
	APIVersion string           `yaml:"apiVersion" json:"apiVersion"`
	Kind       string           `yaml:"kind" json:"kind"`
	Metadata   WorkflowMeta     `yaml:"metadata" json:"metadata"`
	Inputs     []WorkflowInput  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Steps      []WorkflowStep   `yaml:"steps" json:"steps"`
	Outputs    []WorkflowOutput `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

// WorkflowMeta holds identifying metadata for a workflow.
type WorkflowMeta struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// WorkflowInput describes an input parameter accepted by a workflow.
type WorkflowInput struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type" json:"type"`
	Required bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

// WorkflowStep is a single execution unit in a workflow.
type WorkflowStep struct {
	ID        string         `yaml:"id" json:"id"`
	Tool      string         `yaml:"tool" json:"tool"`
	Arguments map[string]any `yaml:"arguments,omitempty" json:"arguments,omitempty"`
	DependsOn []string       `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
}

// WorkflowOutput describes a value exported by a workflow.
type WorkflowOutput struct {
	Name string `yaml:"name" json:"name"`
	From string `yaml:"from" json:"from"`
}
