package codex

import (
	"context"
	stderrors "errors"
	"os"
	"testing"
	"time"

	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
)

func TestRun_CommandConstruction_EchoArgs(t *testing.T) {
	t.Setenv(fakeCodexEnv, "echo_args")

	res, err := Run(context.Background(), Options{
		Prompt:            "hi",
		WorkingDir:        ".",
		Sandbox:           SandboxWorkspaceWrite,
		SessionID:         "sess-1",
		SkipGitRepoCheck:  true,
		ReturnAllMessages: true,
		ImagePaths:        []string{"a.png", "b.png"},
		Model:             "gpt-test",
		Yolo:              true,
		Profile:           "p1",
		Timeout:           5 * time.Second,
		ExecutablePath:    os.Args[0],
		MaxBufferedLines:  100,
	})
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("Run() success=%v, want true", res != nil && res.Success)
	}
	if res.SessionID != "t-123" {
		t.Fatalf("SessionID=%q, want %q", res.SessionID, "t-123")
	}
	if res.AgentMessages != "ok" {
		t.Fatalf("AgentMessages=%q, want %q", res.AgentMessages, "ok")
	}

	if len(res.AllMessages) != 1 {
		t.Fatalf("AllMessages len=%d, want 1", len(res.AllMessages))
	}
	rawArgs, ok := res.AllMessages[0]["args"].([]interface{})
	if !ok {
		t.Fatalf("args type=%T, want []interface{}", res.AllMessages[0]["args"])
	}
	got := make([]string, 0, len(rawArgs))
	for _, a := range rawArgs {
		s, ok := a.(string)
		if !ok {
			t.Fatalf("arg type=%T, want string", a)
		}
		got = append(got, s)
	}

	want := []string{
		"exec",
		"--sandbox", "workspace-write",
		"--cd", ".",
		"--json",
		"--image", "a.png,b.png",
		"--model", "gpt-test",
		"--profile", "p1",
		"--yolo",
		"--skip-git-repo-check",
		"resume", "sess-1",
		"--", "hi",
	}

	if len(got) != len(want) {
		t.Fatalf("args len=%d, want %d\nargs=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d]=%q, want %q\nargs=%v", i, got[i], want[i], got)
		}
	}
}

func TestRun_NoOutputTimeout(t *testing.T) {
	t.Setenv(fakeCodexEnv, "sleep_no_output")

	_, err := Run(context.Background(), Options{
		Prompt:          "hi",
		WorkingDir:      ".",
		Sandbox:         SandboxReadOnly,
		ExecutablePath:  os.Args[0],
		Timeout:         2 * time.Second,
		NoOutputTimeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.NoOutputTimeout {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.NoOutputTimeout)
	}
}

func TestRun_CodexTimeout(t *testing.T) {
	t.Setenv(fakeCodexEnv, "sleep")

	_, err := Run(context.Background(), Options{
		Prompt:         "hi",
		WorkingDir:     ".",
		Sandbox:        SandboxReadOnly,
		ExecutablePath: os.Args[0],
		Timeout:        100 * time.Millisecond,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.CodexTimeout {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.CodexTimeout)
	}
}

func TestRun_InvalidJSONL(t *testing.T) {
	t.Setenv(fakeCodexEnv, "invalid_json")

	_, err := Run(context.Background(), Options{
		Prompt:         "hi",
		WorkingDir:     ".",
		Sandbox:        SandboxReadOnly,
		ExecutablePath: os.Args[0],
		Timeout:        5 * time.Second,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.CodexExecutionFailed {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.CodexExecutionFailed)
	}
	if _, ok := cerr.Data["line"]; !ok {
		t.Fatalf("expected data.line to be present")
	}
}
