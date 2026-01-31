package errors

import (
	"encoding/json"
	stderrors "errors"
	"testing"
)

func TestCodeName(t *testing.T) {
	tests := []struct {
		code Code
		want string
	}{
		{ParseError, "ParseError"},
		{InvalidRequest, "InvalidRequest"},
		{MethodNotFound, "MethodNotFound"},
		{InvalidParams, "InvalidParams"},
		{InternalError, "InternalError"},
		{CodexNotFound, "CodexNotFound"},
		{CodexTimeout, "CodexTimeout"},
		{CodexExecutionFailed, "CodexExecutionFailed"},
		{WorkdirNotFound, "WorkdirNotFound"},
		{WorkdirNotDirectory, "WorkdirNotDirectory"},
		{ImageNotFound, "ImageNotFound"},
		{InvalidSandboxMode, "InvalidSandboxMode"},
		{ParameterProhibited, "ParameterProhibited"},
		{SessionNotFound, "SessionNotFound"},
		{NoOutputTimeout, "NoOutputTimeout"},
		{Code(0), "UnknownError"},
		{Code(-999999), "UnknownError"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.code.Name(); got != tt.want {
				t.Fatalf("Code(%d).Name() = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestError_JSONString(t *testing.T) {
	err := New(InvalidParams, "bad input").
		WithData("field", "PROMPT").
		WithData("reason", "required")

	var got map[string]any
	if unmarshalErr := json.Unmarshal([]byte(err.Error()), &got); unmarshalErr != nil {
		t.Fatalf("Error() output is not valid JSON: %v\noutput: %s", unmarshalErr, err.Error())
	}

	if got["code"] != float64(InvalidParams) {
		t.Fatalf("code = %v, want %v", got["code"], InvalidParams)
	}
	if got["name"] != InvalidParams.Name() {
		t.Fatalf("name = %v, want %v", got["name"], InvalidParams.Name())
	}
	if got["message"] != "bad input" {
		t.Fatalf("message = %v, want %v", got["message"], "bad input")
	}

	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data missing or wrong type: %T", got["data"])
	}
	if data["field"] != "PROMPT" {
		t.Fatalf("data.field = %v, want %v", data["field"], "PROMPT")
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := stderrors.New("root cause")
	err := Wrap(CodexExecutionFailed, "failed", cause)
	if !stderrors.Is(err, cause) {
		t.Fatalf("expected errors.Is to match wrapped cause")
	}
}
