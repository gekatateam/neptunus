package sql

import (
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v4"
	pgxstd "github.com/jackc/pgx/v4/stdlib"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
	ora "github.com/sijms/go-ora/v2"
)

func OpenDB(driverName, dsn, user, pass string, tlsConfig *tls.Config) (*sqlx.DB, error) {
	var db *sql.DB

	switch driverName {
	case "pgx", "postgres":
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", driverName, err)
		}

		if len(user) > 0 && len(pass) > 0 {
			cfg.User = user
			cfg.Password = pass
		}

		cfg.TLSConfig = tlsConfig
		db = pgxstd.OpenDB(*cfg)
	case "mysql":
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", driverName, err)
		}

		if len(user) > 0 && len(pass) > 0 {
			cfg.User = user
			cfg.Passwd = pass
		}

		cfg.TLS = tlsConfig
		connr, err := mysql.NewConnector(cfg)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", driverName, err)
		}

		db = sql.OpenDB(connr)
	case "clickhouse":
		cfg, err := clickhouse.ParseDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", driverName, err)
		}

		if len(user) > 0 && len(pass) > 0 {
			cfg.Auth.Username = user
			cfg.Auth.Password = pass
		}

		cfg.TLS = tlsConfig
		db = clickhouse.OpenDB(cfg)
	case "sqlserver":
		cfg, err := msdsn.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", driverName, err)
		}

		if len(user) > 0 && len(pass) > 0{
			cfg.User = user
			cfg.Password = pass
		}

		cfg.TLSConfig = tlsConfig
		db = sql.OpenDB(mssql.NewConnectorConfig(cfg))
	case "oracle", "ora", "goracle":
		u, err := url.Parse(dsn)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", driverName, err)
		}

		if len(user) > 0 && len(pass) > 0{
			u.User = url.UserPassword(user, pass)
		}

		driverName = "ora" // https://github.com/jmoiron/sqlx/blob/master/bind.go#L27
		oraConnector := ora.NewConnector(u.String()).(*ora.OracleConnector)
		oraConnector.WithTLSConfig(tlsConfig)
		db = sql.OpenDB(oraConnector)
	default:
		return nil, errors.New("unknown driver - " + driverName)
	}

	return sqlx.NewDb(db, driverName), nil
}

func BindNamed(query string, args any, querier sqlx.ExtContext) (string, []any, error) {
	q, a, err := sqlx.Named(query, args)
	if err != nil {
		return "", nil, fmt.Errorf("sqlx.Named: %w", err)
	}

	q, a, err = sqlx.In(q, a...)
	if err != nil {
		return "", nil, fmt.Errorf("sqlx.In: %w", err)
	}

	return querier.Rebind(q), a, nil
}

type QueryInfo struct {
	Query string `mapstructure:"query"`
	File  string `mapstructure:"file"`
}

func (q *QueryInfo) Init() error {
	if len(q.File) > 0 {
		rawQuery, err := os.ReadFile(q.File)
		if err != nil {
			return fmt.Errorf("file reading failed: %w", err)
		}
		q.Query = string(rawQuery)
	}
	return nil
}
