package mcp

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/w31r4/codex-mcp-go/internal/config"
)

func TestStatsTool_AppearsInToolsList(t *testing.T) {
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

	found := false
	for _, tool := range res.Tools {
		if tool.Name == "stats" {
			found = true
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Fatalf("stats tool should be read-only")
			}
			break
		}
	}
	if !found {
		t.Fatalf("stats tool not found in tools/list")
	}
}
