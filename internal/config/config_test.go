package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_Validate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestSecurityConfig_IsModelAllowed(t *testing.T) {
	sec := SecurityConfig{AllowedModels: nil}
	if sec.IsModelAllowed("gpt-foo") {
		t.Fatalf("expected deny when allowlist is empty")
	}

	sec.AllowedModels = []string{"gpt-foo"}
	if !sec.IsModelAllowed("gpt-foo") {
		t.Fatalf("expected allowlisted model to be allowed")
	}
	if sec.IsModelAllowed("gpt-bar") {
		t.Fatalf("expected non-allowlisted model to be denied")
	}

	sec.AllowedModels = []string{"*"}
	if !sec.IsModelAllowed("anything") {
		t.Fatalf("expected wildcard allowlist to allow any value")
	}
}

func TestSecurityConfig_IsWorkDirAllowed_PrefixBoundary(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "tmp", "allowed")
	sec := SecurityConfig{AllowedWorkDirs: []string{base}}

	if !sec.IsWorkDirAllowed(filepath.Join(base, "repo")) {
		t.Fatalf("expected child to be allowed")
	}
	if sec.IsWorkDirAllowed(base + "2") {
		t.Fatalf("expected prefix lookalike to be denied")
	}
}

func TestLoadFromEnv(t *testing.T) {
	cfg := Default()
	t.Setenv(envServerName, "My Server")
	t.Setenv(envDefaultTimeout, "10")
	t.Setenv(envAllowedModels, "a,b")

	cfg.LoadFromEnv()

	if cfg.Server.Name != "My Server" {
		t.Fatalf("server.name=%q, want %q", cfg.Server.Name, "My Server")
	}
	if cfg.Codex.DefaultTimeoutSeconds != 10 {
		t.Fatalf("codex.default_timeout_seconds=%d, want %d", cfg.Codex.DefaultTimeoutSeconds, 10)
	}
	if len(cfg.Security.AllowedModels) != 2 || cfg.Security.AllowedModels[0] != "a" || cfg.Security.AllowedModels[1] != "b" {
		t.Fatalf("allowed_models=%v, want [a b]", cfg.Security.AllowedModels)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.toml")
	if err := os.WriteFile(path, []byte(`
[server]
name = "X"
version = "1.2.3"

[codex]
default_timeout_seconds = 12
max_timeout_seconds = 34

[security]
allowed_models = ["*"]
default_sandbox = "read-only"
allowed_sandbox_modes = ["read-only"]

[logging]
level = "info"
format = "json"
output = "stderr"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Server.Name != "X" || cfg.Server.Version != "1.2.3" {
		t.Fatalf("server=%+v", cfg.Server)
	}
	if cfg.Codex.DefaultTimeoutSeconds != 12 || cfg.Codex.MaxTimeoutSeconds != 34 {
		t.Fatalf("codex=%+v", cfg.Codex)
	}
	if !cfg.Security.IsModelAllowed("anything") {
		t.Fatalf("expected wildcard models to allow any")
	}
}
