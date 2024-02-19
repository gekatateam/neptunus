package plugins

import (
	"fmt"

	"github.com/gekatateam/neptunus/core"
)

// processors
type processorFunc func() core.Processor

var processors = make(map[string]processorFunc)

func AddProcessor(key string, p processorFunc) {
	if _, exists := processors[key]; exists {
		panic(fmt.Errorf("duplicate processor func added: %v", key))
	}

	processors[key] = p
}

func GetProcessor(key string) (processorFunc, bool) {
	p, ok := processors[key]
	return p, ok
}

// filters
type filterFunc func() core.Filter

var filters = make(map[string]filterFunc)

func AddFilter(key string, f filterFunc) {
	if _, exists := filters[key]; exists {
		panic(fmt.Errorf("duplicate filter func added: %v", key))
	}

	filters[key] = f
}

func GetFilter(key string) (filterFunc, bool) {
	f, ok := filters[key]
	return f, ok
}

// inputs
type inputFunc func() core.Input

var inputs = make(map[string]inputFunc)

func AddInput(key string, i inputFunc) {
	if _, exists := inputs[key]; exists {
		panic(fmt.Errorf("duplicate input func added: %v", key))
	}

	inputs[key] = i
}

func GetInput(key string) (inputFunc, bool) {
	i, ok := inputs[key]
	return i, ok
}

// outputs
type outputFunc func() core.Output

var outputs = make(map[string]outputFunc)

func AddOutput(key string, o outputFunc) {
	if _, exists := outputs[key]; exists {
		panic(fmt.Errorf("duplicate output func added: %v", key))
	}

	outputs[key] = o
}

func GetOutput(key string) (outputFunc, bool) {
	o, ok := outputs[key]
	return o, ok
}

// parsers
type parserFunc func() core.Parser

var parsers = make(map[string]parserFunc)

func AddParser(key string, p parserFunc) {
	if _, exists := parsers[key]; exists {
		panic(fmt.Errorf("duplicate parser func added: %v", key))
	}

	parsers[key] = p
}

func GetParser(key string) (parserFunc, bool) {
	p, ok := parsers[key]
	return p, ok
}

// serializers
type serializerFunc func() core.Serializer

var serializers = make(map[string]serializerFunc)

func AddSerializer(key string, p serializerFunc) {
	if _, exists := serializers[key]; exists {
		panic(fmt.Errorf("duplicate serializer func added: %v", key))
	}

	serializers[key] = p
}

func GetSerializer(key string) (serializerFunc, bool) {
	p, ok := serializers[key]
	return p, ok
}

// keykeepers
type keykeeperFunc func() core.Keykeeper

var keykeepers = make(map[string]keykeeperFunc)

func AddKeykeeper(key string, p keykeeperFunc) {
	if _, exists := keykeepers[key]; exists {
		panic(fmt.Errorf("duplicate keykeeper func added: %v", key))
	}

	keykeepers[key] = p
}

func GetKeykeeper(key string) (keykeeperFunc, bool) {
	p, ok := keykeepers[key]
	return p, ok
}
