package mcp

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"

	cerrors "github.com/w31r4/codex-mcp-go/internal/errors"
)

func TestHandleCodexTool_InvalidPrompt(t *testing.T) {
	_, _, err := handleCodexTool(context.Background(), nil, CodexInput{
		PROMPT: "",
		Cd:     ".",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.InvalidParams {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.InvalidParams)
	}
}

func TestHandleCodexTool_InvalidCd(t *testing.T) {
	_, _, err := handleCodexTool(context.Background(), nil, CodexInput{
		PROMPT: "hi",
		Cd:     "",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.InvalidParams {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.InvalidParams)
	}
}

func TestHandleCodexTool_WorkdirNotFound(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	_, _, err := handleCodexTool(context.Background(), nil, CodexInput{
		PROMPT: "hi",
		Cd:     missing,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.WorkdirNotFound {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.WorkdirNotFound)
	}
	if cerr.Data["path"] != missing {
		t.Fatalf("data.path=%v, want %v", cerr.Data["path"], missing)
	}
}

func TestHandleCodexTool_WorkdirNotDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, _, err := handleCodexTool(context.Background(), nil, CodexInput{
		PROMPT: "hi",
		Cd:     path,
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.WorkdirNotDirectory {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.WorkdirNotDirectory)
	}
}

func TestHandleCodexTool_ModelProhibited(t *testing.T) {
	dir := t.TempDir()
	_, _, err := handleCodexTool(context.Background(), nil, CodexInput{
		PROMPT: "hi",
		Cd:     dir,
		Model:  "gpt-foo",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.ParameterProhibited {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.ParameterProhibited)
	}
}

func TestHandleCodexTool_ImageNotFound(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.png")

	_, _, err := handleCodexTool(context.Background(), nil, CodexInput{
		PROMPT: "hi",
		Cd:     dir,
		Image:  []string{missing},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	var cerr *cerrors.Error
	if !stderrors.As(err, &cerr) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if cerr.Code != cerrors.ImageNotFound {
		t.Fatalf("code=%v, want %v", cerr.Code, cerrors.ImageNotFound)
	}
	if cerr.Data["path"] != missing {
		t.Fatalf("data.path=%v, want %v", cerr.Data["path"], missing)
	}
}
