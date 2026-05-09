package loader_test

import (
	"strings"
	"testing"

	"github.com/matiasmartin00/w7s-mcp/internal/loader"
)

// TestLoad_ValidWorkflow verifies that a valid YAML workflow file loads correctly.
// RED: loader.Load does not exist yet — this test references it to establish contract.
func TestLoad_ValidWorkflow(t *testing.T) {
	wf, err := loader.Load("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("expected no error loading valid workflow, got: %v", err)
	}
	if wf == nil {
		t.Fatal("expected non-nil workflow, got nil")
	}
	if wf.Metadata.Name != "greet-user" {
		t.Errorf("expected metadata.name %q, got %q", "greet-user", wf.Metadata.Name)
	}
	if len(wf.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(wf.Steps))
	}
	if wf.Steps[0].ID != "step-greet" {
		t.Errorf("expected step id %q, got %q", "step-greet", wf.Steps[0].ID)
	}
	if wf.Steps[0].Tool != "hello_world" {
		t.Errorf("expected step tool %q, got %q", "hello_world", wf.Steps[0].Tool)
	}
}

// TestLoad_MissingRequiredField verifies that a workflow missing the required "steps" field
// returns a validation error.
func TestLoad_MissingRequiredField(t *testing.T) {
	_, err := loader.Load("testdata/invalid_missing_steps.yaml")
	if err == nil {
		t.Fatal("expected error for workflow missing steps, got nil")
	}
	if !strings.Contains(err.Error(), "steps") {
		t.Errorf("expected error to mention 'steps', got: %v", err)
	}
}

// TestLoad_FileNotFound verifies that loading a non-existent file returns an error.
func TestLoad_FileNotFound(t *testing.T) {
	_, err := loader.Load("testdata/does_not_exist.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestLoadBytes_WrongType verifies that passing invalid YAML (wrong type for a field)
// returns a validation error.
func TestLoadBytes_WrongType(t *testing.T) {
	// steps must be an array; providing a string should fail schema validation.
	data := []byte(`
apiVersion: workflow.w7s.io/v1alpha1
kind: Workflow
metadata:
  name: bad-type
steps: "not-an-array"
`)
	_, err := loader.LoadBytes(data)
	if err == nil {
		t.Fatal("expected error for wrong type in steps field, got nil")
	}
}

// TestLoadBytes_ValidWorkflow is a triangulation test: verifies LoadBytes independently
// from the file system for a different valid workflow (multiple steps).
func TestLoadBytes_ValidWorkflow(t *testing.T) {
	data := []byte(`
apiVersion: workflow.w7s.io/v1alpha1
kind: Workflow
metadata:
  name: multi-step
steps:
  - id: step-one
    tool: tool_a
  - id: step-two
    tool: tool_b
    dependsOn:
      - step-one
`)
	wf, err := loader.LoadBytes(data)
	if err != nil {
		t.Fatalf("expected no error for valid multi-step workflow, got: %v", err)
	}
	if len(wf.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(wf.Steps))
	}
	if wf.Steps[1].DependsOn[0] != "step-one" {
		t.Errorf("expected dependsOn[0]=%q, got %q", "step-one", wf.Steps[1].DependsOn[0])
	}
}
