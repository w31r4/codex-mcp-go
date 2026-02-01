package progress

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeNotifier struct {
	calls []*mcpsdk.ProgressNotificationParams
	err   error
}

func (f *fakeNotifier) NotifyProgress(ctx context.Context, params *mcpsdk.ProgressNotificationParams) error {
	f.calls = append(f.calls, params)
	return f.err
}

func TestNopReporter_DoesNothing(t *testing.T) {
	Nop.Report(context.Background(), "hi")
}

func TestMCPReporter_MonotonicProgress(t *testing.T) {
	fn := &fakeNotifier{}
	r := NewMCPReporter(fn, "tok")

	r.Report(context.Background(), "starting")
	r.Report(context.Background(), "running")

	if len(fn.calls) != 2 {
		t.Fatalf("calls=%d, want 2", len(fn.calls))
	}
	if fn.calls[0].ProgressToken != "tok" {
		t.Fatalf("token=%v, want %q", fn.calls[0].ProgressToken, "tok")
	}
	if fn.calls[0].Progress != 1 || fn.calls[1].Progress != 2 {
		t.Fatalf("progress=[%v,%v], want [1,2]", fn.calls[0].Progress, fn.calls[1].Progress)
	}
	if fn.calls[0].Message != "starting" || fn.calls[1].Message != "running" {
		t.Fatalf("messages=[%q,%q], want [%q,%q]", fn.calls[0].Message, fn.calls[1].Message, "starting", "running")
	}
}

