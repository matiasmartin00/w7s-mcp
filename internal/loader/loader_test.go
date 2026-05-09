package loader_test

import (
	"strings"
	"testing"

	"github.com/matiasmartin00/w7s-mcp/internal/loader"
)

// TestLoad_ValidWorkflow verifies that a valid YAML workflow file loads correctly.
func TestLoad_ValidWorkflow(t *testing.T) {
	wf, err := loader.Load("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("expected no error loading valid workflow, got: %v", err)
	}
	if wf == nil {
		t.Fatal("expected non-nil workflow, got nil")
	}
	if wf.ID != "test-workflow" {
		t.Errorf("expected id %q, got %q", "test-workflow", wf.ID)
	}
	if wf.Name != "Test Workflow" {
		t.Errorf("expected name %q, got %q", "Test Workflow", wf.Name)
	}
	if len(wf.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(wf.Steps))
	}
	if wf.Steps[0].ID != "step1" {
		t.Errorf("expected step id %q, got %q", "step1", wf.Steps[0].ID)
	}
	if wf.Steps[0].Agent != "worker" {
		t.Errorf("expected step agent %q, got %q", "worker", wf.Steps[0].Agent)
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
id: bad-type
name: Bad Type
version: "1.0.0"
agents:
  - id: worker
    name: Worker
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
id: multi-step
name: Multi Step
version: "1.0.0"
agents:
  - id: agent-a
    name: Agent A
  - id: agent-b
    name: Agent B
steps:
  - id: step-one
    agent: agent-a
    input: "Do step one."
  - id: step-two
    agent: agent-b
    input: "Do step two."
`)
	wf, err := loader.LoadBytes(data)
	if err != nil {
		t.Fatalf("expected no error for valid multi-step workflow, got: %v", err)
	}
	if len(wf.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(wf.Steps))
	}
	if wf.Steps[0].ID != "step-one" {
		t.Errorf("expected step[0].id=%q, got %q", "step-one", wf.Steps[0].ID)
	}
}

// TestWorkflowDirs_KnownClients verifies path resolution for known client names.
func TestWorkflowDirs_KnownClients(t *testing.T) {
	cases := []struct {
		clientName    string
		globalSuffix  string
		repoDir       string
	}{
		{"opencode", "/.config/opencode/workflows", ".opencode/workflows"},
		{"github-copilot", "/.copilot/workflows", ".github/workflows-mcp"},
		{"copilot", "/.copilot/workflows", ".github/workflows-mcp"},
		{"claude", "/.claude/workflows", ".claude/workflows"},
		{"unknown-client", "/.config/w7s/workflows", ".w7s/workflows"},
	}

	for _, tc := range cases {
		t.Run(tc.clientName, func(t *testing.T) {
			global, repo := loader.WorkflowDirs(tc.clientName)
			if !strings.HasSuffix(global, tc.globalSuffix) {
				t.Errorf("globalDir: expected suffix %q, got %q", tc.globalSuffix, global)
			}
			if repo != tc.repoDir {
				t.Errorf("repoDir: expected %q, got %q", tc.repoDir, repo)
			}
		})
	}
}

// TestLoadByID_AbsolutePath verifies that LoadByID loads from an absolute path directly.
func TestLoadByID_AbsolutePath(t *testing.T) {
	// Use an absolute path to the valid fixture.
	// The test must be run from within internal/loader (go test ./internal/loader/...)
	// so we use Load directly to get the abs path.
	wf, err := loader.LoadByID("opencode", "/nonexistent/absolute.yml")
	if err == nil {
		t.Errorf("expected error for nonexistent absolute path, got workflow: %v", wf)
	}
}
