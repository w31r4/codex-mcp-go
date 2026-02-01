package mcp

import (
	"context"
	"os"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestSessionTools_AppearInToolsList(t *testing.T) {
	ctx := context.Background()

	s := NewServer(config.Default())
	c := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client", Version: "test"}, nil)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ss, err := s.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server Connect() failed: %v", err)
	}
	defer ss.Close()

	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client Connect() failed: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() failed: %v", err)
	}

	var (
		foundList   bool
		foundGet    bool
		foundCancel bool
	)
	for _, tool := range res.Tools {
		switch tool.Name {
		case "list_sessions":
			foundList = true
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Fatalf("list_sessions should be read-only")
			}
		case "get_session":
			foundGet = true
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Fatalf("get_session should be read-only")
			}
		case "cancel_session":
			foundCancel = true
			if tool.Annotations == nil {
				t.Fatalf("cancel_session should have annotations")
			}
			if tool.Annotations.ReadOnlyHint {
				t.Fatalf("cancel_session should not be read-only")
			}
			if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != true {
				t.Fatalf("cancel_session should be destructive")
			}
		}
	}
	if !foundList || !foundGet || !foundCancel {
		t.Fatalf("missing tools: list=%v get=%v cancel=%v", foundList, foundGet, foundCancel)
	}
}

func TestSessionTools_BasicBehavior(t *testing.T) {
	ctx := context.Background()

	cfg := config.Default()
	cfg.Codex.ExecutablePath = os.Args[0]

	s := NewServer(cfg)
	c := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client", Version: "test"}, nil)

	t1, t2 := mcpsdk.NewInMemoryTransports()
	ss, err := s.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("server Connect() failed: %v", err)
	}
	defer ss.Close()

	cs, err := c.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client Connect() failed: %v", err)
	}
	defer cs.Close()

	// list_sessions should start empty.
	listRes, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_sessions"})
	if err != nil {
		t.Fatalf("list_sessions failed: %v", err)
	}
	listSC, ok := listRes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("list_sessions structuredContent type=%T, want map", listRes.StructuredContent)
	}
	sessions, ok := listSC["sessions"].([]any)
	if !ok {
		t.Fatalf("list_sessions.sessions type=%T, want array", listSC["sessions"])
	}
	if len(sessions) != 0 {
		t.Fatalf("list_sessions.sessions len=%d, want 0", len(sessions))
	}

	// get_session for unknown should report found=false.
	getRes, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_session",
		Arguments: map[string]any{"SESSION_ID": "missing"},
	})
	if err != nil {
		t.Fatalf("get_session failed: %v", err)
	}
	getSC, ok := getRes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("get_session structuredContent type=%T, want map", getRes.StructuredContent)
	}
	if getSC["found"] != false {
		t.Fatalf("get_session.found=%v, want false", getSC["found"])
	}

	// Start a codex run (new session) and verify tracking uses the real thread_id when completed.
	t.Setenv(fakeCodexEnv, "success_tool_call")
	workdir := t.TempDir()
	_, err = cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT": "hi",
			"cd":     workdir,
		},
	})
	if err != nil {
		t.Fatalf("codex call failed: %v", err)
	}

	listRes, err = cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "list_sessions"})
	if err != nil {
		t.Fatalf("list_sessions failed: %v", err)
	}
	listSC, ok = listRes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("list_sessions structuredContent type=%T, want map", listRes.StructuredContent)
	}
	sessions, ok = listSC["sessions"].([]any)
	if !ok || len(sessions) == 0 {
		t.Fatalf("list_sessions.sessions missing or empty")
	}

	// Find session "t-123" (from the fake codex).
	found := false
	for _, item := range sessions {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["SESSION_ID"] == "t-123" {
			found = true
			if m["state"] != "completed" {
				t.Fatalf("session.state=%v, want %q", m["state"], "completed")
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected to find session SESSION_ID=t-123")
	}

	getRes, err = cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "get_session",
		Arguments: map[string]any{"SESSION_ID": "t-123"},
	})
	if err != nil {
		t.Fatalf("get_session failed: %v", err)
	}
	getSC, ok = getRes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("get_session structuredContent type=%T, want map", getRes.StructuredContent)
	}
	if getSC["found"] != true {
		t.Fatalf("get_session.found=%v, want true", getSC["found"])
	}
}

func TestCancelSession_CancelsRunningSession(t *testing.T) {
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
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		res, _ := cs1.CallTool(callCtx, &mcpsdk.CallToolParams{
			Name: "codex",
			Arguments: map[string]any{
				"PROMPT":          "hi",
				"cd":              workdir,
				"SESSION_ID":      sessionID,
				"timeout_seconds": 5,
			},
		})
		done <- res
	}()

	// Wait until the session shows up as running.
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

	cancelRes, err := cs2.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "cancel_session",
		Arguments: map[string]any{"SESSION_ID": sessionID},
	})
	if err != nil {
		t.Fatalf("cancel_session failed: %v", err)
	}
	cancelSC, ok := cancelRes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("cancel_session structuredContent type=%T, want map", cancelRes.StructuredContent)
	}
	if cancelSC["cancelled"] != true {
		t.Fatalf("cancel_session.cancelled=%v, want true", cancelSC["cancelled"])
	}

	select {
	case res := <-done:
		if res == nil {
			t.Fatalf("codex tool call returned nil result")
		}
		if !res.IsError {
			t.Fatalf("expected codex tool call to end in error after cancellation")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("codex tool call did not return after cancellation")
	}
}

