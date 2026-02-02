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

var globalSessions = session.NewManager(session.DefaultOptions())

type ListSessionsInput struct{}

type ListSessionsOutput struct {
	Sessions []session.View `json:"sessions"`
}

type GetSessionInput struct {
	SessionID string `json:"SESSION_ID"`
}

type GetSessionOutput struct {
	Found   bool               `json:"found"`
	Session session.DetailView `json:"session,omitempty"`
}

type CancelSessionInput struct {
	SessionID string `json:"SESSION_ID"`
}

type CancelSessionOutput struct {
	Cancelled bool         `json:"cancelled"`
	Session   session.View `json:"session,omitempty"`
}

func buildListSessionsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}
}

func buildGetSessionInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"SESSION_ID": {Type: "string", Description: "Session identifier to look up."},
		},
		Required: []string{"SESSION_ID"},
	}
}

func buildCancelSessionInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"SESSION_ID": {Type: "string", Description: "Running session identifier to cancel."},
		},
		Required: []string{"SESSION_ID"},
	}
}

func handleListSessions(ctx context.Context, req *mcp.CallToolRequest, input ListSessionsInput) (result *mcp.CallToolResult, output ListSessionsOutput, err error) {
	ctx, rc := logging.NewRequestContext(ctx, "list_sessions")
	logging.LogRequest(ctx, map[string]any{})
	defer func() {
		success := err == nil
		globalMetrics.RecordRequest("list_sessions", success, time.Since(rc.StartTime))
		if err != nil {
			var cerr *cerrors.Error
			if stderrors.As(err, &cerr) {
				globalMetrics.RecordError(cerr.Code.Name())
			}
		}
		logging.LogResponse(ctx, map[string]any{"success": success}, err)
	}()

	output.Sessions = globalSessions.List()
	return nil, output, nil
}

func handleGetSession(ctx context.Context, req *mcp.CallToolRequest, input GetSessionInput) (result *mcp.CallToolResult, output GetSessionOutput, err error) {
	ctx, rc := logging.NewRequestContext(ctx, "get_session")
	logging.LogRequest(ctx, map[string]any{"session_id": strings.TrimSpace(input.SessionID)})
	defer func() {
		success := err == nil
		globalMetrics.RecordRequest("get_session", success, time.Since(rc.StartTime))
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
		return nil, GetSessionOutput{}, cerrors.ErrInvalidParams("SESSION_ID is required")
	}

	s, ok := globalSessions.GetDetail(input.SessionID, 20)
	output.Found = ok
	if ok {
		output.Session = s
	}
	return nil, output, nil
}

func handleCancelSession(ctx context.Context, req *mcp.CallToolRequest, input CancelSessionInput) (result *mcp.CallToolResult, output CancelSessionOutput, err error) {
	ctx, rc := logging.NewRequestContext(ctx, "cancel_session")
	logging.LogRequest(ctx, map[string]any{"session_id": strings.TrimSpace(input.SessionID)})
	defer func() {
		success := err == nil
		globalMetrics.RecordRequest("cancel_session", success, time.Since(rc.StartTime))
		if err != nil {
			var cerr *cerrors.Error
			if stderrors.As(err, &cerr) {
				globalMetrics.RecordError(cerr.Code.Name())
			}
		}
		logging.LogResponse(ctx, map[string]any{"success": success, "cancelled": output.Cancelled}, err)
	}()

	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.SessionID == "" {
		return nil, CancelSessionOutput{}, cerrors.ErrInvalidParams("SESSION_ID is required")
	}

	cancelled, cancelErr := globalSessions.Cancel(input.SessionID)
	if cancelErr != nil {
		return nil, CancelSessionOutput{}, cancelErr
	}
	output.Cancelled = cancelled

	if s, ok := globalSessions.Get(input.SessionID); ok {
		output.Session = s
	}
	return nil, output, nil
}
