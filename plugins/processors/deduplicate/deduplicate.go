package deduplicate

import (
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
)

// redis client SetNX() method returns RedisBool
// where false means than key already exists
//
// redis metrics needs here
type Deduplicate struct {
	*core.BaseProcessor `mapstructure:"-"`
	IdempotencyKey      string `mapstructure:"idempotency_key"`
	Redis               Redis  `mapstructure:"redis"`
	Retryer *retryer.Retryer `mapstructure:",squash"`

	id     uint64
	client redis.UniversalClient
}

type Redis struct {
	Servers          []string      `mapstructure:"servers"`
	Username         string        `mapstructure:"username"`
	Password         string        `mapstructure:"password"`
	Keyspace         string        `mapstructure:"keyspace"`
	TTL              time.Duration `mapstructure:"ttl"`
	Timeout          time.Duration `mapstructure:"timeout"`
	ConnsMaxIdleTime time.Duration `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime time.Duration `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen     int           `mapstructure:"conns_max_open"`
	ConnsMaxIdle     int           `mapstructure:"conns_max_idle"`
}

func (p *Deduplicate) Init() error {
	return nil
}

func (p *Deduplicate) Close() error {
	return p.client.Close()
}

func (p *Deduplicate) SetId(id uint64) {
	p.id = id
}

func (p *Deduplicate) Run() {
	for e := range p.In {
		now := time.Now()
		e.Done()
		p.Log.Debug("event dropped",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("deduplicate", func() core.Processor {
		return &Deduplicate{
			Redis: Redis{
				Keyspace:         "neptunus:deduplicate",
				ConnsMaxIdleTime: 10 * time.Minute,
				ConnsMaxLifetime: 10 * time.Minute,
				ConnsMaxOpen:     2,
				ConnsMaxIdle:     1,
			},
		}
	})
}
