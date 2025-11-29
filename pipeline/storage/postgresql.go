package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	pgxstd "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
	pkg "github.com/gekatateam/neptunus/pkg/tls"
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
	timeout   = time.Second * 30
	connlimit = 1

	migrate = `
CREATE TABLE IF NOT EXISTS pipelines (
	created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	, updated_at  TIMESTAMP WITH TIME ZONE
	, deleted_at  TIMESTAMP WITH TIME ZONE
	, id          TEXT CONSTRAINT unique_ids_only UNIQUE
	, run         BOOLEAN
	, lines       INTEGER
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

	check = `
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

type storedPipeline struct {
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   sql.NullTime   `db:"updated_at"`
	DeletedAt   sql.NullTime   `db:"deleted_at"`
	Id          string         `db:"id"`
	Run         bool           `db:"run"`
	Lines       int            `db:"lines"`
	Buffer      int            `db:"buffer"`
	Consistency string         `db:"consistency"`
	LogLevel    string         `db:"log_level"`
	Vars        types.JSONText `db:"vars"`
	Keykeepers  types.JSONText `db:"keykeepers"`
	Inputs      types.JSONText `db:"inputs"`
	Processors  types.JSONText `db:"processors"`
	Outputs     types.JSONText `db:"outputs"`
}

type postgresql struct {
	db *sqlx.DB
}

func PostgreSQL(cfg config.PostgresqlStorage) (*postgresql, error) {
	config, err := pgx.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}

	if cfg.TLSEnable {
		tls, err := pkg.NewConfigBuilder().
			RootCaFile(cfg.TLSCAFile).
			KeyPairFile(cfg.TLSCertFile, cfg.TLSKeyFile).
			SkipVerify(cfg.TLSInsecureSkipVerify).
			ServerName(cfg.TLSServerName).
			MinMaxVersion(cfg.TLSMinVersion, "").
			Build()
		if err != nil {
			return nil, fmt.Errorf("tls: %w", err)
		}

		config.TLSConfig = tls
	}

	config.User = cfg.Username
	config.Password = cfg.Password

	db := pgxstd.OpenDB(*config)
	db.SetConnMaxIdleTime(timeout)
	db.SetConnMaxLifetime(timeout * 2)
	db.SetMaxIdleConns(connlimit)
	db.SetMaxOpenConns(connlimit)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	if cfg.Migrate {
		_, err := db.ExecContext(ctx, migrate)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}

	return &postgresql{db: sqlx.NewDb(db, "pgx")}, nil
}

func (s *postgresql) List() ([]*config.Pipeline, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	storedPipes := make([]storedPipeline, 0)
	err := s.db.SelectContext(ctx, &storedPipes, list)
	if err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	pipes := make([]*config.Pipeline, 0, len(storedPipes))
	for _, v := range storedPipes {
		p, err := storedToInternal(v)
		if err != nil {
			return nil, &pipeline.ValidationError{Err: err}
		}

		pipes = append(pipes, p)
	}

	return pipes, nil
}

func (s *postgresql) Get(id string) (*config.Pipeline, error)
func (s *postgresql) Add(pipe *config.Pipeline) error
func (s *postgresql) Update(pipe *config.Pipeline) error
func (s *postgresql) Delete(id string) error

func (s *postgresql) Close() error {
	return s.db.Close()
}

func storedToInternal(p storedPipeline) (*config.Pipeline, error) {
	cfg := config.Pipeline{
		Settings: config.PipeSettings{
			Id:          p.Id,
			Lines:       p.Lines,
			Run:         p.Run,
			Buffer:      p.Buffer,
			Consistency: p.Consistency,
			LogLevel:    p.LogLevel,
		},
	}

	if err := p.Vars.Unmarshal(&cfg.Vars); err != nil {
		return nil, fmt.Errorf("unmarshal vars: %w", err)
	}

	if err := p.Keykeepers.Unmarshal(&cfg.Keykeepers); err != nil {
		return nil, fmt.Errorf("unmarshal keykeepers: %w", err)
	}

	if err := p.Inputs.Unmarshal(&cfg.Inputs); err != nil {
		return nil, fmt.Errorf("unmarshal inputs: %w", err)
	}

	if err := p.Processors.Unmarshal(&cfg.Processors); err != nil {
		return nil, fmt.Errorf("unmarshal processors: %w", err)
	}

	if err := p.Outputs.Unmarshal(&cfg.Outputs); err != nil {
		return nil, fmt.Errorf("unmarshal outputs: %w", err)
	}

	return &cfg, nil
}
