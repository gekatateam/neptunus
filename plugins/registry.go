package plugins

import (
	"fmt"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
)

// processors
type processorFunc func(config map[string]any, alias, pipeline string, log logger.Logger) (core.Processor, error)

var processors = make(map[string]processorFunc)

func AddProcessor(key string, p processorFunc) {
	_, exists := processors[key]
	if exists {
		panic(fmt.Errorf("duplicate processor func added: %v", key))
	}

	processors[key] = p
}

func GetProcessor(key string) (processorFunc, bool) {
	p, ok := processors[key]
	return p, ok
}

// filters
type filterFunc func(config map[string]any, alias, pipeline string, log logger.Logger) (core.Filter, error)

var filters = make(map[string]filterFunc)

func AddFilter(key string, f filterFunc) {
	_, exists := filters[key]
	if exists {
		panic(fmt.Errorf("duplicate filter func added: %v", key))
	}

	filters[key] = f
}

func GetFilter(key string) (filterFunc, bool) {
	f, ok := filters[key]
	return f, ok
}

// inputs
type inputFunc func(config map[string]any, alias, pipeline string, parser core.Parser, log logger.Logger) (core.Input, error)

var inputs = make(map[string]inputFunc)

func AddInput(key string, i inputFunc) {
	_, exists := inputs[key]
	if exists {
		panic(fmt.Errorf("duplicate input func added: %v", key))
	}

	inputs[key] = i
}

func GetInput(key string) (inputFunc, bool) {
	i, ok := inputs[key]
	return i, ok
}

// outputs
type outputFunc func(config map[string]any, alias, pipeline string, log logger.Logger) (core.Output, error)

var outputs = make(map[string]outputFunc)

func AddOutput(key string, o outputFunc) {
	_, exists := outputs[key]
	if exists {
		panic(fmt.Errorf("duplicate output func added: %v", key))
	}

	outputs[key] = o
}

func GetOutput(key string) (outputFunc, bool) {
	o, ok := outputs[key]
	return o, ok
}

// parsers
type parserFunc func(config map[string]any, alias, pipeline string, log logger.Logger) (core.Parser, error)

var parsers = make(map[string]parserFunc)

func AddParser(key string, p parserFunc) {
	_, exists := parsers[key]
	if exists {
		panic(fmt.Errorf("duplicate parser func added: %v", key))
	}

	parsers[key] = p
}

func GetParser(key string) (parserFunc, bool) {
	p, ok := parsers[key]
	return p, ok
}
