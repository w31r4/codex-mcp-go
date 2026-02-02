package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/w31r4/codex-mcp-go/internal/codex"
)

func TestApplySafeLocalPreset_OverridesSandboxAndYolo(t *testing.T) {
	cfg := Default()
	cfg.Security.DefaultSandbox = codex.SandboxDangerFullAccess
	cfg.Security.DisableYolo = false

	cfg.Security.AllowedWorkDirs = []string{"/tmp"}

	if err := ApplySafeLocalPreset(cfg, ""); err != nil {
		t.Fatalf("ApplySafeLocalPreset() failed: %v", err)
	}
	if cfg.Security.DefaultSandbox != codex.SandboxReadOnly {
		t.Fatalf("default_sandbox=%q, want %q", cfg.Security.DefaultSandbox, codex.SandboxReadOnly)
	}
	if cfg.Security.DisableYolo != true {
		t.Fatalf("disable_yolo=%v, want true", cfg.Security.DisableYolo)
	}
	// Existing allowed work dirs should remain unless a root override is provided.
	if len(cfg.Security.AllowedWorkDirs) != 1 || cfg.Security.AllowedWorkDirs[0] != "/tmp" {
		t.Fatalf("allowed_work_dirs=%v, want [/tmp]", cfg.Security.AllowedWorkDirs)
	}
}

func TestApplySafeLocalPreset_RootOverride(t *testing.T) {
	cfg := Default()
	if err := ApplySafeLocalPreset(cfg, "/a,/b"); err != nil {
		t.Fatalf("ApplySafeLocalPreset() failed: %v", err)
	}
	if len(cfg.Security.AllowedWorkDirs) != 2 || cfg.Security.AllowedWorkDirs[0] != "/a" || cfg.Security.AllowedWorkDirs[1] != "/b" {
		t.Fatalf("allowed_work_dirs=%v, want [/a /b]", cfg.Security.AllowedWorkDirs)
	}
}

func TestApplySafeLocalPreset_DefaultsWorkDirWhenEmpty(t *testing.T) {
	cfg := Default()
	cfg.Security.AllowedWorkDirs = nil

	if err := ApplySafeLocalPreset(cfg, ""); err != nil {
		t.Fatalf("ApplySafeLocalPreset() failed: %v", err)
	}
	if len(cfg.Security.AllowedWorkDirs) == 0 {
		t.Fatalf("expected allowed_work_dirs to be set")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		if cfg.Security.AllowedWorkDirs[0] != filepath.Clean(home) {
			t.Fatalf("allowed_work_dirs[0]=%q, want home %q", cfg.Security.AllowedWorkDirs[0], filepath.Clean(home))
		}
		return
	}
	if cfg.Security.AllowedWorkDirs[0] != "." {
		t.Fatalf("allowed_work_dirs[0]=%q, want %q", cfg.Security.AllowedWorkDirs[0], ".")
	}
}
