package mcp

import (
	"context"
	"os"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestTailSession_ReturnsEntriesForRunningSession(t *testing.T) {
	ctx := context.Background()

	cfg := config.Default()
	cfg.Codex.ExecutablePath = os.Args[0]

	s := NewServer(cfg)

	c1 := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client1", Version: "test"}, nil)
	c2 := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client2", Version: "test"}, nil)

	t1a, t2a := mcpsdk.NewInMemoryTransports()
	t1b, t2b := mcpsdk.NewInMemoryTransports()

	ss1, err := s.Connect(ctx, t1a, nil)
	if err != nil {
		t.Fatalf("server Connect(a) failed: %v", err)
	}
	defer ss1.Close()
	ss2, err := s.Connect(ctx, t1b, nil)
	if err != nil {
		t.Fatalf("server Connect(b) failed: %v", err)
	}
	defer ss2.Close()

	cs1, err := c1.Connect(ctx, t2a, nil)
	if err != nil {
		t.Fatalf("client1 Connect() failed: %v", err)
	}
	defer cs1.Close()
	cs2, err := c2.Connect(ctx, t2b, nil)
	if err != nil {
		t.Fatalf("client2 Connect() failed: %v", err)
	}
	defer cs2.Close()

	t.Setenv(fakeCodexEnv, "sleep")

	workdir := t.TempDir()
	sessionID := "s123"

	done := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		res, _ := cs1.CallTool(callCtx, &mcpsdk.CallToolParams{
			Name: "codex",
			Arguments: map[string]any{
				"PROMPT":     "hi",
				"cd":         workdir,
				"SESSION_ID": sessionID,
			},
		})
		done <- res
	}()

	// Wait until the session is running.
	deadline := time.Now().Add(2 * time.Second)
	for {
		res, err := cs2.CallTool(ctx, &mcpsdk.CallToolParams{
			Name:      "get_session",
			Arguments: map[string]any{"SESSION_ID": sessionID},
		})
		if err == nil {
			sc, ok := res.StructuredContent.(map[string]any)
			if ok && sc["found"] == true {
				if sess, ok := sc["session"].(map[string]any); ok && sess["state"] == "running" {
					break
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("session did not enter running state in time")
		}
		time.Sleep(20 * time.Millisecond)
	}

	tailRes, err := cs2.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "tail_session",
		Arguments: map[string]any{
			"SESSION_ID": sessionID,
			"cursor":     0,
			"limit":      50,
		},
	})
	if err != nil {
		t.Fatalf("tail_session failed: %v", err)
	}
	if tailRes.IsError {
		t.Fatalf("tail_session returned isError=true")
	}
	sc, ok := tailRes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("tail_session structuredContent type=%T, want map", tailRes.StructuredContent)
	}
	if sc["found"] != true {
		t.Fatalf("tail_session.found=%v, want true", sc["found"])
	}
	if sc["state"] != "running" {
		t.Fatalf("tail_session.state=%v, want %q", sc["state"], "running")
	}
	entries, ok := sc["entries"].([]any)
	if !ok {
		t.Fatalf("tail_session.entries type=%T, want array", sc["entries"])
	}
	if len(entries) == 0 {
		t.Fatalf("expected tail_session.entries to be non-empty")
	}
	next, ok := sc["next_cursor"].(float64)
	if !ok || next <= 0 {
		t.Fatalf("tail_session.next_cursor=%v, want > 0", sc["next_cursor"])
	}

	// Cleanup.
	_, _ = cs2.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "cancel_session",
		Arguments: map[string]any{"SESSION_ID": sessionID},
	})
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("codex tool call did not return after cancellation")
	}
}
