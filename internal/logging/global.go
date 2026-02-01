package logging

import (
	"log/slog"
	"sync/atomic"
)

var globalLogger atomic.Value // stores Logger

func init() {
	globalLogger.Store(Logger(&SlogLogger{logger: slog.Default()}))
}

func SetGlobalLogger(l Logger) {
	if l == nil {
		return
	}
	globalLogger.Store(l)
}

func GetLogger() Logger {
	if v := globalLogger.Load(); v != nil {
		return v.(Logger)
	}
	return &SlogLogger{logger: slog.Default()}
}
