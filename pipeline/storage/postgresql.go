package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

const (
	timeout   = time.Second * 30
	connlimit = 1

	migrate = `
CREATE TABLE IF NOT EXISTS pipelines (
	created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	, updated_at  TIMESTAMP WITH TIME ZONE
	, deleted_at  TIMESTAMP WITH TIME ZONE
	, id          TEXT NOT NULL
	, run         BOOLEAN NOT NULL
	, lines       INTEGER NOT NULL
	, buffer      INTEGER NOT NULL
	, consistency TEXT NOT NULL
	, log_level   TEXT NOT NULL
	, vars        JSON NOT NULL
	, keykeepers  JSON NOT NULL
	, inputs      JSON NOT NULL
	, processors  JSON NOT NULL
	, outputs     JSON NOT NULL
	, CONSTRAINT unique_ids_only UNIQUE (id)
	, CONSTRAINT not_empty_id CHECK (id <> '')
);`

	add = `
INSERT INTO pipelines 
(id, lines, run, buffer, consistency, log_level, vars, keykeepers, inputs, processors, outputs)
VALUES
(:id, :lines, :run, :buffer, :consistency, :log_level, :vars, :keykeepers, :inputs, :processors, :outputs);`

	check = `
SELECT id, deleted_at FROM pipelines
WHERE id = $1;`

	remove = `
DELETE FROM pipelines
WHERE id = $1;`

	delete = `
UPDATE pipelines 
SET
	deleted_at = NOW()
WHERE id = $1;`

	update = `
UPDATE pipelines 
SET
	updated_at = NOW()
	, lines       = :lines
	, run         = :run
	, buffer      = :buffer
	, consistency = :consistency
	, log_level   = :log_level
	, vars        = :vars
	, keykeepers  = :keykeepers
	, inputs      = :inputs
	, processors  = :processors
	, outputs     = :outputs
WHERE id = :id;`

	get = `
SELECT id, lines, run, buffer, consistency, log_level, vars, keykeepers, inputs, processors, outputs
FROM pipelines
WHERE id = $1 and deleted_at IS NULL;`

	list = `
SELECT id, lines, run, buffer, consistency, log_level, vars, keykeepers, inputs, processors, outputs
FROM pipelines
WHERE deleted_at IS NULL;`
)

var (
	emptyVars = types.JSONText("{}")
	emptyList = types.JSONText("[]")
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
	if err := s.db.SelectContext(ctx, &storedPipes, list); err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	pipes := make([]*config.Pipeline, 0, len(storedPipes))
	for _, v := range storedPipes {
		p, err := storedToConfig(v)
		if err != nil {
			return nil, &pipeline.ValidationError{Err: err}
		}

		pipes = append(pipes, p)
	}

	return pipes, nil
}

func (s *postgresql) Get(id string) (*config.Pipeline, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	storedPipe := storedPipeline{}
	if err := s.db.GetContext(ctx, &storedPipe, get, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &pipeline.NotFoundError{Err: errors.New("no rows returned, there is no such pipeline")}
		}
		return nil, &pipeline.IOError{Err: err}
	}

	pipe, err := storedToConfig(storedPipe)
	if err != nil {
		return nil, &pipeline.ValidationError{Err: err}
	}

	return pipe, nil
}

func (s *postgresql) Add(pipe *config.Pipeline) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer tx.Rollback()
	storedPipe := storedPipeline{}

	if err := tx.GetContext(ctx, &storedPipe, check, pipe.Settings.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			goto CLEANED_UP
		}
		return &pipeline.IOError{Err: err}
	}

	if len(storedPipe.Id) > 0 && !storedPipe.DeletedAt.Valid {
		return &pipeline.ConflictError{Err: errors.New("pipeline already exists")}
	}

	if len(storedPipe.Id) > 0 && storedPipe.DeletedAt.Valid {
		if _, err := tx.ExecContext(ctx, remove, storedPipe.Id); err != nil {
			return &pipeline.IOError{Err: err}
		}
	}

