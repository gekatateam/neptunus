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
			LogLevel:  "info",
			LogFormat: "logfmt",
			MgmtAddr:  ":9600",
		},
		PipeCfg: PipeCfg{
			Storage: "fs",
			File: FileStorage{
				Directory: ".pipelines",
				Extention: ".toml",
			},
		},
	}
)

type Config struct {
	Common  Common  `toml:"common"   yaml:"common"`
	PipeCfg PipeCfg `toml:"pipeline" yaml:"pipeline"`
}

type Common struct {
	LogLevel  string         `toml:"log_level"       yaml:"log_level"`
	LogFormat string         `toml:"log_format"      yaml:"log_format"`
	LogFields map[string]any `toml:"log_fields"      yaml:"log_fields"`
	MgmtAddr  string         `toml:"manager_address" yaml:"manager_address"`
}

type PipeCfg struct {
	Storage string      `toml:"storage" yaml:"storage"`
	File    FileStorage `toml:"fs"      yaml:"fs"`
}

type FileStorage struct {
	Directory string `toml:"directory" yaml:"directory"`
	Extention string `toml:"extention" yaml:"extention"`
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
