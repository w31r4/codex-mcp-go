package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseLevel(tt.in); got != tt.want {
				t.Fatalf("parseLevel(%q)=%v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNew_FileOutput_WritesToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	l, err := New(Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: path,
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	l.Info("hello", "value", 1)

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile failed: %v", readErr)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("log file does not contain message; got: %s", string(data))
	}
}

func TestNew_FileOutput_MissingPathErrors(t *testing.T) {
	_, err := New(Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: "",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
