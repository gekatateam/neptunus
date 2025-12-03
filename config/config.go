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
			Storage:  "fs",
			FailFast: false,
			File: FileStorage{
				Directory: ".pipelines",
				Extension: ".toml",
			},
		},
	}
)

type Config struct {
	Common  Common  `toml:"common"  yaml:"common"  json:"common"`
	Runtime Runtime `toml:"runtime" yaml:"runtime" json:"runtime"`
	Engine  Engine  `toml:"engine"  yaml:"engine"  json:"engine"`
}

type Common struct {
	LogLevel    string            `toml:"log_level"    yaml:"log_level"    json:"log_level"`
	LogFormat   string            `toml:"log_format"   yaml:"log_format"   json:"log_format"`
	LogFields   map[string]string `toml:"log_fields"   yaml:"log_fields"   json:"log_fields"`
	LogReplaces map[string]string `toml:"log_replaces" yaml:"log_replaces" json:"log_replaces"`
	HttpPort    string            `toml:"http_port"    yaml:"http_port"    json:"http_port"`
}

type Runtime struct {
	GCPercent  string `toml:"gcpercent"   yaml:"gcpercent"   json:"gcpercent"`
	MemLimit   string `toml:"memlimit"    yaml:"memlimit"    json:"memlimit"`
	MaxThreads int    `toml:"maxthreads"  yaml:"maxthreads"  json:"maxthreads"`
	MaxProcs   int    `toml:"maxprocs"    yaml:"maxprocs"    json:"maxprocs"`
}

type Engine struct {
	Storage    string            `toml:"storage"    yaml:"storage"    json:"storage"`
	FailFast   bool              `toml:"fail_fast"  yaml:"fail_fast"  json:"fail_fast"`
	File       FileStorage       `toml:"fs"         yaml:"fs"         json:"fs"`
	Postgresql PostgresqlStorage `toml:"postgresql" yaml:"postgresql" json:"postgresql"`
}

type FileStorage struct {
	Directory string `toml:"directory" yaml:"directory" json:"directory"`
	Extension string `toml:"extension" yaml:"extension" json:"extension"`
}

type PostgresqlStorage struct {
	Instance string `toml:"instance" yaml:"instance" json:"instance"`
	DSN      string `toml:"dsn"      yaml:"dsn"      json:"dsn"`
	Username string `toml:"username" yaml:"username" json:"username"`
	Password string `toml:"password" yaml:"password" json:"password"`
	Migrate  bool   `toml:"migrate"  yaml:"migrate"  json:"migrate"`
}

func ReadConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	buf = []byte(os.ExpandEnv(string(buf)))
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
		return &config, fmt.Errorf("unknown configuration file extension: %v", e)
	}

	return &config, nil
}
