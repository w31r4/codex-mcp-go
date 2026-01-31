package errors

import (
	"encoding/json"
	"fmt"
)

// Error is a structured error for codex-mcp-go.
//
// It is intentionally serializable so it can be embedded into tool error text
// (and later into structured tool output) without losing the numeric code.
type Error struct {
	Code    Code           `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`

	cause error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	payload := map[string]any{
		"code":    int(e.Code),
		"name":    e.Code.Name(),
		"message": e.Message,
	}
	if len(e.Data) > 0 {
		payload["data"] = e.Data
	}

	b, err := json.Marshal(payload)
	if err == nil {
		return string(b)
	}

	// Best-effort fallback; should be rare.
	if len(e.Data) > 0 {
		return fmt.Sprintf("[%d %s] %s (data=%v)", e.Code, e.Code.Name(), e.Message, e.Data)
	}
	return fmt.Sprintf("[%d %s] %s", e.Code, e.Code.Name(), e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// WithCause sets the wrapped root cause (server-side only).
func (e *Error) WithCause(cause error) *Error {
	if e == nil {
		return nil
	}
	e.cause = cause
	return e
}

// WithData adds a structured diagnostic field.
func (e *Error) WithData(key string, value any) *Error {
	if e == nil {
		return nil
	}
	if e.Data == nil {
		e.Data = make(map[string]any)
	}
	e.Data[key] = value
	return e
}
