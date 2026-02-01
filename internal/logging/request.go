package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type requestContextKey struct{}

type RequestContext struct {
	RequestID string
	ToolName  string
	StartTime time.Time
	Logger    Logger
}

func NewRequestContext(ctx context.Context, toolName string) (context.Context, *RequestContext) {
	reqID := newRequestID()
	logger := GetLogger().With("request_id", reqID, "tool", toolName)
	rc := &RequestContext{
		RequestID: reqID,
		ToolName:  toolName,
		StartTime: time.Now(),
		Logger:    logger,
	}
	return context.WithValue(ctx, requestContextKey{}, rc), rc
}

func GetRequestContext(ctx context.Context) *RequestContext {
	if rc, ok := ctx.Value(requestContextKey{}).(*RequestContext); ok {
		return rc
	}
	return nil
}

func LogRequest(ctx context.Context, input any) {
	rc := GetRequestContext(ctx)
	if rc == nil {
		return
	}
	rc.Logger.Info("tool request received", "input", input)
}

func LogResponse(ctx context.Context, output any, err error) {
	rc := GetRequestContext(ctx)
	if rc == nil {
		return
	}
	duration := time.Since(rc.StartTime).Milliseconds()
	if err != nil {
		rc.Logger.Error("tool request failed", "error", err.Error(), "duration_ms", duration)
		return
	}
	rc.Logger.Info("tool request completed", "duration_ms", duration, "output", output)
}

func newRequestID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Best-effort fallback.
		return "00000000"
	}
	return hex.EncodeToString(b[:])
}
