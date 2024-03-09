package sql

import (
	"time"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

type Sql struct {
	*core.BaseInput  `mapstructure:"-"`
	Dsn              string        `mapstructure:"dsn"`
	Driver           string        `mapstructure:"driver"`
	ConnsMaxIdleTime time.Duration `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime time.Duration `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen     int           `mapstructure:"conns_max_open"`
	ConnsMaxIdle     int           `mapstructure:"conns_max_idle"`
	Transactional    bool          `mapstructure:"transactional"`

	OnInit struct {
		Query      string         `mapstructure:"query"`
		File       string         `mapstructure:"file"`
		KeepValues map[string]any `mapstructure:"keep_values"`
	} `mapstructure:"on_init"`

	OnPoll struct {
		Query      string         `mapstructure:"query"`
		File       string         `mapstructure:"file"`
		KeepValues map[string]any `mapstructure:"keep_values"`
	} `mapstructure:"on_poll"`

	OnDelivery struct {
		Query string `mapstructure:"query"`
		File  string `mapstructure:"file"`
	} `mapstructure:"on_delivery"`
}

func (i *Sql) Init() error
func (i *Sql) Close() error
func (i *Sql) Run()

func init() {
	plugins.AddInput("sql", func() core.Input {
		return &Sql{}
	})
}
