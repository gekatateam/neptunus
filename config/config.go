package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
)

var (
	defaultConfig = Config{
		Common: Common{
			LogLevel:    "info",
			LogFormat:   "logfmt",
		},
		Pipes: nil,
	}
)

type Config struct {
	Common Common    `toml:"common"    yaml:"common"`
	Pipes  []PipeCfg `toml:"pipelines" yaml:"pipelines"`
}

type Common struct {
	LogLevel  string `toml:"log_level"  yaml:"log_level"`
	LogFormat string `toml:"log_format" yaml:"log_format"`
}

type PipeCfg struct {
	Id     string `toml:"id" yaml:"id"`
	Config struct {
		File string `toml:"file" yaml:"file"`
	} `toml:"config" yaml:"config"`
}

func ReadConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	config := defaultConfig

	switch e := filepath.Ext(file); e {
	case ".toml":
		if err := toml.Unmarshal(buf, &config); err != nil {
			return &config, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(buf, &config); err != nil {
			return &config, err
		}
	default:
		return &config, fmt.Errorf("unknown configuration file extention: %v", e)
	}

	return &config, nil
}
