package mcp

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
	"github.com/w31r4/codex-mcp-go/internal/logging"
	"github.com/w31r4/codex-mcp-go/internal/session"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TailSessionInput struct {
	SessionID string `json:"SESSION_ID"`
	Cursor    *int64 `json:"cursor,omitempty"`
	Limit     *int   `json:"limit,omitempty"`
}

type TailSessionOutput struct {
	Found         bool                          `json:"found"`
	SessionID     string                        `json:"SESSION_ID"`
	State         session.State                 `json:"state,omitempty"`
	Entries       []session.DiagnosticEntryView `json:"entries,omitempty"`
	NextCursor    uint64                        `json:"next_cursor"`
	Dropped       bool                          `json:"dropped,omitempty"`
	DroppedBefore uint64                        `json:"dropped_before,omitempty"`
}

func buildTailSessionInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"SESSION_ID": {Type: "string", Description: "Session identifier to tail."},
			"cursor":     {Type: "number", Description: "Return entries with seq > cursor. Start with 0."},
			"limit":      {Type: "number", Description: "Maximum number of entries to return (default 50, max 200)."},
		},
		Required: []string{"SESSION_ID"},
	}
}

func handleTailSession(ctx context.Context, req *mcp.CallToolRequest, input TailSessionInput) (result *mcp.CallToolResult, output TailSessionOutput, err error) {
	ctx, rc := logging.NewRequestContext(ctx, "tail_session")
	logging.LogRequest(ctx, map[string]any{
		"session_id": strings.TrimSpace(input.SessionID),
	})
	defer func() {
		success := err == nil
		globalMetrics.RecordRequest("tail_session", success, time.Since(rc.StartTime))
		if err != nil {
			var cerr *cerrors.Error
			if stderrors.As(err, &cerr) {
				globalMetrics.RecordError(cerr.Code.Name())
			}
		}
		logging.LogResponse(ctx, map[string]any{"success": success, "found": output.Found}, err)
	}()

	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.SessionID == "" {
		return nil, TailSessionOutput{}, cerrors.ErrInvalidParams("SESSION_ID is required")
	}

	cursor := uint64(0)
	if input.Cursor != nil && *input.Cursor > 0 {
		cursor = uint64(*input.Cursor)
	}
	limit := 50
	if input.Limit != nil && *input.Limit > 0 {
		limit = *input.Limit
	}

	entries, next, dropped, droppedBefore, state, found := globalSessions.TailDiagnostics(input.SessionID, cursor, limit)

	output.Found = found
	output.SessionID = input.SessionID
	output.State = state
	output.Entries = entries
	output.NextCursor = next
	output.Dropped = dropped
	output.DroppedBefore = droppedBefore

	return nil, output, nil
}
