package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
)

func TestCodexTool_WorkdirLock_Reject(t *testing.T) {
	ctx := context.Background()

	cfg := config.Default()
	cfg.Codex.ExecutablePath = os.Args[0]
	cfg.Codex.WorkdirLockMode = "reject"

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

	// Wait until the session shows up as running (ensures the first call acquired the lock).
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

	// A concurrent call in the same cd should be rejected.
	res2, err := cs2.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT": "hi",
			"cd":     workdir,
		},
	})
	if err != nil {
		t.Fatalf("second codex call failed: %v", err)
	}
	if res2 == nil || !res2.IsError {
		t.Fatalf("expected second codex call to be rejected with isError=true")
	}
	if len(res2.Content) == 0 {
		t.Fatalf("expected error content")
	}
	tc, ok := res2.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("error content type=%T, want *TextContent", res2.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &payload); err != nil {
		t.Fatalf("error payload is not JSON: %v (%q)", err, tc.Text)
	}
	if payload["code"] != float64(cerrors.WorkdirBusy) || payload["name"] != cerrors.WorkdirBusy.Name() {
		t.Fatalf("error=%v, want code/name for %s", payload, cerrors.WorkdirBusy.Name())
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

func TestCodexTool_WorkdirLock_Queue_AllowsSecondAfterRelease(t *testing.T) {
	ctx := context.Background()

	cfg := config.Default()
	cfg.Codex.ExecutablePath = os.Args[0]
	cfg.Codex.WorkdirLockMode = "queue"
	cfg.Codex.WorkdirLockTimeoutSeconds = 2

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

	workdir := t.TempDir()
	session1 := "s1"

	t.Setenv(fakeCodexEnv, "sleep")
	firstDone := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		res, _ := cs1.CallTool(callCtx, &mcpsdk.CallToolParams{
			Name: "codex",
			Arguments: map[string]any{
				"PROMPT":     "hi",
				"cd":         workdir,
				"SESSION_ID": session1,
			},
		})
		firstDone <- res
	}()

	// Wait until the first session is running.
	deadline := time.Now().Add(2 * time.Second)
	for {
		res, err := cs2.CallTool(ctx, &mcpsdk.CallToolParams{
			Name:      "get_session",
			Arguments: map[string]any{"SESSION_ID": session1},
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

	// Switch the fake codex behavior for the second run: it should complete quickly once it acquires the lock.
	t.Setenv(fakeCodexEnv, "success_tool_call")

	secondDone := make(chan *mcpsdk.CallToolResult, 1)
	go func() {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		res, _ := cs2.CallTool(callCtx, &mcpsdk.CallToolParams{
			Name: "codex",
			Arguments: map[string]any{
				"PROMPT":     "hi",
				"cd":         workdir,
				"SESSION_ID": "s2",
			},
		})
		secondDone <- res
	}()

	// Release the lock by cancelling the first run.
	time.Sleep(100 * time.Millisecond)
	_, _ = cs2.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "cancel_session",
		Arguments: map[string]any{"SESSION_ID": session1},
	})

	select {
	case res := <-secondDone:
		if res == nil || res.IsError {
			t.Fatalf("expected queued codex call to succeed, got res=%v", res)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("queued codex call did not complete in time")
	}

	select {
	case <-firstDone:
	case <-time.After(3 * time.Second):
		t.Fatalf("first codex call did not return after cancellation")
	}
}
