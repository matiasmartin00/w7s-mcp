// Package loader provides functions to load and validate workflow YAML files.
package loader

import (
	_ "embed"
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/matiasmartin00/w7s-mcp/internal/workflow"
)

//go:embed workflow.schema.json
var schemaBytes []byte

// LoadByID resolves and loads a workflow by its ID or absolute path,
// searching directories determined by the MCP clientName.
func LoadByID(clientName, workflowID string) (*workflow.Workflow, error) {
	globalDir, repoDir := WorkflowDirs(clientName)
	return LoadByIDFromDirs(globalDir, repoDir, workflowID)
}

// LoadByIDFromDirs resolves and loads a workflow by ID using explicit directory paths.
// This variant is intended for testing, allowing callers to inject temp dirs without
// touching real home-directory paths.
//
// Resolution order:
//  1. Absolute path — if workflowID starts with '/', use it directly.
//  2. Global dir   — {globalDir}/{workflowID}.yml  (takes precedence over repo dir).
//  3. Repo dir     — {repoDir}/{workflowID}.yml
func LoadByIDFromDirs(globalDir, repoDir, workflowID string) (*workflow.Workflow, error) {
	// 1. Absolute path: if workflowID looks like an absolute path, try directly.
	if len(workflowID) > 0 && workflowID[0] == '/' {
		if _, err := os.Stat(workflowID); err == nil {
			return Load(workflowID)
		}
		return nil, fmt.Errorf("workflow file not found at absolute path %q; ensure the file exists", workflowID)
	}

	// 2. Global dir: {globalDir}/{workflowID}.yml
	globalPath := globalDir + "/" + workflowID + ".yml"
	if _, err := os.Stat(globalPath); err == nil {
		return Load(globalPath)
	}

	// 3. Repo dir: {repoDir}/{workflowID}.yml (relative to cwd)
	repoPath := repoDir + "/" + workflowID + ".yml"
	if _, err := os.Stat(repoPath); err == nil {
		return Load(repoPath)
	}

	return nil, fmt.Errorf(
		"workflow %q not found; searched %q and %q — create a file named %s.yml in one of those directories",
		workflowID, globalDir, repoDir, workflowID,
	)
}

// Load reads a YAML workflow file from path, validates it against the JSON Schema,
// and returns a typed *workflow.Workflow. It wraps any underlying error with context.
func Load(path string) (*workflow.Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading workflow file: %w", err)
	}
	return LoadBytes(data)
}

// LoadBytes parses and validates a YAML workflow from an in-memory byte slice.
// The pipeline is: YAML → map[string]any → JSON → schema validation → typed struct.
func LoadBytes(data []byte) (*workflow.Workflow, error) {
	// 1. Parse YAML into a generic map to enable JSON schema validation.
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// 2. Convert to JSON.
	jsonData, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("converting to JSON: %w", err)
	}

	// 3. Validate the JSON document against the embedded schema.
	if err := validateSchema(jsonData); err != nil {
		return nil, err
	}

	// 4. Unmarshal YAML into the typed workflow struct.
	var wf workflow.Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("decoding workflow: %w", err)
	}
	return &wf, nil
}

// validateSchema compiles the embedded JSON Schema and validates jsonData against it.
func validateSchema(jsonData []byte) error {
	compiler := jsonschema.NewCompiler()

	// UnmarshalJSON parses the schema bytes into the format expected by AddResource.
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return fmt.Errorf("parsing embedded schema: %w", err)
	}

	if err := compiler.AddResource("workflow.schema.json", schemaDoc); err != nil {
		return fmt.Errorf("loading embedded schema: %w", err)
	}

	schema, err := compiler.Compile("workflow.schema.json")
	if err != nil {
		return fmt.Errorf("compiling schema: %w", err)
	}

	// UnmarshalJSON parses the document for schema validation.
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("preparing document for validation: %w", err)
	}

	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}
	return nil
}
