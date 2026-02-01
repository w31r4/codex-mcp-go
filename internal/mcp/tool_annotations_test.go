package mcp

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestCodexTool_HasConservativeAnnotations(t *testing.T) {
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
	for _, tool := range res.Tools {
		if tool.Name == "codex" {
			codexTool = tool
			break
		}
	}
	if codexTool == nil {
		t.Fatalf("codex tool not found in tools/list")
	}
	if codexTool.Annotations == nil {
		t.Fatalf("codex tool annotations should be present")
	}

	if codexTool.Annotations.ReadOnlyHint {
		t.Fatalf("codex tool should not be read-only")
	}
	if codexTool.Annotations.DestructiveHint == nil || *codexTool.Annotations.DestructiveHint != true {
		t.Fatalf("codex tool should be marked destructive")
	}
	if codexTool.Annotations.IdempotentHint {
		t.Fatalf("codex tool should not be idempotent")
	}
	if codexTool.Annotations.OpenWorldHint == nil || *codexTool.Annotations.OpenWorldHint != true {
		t.Fatalf("codex tool should be marked open-world")
	}
}

