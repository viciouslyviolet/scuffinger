package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"scuffinger/internal/config"
	"scuffinger/internal/metrics"
)

// Format represents a log output format.
type Format string

const (
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
	FormatYAML  Format = "yaml"
)

// Logger wraps slog.Logger with caller-tracking Debug and configurable output format.
type Logger struct {
	sl *slog.Logger
}

// New creates a Logger that writes to os.Stderr using the format and level from cfg.
// Logs are sent to stderr by default so they never interfere with command output on stdout.
func New(cfg config.LogConfig) *Logger {
	return newLogger(cfg, os.Stderr)
}

// NewWithWriter creates a Logger that writes to w (useful for testing).
func NewWithWriter(cfg config.LogConfig, w io.Writer) *Logger {
	return newLogger(cfg, w)
}

func newLogger(cfg config.LogConfig, w io.Writer) *Logger {
	level := parseLevel(cfg.Level)
	format := Format(strings.ToLower(cfg.Format))

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	switch format {
	case FormatYAML:
		handler = NewYAMLHandler(w, opts)
	case FormatPlain:
		handler = slog.NewTextHandler(w, opts)
	default: // JSON is the default
		handler = slog.NewJSONHandler(w, opts)
	}

	return &Logger{sl: slog.New(handler)}
}

// Debug logs at DEBUG level. It automatically captures the caller's function
// name, file path, and line number via runtime reflection.
// keysAndValues are optional key/value pairs appended as structured metadata.
//
//	logger.Debug("cache miss", "key", "user:42", "latency_ms", 12)
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	metrics.IncLogMessage("debug")
	if !l.sl.Enabled(context.Background(), slog.LevelDebug) {
		return
	}

	// Capture caller information via runtime reflection.
	callerAttrs := callerFields(1)
	attrs := make([]any, 0, len(callerAttrs)+len(keysAndValues))
	attrs = append(attrs, callerAttrs...)
	attrs = append(attrs, keysAndValues...)

	l.sl.Debug(msg, attrs...)
}

// Info logs at INFO level.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	metrics.IncLogMessage("info")
	l.sl.Info(msg, keysAndValues...)
}

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, keysAndValues ...any) {
	metrics.IncLogMessage("warn")
	l.sl.Warn(msg, keysAndValues...)
}

// Error logs at ERROR level.
func (l *Logger) Error(msg string, keysAndValues ...any) {
	metrics.IncLogMessage("error")
	l.sl.Error(msg, keysAndValues...)
}

// With returns a new Logger that always includes the given key/value pairs.
func (l *Logger) With(keysAndValues ...any) *Logger {
	return &Logger{sl: l.sl.With(keysAndValues...)}
}

// Slog returns the underlying *slog.Logger for interop with libraries that
// accept one directly.
func (l *Logger) Slog() *slog.Logger {
	return l.sl
}

// ── helpers ──────────────────────────────────────────────────────────────────

// callerFields returns key/value pairs with function, file, and line of the
// caller at the given skip depth (0 = callerFields itself).
func callerFields(skip int) []any {
	// +1 because callerFields is itself one frame.
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return nil
	}

	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = shortFuncName(fn.Name())
	}

	return []any{
		"caller.function", funcName,
		"caller.file", shortFilePath(file),
		"caller.line", line,
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// shortFuncName returns the last package-qualified portion of a full function name.
// "scuffinger/internal/services.(*Manager).ConnectAll" → "services.(*Manager).ConnectAll"
func shortFuncName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// shortFilePath trims to a project-relative path when possible.
// "/home/user/GolandProjects/scuffinger/cmd/serve.go" → "cmd/serve.go"
func shortFilePath(path string) string {
	const marker = "scuffinger/"
	if i := strings.LastIndex(path, marker); i >= 0 {
		return path[i+len(marker):]
	}
	return path
}
