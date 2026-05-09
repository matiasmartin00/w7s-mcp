package domain_test

import (
	"regexp"
	"testing"

	"github.com/matiasmartin00/w7s-mcp/internal/domain"
)

// ---------------------------------------------------------------------------
// RunStatus and StepStatus constants
// ---------------------------------------------------------------------------

func TestRunStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      domain.RunStatus
		expected string
	}{
		{"running", domain.RunStatusRunning, "running"},
		{"done", domain.RunStatusDone, "done"},
		{"escalated", domain.RunStatusEscalated, "escalated"},
		{"failed", domain.RunStatusFailed, "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("RunStatus %q: got %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestStepStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      domain.StepStatus
		expected string
	}{
		{"pending", domain.StepStatusPending, "pending"},
		{"running", domain.StepStatusRunning, "running"},
		{"done", domain.StepStatusDone, "done"},
		{"failed", domain.StepStatusFailed, "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.got) != tt.expected {
				t.Errorf("StepStatus %q: got %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewRunID
// ---------------------------------------------------------------------------

var uuidV4Re = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewRunID_Format(t *testing.T) {
	id := domain.NewRunID()
	if !uuidV4Re.MatchString(id) {
		t.Errorf("NewRunID() = %q; want UUID v4 format", id)
	}
}

func TestNewRunID_Unique(t *testing.T) {
	a := domain.NewRunID()
	b := domain.NewRunID()
	if a == b {
		t.Errorf("NewRunID() returned duplicate IDs: %q", a)
	}
}

// ---------------------------------------------------------------------------
// ValidationError
// ---------------------------------------------------------------------------

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		ve       domain.ValidationError
		expected string
	}{
		{
			name:     "basic",
			ve:       domain.ValidationError{Field: "run_id", Code: domain.ErrCodeRequired, Message: "run_id is required"},
			expected: "run_id: required: run_id is required",
		},
		{
			name:     "not_found",
			ve:       domain.ValidationError{Field: "workflow_id", Code: domain.ErrCodeNotFound, Message: "workflow not found"},
			expected: "workflow_id: not_found: workflow not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ve.Error(); got != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StartRunRequest.Validate
// ---------------------------------------------------------------------------

func TestStartRunRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.StartRunRequest
		wantErr bool
		errCode string
		errField string
	}{
		{
			name:    "valid",
			req:     domain.StartRunRequest{WorkflowID: "wf-1", Task: "do something"},
			wantErr: false,
		},
		{
			name:     "missing workflow_id",
			req:      domain.StartRunRequest{Task: "do something"},
			wantErr:  true,
			errCode:  domain.ErrCodeRequired,
			errField: "workflow_id",
		},
		{
			name:     "missing task",
			req:      domain.StartRunRequest{WorkflowID: "wf-1"},
			wantErr:  true,
			errCode:  domain.ErrCodeRequired,
			errField: "task",
		},
		{
			name:     "both missing",
			req:      domain.StartRunRequest{},
			wantErr:  true,
			errCode:  domain.ErrCodeRequired,
			errField: "workflow_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				ve, ok := err.(domain.ValidationError)
				if !ok {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if ve.Code != tt.errCode {
					t.Errorf("Code = %q, want %q", ve.Code, tt.errCode)
				}
				if ve.Field != tt.errField {
					t.Errorf("Field = %q, want %q", ve.Field, tt.errField)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetNextStepRequest.Validate
// ---------------------------------------------------------------------------

func TestGetNextStepRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.GetNextStepRequest
		wantErr bool
	}{
		{"valid", domain.GetNextStepRequest{RunID: "run-123"}, false},
		{"missing run_id", domain.GetNextStepRequest{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				ve, ok := err.(domain.ValidationError)
				if !ok {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if ve.Field != "run_id" {
					t.Errorf("Field = %q, want run_id", ve.Field)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CompleteStepRequest.Validate
// ---------------------------------------------------------------------------

func TestCompleteStepRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      domain.CompleteStepRequest
		wantErr  bool
		errField string
	}{
		{
			name:    "valid",
			req:     domain.CompleteStepRequest{RunID: "r", StepID: "s", Output: "out"},
			wantErr: false,
		},
		{
			name:     "missing run_id",
			req:      domain.CompleteStepRequest{StepID: "s", Output: "out"},
			wantErr:  true,
			errField: "run_id",
		},
		{
			name:     "missing step_id",
			req:      domain.CompleteStepRequest{RunID: "r", Output: "out"},
			wantErr:  true,
			errField: "step_id",
		},
		{
			name:     "missing output",
			req:      domain.CompleteStepRequest{RunID: "r", StepID: "s"},
			wantErr:  true,
			errField: "output",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				ve, ok := err.(domain.ValidationError)
				if !ok {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if ve.Field != tt.errField {
					t.Errorf("Field = %q, want %q", ve.Field, tt.errField)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FailStepRequest.Validate
// ---------------------------------------------------------------------------

func TestFailStepRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      domain.FailStepRequest
		wantErr  bool
		errField string
	}{
		{
			name:    "valid without reason",
			req:     domain.FailStepRequest{RunID: "r", StepID: "s"},
			wantErr: false,
		},
		{
			name:    "valid with reason",
			req:     domain.FailStepRequest{RunID: "r", StepID: "s", Reason: "oops"},
			wantErr: false,
		},
		{
			name:     "missing run_id",
			req:      domain.FailStepRequest{StepID: "s"},
			wantErr:  true,
			errField: "run_id",
		},
		{
			name:     "missing step_id",
			req:      domain.FailStepRequest{RunID: "r"},
			wantErr:  true,
			errField: "step_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				ve, ok := err.(domain.ValidationError)
				if !ok {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if ve.Field != tt.errField {
					t.Errorf("Field = %q, want %q", ve.Field, tt.errField)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetRunStatusRequest.Validate
// ---------------------------------------------------------------------------

func TestGetRunStatusRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.GetRunStatusRequest
		wantErr bool
	}{
		{"valid run_id only", domain.GetRunStatusRequest{RunID: "r"}, false},
		{"valid workflow_id only", domain.GetRunStatusRequest{WorkflowID: "wf-1"}, false},
		{"valid both", domain.GetRunStatusRequest{RunID: "r", WorkflowID: "wf-1"}, false},
		{"neither provided", domain.GetRunStatusRequest{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				ve, ok := err.(domain.ValidationError)
				if !ok {
					t.Fatalf("expected ValidationError, got %T", err)
				}
				if ve.Code != domain.ErrCodeRequired {
					t.Errorf("Code = %q, want %q", ve.Code, domain.ErrCodeRequired)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Error code constants
// ---------------------------------------------------------------------------

func TestErrorCodeConstants(t *testing.T) {
	if domain.ErrCodeRequired != "required" {
		t.Errorf("ErrCodeRequired = %q, want \"required\"", domain.ErrCodeRequired)
	}
	if domain.ErrCodeInvalid != "invalid" {
		t.Errorf("ErrCodeInvalid = %q, want \"invalid\"", domain.ErrCodeInvalid)
	}
	if domain.ErrCodeNotFound != "not_found" {
		t.Errorf("ErrCodeNotFound = %q, want \"not_found\"", domain.ErrCodeNotFound)
	}
	if domain.ErrCodeInvalidState != "invalid_state" {
		t.Errorf("ErrCodeInvalidState = %q, want \"invalid_state\"", domain.ErrCodeInvalidState)
	}
}
