package starlark

import (
	"errors"
	"log/slog"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/gekatateam/neptunus/plugins/common/elog"
)

/*
Package log provides plugin logger into Starlark code

there is a four functions - `debug`, `info`, `warn` and `error` - each for concrete level
each func signature is `logfunc(msg, event=None, error=None)`, where:
 - `msg` - is a message to log; it is alvays required
 - `event` - is an optional Event; if passed, event attrs (id, uuid, routing key) will be added to log row
 - `error` - is an optional Error; if passed, `error` key will be added to row
*/

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
