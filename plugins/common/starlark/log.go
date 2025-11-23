package starlark

import (
	"errors"
	"log/slog"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/gekatateam/neptunus/plugins/common/elog"
)

const loggerKey = "log.logger"

var Log = &starlarkstruct.Module{
	Name: "log",
	Members: starlark.StringDict{
		"debug": starlark.NewBuiltin("debug", LogFunc),
		"info":  starlark.NewBuiltin("info", LogFunc),
		"warn":  starlark.NewBuiltin("warn", LogFunc),
		"error": starlark.NewBuiltin("error", LogFunc),
	},
}

func LogFunc(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	logger := t.Local(loggerKey).(*slog.Logger)

	var (
		msg   string
		event *Event
		error Error
	)

	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "msg", &msg, "event?", &event, "error?", &error); err != nil {
		return starlark.None, err
	}

	var keys []any
	if error.Truth() {
		keys = append(keys, "error", errors.New(error.String()))
	}
	if event != nil {
		keys = append(keys, elog.EventGroup(event.event))
	}

	switch fn.Name() {
	case "debug":
		logger.Debug(msg, keys...)
	case "info":
		logger.Info(msg, keys...)
	case "warn":
		logger.Warn(msg, keys...)
	case "error":
		logger.Error(msg, keys...)
	}

	return starlark.None, nil
}
