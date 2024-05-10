package sql

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	csql "github.com/gekatateam/neptunus/plugins/common/sql"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Sql struct {
	*core.BaseOutput `mapstructure:"-"`
	Dsn              string        `mapstructure:"dsn"`
	Driver           string        `mapstructure:"driver"`
	ConnsMaxIdleTime time.Duration `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime time.Duration `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen     int           `mapstructure:"conns_max_open"`
	ConnsMaxIdle     int           `mapstructure:"conns_max_idle"`
	Timeout          time.Duration `mapstructure:"timeout"`

	OnInit csql.QueryInfo `mapstructure:"on_init"`
	OnPush csql.QueryInfo `mapstructure:"on_push"`

	*tls.TLSClientConfig          `mapstructure:",squash"`
	*batcher.Batcher[*core.Event] `mapstructure:",squash"`

	db *sqlx.DB
}

func (o *Sql) Init() error {
	if len(o.Dsn) == 0 {
		return errors.New("dsn required")
	}

	if len(o.Driver) == 0 {
		return errors.New("driver required")
	}

	if err := o.OnInit.Init(); err != nil {
		return fmt.Errorf("onInit: %w", err)
	}

	if err := o.OnPush.Init(); err != nil {
		return fmt.Errorf("onPush: %w", err)
	}

	tlsConfig, err := o.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	db, err := csql.OpenDB(o.Driver, o.Dsn, tlsConfig)
	if err != nil {
		return err
	}

	db.DB.SetConnMaxIdleTime(o.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(o.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(o.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(o.ConnsMaxOpen)

	if err := db.Ping(); err != nil {
		return err
	}
	o.db = db

	return nil
}

func (o *Sql) Run() {
	for e := range o.In {

	}
}

func (o *Sql) Close() error {
	return nil
}

func init() {
	plugins.AddOutput("sql", func() core.Output {
		return &Sql{
			ConnsMaxIdleTime: 10 * time.Minute,
			ConnsMaxLifetime: 10 * time.Minute,
			ConnsMaxOpen:     2,
			ConnsMaxIdle:     1,
			Timeout:          30 * time.Second,

			TLSClientConfig: &tls.TLSClientConfig{},
			Batcher: &batcher.Batcher[*core.Event]{
				Buffer:   100,
				Interval: 5 * time.Second,
			},
		}
	})
}
