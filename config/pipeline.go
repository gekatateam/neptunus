package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
)

// TODO: rewrite to structs
type Pipeline struct {
	Inputs     []PluginSet `toml:"inputs"     yaml:"inputs"`
	Processors []PluginSet `toml:"processors" yaml:"processors"`
	Outputs    []PluginSet `toml:"outputs"    yaml:"outputs"`
}

type PluginSet map[string]Plugin

type Plugin map[string]any 

func (p Plugin) Alias() string {
	aliasRaw, ok := p["alias"]
	if !ok {
		return ""
	}
	alias, ok := aliasRaw.(string)
	if !ok {
		return ""
	}
	return alias
}

func (p Plugin) Filters() PluginSet {
	filtersRaw, ok := p["filters"]
	if !ok {
		return nil
	}
	filtersSet, ok := filtersRaw.(map[string]any)
	if !ok {
		return nil
	}
	var filters = make(PluginSet, len(filtersSet))
	for key, value := range filtersSet {
		filterCfg, ok := value.(map[string]any)
		if !ok {
			filters[key] = Plugin{}
			continue
		}
		filters[key] = Plugin(filterCfg)
	}
	return filters
}

func ReadPipeline(file string) (*Pipeline, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	pipeline := Pipeline{}

	switch e := filepath.Ext(file); e {
	case ".toml":
		if err := toml.Unmarshal(buf, &pipeline); err != nil {
			return &pipeline, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(buf, &pipeline); err != nil {
			return &pipeline, err
		}
	default:
		return &pipeline, fmt.Errorf("unknown pipeline file extention: %v", e)
	}

	return &pipeline, nil
}