CLEANED_UP:
	storedPipe, err = configToStored(pipe)
	if err != nil {
		return &pipeline.ValidationError{Err: err}
	}

	if _, err := tx.NamedExecContext(ctx, add, storedPipe); err != nil {
		return &pipeline.IOError{Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &pipeline.IOError{Err: err}
	}

	return nil
}

func (s *postgresql) Update(pipe *config.Pipeline) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer tx.Rollback()
	storedPipe := storedPipeline{}

	if err := tx.GetContext(ctx, &storedPipe, check, pipe.Settings.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &pipeline.NotFoundError{Err: errors.New("no rows returned, nothing to delete")}
		}
		return &pipeline.IOError{Err: err}
	}

	if storedPipe.DeletedAt.Valid {
		return &pipeline.NotFoundError{Err: fmt.Errorf("pipeline already deleted at %v", storedPipe.DeletedAt.Time)}
	}

	storedPipe, err = configToStored(pipe)
	if err != nil {
		return &pipeline.ValidationError{Err: err}
	}

	if _, err := tx.NamedExecContext(ctx, update, storedPipe); err != nil {
		return &pipeline.IOError{Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &pipeline.IOError{Err: err}
	}

	return nil
}

func (s *postgresql) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer tx.Rollback()
	storedPipe := storedPipeline{}

	if err := tx.GetContext(ctx, &storedPipe, check, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &pipeline.NotFoundError{Err: errors.New("no rows returned, nothing to delete")}
		}
		return &pipeline.IOError{Err: err}
	}

	if storedPipe.DeletedAt.Valid {
		return &pipeline.NotFoundError{Err: fmt.Errorf("pipeline already deleted at %v", storedPipe.DeletedAt.Time)}
	}

	if _, err := tx.ExecContext(ctx, delete, id); err != nil {
		return &pipeline.IOError{Err: err}
	}

	if err := tx.Commit(); err != nil {
		return &pipeline.IOError{Err: err}
	}

	return nil
}

func (s *postgresql) Close() error {
	return s.db.Close()
}

func storedToConfig(p storedPipeline) (*config.Pipeline, error) {
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

	return config.SetPipelineDefaults(&cfg), nil
}

func configToStored(c *config.Pipeline) (storedPipeline, error) {
	pipe := storedPipeline{
		Id:          c.Settings.Id,
		Run:         c.Settings.Run,
		Lines:       c.Settings.Lines,
		Buffer:      c.Settings.Buffer,
		Consistency: c.Settings.Consistency,
		LogLevel:    c.Settings.LogLevel,
		Vars:        types.JSONText{},
		Keykeepers:  types.JSONText{},
		Inputs:      types.JSONText{},
		Processors:  types.JSONText{},
		Outputs:     types.JSONText{},
	}

	if len(c.Vars) == 0 {
		pipe.Vars = emptyVars
	} else {
		vars, err := json.Marshal(c.Vars)
		if err != nil {
			return storedPipeline{}, fmt.Errorf("marshal vars: %w", err)
		}
		pipe.Vars = vars
	}

	if len(c.Keykeepers) == 0 {
		pipe.Keykeepers = emptyList
	} else {
		keykeepers, err := json.Marshal(c.Keykeepers)
		if err != nil {
			return storedPipeline{}, fmt.Errorf("marshal keykeepers: %w", err)
		}
		pipe.Keykeepers = keykeepers
	}

	if len(c.Inputs) == 0 {
		pipe.Inputs = emptyList
	} else {
		inputs, err := json.Marshal(c.Inputs)
		if err != nil {
			return storedPipeline{}, fmt.Errorf("marshal inputs: %w", err)
		}
		pipe.Inputs = inputs
	}

	if len(c.Processors) == 0 {
		pipe.Processors = emptyList
	} else {
		processors, err := json.Marshal(c.Processors)
		if err != nil {
			return storedPipeline{}, fmt.Errorf("marshal processors: %w", err)
		}
		pipe.Processors = processors
	}

	if len(c.Outputs) == 0 {
		pipe.Outputs = emptyList
	} else {
		outputs, err := json.Marshal(c.Outputs)
		if err != nil {
			return storedPipeline{}, fmt.Errorf("marshal outputs: %w", err)
		}
		pipe.Outputs = outputs
	}

	return pipe, nil
}
