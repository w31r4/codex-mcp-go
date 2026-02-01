package mcp

import (
	"fmt"
	"os"
	"testing"
)

const fakeCodexEnv = "CODEX_MCP_FAKE_CODEX"

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeCodexEnv); mode != "" && len(os.Args) > 1 && os.Args[1] == "exec" {
		runFakeCodex(mode)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runFakeCodex(mode string) {
	switch mode {
	case "success_tool_call":
		fmt.Fprintln(os.Stdout, `{"thread_id":"t-123","item":{"type":"tool_call","name":"x"}}`)
		fmt.Fprintln(os.Stdout, `{"thread_id":"t-123","item":{"type":"agent_message","text":"hello from codex"}}`)
	default:
		fmt.Fprintln(os.Stdout, `{"thread_id":"t-123","item":{"type":"agent_message","text":"hello from codex"}}`)
	}
}

