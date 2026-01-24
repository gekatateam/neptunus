package plugins

import (
	"fmt"

	"github.com/gekatateam/neptunus/core"
)

type (
	processorFunc    func() core.Processor
	filterFunc       func() core.Filter
	inputFunc        func() core.Input
	outputFunc       func() core.Output
	parserFunc       func() core.Parser
	serializerFunc   func() core.Serializer
	keykeeperFunc    func() core.Keykeeper
	lookupFunc       func() core.Lookup
	compressorFunc   func() core.Compressor
	decompressorFunc func() core.Decompressor
)

var (
	processors    = make(map[string]processorFunc)
	filters       = make(map[string]filterFunc)
	inputs        = make(map[string]inputFunc)
	outputs       = make(map[string]outputFunc)
	parsers       = make(map[string]parserFunc)
	serializers   = make(map[string]serializerFunc)
	keykeepers    = make(map[string]keykeeperFunc)
	lookups       = make(map[string]lookupFunc)
	compressors   = make(map[string]compressorFunc)
	decompressors = make(map[string]decompressorFunc)
)

// processors
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

// lookups
func AddLookup(key string, p lookupFunc) {
	if _, exists := lookups[key]; exists {
		panic(fmt.Errorf("duplicate lookup func added: %v", key))
	}

	lookups[key] = p
}

func GetLookup(key string) (lookupFunc, bool) {
	p, ok := lookups[key]
	return p, ok
}

// compressors
func AddCompressor(key string, p compressorFunc) {
	if _, exists := compressors[key]; exists {
		panic(fmt.Errorf("duplicate compressor func added: %v", key))
	}

	compressors[key] = p
}

func GetCompressor(key string) (compressorFunc, bool) {
	p, ok := compressors[key]
	return p, ok
}

// decompressors
func AddDecompressor(key string, p decompressorFunc) {
	if _, exists := decompressors[key]; exists {
		panic(fmt.Errorf("duplicate decompressor func added: %v", key))
	}

	decompressors[key] = p
}

func GetDecompressor(key string) (decompressorFunc, bool) {
	p, ok := decompressors[key]
	return p, ok
}
