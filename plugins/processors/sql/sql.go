package sql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	dbstats "github.com/gekatateam/neptunus/plugins/common/metrics"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	csql "github.com/gekatateam/neptunus/plugins/common/sql"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

const tablePlaceholder = ":table_name"

type Sql struct {
	*core.BaseProcessor `mapstructure:"-"`
	EnableMetrics       bool          `mapstructure:"enable_metrics"`
	Dsn                 string        `mapstructure:"dsn"`
	Driver              string        `mapstructure:"driver"`
	Shared              bool          `mapstructure:"shared"`
	ConnsMaxIdleTime    time.Duration `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime    time.Duration `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen        int           `mapstructure:"conns_max_open"`
	ConnsMaxIdle        int           `mapstructure:"conns_max_idle"`
	QueryTimeout        time.Duration `mapstructure:"query_timeout"`

	TablePlaceholder string            `mapstructure:"table_placeholder"`
	TableLabel       string            `mapstructure:"table_label"`
	OnEvent          csql.QueryInfo    `mapstructure:"on_event"`
	Columns          map[string]string `mapstructure:"columns"`
	Fields           map[string]string `mapstructure:"fields"`

	*tls.TLSClientConfig `mapstructure:",squash"`
	*retryer.Retryer     `mapstructure:",squash"`

	db *sqlx.DB
	id uint64
}

func (p *Sql) Init() error {
	if len(p.Dsn) == 0 {
		return errors.New("dsn required")
	}

	if len(p.Driver) == 0 {
		return errors.New("driver required")
	}

	if len(p.OnEvent.File) == 0 && len(p.OnEvent.Query) == 0 {
		return errors.New("onEvent.query or onEvent.file requred")
	}

	if err := p.OnEvent.Init(); err != nil {
		return fmt.Errorf("onEvent: %w", err)
	}

	if len(p.TablePlaceholder) == 0 {
		p.TablePlaceholder = tablePlaceholder
	}

	tlsConfig, err := p.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	db, err := csql.OpenDB(p.Driver, p.Dsn, tlsConfig)
	if err != nil {
		return err
	}

	db.DB.SetConnMaxIdleTime(p.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(p.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(p.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(p.ConnsMaxOpen)

	p.db = db
	if p.Shared {
		p.db = clientStorage.CompareAndStore(p.id, db)
	}

	if err := p.db.Ping(); err != nil {
		defer p.db.Close()
		return err
	}

	testArgs := map[string]any{}
	for k, v := range p.Columns {
		testArgs[k] = v
	}

	if _, _, err := csql.BindNamed(
		strings.Replace(p.OnEvent.Query, p.TablePlaceholder, "TEST_TABLE_NAME", 1),
		testArgs, p.db); err != nil {
		defer p.db.Close()
		return fmt.Errorf("query test binding failed: %w", err)
	}

	return nil
}

func (p *Sql) SetId(id uint64) {
	p.id = id
}

func (p *Sql) Run() {
	if p.EnableMetrics {
		dbstats.RegisterDB(p.Pipeline, p.Alias, p.Driver, p.db)
		defer dbstats.UnregisterDB(p.Pipeline, p.Alias, p.Driver)
	}

	for e := range p.In {
		now := time.Now()

		rawArgs := make(map[string]any, len(p.Columns))
		for column, field := range p.Columns {
			// if field does not exists, nil returns
			value, _ := e.GetField(field)
			rawArgs[column] = value
		}

		var tableName string
		if len(p.TableLabel) > 0 {
			label, ok := e.GetLabel(p.TableLabel)
			if !ok {
				p.Log.Error("query preparation failed",
					"error", fmt.Errorf("event does not contains %v label", p.TableLabel),
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				e.StackError(fmt.Errorf("event does not contains %v label", p.TableLabel))
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
			tableName = label
		}

		fetchedRows := make(map[string][]any)

		err := p.Retryer.Do("exec query", p.Log, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), p.QueryTimeout)
			defer cancel()

			// after init tests, error normally does not occur here
			query, args, err := csql.BindNamed(
				strings.Replace(p.OnEvent.Query, p.TablePlaceholder, tableName, 1),
				rawArgs, p.db)
			if err != nil {
				return fmt.Errorf("query binding failed: %w", err)
			}

			p.Log.Debug(fmt.Sprintf("binded query: %v", query))

			rows, err := p.db.QueryContext(ctx, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				fetchedRow := make(map[string]any)
				if err := sqlx.MapScan(rows, fetchedRow); err != nil {
					return err
				}

				for k, v := range fetchedRow {
					if r, ok := fetchedRows[k]; ok {
						fetchedRows[k] = append(r, v)
					} else {
						fetchedRows[k] = []any{v}
					}
				}
			}

			return nil
		})

		if err != nil {
			p.Log.Error("sql exec failed",
				"error", err,
				slog.Group("event",
					"id", e.Id,
					"key", e.RoutingKey,
				),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		for field, column := range p.Fields {
			if val, ok := fetchedRows[column]; ok {
				if err := e.SetField(field, any(val)); err != nil {
					p.Log.Warn("set field failed",
						"error", fmt.Errorf("set field failed: %w", err),
						slog.Group("event",
							"id", e.Id,
							"key", e.RoutingKey,
						),
					)
				}
			}
		}
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func (p *Sql) Close() error {
	return p.db.Close()
}

func init() {
	plugins.AddProcessor("sql", func() core.Processor {
		return &Sql{
			Shared:           true,
			ConnsMaxIdleTime: 10 * time.Minute,
			ConnsMaxLifetime: 10 * time.Minute,
			ConnsMaxOpen:     2,
			ConnsMaxIdle:     1,
			QueryTimeout:     10 * time.Second,
			TablePlaceholder: tablePlaceholder,

			TLSClientConfig: &tls.TLSClientConfig{},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
