package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gekatateam/neptunus/config"
)

const defaultTableName = "NEPTUNUS_PIPELINES"

type postgresStorage struct {
	table string
	pool  *pgxpool.Pool
}

func Postgres(dsn, table string) (*postgresStorage, error) {
	if len(table) == 0 {
		table = defaultTableName
	}

	if len(dsn) == 0 {
		return nil, errors.New("DSN required")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	return &postgresStorage{
		pool:  pool,
		table: table,
	}, nil
}

func (s *postgresStorage) List() ([]*config.Pipeline, error)
func (s *postgresStorage) Get(id string) (*config.Pipeline, error)
func (s *postgresStorage) Add(pipe *config.Pipeline) error
func (s *postgresStorage) Update(pipe *config.Pipeline) error
func (s *postgresStorage) Delete(id string) error
func (s *postgresStorage) Close() error
