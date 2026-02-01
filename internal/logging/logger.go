package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	With(args ...any) Logger

	// Slog returns the underlying slog logger (for SDK integration).
	Slog() *slog.Logger
}

type Config struct {
	// Level: debug, info, warn, error
	Level string `toml:"level"`
	// Format: json, text
	Format string `toml:"format"`
	// Output: stdout, stderr, file
	Output string `toml:"output"`
	// FilePath is used when Output=file.
	FilePath string `toml:"file_path"`
}

func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "json",
		Output: "stderr",
	}
}

type SlogLogger struct {
	logger *slog.Logger
}

func New(cfg Config) (*SlogLogger, error) {
	level := parseLevel(cfg.Level)
	writer, err := selectWriter(cfg.Output, cfg.FilePath)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	case "json", "":
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	return &SlogLogger{logger: slog.New(handler)}, nil
}

func (l *SlogLogger) Slog() *slog.Logger {
	return l.logger
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{logger: l.logger.With(args...)}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

func selectWriter(output string, filePath string) (io.Writer, error) {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "stdout":
		return os.Stdout, nil
	case "stderr", "":
		return os.Stderr, nil
	case "file":
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return os.Stderr, nil
	}
}
