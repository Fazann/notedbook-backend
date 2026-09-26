package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Fazann/notedbook-backend/config"
)

// Log is the shared structured logger. Output is JSON in production and
// human-readable text elsewhere, e.g.
//
//	time=2026-09-26T10:00:00Z level=INFO source=main.go:25 msg="migration success" tables=2
var Log *slog.Logger

func init() {
	opts := &slog.HandlerOptions{
		Level:     parseLevel(os.Getenv("LOG_LEVEL")),
		AddSource: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Shorten source to "dir/file.go:line".
			if src, ok := a.Value.Any().(*slog.Source); ok {
				src.File = filepath.Join(filepath.Base(filepath.Dir(src.File)), filepath.Base(src.File))
			}
			return a
		},
	}

	var handler slog.Handler
	if config.Environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	Log = slog.New(handler).With(slog.String("env", config.Environment))
	slog.SetDefault(Log)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		if config.Environment == "production" {
			return slog.LevelInfo
		}
		return slog.LevelDebug
	}
}

// Helpers take a message plus key/value pairs:
//
//	logger.Info("user created", "user_id", id)
//	logger.Error("upload failed", "err", err, "file", name)
func Debug(msg string, args ...any) { log(slog.LevelDebug, msg, args...) }
func Info(msg string, args ...any)  { log(slog.LevelInfo, msg, args...) }
func Warn(msg string, args ...any)  { log(slog.LevelWarn, msg, args...) }
func Error(msg string, args ...any) { log(slog.LevelError, msg, args...) }

// Fatal logs at error level and exits the program with status 1.
func Fatal(msg string, args ...any) {
	log(slog.LevelError, msg, args...)
	os.Exit(1)
}

// With returns a child logger that always includes the given attributes,
// e.g. logger.With("module", "auth").Info("login ok").
func With(args ...any) *slog.Logger { return Log.With(args...) }

// log records the caller of the helper (not this file) as the source.
func log(level slog.Level, msg string, args ...any) {
	ctx := context.Background()
	if !Log.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = Log.Handler().Handle(ctx, r)
}
