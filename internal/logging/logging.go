package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Config keys are read from environment variables:
// - PICOBUS_LOG_LEVEL: trace, debug, info, warn, error
// - PICOBUS_LOG_FORMAT: json or text

// Init configures the package-global slog logger.
func Init() {
	level := strings.ToLower(os.Getenv("PICOBUS_LOG_LEVEL"))
	format := strings.ToLower(os.Getenv("PICOBUS_LOG_FORMAT"))

	if level == "" {
		level = "info"
	}

	if format == "" {
		format = "text"
	}

	options := &slog.HandlerOptions{
		AddSource: true,
		Level:     parseLevel(level),
	}

	var handler slog.Handler

	// ensure handler reflects requested format
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, options)
	case "text":
		handler = slog.NewTextHandler(os.Stderr, options)
	default:
		panic("unknown log format: " + format)
	}

	slog.SetDefault(slog.New(handler))
}

// Logger returns the package-global logger.
func Logger() *slog.Logger { return slog.Default() }

func parseLevel(s string) slog.Level {
	switch s {
	case "trace":
		return slog.Level(-4)
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		panic("unknown log level: " + s)
	}
}

func Trace(msg string, args ...any) { Logger().Log(context.Background(), slog.Level(-4), msg, args...) }
func Debug(msg string, args ...any) { Logger().Debug(msg, args...) }
func Info(msg string, args ...any)  { Logger().Info(msg, args...) }
func Warn(msg string, args ...any)  { Logger().Warn(msg, args...) }
func Error(msg string, args ...any) { Logger().Error(msg, args...) }
