package sql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	dbstats "github.com/gekatateam/neptunus/plugins/common/metrics"
	"github.com/gekatateam/neptunus/plugins/common/pool"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	csql "github.com/gekatateam/neptunus/plugins/common/sql"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

const tablePlaceholder = ":table_name"

type Sql struct {
	*core.BaseOutput              `mapstructure:"-"`
	*csql.Connector               `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`
	*retryer.Retryer              `mapstructure:",squash"`
	EnableMetrics                 bool              `mapstructure:"enable_metrics"`
	IdleTimeout                   time.Duration     `mapstructure:"idle_timeout"`
	TablePlaceholder              string            `mapstructure:"table_placeholder"`
	OnInit                        csql.QueryInfo    `mapstructure:"on_init"`
	OnPush                        csql.QueryInfo    `mapstructure:"on_push"`
	Columns                       map[string]string `mapstructure:"columns"`

	queryersPool *pool.Pool[*core.Event, string]
	db           *sqlx.DB
}

func (o *Sql) Init() error {
	if err := o.OnInit.Init(false); err != nil {
		return fmt.Errorf("onInit: %w", err)
	}

	if err := o.OnPush.Init(true); err != nil {
		return fmt.Errorf("onPush: %w", err)
	}

	if len(o.TablePlaceholder) == 0 {
		o.TablePlaceholder = tablePlaceholder
	}

	if o.IdleTimeout > 0 && o.IdleTimeout < time.Minute {
		o.IdleTimeout = time.Minute
	}

	if o.Batcher.Buffer <= 0 {
		o.Batcher.Buffer = 1
	}

	db, err := o.Connector.Init()
	if err != nil {
		return err
	}
	o.db = db

	testArgs := map[string]any{}
	for k, v := range o.Columns {
		testArgs[k] = v
	}

	if _, _, err := csql.BindNamed(
		strings.Replace(o.OnPush.Query, o.TablePlaceholder, "TEST_TABLE_NAME", 1),
		testArgs, o.db); err != nil {
		o.db.Close()
		return fmt.Errorf("query test binding failed: %w", err)
	}

	if len(o.OnInit.Query) > 0 {
		if err := o.init(); err != nil {
			o.db.Close()
			return fmt.Errorf("onInit query failed: %w", err)
		}
	}

	o.queryersPool = pool.New(o.newQueryer)

	return nil
}

func (o *Sql) Run() {
	clearTicker := time.NewTicker(time.Minute)
	if o.IdleTimeout == 0 {
		clearTicker.Stop()
	}

	if o.EnableMetrics {
		dbstats.RegisterDB(o.Pipeline, o.Alias, o.Driver, o.db)
		defer dbstats.UnregisterDB(o.Pipeline, o.Alias, o.Driver)
	}

MAIN_LOOP:
	for {
		select {
		case e, ok := <-o.In:
			if !ok {
				clearTicker.Stop()
				break MAIN_LOOP
			}
			o.queryersPool.Get(e.RoutingKey).Push(e)
		case <-clearTicker.C:
			for _, key := range o.queryersPool.Keys() {
				if time.Since(o.queryersPool.LastWrite(key)) > o.IdleTimeout {
					o.queryersPool.Remove(key)
				}
			}
		}
	}

	o.queryersPool.Close()
}

func (o *Sql) Close() error {
	return o.db.Close()
}

func (o *Sql) init() error {
	ctx, cancel := context.WithTimeout(context.Background(), o.QueryTimeout)
	defer cancel()

	_, err := o.db.ExecContext(ctx, o.OnInit.Query)
	return err
}

func (o *Sql) newQueryer(key string) pool.Runner[*core.Event] {
	return &querier{
		BaseOutput: o.BaseOutput,
		Batcher:    o.Batcher,
		Retryer:    o.Retryer,
		db:         o.db,
		query:      strings.Replace(o.OnPush.Query, o.TablePlaceholder, key, 1),
		columns:    o.Columns,
		tableName:  key,
		timeout:    o.QueryTimeout,
		input:      make(chan *core.Event),
	}
}

func init() {
	plugins.AddOutput("sql", func() core.Output {
		return &Sql{
			IdleTimeout:      5 * time.Minute,
			TablePlaceholder: tablePlaceholder,
			Connector: &csql.Connector{
				ConnsMaxIdleTime: 10 * time.Minute,
				ConnsMaxLifetime: 10 * time.Minute,
				ConnsMaxOpen:     2,
				ConnsMaxIdle:     1,
				QueryTimeout:     30 * time.Second,
				TLSClientConfig:  &tls.TLSClientConfig{},
			},
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
