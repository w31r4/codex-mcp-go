package metrics

import (
	"testing"
	"time"
)

func TestMetrics_RecordRequestAndSnapshot(t *testing.T) {
	m := New()

	m.RecordRequest("codex", true, 10*time.Millisecond)
	m.RecordRequest("codex", false, 5*time.Millisecond)
	m.RecordRequest("stats", true, 1*time.Millisecond)

	s := m.Snapshot()
	if s.TotalRequests != 3 {
		t.Fatalf("TotalRequests=%d, want %d", s.TotalRequests, 3)
	}
	if s.SuccessRequests != 2 {
		t.Fatalf("SuccessRequests=%d, want %d", s.SuccessRequests, 2)
	}
	if s.FailedRequests != 1 {
		t.Fatalf("FailedRequests=%d, want %d", s.FailedRequests, 1)
	}
	if s.MinLatencyMs != 1 {
		t.Fatalf("MinLatencyMs=%d, want %d", s.MinLatencyMs, 1)
	}
	if s.MaxLatencyMs != 10 {
		t.Fatalf("MaxLatencyMs=%d, want %d", s.MaxLatencyMs, 10)
	}
	if s.AvgLatencyMs != (10+5+1)/3 {
		t.Fatalf("AvgLatencyMs=%d, want %d", s.AvgLatencyMs, (10+5+1)/3)
	}

	if s.ToolCalls["codex"] != 2 {
		t.Fatalf("ToolCalls[codex]=%d, want %d", s.ToolCalls["codex"], 2)
	}
	if s.ToolCalls["stats"] != 1 {
		t.Fatalf("ToolCalls[stats]=%d, want %d", s.ToolCalls["stats"], 1)
	}
}

func TestMetrics_RecordError(t *testing.T) {
	m := New()
	m.RecordError("InvalidParams")
	m.RecordError("InvalidParams")
	m.RecordError("CodexNotFound")

	s := m.Snapshot()
	if s.ErrorCounts["InvalidParams"] != 2 {
		t.Fatalf("ErrorCounts[InvalidParams]=%d, want %d", s.ErrorCounts["InvalidParams"], 2)
	}
	if s.ErrorCounts["CodexNotFound"] != 1 {
		t.Fatalf("ErrorCounts[CodexNotFound]=%d, want %d", s.ErrorCounts["CodexNotFound"], 1)
	}
}
