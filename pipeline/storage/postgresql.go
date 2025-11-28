package storage

import (
	"errors"

	"github.com/gekatateam/neptunus/config"
)

// type Pipeline struct {
// 	Settings   PipeSettings   `toml:"settings"   yaml:"settings"   json:"settings"`
// 	Vars       map[string]any `toml:"vars"       yaml:"vars"       json:"vars"`
// 	Inputs     []PluginSet    `toml:"inputs"     yaml:"inputs"     json:"inputs"`
// 	Processors []PluginSet    `toml:"processors" yaml:"processors" json:"processors"`
// 	Outputs    []PluginSet    `toml:"outputs"    yaml:"outputs"    json:"outputs"`
// 	Keykeepers []PluginSet    `toml:"keykeepers" yaml:"keykeepers" json:"keykeepers"`
// }

// type PipeSettings struct {
// 	Id          string `toml:"id"          yaml:"id"          json:"id"`
// 	Lines       int    `toml:"lines"       yaml:"lines"       json:"lines"`
// 	Run         bool   `toml:"run"         yaml:"run"         json:"run"`
// 	Buffer      int    `toml:"buffer"      yaml:"buffer"      json:"buffer"`
// 	Consistency string `toml:"consistency" yaml:"consistency" json:"consistency"`
// 	LogLevel    string `toml:"log_level"   yaml:"log_level"   json:"log_level"`
// }

const (
	migrate = `
CREATE TABLE IF NOT EXISTS pipelines (
	created_at    TIMESTAMP WITH TIME ZONE
	, updated_at  TIMESTAMP WITH TIME ZONE
	, deleted_at  TIMESTAMP WITH TIME ZONE
	, id          TEXT
	, lines       INTEGER
	, run         BOOLEAN
	, buffer      INTEGER
	, consistency TEXT
	, log_level   TEXT
	, vars        JSON
	, keykeepers  JSON
	, inputs      JSON
	, processors  JSON
	, outputs     JSON
);`
)

type postgresql struct {
}

func PostgreSQL(cfg config.PostgresqlStorage) (*postgresql, error) {
	panic(errors.ErrUnsupported)
}
