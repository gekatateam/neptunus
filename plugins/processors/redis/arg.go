package redisproc

import (
	"fmt"
	"regexp"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins/common/convert"
)

var fieldRequest = regexp.MustCompile(`^{{\s(.+)\s}}$`)

type pluginArg struct {
	isField bool
	value   any
}

func newPluginArg(rawArg any) pluginArg {
	if v, ok := rawArg.(string); ok {
		if match := fieldRequest.FindStringSubmatch(v); len(match) == 2 {
			return pluginArg{
				isField: true,
				value:   match[1],
			}
		}
	}

	return pluginArg{
		isField: false,
		value:   rawArg,
	}
}

func commandArgs(pluginArgs []pluginArg, e *core.Event) ([]any, error) {
	commandArgs := make([]any, 0, len(pluginArgs))

	for _, v := range pluginArgs {
		if v.isField {
			rawField, err := e.GetField(v.value.(string))
			if err != nil {
				return nil, fmt.Errorf("field %v not found in event: %w", v.value, err)
			}

			switch field := rawField.(type) {
			case []any:
				commandArgs = append(commandArgs, field...)
			case map[string]any:
				for k, v := range field {
					commandArgs = append(commandArgs, k, v)
				}
			default:
				commandArgs = append(commandArgs, field)
			}

			continue
		}
		commandArgs = append(commandArgs, v.value)
	}
	return commandArgs, nil
}

func unpackResult(cmdResult any) (any, error) {
	if cmdResult == nil {
		return nil, nil
	}

	// https://github.com/redis/go-redis/blob/master/internal/proto/reader.go#L267
	if redisMap, ok := cmdResult.(map[any]any); ok {
		dataMap := make(map[string]any, len(redisMap))
		for k, v := range redisMap {
			key, err := convert.AnyToString(k)
			if err != nil {
				return nil, fmt.Errorf("%v: %w", k, err)
			}
			dataMap[key] = v
		}
		return dataMap, nil
	}

	return cmdResult, nil
}
