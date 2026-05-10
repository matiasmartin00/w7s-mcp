package parser_test

import (
	"testing"

	"github.com/matiasmartin00/w7s-mcp/internal/parser"
)

// ── Interpolate ──────────────────────────────────────────────────────────────

func TestInterpolate_ReplacesKnownVariables(t *testing.T) {
	t.Parallel()
	tmpl := "Task: {{task}}\nScope: {{scope}}"
	vars := map[string]string{"task": "implement X", "scope": "auth module"}
	got := parser.Interpolate(tmpl, vars)
	want := "Task: implement X\nScope: auth module"
	if got != want {
		t.Errorf("Interpolate() = %q, want %q", got, want)
	}
}

func TestInterpolate_LeavesUnknownPlaceholderUnchanged(t *testing.T) {
	t.Parallel()
	tmpl := "Task: {{task}}\nFiles: {{files}}"
	vars := map[string]string{"task": "implement X"}
	got := parser.Interpolate(tmpl, vars)
	want := "Task: implement X\nFiles: {{files}}"
	if got != want {
		t.Errorf("Interpolate() = %q, want %q", got, want)
	}
}

func TestInterpolate_EmptyVarsLeavesAllPlaceholders(t *testing.T) {
	t.Parallel()
	tmpl := "{{a}} {{b}}"
	got := parser.Interpolate(tmpl, map[string]string{})
	if got != tmpl {
		t.Errorf("Interpolate() = %q, want %q", got, tmpl)
	}
}

func TestInterpolate_NoPlaceholdersReturnedAsIs(t *testing.T) {
	t.Parallel()
	tmpl := "no placeholders here"
	got := parser.Interpolate(tmpl, map[string]string{"x": "y"})
	if got != tmpl {
		t.Errorf("Interpolate() = %q, want %q", got, tmpl)
	}
}

func TestInterpolate_RepeatedPlaceholder(t *testing.T) {
	t.Parallel()
	tmpl := "{{x}} and {{x}}"
	vars := map[string]string{"x": "hello"}
	got := parser.Interpolate(tmpl, vars)
	want := "hello and hello"
	if got != want {
		t.Errorf("Interpolate() = %q, want %q", got, want)
	}
}

func TestInterpolate_EmptyTemplate(t *testing.T) {
	t.Parallel()
	got := parser.Interpolate("", map[string]string{"x": "y"})
	if got != "" {
		t.Errorf("Interpolate() = %q, want empty string", got)
	}
}

func TestInterpolateStrict_ReplacesKnownVariables(t *testing.T) {
	t.Parallel()
	tmpl := "Task: {{task}}\nRun: {{run_id}}"
	vars := map[string]string{"task": "implement X", "run_id": "run-123"}
	got, err := parser.InterpolateStrict(tmpl, vars)
	if err != nil {
		t.Fatalf("InterpolateStrict() unexpected error: %v", err)
	}
	want := "Task: implement X\nRun: run-123"
	if got != want {
		t.Errorf("InterpolateStrict() = %q, want %q", got, want)
	}
}

func TestInterpolateStrict_ReturnsErrorOnMissingVariable(t *testing.T) {
	t.Parallel()
	tmpl := "Task: {{task}}\nScope: {{scope}}"
	vars := map[string]string{"task": "implement X"}
	_, err := parser.InterpolateStrict(tmpl, vars)
	if err == nil {
		t.Fatal("InterpolateStrict() expected error, got nil")
	}
	missingErr, ok := err.(parser.ErrMissingVariable)
	if !ok {
		t.Fatalf("InterpolateStrict() error type = %T, want parser.ErrMissingVariable", err)
	}
	if missingErr.Variable != "scope" {
		t.Fatalf("missing variable = %q, want %q", missingErr.Variable, "scope")
	}
}

// ── Extract ──────────────────────────────────────────────────────────────────

func TestExtract_CapturesMatchingVariables(t *testing.T) {
	t.Parallel()
	output := "STATUS: done\nSCOPE: auth module\nFILES: src/auth.go"
	patterns := map[string]string{
		"scope": `SCOPE:\s*(.+)`,
		"files": `FILES:\s*(.+)`,
	}
	got := parser.Extract(output, patterns)
	if got["scope"] != "auth module" {
		t.Errorf("Extract scope = %q, want %q", got["scope"], "auth module")
	}
	if got["files"] != "src/auth.go" {
		t.Errorf("Extract files = %q, want %q", got["files"], "src/auth.go")
	}
}

func TestExtract_NonMatchingPatternNotIncluded(t *testing.T) {
	t.Parallel()
	output := "STATUS: done"
	patterns := map[string]string{
		"scope": `SCOPE:\s*(.+)`,
	}
	got := parser.Extract(output, patterns)
	if _, ok := got["scope"]; ok {
		t.Errorf("Extract: unexpected key 'scope' in result")
	}
}

func TestExtract_InvalidRegexSkipped(t *testing.T) {
	t.Parallel()
	output := "SCOPE: auth"
	patterns := map[string]string{
		"bad":   `[invalid`,
		"scope": `SCOPE:\s*(.+)`,
	}
	got := parser.Extract(output, patterns)
	if _, ok := got["bad"]; ok {
		t.Errorf("Extract: invalid regex key should not appear in result")
	}
	if got["scope"] != "auth" {
		t.Errorf("Extract scope = %q, want %q", got["scope"], "auth")
	}
}

func TestExtract_EmptyOutputReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	got := parser.Extract("", map[string]string{"x": `X:\s*(.+)`})
	if len(got) != 0 {
		t.Errorf("Extract on empty output: expected empty map, got %v", got)
	}
}

func TestExtract_EmptyPatternsReturnsEmptyMap(t *testing.T) {
	t.Parallel()
	got := parser.Extract("some output", map[string]string{})
	if len(got) != 0 {
		t.Errorf("Extract with no patterns: expected empty map, got %v", got)
	}
}

func TestExtract_NoCaptureGroupSkipped(t *testing.T) {
	t.Parallel()
	// Pattern with no capture group
	output := "SCOPE: auth"
	patterns := map[string]string{
		"scope": `SCOPE:\s*.+`, // no capture group → len(m) < 2
	}
	got := parser.Extract(output, patterns)
	if _, ok := got["scope"]; ok {
		t.Errorf("Extract: pattern without capture group should not produce a variable")
	}
}

func TestExtract_EmptyCaptureSkipped(t *testing.T) {
	t.Parallel()
	// Pattern matches but capture is empty string after trim
	output := "SCOPE:   "
	patterns := map[string]string{
		"scope": `SCOPE:\s*(.*)`,
	}
	got := parser.Extract(output, patterns)
	if _, ok := got["scope"]; ok {
		t.Errorf("Extract: empty capture group should not produce a variable")
	}
}

func TestExtract_MultilineOutput(t *testing.T) {
	t.Parallel()
	output := "Some preamble\nSTATUS: done\nCHANGES: refactored auth and added tests\nMore text"
	patterns := map[string]string{
		"changes": `CHANGES:\s*(.+)`,
	}
	got := parser.Extract(output, patterns)
	if got["changes"] != "refactored auth and added tests" {
		t.Errorf("Extract changes = %q, want %q", got["changes"], "refactored auth and added tests")
	}
}
