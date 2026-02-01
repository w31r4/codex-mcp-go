package session

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
)

func TestManager_StartAndComplete(t *testing.T) {
	m := NewManager(Options{MaxRunning: 2, TTL: time.Minute})

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := m.Start("s1", "/tmp", "read-only", cancel); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if got, ok := m.Get("s1"); !ok {
		t.Fatalf("Get() should find session")
	} else if got.State != StateRunning {
		t.Fatalf("state=%v, want %v", got.State, StateRunning)
	}

	if !m.MarkCompleted("s1", 123, 2) {
		t.Fatalf("MarkCompleted() returned false")
	}
	if got, ok := m.Get("s1"); !ok {
		t.Fatalf("Get() should find session")
	} else {
		if got.State != StateCompleted {
			t.Fatalf("state=%v, want %v", got.State, StateCompleted)
		}
		if got.ExecutionTimeMs != 123 {
			t.Fatalf("execution_time_ms=%v, want %v", got.ExecutionTimeMs, int64(123))
		}
		if got.ToolCallCount != 2 {
			t.Fatalf("tool_call_count=%v, want %v", got.ToolCallCount, 2)
		}
		if got.EndedAt == "" {
			t.Fatalf("ended_at should be set")
		}
	}
}

func TestManager_UpdateID(t *testing.T) {
	m := NewManager(Options{MaxRunning: 2, TTL: time.Minute})

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := m.Start("tmp_1", "/tmp", "read-only", cancel); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if ok, err := m.UpdateID("tmp_1", "real_1"); err != nil || !ok {
		t.Fatalf("UpdateID() ok=%v err=%v, want ok=true err=nil", ok, err)
	}
	if _, ok := m.Get("tmp_1"); ok {
		t.Fatalf("old ID should not be found")
	}
	if _, ok := m.Get("real_1"); !ok {
		t.Fatalf("new ID should be found")
	}
}

func TestManager_ConcurrencyLimit(t *testing.T) {
	m := NewManager(Options{MaxRunning: 1, TTL: time.Minute})

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := m.Start("s1", "/tmp", "read-only", cancel); err != nil {
		t.Fatalf("Start(s1) failed: %v", err)
	}
	_, err := m.Start("s2", "/tmp", "read-only", cancel)
	if err == nil {
		t.Fatalf("expected Start(s2) to fail")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.SessionLimitExceeded {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.SessionLimitExceeded)
	}
}

func TestManager_Cancel(t *testing.T) {
	m := NewManager(Options{MaxRunning: 2, TTL: time.Minute})

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := m.Start("s1", "/tmp", "read-only", cancel); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	ok, err := m.Cancel("s1")
	if err != nil || !ok {
		t.Fatalf("Cancel() ok=%v err=%v, want ok=true err=nil", ok, err)
	}
	if got, ok := m.Get("s1"); !ok {
		t.Fatalf("Get() should find session")
	} else if got.State != StateCancelled {
		t.Fatalf("state=%v, want %v", got.State, StateCancelled)
	}
	ok, err = m.Cancel("s1")
	if err != nil {
		t.Fatalf("Cancel() second call err=%v, want nil", err)
	}
	if ok {
		t.Fatalf("Cancel() second call ok=true, want false")
	}
}

func TestManager_CleanupExpired(t *testing.T) {
	m := NewManager(Options{MaxRunning: 2, TTL: time.Second})

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := m.Start("s1", "/tmp", "read-only", cancel); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if !m.MarkCompleted("s1", 1, 0) {
		t.Fatalf("MarkCompleted() returned false")
	}
	removed := m.CleanupExpired(time.Now().Add(2 * time.Hour))
	if removed != 1 {
		t.Fatalf("removed=%d, want 1", removed)
	}
	if _, ok := m.Get("s1"); ok {
		t.Fatalf("session should be cleaned up")
	}
}
