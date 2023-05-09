package config

import "time"

type Config struct {
	Common Common `toml:"common"   yaml:"common"`
}

type Common struct {
	LogLevel  string `toml:"log_level"       yaml:"log_level"`
	LogFormat string `toml:"log_format"      yaml:"log_format"`
	//	MgmtAddr    string        `toml:"manager_address" yaml:"manager_address"`
	//	LogRequests bool          `toml:"log_requests"    yaml:"log_requests"`
	StopTimeout time.Duration `toml:"stop_timeout"    yaml:"stop_timeout"`
}
