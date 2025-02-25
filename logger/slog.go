package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/gekatateam/neptunus/pkg/prettylog"

	"github.com/gekatateam/neptunus/config"
)

var Default = slog.New(prettylog.NewHandler(&slog.HandlerOptions{
	Level:     slog.LevelInfo,
	AddSource: false,
}))

func Init(cfg config.Common) error {
	var opts = &slog.HandlerOptions{}
	var handler slog.Handler = nil

	switch l := cfg.LogLevel; l {
	case "debug":
		opts.Level = slog.LevelDebug
	case "info":
		opts.Level = slog.LevelInfo
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		return fmt.Errorf("unknown log level: %v", l)
	}

	switch f := cfg.LogFormat; f {
	case "logfmt":
		opts.ReplaceAttr = attrReplacer
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		opts.ReplaceAttr = attrReplacer
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "pretty":
		handler = prettylog.NewHandler(opts)
	default:
		return fmt.Errorf("unknown log format: %v", f)
	}

	logger := slog.New(handler)
	if len(cfg.LogFields) > 0 {
		for k, v := range cfg.LogFields {
			logger = logger.With(k, v)
		}
	}

	Default = logger

	return nil
}

func Mock() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func attrReplacer(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		a.Key = "@timestamp"
	}

	if a.Key == slog.MessageKey {
		a.Key = "message"
	}

	return a
}
