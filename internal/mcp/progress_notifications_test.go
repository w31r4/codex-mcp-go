package mcp

import (
	"context"
	"os"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestCodexTool_EmitsProgressNotificationsWhenTokenSet(t *testing.T) {
	ctx := context.Background()

	cfg := config.Default()
	cfg.Codex.ExecutablePath = os.Args[0]

	s := NewServer(cfg)

	progressCh := make(chan *mcpsdk.ProgressNotificationParams, 16)
	c := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client", Version: "test"}, &mcpsdk.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcpsdk.ProgressNotificationClientRequest) {
			select {
			case progressCh <- req.Params:
			default:
			}
		},
	})

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

	t.Setenv(fakeCodexEnv, "success_tool_call")

	workdir := t.TempDir()
	params := &mcpsdk.CallToolParams{
		Meta: mcpsdk.Meta{"progressToken": "pt1"},
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT": "hi",
			"cd":     workdir,
		},
	}

	done := make(chan struct{})
	go func() {
		_, _ = cs.CallTool(ctx, params)
		close(done)
	}()

	select {
	case p := <-progressCh:
		if p == nil {
			t.Fatalf("got nil progress params")
		}
		if p.ProgressToken != "pt1" {
			t.Fatalf("progressToken=%v, want %q", p.ProgressToken, "pt1")
		}
		if p.Progress <= 0 {
			t.Fatalf("progress=%v, want > 0", p.Progress)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for progress notification")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for codex tool call to finish")
	}
}
