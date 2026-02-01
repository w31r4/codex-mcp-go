package mcp

import (
	"context"
	"os"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestCodexTool_OutputSchemaShape(t *testing.T) {
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

	var codexTool *mcpsdk.Tool
	for i := range res.Tools {
		if res.Tools[i].Name == "codex" {
			codexTool = res.Tools[i]
			break
		}
	}
	if codexTool == nil {
		t.Fatalf("codex tool not found in tools/list")
	}

	schema, ok := codexTool.OutputSchema.(map[string]any)
	if !ok {
		t.Fatalf("codex outputSchema type=%T, want map", codexTool.OutputSchema)
	}
	if schema["type"] != "object" {
		t.Fatalf("outputSchema.type=%v, want %q", schema["type"], "object")
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("outputSchema.properties type=%T, want map", schema["properties"])
	}

	wantFields := []string{
		"success",
		"SESSION_ID",
		"agent_messages",
		"all_messages",
		"execution_time_ms",
		"tool_call_count",
	}
	for _, key := range wantFields {
		if _, ok := props[key]; !ok {
			t.Fatalf("outputSchema missing property %q", key)
		}
	}
}

func TestCodexTool_Call_ReturnsStructuredContentAndTextContent(t *testing.T) {
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

	t.Setenv("CODEX_MCP_FAKE_CODEX", "success_tool_call")

	workdir := t.TempDir()
	out, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: "codex",
		Arguments: map[string]any{
			"PROMPT":             "hi",
			"cd":                 workdir,
			"return_all_messages": true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if out.IsError {
		t.Fatalf("CallTool() returned isError=true")
	}
	if len(out.Content) != 1 {
		t.Fatalf("content len=%d, want 1", len(out.Content))
	}
	tc, ok := out.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("content[0] type=%T, want *TextContent", out.Content[0])
	}
	if tc.Text != "hello from codex" {
		t.Fatalf("content[0].text=%q, want %q", tc.Text, "hello from codex")
	}

	sc, ok := out.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent type=%T, want map", out.StructuredContent)
	}
	if sc["success"] != true {
		t.Fatalf("structuredContent.success=%v, want true", sc["success"])
	}
	if sc["SESSION_ID"] != "t-123" {
		t.Fatalf("structuredContent.SESSION_ID=%v, want %q", sc["SESSION_ID"], "t-123")
	}
	if sc["agent_messages"] != "hello from codex" {
		t.Fatalf("structuredContent.agent_messages=%v, want %q", sc["agent_messages"], "hello from codex")
	}

	if _, ok := sc["execution_time_ms"].(float64); !ok {
		t.Fatalf("structuredContent.execution_time_ms type=%T, want number", sc["execution_time_ms"])
	}
	if sc["tool_call_count"] != float64(1) {
		t.Fatalf("structuredContent.tool_call_count=%v, want %v", sc["tool_call_count"], float64(1))
	}

	all, ok := sc["all_messages"].([]any)
	if !ok {
		t.Fatalf("structuredContent.all_messages type=%T, want array", sc["all_messages"])
	}
	if len(all) != 2 {
		t.Fatalf("structuredContent.all_messages len=%d, want 2", len(all))
	}
}
