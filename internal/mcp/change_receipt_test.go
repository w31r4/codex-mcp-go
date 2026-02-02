package mcp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestCodexTool_ChangeReceipt_GitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "test")

	path := filepath.Join(repo, "file.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit("add", "file.txt")
	runGit("commit", "-m", "init")

	// Create an unstaged modification that should appear in the receipt.
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

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

	t.Setenv(fakeCodexEnv, "success_tool_call")

	out, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT":      "hi",
			"cd":          repo,
			"return_diff": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if out.IsError {
		t.Fatalf("CallTool() returned isError=true")
	}

	sc, ok := out.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent type=%T, want map", out.StructuredContent)
	}

	cr, ok := sc["change_receipt"].(map[string]any)
	if !ok {
		t.Fatalf("change_receipt type=%T, want map", sc["change_receipt"])
	}
	if cr["receipt_available"] != true {
		t.Fatalf("change_receipt.receipt_available=%v, want true", cr["receipt_available"])
	}
	if cr["git_status"] == "" {
		t.Fatalf("change_receipt.git_status is empty")
	}
	if cr["diff_stat"] == "" {
		t.Fatalf("change_receipt.diff_stat is empty")
	}
	if cr["diff"] == "" {
		t.Fatalf("change_receipt.diff is empty")
	}

	changed, ok := cr["changed_files"].([]any)
	if !ok {
		t.Fatalf("change_receipt.changed_files type=%T, want array", cr["changed_files"])
	}
	found := false
	for _, item := range changed {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["path"] == "file.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected changed_files to include file.txt, got=%v", changed)
	}
}

func TestCodexTool_ChangeReceipt_NonGitDir(t *testing.T) {
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

	t.Setenv(fakeCodexEnv, "success_tool_call")

	out, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT":      "hi",
			"cd":          t.TempDir(),
			"return_diff": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if out.IsError {
		t.Fatalf("CallTool() returned isError=true")
	}

	sc, ok := out.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent type=%T, want map", out.StructuredContent)
	}

	cr, ok := sc["change_receipt"].(map[string]any)
	if !ok {
		t.Fatalf("change_receipt type=%T, want map", sc["change_receipt"])
	}
	if cr["receipt_available"] != false {
		t.Fatalf("change_receipt.receipt_available=%v, want false", cr["receipt_available"])
	}
}

func TestCodexTool_ChangeReceipt_GitMissing(t *testing.T) {
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

	// Simulate "git not found" without affecting codex (we set executable_path).
	t.Setenv("PATH", "")
	t.Setenv(fakeCodexEnv, "success_tool_call")

	out, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT":      "hi",
			"cd":          t.TempDir(),
			"return_diff": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if out.IsError {
		t.Fatalf("CallTool() returned isError=true")
	}

	sc, ok := out.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent type=%T, want map", out.StructuredContent)
	}

	cr, ok := sc["change_receipt"].(map[string]any)
	if !ok {
		t.Fatalf("change_receipt type=%T, want map", sc["change_receipt"])
	}
	if cr["receipt_available"] != false {
		t.Fatalf("change_receipt.receipt_available=%v, want false", cr["receipt_available"])
	}
	if cr["receipt_error"] != "git not found" {
		t.Fatalf("change_receipt.receipt_error=%v, want %q", cr["receipt_error"], "git not found")
	}
}

func TestCodexTool_ChangeReceipt_TruncatesDiff(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	repo := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "test")

	path := filepath.Join(repo, "big.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit("add", "big.txt")
	runGit("commit", "-m", "init")

	// Create a large diff (>64KiB).
	blob := strings.Repeat("a", 100000) + "\n"
	if err := os.WriteFile(path, []byte(blob), 0o644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

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

	t.Setenv(fakeCodexEnv, "success_tool_call")

	out, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT":      "hi",
			"cd":          repo,
			"return_diff": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if out.IsError {
		t.Fatalf("CallTool() returned isError=true")
	}

	sc, ok := out.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent type=%T, want map", out.StructuredContent)
	}

	cr, ok := sc["change_receipt"].(map[string]any)
	if !ok {
		t.Fatalf("change_receipt type=%T, want map", sc["change_receipt"])
	}
	if cr["receipt_available"] != true {
		t.Fatalf("change_receipt.receipt_available=%v, want true", cr["receipt_available"])
	}
	if cr["diff"] == "" {
		t.Fatalf("change_receipt.diff is empty")
	}
	if cr["diff_truncated"] != true {
		t.Fatalf("change_receipt.diff_truncated=%v, want true", cr["diff_truncated"])
	}
}
