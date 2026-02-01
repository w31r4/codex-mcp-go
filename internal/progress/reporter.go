package progress

import (
	"context"
	"sync/atomic"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Reporter emits best-effort progress updates for long-running operations.
//
// Implementations MUST NOT panic and MUST NOT return errors to callers; failures
// to notify are best-effort.
type Reporter interface {
	Report(ctx context.Context, message string)
}

type nopReporter struct{}

func (nopReporter) Report(context.Context, string) {}

// Nop is a Reporter that does nothing.
var Nop Reporter = nopReporter{}

type ProgressNotifier interface {
	NotifyProgress(ctx context.Context, params *mcpsdk.ProgressNotificationParams) error
}

// MCPReporter emits MCP notifications/progress for a given progress token.
// Each call to Report increments the progress counter by 1 (monotonic).
type MCPReporter struct {
	notifier ProgressNotifier
	token    any
	seq      atomic.Uint64
}

func NewMCPReporter(notifier ProgressNotifier, progressToken any) Reporter {
	if notifier == nil || progressToken == nil {
		return Nop
	}
	return &MCPReporter{
		notifier: notifier,
		token:    progressToken,
	}
}

func (r *MCPReporter) Report(ctx context.Context, message string) {
	if r == nil || r.notifier == nil || r.token == nil {
		return
	}

	progress := float64(r.seq.Add(1))
	_ = r.notifier.NotifyProgress(ctx, &mcpsdk.ProgressNotificationParams{
		ProgressToken: r.token,
		Message:       message,
		Progress:      progress,
		Total:         0,
	})
}

