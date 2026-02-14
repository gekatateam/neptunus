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

const (
	ModeHorizontal = "horizontal"
	ModeVertical   = "vertical"
)

type Sql struct {
	*core.BaseLookup `mapstructure:"-"`
	*csql.Connector  `mapstructure:",squash"`
	OnUpdate         csql.QueryInfo `mapstructure:"on_update"`
	Mode             string         `mapstructure:"mode"`
	KeyColumn        string         `mapstructure:"key_column"`

	db *sqlx.DB
}

func (l *Sql) Init() error {
	if err := l.OnUpdate.Init(true); err != nil {
		return fmt.Errorf("onUpdate: %w", err)
	}

	switch l.Mode {
	case ModeHorizontal:
	case ModeVertical:
		if len(l.KeyColumn) == 0 {
			return errors.New("key column required")
		}
	default:
		return fmt.Errorf("unknown mode: %v", l.Mode)
	}

	db, err := l.Connector.Init()
	if err != nil {
		return err
	}

	l.db = db
	return nil
}

func (l *Sql) Close() error {
	return l.db.Close()
}

func (l *Sql) Update() (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.QueryTimeout)
	defer cancel()

	rows, err := l.db.QueryContext(ctx, l.OnUpdate.Query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	switch l.Mode {
	case ModeHorizontal:
		return l.scanHorizontal(rows)
	case ModeVertical:
		return l.scanVertical(rows)
	default:
		panic(fmt.Errorf("totally unexpected - how you start it with this mode? %v", l.Mode))
	}
}

func (l *Sql) scanHorizontal(rows *sql.Rows) ([]any, error) {
	result := []any{}

	for rows.Next() {
		fetchedRow := make(map[string]any)
		if err := sqlx.MapScan(rows, fetchedRow); err != nil {
			return nil, fmt.Errorf("sqlx.MapScan: %w", err)
		}

		result = append(result, fetchedRow)
	}

	return result, nil
}

func (l *Sql) scanVertical(rows *sql.Rows) (map[string]any, error) {
	result := map[string]any{}

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
				Connector: &csql.Connector{
					ConnsMaxIdleTime: 10 * time.Minute,
					ConnsMaxLifetime: 10 * time.Minute,
					ConnsMaxOpen:     2,
					ConnsMaxIdle:     1,
					QueryTimeout:     30 * time.Second,
					TLSClientConfig:  &tls.TLSClientConfig{},
				},
				Mode: ModeVertical,
			},
			Interval: 30 * time.Second,
		}
	})
}
