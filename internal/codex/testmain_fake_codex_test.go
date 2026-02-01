package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const fakeCodexEnv = "CODEX_MCP_FAKE_CODEX"

func TestMain(m *testing.M) {
	if mode := strings.TrimSpace(os.Getenv(fakeCodexEnv)); mode != "" && len(os.Args) > 1 && os.Args[1] == "exec" {
		runFakeCodex(mode)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runFakeCodex(mode string) {
	switch mode {
	case "echo_args":
		out := map[string]any{
			"thread_id": "t-123",
			"args":      os.Args[1:],
			"item": map[string]any{
				"type": "agent_message",
				"text": "ok",
			},
		}
		b, _ := json.Marshal(out)
		fmt.Fprintln(os.Stdout, string(b))
	case "sleep_no_output":
		time.Sleep(30 * time.Second)
	case "sleep":
		out := map[string]any{
			"thread_id": "t-123",
			"item": map[string]any{
				"type": "agent_message",
				"text": "ok",
			},
		}
		b, _ := json.Marshal(out)
		fmt.Fprintln(os.Stdout, string(b))
		time.Sleep(30 * time.Second)
	case "invalid_json":
		fmt.Fprintln(os.Stdout, "not-json")
		out := map[string]any{
			"thread_id": "t-123",
			"item": map[string]any{
				"type": "agent_message",
				"text": "ok",
			},
		}
		b, _ := json.Marshal(out)
		fmt.Fprintln(os.Stdout, string(b))
	default:
		out := map[string]any{
			"thread_id": "t-123",
			"item": map[string]any{
				"type": "agent_message",
				"text": "ok",
			},
		}
		b, _ := json.Marshal(out)
		fmt.Fprintln(os.Stdout, string(b))
	}
}

