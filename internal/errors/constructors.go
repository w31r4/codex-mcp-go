package errors

import (
	"fmt"
	"time"
)

// New constructs a new structured error.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf constructs a formatted structured error.
func Newf(code Code, format string, args ...any) *Error {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap constructs a structured error that wraps an underlying cause.
func Wrap(code Code, message string, cause error) *Error {
	return New(code, message).WithCause(cause)
}

func ErrInvalidParams(message string) *Error {
	return New(InvalidParams, message)
}

func ErrCodexNotFound(cause error) *Error {
	return Wrap(CodexNotFound, "codex executable not found in PATH", cause)
}

func ErrCodexTimeout(timeout time.Duration) *Error {
	return New(CodexTimeout, "codex execution timed out").
		WithData("timeout_seconds", int(timeout.Seconds()))
}

func ErrNoOutputTimeout(timeout time.Duration) *Error {
	return New(NoOutputTimeout, "no output received within timeout").
		WithData("timeout_seconds", int(timeout.Seconds()))
}

func ErrCodexExecutionFailed(message string, cause error) *Error {
	if cause == nil {
		return New(CodexExecutionFailed, message)
	}
	return Wrap(CodexExecutionFailed, message, cause)
}

func ErrWorkdirNotFound(path string) *Error {
	return New(WorkdirNotFound, "working directory does not exist").
		WithData("path", path)
}

func ErrWorkdirNotDirectory(path string) *Error {
	return New(WorkdirNotDirectory, "working directory is not a directory").
		WithData("path", path)
}

func ErrImageNotFound(path string) *Error {
	return New(ImageNotFound, "image file does not exist").
		WithData("path", path)
}

func ErrInvalidSandboxMode(mode string, valid []string) *Error {
	return New(InvalidSandboxMode, "invalid sandbox mode").
		WithData("provided", mode).
		WithData("valid_modes", valid)
}

func ErrParameterProhibited(param string, reason string) *Error {
	return New(ParameterProhibited, "parameter is prohibited").
		WithData("parameter", param).
		WithData("reason", reason)
}

func ErrSessionNotFound(sessionID string) *Error {
	return New(SessionNotFound, "session not found").
		WithData("SESSION_ID", sessionID)
}

func ErrSessionLimitExceeded(maxRunning int, running int) *Error {
	return New(SessionLimitExceeded, "too many concurrent sessions").
		WithData("max_running", maxRunning).
		WithData("running", running)
}
