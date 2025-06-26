package redisproc

import (
	"context"
	"errors"
	"fmt"
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

var clientStorage = sharedstorage.New[redis.UniversalClient]()

type Redis struct {
	*core.BaseProcessor `mapstructure:"-"`
	Servers             []string      `mapstructure:"servers"`
	Username            string        `mapstructure:"username"`
	Password            string        `mapstructure:"password"`
	Timeout             time.Duration `mapstructure:"timeout"`
	ConnsMaxIdleTime    time.Duration `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime    time.Duration `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen        int           `mapstructure:"conns_max_open"`
	ConnsMaxIdle        int           `mapstructure:"conns_max_idle"`

	Command  []any  `mapstructure:"command"`
	ResultTo string `mapstructure:"result_to"`

	*tls.TLSClientConfig `mapstructure:",squash"`
	*retryer.Retryer     `mapstructure:",squash"`

	id     uint64
	client redis.UniversalClient
	args   []pluginArg
}

func (p *Redis) Init() error {
	if len(p.Command) == 0 {
		return errors.New("command required")
	}

	if len(p.Servers) == 0 {
		return errors.New("at least one Redis server address required")
	}

	p.args = make([]pluginArg, 0, len(p.Command))
	for _, v := range p.Command {
		p.args = append(p.args, newPluginArg(v))
	}

	tlsConfig, err := p.TLSClientConfig.Config()
	if err != nil {
		return err
	}

	// also, pool options:
	// MinIdleConns    int
	// MaxActiveConns  int
	p.client = clientStorage.CompareAndStore(p.id, redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:                 p.Servers,
		Username:              p.Username,
		Password:              p.Password,
		MaxRetries:            0,
		DialTimeout:           p.Timeout,
		ReadTimeout:           p.Timeout,
		WriteTimeout:          p.Timeout,
		ContextTimeoutEnabled: true,
		PoolSize:              p.ConnsMaxOpen,
		PoolTimeout:           p.Timeout,
		MaxIdleConns:          p.ConnsMaxIdle,
		ConnMaxIdleTime:       p.ConnsMaxIdleTime,
		ConnMaxLifetime:       p.ConnsMaxLifetime,
		TLSConfig:             tlsConfig,
	}))

	if err := p.client.Ping(context.Background()).Err(); err != nil {
		defer p.client.Close()
		return err
	}

	return nil
}

func (p *Redis) Close() error {
	if clientStorage.Leave(p.id) {
		return p.client.Close()
	}
	return nil
}

func (p *Redis) SetId(id uint64) {
	p.id = id
}

func (p *Redis) Run() {
	for e := range p.In {
		now := time.Now()

		args, err := commandArgs(p.args, e)
		if err != nil {
			p.Log.Error("command args preparation failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		var result any
		err = p.Retryer.Do("exec command", p.Log, func() error {
			r, err := p.client.Do(context.Background(), args...).Result()
			result = r

			if err != nil && !errors.Is(err, redis.Nil) {
				return err
			}

			return nil
		})

		if err != nil {
			p.Log.Error("command execution failed",
				"error", err,
				elog.EventGroup(e),
			)
			e.StackError(err)
			p.Out <- e
			p.Observe(metrics.EventFailed, time.Since(now))
			continue
		}

		if len(p.ResultTo) > 0 {
			field, err := unpackResult(result)
			if err != nil {
				p.Log.Error("command result unpack failed",
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}

			if err := e.SetField(p.ResultTo, field); err != nil {
				p.Log.Error(fmt.Sprintf("set field %v failed", p.ResultTo),
					"error", err,
					elog.EventGroup(e),
				)
				e.StackError(err)
				p.Out <- e
				p.Observe(metrics.EventFailed, time.Since(now))
				continue
			}
		}

		p.Log.Debug("command execution completed",
			elog.EventGroup(e),
		)
		p.Out <- e
		p.Observe(metrics.EventAccepted, time.Since(now))
	}
}

func init() {
	plugins.AddProcessor("redis", func() core.Processor {
		return &Redis{
			ConnsMaxIdleTime: 10 * time.Minute,
			ConnsMaxLifetime: 10 * time.Minute,
			ConnsMaxOpen:     2,
			ConnsMaxIdle:     1,
			Timeout:          30 * time.Second,
			TLSClientConfig:  &tls.TLSClientConfig{},
			Retryer: &retryer.Retryer{
				RetryAttempts: 0,
				RetryAfter:    5 * time.Second,
			},
		}
	})
}
