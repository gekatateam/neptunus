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
	created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	, updated_at  TIMESTAMP WITH TIME ZONE
	, deleted_at  TIMESTAMP WITH TIME ZONE
	, id          TEXT CONSTRAINT unique_ids_only UNIQUE
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

	add = `
INSERT INTO pipelines 
(id, lines, run, buffer, consistency, log_level, vars, keykeepers, inputs, processors, outputs)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);`

	add_check = `
SELECT id, deleted_at FROM pipelines
WHERE id = $1;`

	delete = `
DELETE FROM pipelines
WHERE id = $1;`

	update = `
UPDATE pipelines 
SET
	updated_at = NOW()
	, lines       = $2
	, run         = $3
	, buffer      = $4
	, consistency = $5
	, log_level   = $6
	, vars        = $7
	, keykeepers  = $8
	, inputs      = $9
	, processors  = $10
	, outputs     = $11
WHERE id = $1;`

	get = `
SELECT lines, run, buffer, consistency, log_level, vars, keykeepers, inputs, processors, outputs
FROM pipelines
WHERE id = $1 and deleted_at IS NULL;`

	list = `
SELECT id, lines, run, buffer, consistency, log_level, vars, keykeepers, inputs, processors, outputs
FROM pipelines
WHERE deleted_at IS NULL;`
)

type postgresql struct {
}

func PostgreSQL(cfg config.PostgresqlStorage) (*postgresql, error) {
	panic(errors.ErrUnsupported)
}

func (s *postgresql) List() ([]*config.Pipeline, error)
func (s *postgresql) Get(id string) (*config.Pipeline, error)
func (s *postgresql) Add(pipe *config.Pipeline) error
func (s *postgresql) Update(pipe *config.Pipeline) error
func (s *postgresql) Delete(id string) error
func (s *postgresql) Close() error
