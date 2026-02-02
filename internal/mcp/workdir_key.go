package mcp

import (
	"context"
	"path/filepath"
	"time"

	"github.com/w31r4/codex-mcp-go/internal/receipt"
)

func workdirKey(ctx context.Context, cd string) string {
	cd = normalizeWorkdir(cd)

	// Prefer git root when available so concurrent runs in the same repo
	// (even from different subdirs) are mutually excluded.
	if root, ok, err := receipt.GitRoot(ctx, cd, 2*time.Second); err == nil && ok {
		if normalized := normalizeWorkdir(root); normalized != "" {
			return normalized
		}
	}
	return cd
}

func normalizeWorkdir(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil && abs != "" {
		path = abs
	}
	path = filepath.Clean(path)
	if real, err := filepath.EvalSymlinks(path); err == nil && real != "" {
		path = real
	}
	return filepath.Clean(path)
}
