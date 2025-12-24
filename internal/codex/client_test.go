package codex

import "testing"

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
