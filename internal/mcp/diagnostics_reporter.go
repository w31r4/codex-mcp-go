package mcp

import (
	"context"
	"strings"

	"github.com/w31r4/codex-mcp-go/internal/progress"
	"github.com/w31r4/codex-mcp-go/internal/session"
)

// diagnosticsReporter mirrors progress updates into the session manager as diagnostic events.
// It MUST be best-effort and never panic.
type diagnosticsReporter struct {
	next         progress.Reporter
	getSessionID func() string
}

func (r diagnosticsReporter) Report(ctx context.Context, message string) {
	defer func() { _ = recover() }()
	if r.getSessionID != nil {
		if id := strings.TrimSpace(r.getSessionID()); id != "" {
			globalSessions.AppendDiagnostic(id, session.DiagnosticProgress, message)
		}
	}
	if r.next != nil {
		r.next.Report(ctx, message)
	}
}
