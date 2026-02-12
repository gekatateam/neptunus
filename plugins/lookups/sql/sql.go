package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/convert"
	csql "github.com/gekatateam/neptunus/plugins/common/sql"
	"github.com/gekatateam/neptunus/plugins/common/tls"
	"github.com/gekatateam/neptunus/plugins/core/lookup"
)

type Sql struct {
	*core.BaseLookup `mapstructure:"-"`
	EnableMetrics    bool           `mapstructure:"enable_metrics"`
	Driver           string         `mapstructure:"driver"`
	Dsn              string         `mapstructure:"dsn"`
	Username         string         `mapstructure:"username"`
	Password         string         `mapstructure:"password"`
	ConnsMaxIdleTime time.Duration  `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime time.Duration  `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen     int            `mapstructure:"conns_max_open"`
	ConnsMaxIdle     int            `mapstructure:"conns_max_idle"`
	Timeout          time.Duration  `mapstructure:"timeout"`
	OnUpdate         csql.QueryInfo `mapstructure:"on_update"`

	Mode      string `mapstructure:"mode"`
	KeyColumn string `mapstructure:"key_column"`

	*tls.TLSClientConfig `mapstructure:",squash"`

	db *sqlx.DB
}

func (l *Sql) Init() error {
	if len(l.Dsn) == 0 {
		return errors.New("dsn required")
	}

	if len(l.Driver) == 0 {
		return errors.New("driver required")
	}

	if len(l.OnUpdate.File) == 0 && len(l.OnUpdate.Query) == 0 {
		return errors.New("onUpdate.query or onUpdate.file required")
	}

	if err := l.OnUpdate.Init(); err != nil {
		return fmt.Errorf("onUpdate: %w", err)
	}

	switch l.Mode {
	case "horizontal":
	case "vertical":
		if len(l.KeyColumn) == 0 {
			return errors.New("key column required")
		}
	default:
		return fmt.Errorf("unknown mode: %v", l.Mode)
	}

	tlsConfig, err := l.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	db, err := csql.OpenDB(l.Driver, l.Dsn, l.Username, l.Password, tlsConfig)
	if err != nil {
		return err
	}

	db.DB.SetConnMaxIdleTime(l.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(l.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(l.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(l.ConnsMaxOpen)

	if err := db.Ping(); err != nil {
		defer db.Close()
		return err
	}
	l.db = db

	return nil
}

func (l *Sql) Close() error {
	return l.db.Close()
}

func (l *Sql) Update() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.Timeout)
	defer cancel()

	rows, err := l.db.QueryContext(ctx, l.OnUpdate.Query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	switch l.Mode {
	case "horizotal":
		return l.scanHorizontal(rows)
	case "vertical":
		return l.scanVertical(rows)
	default:
		panic(fmt.Errorf("totally unexpected - how you start it with this mode? %v", l.Mode))
	}
}

func (l *Sql) scanHorizontal(rows *sql.Rows) ([]map[string]any, error) {
	result := []map[string]any{}

	for rows.Next() {
		fetchedRow := make(map[string]any)
		if err := sqlx.MapScan(rows, fetchedRow); err != nil {
			return nil, fmt.Errorf("sqlx.MapScan: %w", err)
		}

		result = append(result, fetchedRow)
	}

	return result, nil
}

func (l *Sql) scanVertical(rows *sql.Rows) (map[string]map[string]any, error) {
	result := map[string]map[string]any{}

	for rows.Next() {
		fetchedRow := make(map[string]any)
		if err := sqlx.MapScan(rows, fetchedRow); err != nil {
			return nil, fmt.Errorf("sqlx.MapScan: %w", err)
		}

		keyRaw, ok := fetchedRow[l.KeyColumn]
		if !ok {
			return nil, fmt.Errorf("fetched row does not contains configured key column %v", l.KeyColumn)
		}

		key, err := convert.AnyToString(keyRaw)
		if err != nil {
			return nil, fmt.Errorf("cannot convert key to string: %w", err)
		}

		delete(fetchedRow, l.KeyColumn)
		result[key] = fetchedRow
	}

	return result, nil
}

func init() {
	plugins.AddLookup("sql", func() core.Lookup {
		return &lookup.Lookup{
			LazyLookup: &Sql{
				ConnsMaxIdleTime: 10 * time.Minute,
				ConnsMaxLifetime: 10 * time.Minute,
				ConnsMaxOpen:     2,
				ConnsMaxIdle:     1,
				Timeout:          30 * time.Second,
				Mode:             "vertical",
				TLSClientConfig:  &tls.TLSClientConfig{},
			},
			Interval: 30 * time.Second,
		}
	})
}
