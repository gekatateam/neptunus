package deduplicate

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/elog"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	"github.com/gekatateam/neptunus/plugins/common/sharedstorage"
	"github.com/gekatateam/neptunus/plugins/common/tls"
)

var clientStorage = sharedstorage.New[redis.UniversalClient, uint64]()

type Deduplicate struct {
	*core.BaseProcessor `mapstructure:"-"`
	IdempotencyKey      string `mapstructure:"idempotency_key"`
	Redis               Redis  `mapstructure:"redis"`
	*retryer.Retryer    `mapstructure:",squash"`

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

	*tls.TLSClientConfig `mapstructure:",squash"`
}

func (p *Deduplicate) Init() error {
	if len(p.IdempotencyKey) == 0 {
		return errors.New("idempotency_key required")
	}

	if len(p.Redis.Servers) == 0 {
		return errors.New("at least one Redis server address required")
	}

	tlsConfig, err := p.Redis.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	if len(p.Redis.Keyspace) > 0 && !strings.HasSuffix(p.Redis.Keyspace, ":") {
		p.Redis.Keyspace = p.Redis.Keyspace + ":"
	}

	// also, pool options:
	// MinIdleConns    int
	// MaxActiveConns  int
	p.client = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:                 p.Redis.Servers,
		Username:              p.Redis.Username,
		Password:              p.Redis.Password,
		MaxRetries:            0,
		DialTimeout:           p.Redis.Timeout,
		ReadTimeout:           p.Redis.Timeout,
		WriteTimeout:          p.Redis.Timeout,
		ContextTimeoutEnabled: true,
		PoolSize:              p.Redis.ConnsMaxOpen,
		PoolTimeout:           p.Redis.Timeout,
		MaxIdleConns:          p.Redis.ConnsMaxIdle,
		ConnMaxIdleTime:       p.Redis.ConnsMaxIdleTime,
		ConnMaxLifetime:       p.Redis.ConnsMaxLifetime,
		TLSConfig:             tlsConfig,
	})

	p.client = clientStorage.CompareAndStore(p.id, p.client)

	if err := p.client.Ping(context.Background()).Err(); err != nil {
		defer p.client.Close()
		return err
	}

	return nil
}

func (p *Deduplicate) Close() error {
	if clientStorage.Leave(p.id) {
		return p.client.Close()
	}
	return nil
}

func (p *Deduplicate) SetId(id uint64) {
	p.id = id
}

func (p *Deduplicate) Run() {
	var now time.Time
	for e := range p.In {
		now = time.Now()
		e.SetLabel("::duplicate", "false")

		key, ok := e.GetLabel(p.IdempotencyKey)
		if !ok {
			p.Log.Debug("event has no configured label, skipped",
				elog.EventGroup(e),
			)
			p.Out <- e
			p.Observe(metrics.EventAccepted, time.Since(now))
			continue
		}

		var unique bool
		err := p.Retryer.Do("set key", p.Log, func() error {
			boolCmd := p.client.SetNX(context.Background(), p.Redis.Keyspace+key, time.Now().String(), p.Redis.TTL)
			if err := boolCmd.Err(); err != nil {
				return err
			}

			// false means than key not set because key already exists
			unique = boolCmd.Val()
			return nil
		})

		if err != nil {
			p.Log.Error("redis cmd exec failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		if !unique {
			p.Log.Debug("duplicate event found",
				elog.EventGroup(e),
			)
			e.SetLabel("::duplicate", "true")
		}
		p.Out <- e
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
				Timeout:          30 * time.Second,
				TTL:              1 * time.Hour,
				TLSClientConfig:  &tls.TLSClientConfig{},
			},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
