package logger

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"

	dynamic "github.com/gekatateam/dynamic-level-handler"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pkg/prettylog"
)

var Default = slog.New(prettylog.NewHandler(&slog.HandlerOptions{
	Level:     slog.LevelInfo,
	AddSource: false,
}))

func Init(cfg config.Common) error {
	var opts = &slog.HandlerOptions{}
	var handler slog.Handler = nil

	level, err := LevelToLeveler(cfg.LogLevel)
	if err != nil {
		return err
	}

	replaces := make(map[*regexp.Regexp]string, len(cfg.LogReplaces))
	for k, v := range cfg.LogReplaces {
		rex, err := regexp.Compile(k)
		if err != nil {
			return fmt.Errorf("logReplaces: %v: %w", k, err)
		}
		replaces[rex] = v
	}

	opts.Level = level

	switch f := cfg.LogFormat; f {
	case "logfmt":
		opts.ReplaceAttr = attrReplacer(replaces)
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		opts.ReplaceAttr = attrReplacer(replaces)
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "pretty":
		handler = prettylog.NewHandler(opts)
	default:
		return fmt.Errorf("unknown log format: %v", f)
	}

	logger := slog.New(dynamic.New(handler))
	if len(cfg.LogFields) > 0 {
		for k, v := range cfg.LogFields {
			logger = logger.With(k, v)
		}
	}

	Default = logger

	return nil
}

func Mock() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func attrReplacer(replaces map[*regexp.Regexp]string) func(groups []string, a slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			a.Key = "@timestamp"
		}

		if a.Key == slog.MessageKey {
			a.Key = "message"

			value := a.Value.String()
			for rex, replace := range replaces {
				value = rex.ReplaceAllString(value, replace)
			}
			a.Value = slog.StringValue(value)
		}

		return a
	}
}

func LevelToLeveler(level string) (slog.Leveler, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return nil, fmt.Errorf("unknown log level: %v", level)
	}
}

func ShouldLevelToLeveler(level string) slog.Leveler {
	l, err := LevelToLeveler(level)
	if err != nil {
		return nil
	}
	return l
}
