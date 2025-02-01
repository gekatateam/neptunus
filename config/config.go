package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
)

var (
	Default = Config{
		Common: Common{
			LogLevel:  "info",
			LogFormat: "logfmt",
			HttpPort:  ":9600",
		},
		Engine: Engine{
			Storage: "fs",
			File: FileStorage{
				Directory: ".pipelines",
				Extention: ".toml",
			},
		},
	}
)

type Config struct {
	Common Common `toml:"common" yaml:"common" json:"common"`
	Engine Engine `toml:"engine" yaml:"engine" json:"engine"`
}

type Common struct {
	LogLevel  string            `toml:"log_level"  yaml:"log_level"  json:"log_level"`
	LogFormat string            `toml:"log_format" yaml:"log_format" json:"log_format"`
	LogFields map[string]string `toml:"log_fields" yaml:"log_fields" json:"log_fields"`
	HttpPort  string            `toml:"http_port"  yaml:"http_port"  json:"http_port"`
}

type Engine struct {
	Storage string      `toml:"storage" yaml:"storage" json:"storage"`
	File    FileStorage `toml:"fs"      yaml:"fs"      json:"fs"`
}

type FileStorage struct {
	Directory string `toml:"directory" yaml:"directory" json:"directory"`
	Extention string `toml:"extention" yaml:"extention" json:"extention"`
}

func ReadConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	config := Default

	switch e := filepath.Ext(file); e {
	case ".json":
		if err := json.Unmarshal(buf, &config); err != nil {
			return &config, err
		}
	case ".toml":
		if err := toml.Unmarshal(buf, &config); err != nil {
			return &config, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(buf, &config); err != nil {
			return &config, err
		}
	case "json":
		if err := json.Unmarshal(buf, &config); err != nil {
			return &config, err
		}
	default:
		return &config, fmt.Errorf("unknown configuration file extention: %v", e)
	}

	return &config, nil
}
