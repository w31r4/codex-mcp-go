package codex

import (
	"context"
	"errors"
	"testing"

	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
)

func TestIsValidSandbox(t *testing.T) {
	tests := []struct {
		name    string
		sandbox string
		want    bool
	}{
		{"valid read-only", "read-only", true},
		{"valid workspace-write", "workspace-write", true},
		{"valid danger-full-access", "danger-full-access", true},
		{"invalid network-only", "network-only", false},
		{"invalid write (old value)", "write", false},
		{"invalid full (old value)", "full", false},
		{"invalid empty string", "", false},
		{"invalid case sensitive READ-ONLY", "READ-ONLY", false},
		{"invalid with spaces", " read-only", false},
		{"invalid random string", "sandbox", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSandbox(tt.sandbox); got != tt.want {
				t.Errorf("IsValidSandbox(%q) = %v, want %v", tt.sandbox, got, tt.want)
			}
		})
	}
}

func TestValidSandboxModes(t *testing.T) {
	// Ensure the slice contains exactly the expected values
	expected := []string{"read-only", "workspace-write", "danger-full-access"}
	if len(ValidSandboxModes) != len(expected) {
		t.Errorf("ValidSandboxModes has %d elements, want %d", len(ValidSandboxModes), len(expected))
	}
	for i, v := range expected {
		if ValidSandboxModes[i] != v {
			t.Errorf("ValidSandboxModes[%d] = %q, want %q", i, ValidSandboxModes[i], v)
		}
	}
}

func TestSandboxConstants(t *testing.T) {
	// Verify constants match expected values
	if SandboxReadOnly != "read-only" {
		t.Errorf("SandboxReadOnly = %q, want %q", SandboxReadOnly, "read-only")
	}
	if SandboxWorkspaceWrite != "workspace-write" {
		t.Errorf("SandboxWorkspaceWrite = %q, want %q", SandboxWorkspaceWrite, "workspace-write")
	}
	if SandboxDangerFullAccess != "danger-full-access" {
		t.Errorf("SandboxDangerFullAccess = %q, want %q", SandboxDangerFullAccess, "danger-full-access")
	}
}

func TestRun_InvalidSandbox_ReturnsStructuredError(t *testing.T) {
	_, err := Run(context.Background(), Options{
		Prompt:     "hi",
		WorkingDir: ".",
		Sandbox:    "network-only",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.InvalidSandboxMode {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.InvalidSandboxMode)
	}
	if cerr.Data["provided"] != "network-only" {
		t.Fatalf("data.provided=%v, want %v", cerr.Data["provided"], "network-only")
	}
}

func TestRun_CodexNotFound_ReturnsStructuredError(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := Run(context.Background(), Options{
		Prompt:     "hi",
		WorkingDir: ".",
		Sandbox:    SandboxReadOnly,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.CodexNotFound {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.CodexNotFound)
	}
}
