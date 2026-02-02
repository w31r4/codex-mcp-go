package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/w31r4/codex-mcp-go/internal/codex"
	"github.com/w31r4/codex-mcp-go/internal/logging"
)

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Codex    CodexConfig    `toml:"codex"`
	Security SecurityConfig `toml:"security"`
	Logging  logging.Config `toml:"logging"`
}

type ServerConfig struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type CodexConfig struct {
	// Timeouts are in seconds.
	DefaultTimeoutSeconds         int `toml:"default_timeout_seconds"`
	MaxTimeoutSeconds             int `toml:"max_timeout_seconds"`
	DefaultNoOutputTimeoutSeconds int `toml:"default_no_output_timeout_seconds"`

	MaxBufferedLines int    `toml:"max_buffered_lines"`
	ExecutablePath   string `toml:"executable_path"`
}

type SecurityConfig struct {
	AllowedModels       []string `toml:"allowed_models"`
	AllowedProfiles     []string `toml:"allowed_profiles"`
	DefaultSandbox      string   `toml:"default_sandbox"`
	AllowedSandboxModes []string `toml:"allowed_sandbox_modes"`
	AllowedWorkDirs     []string `toml:"allowed_work_dirs"`
	DisableYolo         bool     `toml:"disable_yolo"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Name:    "Codex MCP Server-from guda.studio",
			Version: "0.2.0",
		},
		Codex: CodexConfig{
			DefaultTimeoutSeconds:         1800,
			MaxTimeoutSeconds:             1800,
			DefaultNoOutputTimeoutSeconds: 0,
			MaxBufferedLines:              100,
			ExecutablePath:                "",
		},
		Security: SecurityConfig{
			AllowedModels:       nil, // deny all by default
			AllowedProfiles:     nil, // deny all by default
			DefaultSandbox:      codex.SandboxReadOnly,
			AllowedSandboxModes: []string{codex.SandboxReadOnly, codex.SandboxWorkspaceWrite, codex.SandboxDangerFullAccess},
			AllowedWorkDirs:     nil, // allow all by default
			DisableYolo:         false,
		},
		Logging: logging.DefaultConfig(),
	}
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	if strings.TrimSpace(c.Server.Name) == "" {
		return fmt.Errorf("server.name is required")
	}
	if strings.TrimSpace(c.Server.Version) == "" {
		return fmt.Errorf("server.version is required")
	}

	if c.Codex.DefaultTimeoutSeconds < 0 {
		return fmt.Errorf("codex.default_timeout_seconds must be >= 0")
	}
	if c.Codex.MaxTimeoutSeconds <= 0 {
		return fmt.Errorf("codex.max_timeout_seconds must be > 0")
	}
	if c.Codex.DefaultNoOutputTimeoutSeconds < 0 {
		return fmt.Errorf("codex.default_no_output_timeout_seconds must be >= 0")
	}
	if c.Codex.MaxBufferedLines < 0 {
		return fmt.Errorf("codex.max_buffered_lines must be >= 0")
	}

	if c.Security.DefaultSandbox == "" {
		return fmt.Errorf("security.default_sandbox is required")
	}
	if !codex.IsValidSandbox(c.Security.DefaultSandbox) {
		return fmt.Errorf("security.default_sandbox must be one of %v", codex.ValidSandboxModes)
	}
	if len(c.Security.AllowedSandboxModes) == 0 {
		return fmt.Errorf("security.allowed_sandbox_modes must not be empty")
	}
	for _, mode := range c.Security.AllowedSandboxModes {
		if !codex.IsValidSandbox(mode) {
			return fmt.Errorf("security.allowed_sandbox_modes contains invalid value %q (valid: %v)", mode, codex.ValidSandboxModes)
		}
	}
	if !containsString(c.Security.AllowedSandboxModes, c.Security.DefaultSandbox) {
		return fmt.Errorf("security.default_sandbox %q must be included in security.allowed_sandbox_modes", c.Security.DefaultSandbox)
	}

	for _, dir := range c.Security.AllowedWorkDirs {
		if strings.TrimSpace(dir) == "" {
			return fmt.Errorf("security.allowed_work_dirs contains an empty entry")
		}
	}

	if strings.EqualFold(strings.TrimSpace(c.Logging.Output), "file") && strings.TrimSpace(c.Logging.FilePath) == "" {
		return fmt.Errorf("logging.file_path is required when logging.output=file")
	}

	return nil
}

func (s SecurityConfig) IsModelAllowed(model string) bool {
	return isAllowlisted(s.AllowedModels, model)
}

func (s SecurityConfig) IsProfileAllowed(profile string) bool {
	return isAllowlisted(s.AllowedProfiles, profile)
}

func (s SecurityConfig) IsSandboxAllowed(mode string) bool {
	return containsString(s.AllowedSandboxModes, mode)
}

func (s SecurityConfig) IsWorkDirAllowed(workDir string) bool {
	if len(s.AllowedWorkDirs) == 0 {
		return true
	}

	path := filepath.Clean(workDir)
	for _, prefix := range s.AllowedWorkDirs {
		prefix = filepath.Clean(prefix)
		if prefix == "." || prefix == string(filepath.Separator) {
			return true
		}
		if path == prefix {
			return true
		}
		sep := string(filepath.Separator)
		if strings.HasPrefix(path, prefix+sep) {
			return true
		}
	}
	return false
}

func isAllowlisted(allowlist []string, value string) bool {
	if value == "" {
		return true
	}
	if len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" {
			return true
		}
		if allowed == value {
			return true
		}
	}
	return false
}

func containsString(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}
