package sql

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/plugins/common/tls"
)

type Connector struct {
	Driver               string        `mapstructure:"driver"`
	Dsn                  string        `mapstructure:"dsn"`
	Username             string        `mapstructure:"username"`
	Password             string        `mapstructure:"password"`
	ConnsMaxIdleTime     time.Duration `mapstructure:"conns_max_idle_time"`
	ConnsMaxLifetime     time.Duration `mapstructure:"conns_max_life_time"`
	ConnsMaxOpen         int           `mapstructure:"conns_max_open"`
	ConnsMaxIdle         int           `mapstructure:"conns_max_idle"`
	QueryTimeout         time.Duration `mapstructure:"query_timeout"`
	*tls.TLSClientConfig `mapstructure:",squash"`
}

func (c *Connector) Init() (*sqlx.DB, error) {
	if len(c.Dsn) == 0 {
		return nil, errors.New("dsn required")
	}

	if len(c.Driver) == 0 {
		return nil, errors.New("driver required")
	}

	tlsConfig, err := c.TLSClientConfig.Config()
	if err != nil {
		return nil, err
	}

	db, err := OpenDB(c.Driver, c.Dsn, c.Username, c.Password, tlsConfig)
	if err != nil {
		return nil, err
	}

	if c.QueryTimeout <= 0 {
		return nil, errors.New("queryTimeout cannot be less or equal to zero")
	}

	db.DB.SetConnMaxIdleTime(c.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(c.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(c.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(c.ConnsMaxOpen)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
