package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/w31r4/codex-mcp-go/internal/codex"
)

// ApplySafeLocalPreset enforces safer defaults for local usage.
//
// It is intentionally conservative, but avoids breaking existing setups:
// - Always enforces read-only as the default sandbox.
// - Always disables yolo.
// - Restricts allowed work dirs only when:
//   - safeLocalRoot is provided, or
//   - allowed work dirs are currently empty (then defaults to $HOME).
func ApplySafeLocalPreset(cfg *Config, safeLocalRoot string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	cfg.Security.DefaultSandbox = codex.SandboxReadOnly
	cfg.Security.DisableYolo = true

	safeLocalRoot = strings.TrimSpace(safeLocalRoot)
	if safeLocalRoot != "" {
		cfg.Security.AllowedWorkDirs = splitCSV(safeLocalRoot)
		return nil
	}
	if len(cfg.Security.AllowedWorkDirs) > 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		cfg.Security.AllowedWorkDirs = []string{filepath.Clean(home)}
		return nil
	}

	// Best-effort fallback: allow current directory only.
	cfg.Security.AllowedWorkDirs = []string{"."}
	return nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
