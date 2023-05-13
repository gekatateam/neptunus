package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
)

var (
	defaultConfig = Config{
		Common: Common{
			LogLevel:    "info",
			LogFormat:   "logfmt",
			StopTimeout: 5 * time.Second,
		},
		Pipelines: nil,
	}
)

type Config struct {
	Common    Common     `toml:"common"    yaml:"common"`
	Pipelines []Pipeline `toml:"pipelines" yaml:"pipelines"`
}

type Common struct {
	LogLevel    string        `toml:"log_level"    yaml:"log_level"`
	LogFormat   string        `toml:"log_format"   yaml:"log_format"`
	StopTimeout time.Duration `toml:"stop_timeout" yaml:"stop_timeout"`
}

type Pipeline struct {
	Id     string `toml:"id"     yaml:"id"`
	Config string `toml:"config" yaml:"config"`
	Lines  int    `toml:"lines"  yaml:"lines"`
}

func Read(file string) (*Config, error) {
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
