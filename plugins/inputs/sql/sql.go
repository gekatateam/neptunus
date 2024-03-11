package sql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/plugins"
)

const (
	keepFirst = iota + 1
	keepLast
	keepAll
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
	Timeout          time.Duration `mapstructure:"timeout"`

	OnInit     QueryInfo  `mapstructure:"on_init"`
	OnPoll     QueryInfo  `mapstructure:"on_poll"`
	OnDelivery QueryInfo  `mapstructure:"on_delivery"`
	KeepValues KeepValues `mapstructure:"keep_values"`

	keepIndex  map[string]int
	keepValues map[string]any

	db *sqlx.DB
}

type QueryInfo struct {
	Query      string   `mapstructure:"query"`
	File       string   `mapstructure:"file"`
}

type KeepValues struct {
	Last  []string `mapstructure:"last"`
	First []string `mapstructure:"first"`
	All   []string `mapstructure:"all"`
}

func (i *Sql) Init() error {
	if len(i.Dsn) == 0 {
		return errors.New("dsn required")
	}

	if len(i.Driver) == 0 {
		return errors.New("driver required")
	}

	if len(i.OnPoll.File) == 0 && len(i.OnPoll.Query) == 0 {
		return errors.New("onPoll.query or onPoll.file requred")
	}

	i.keepValues = make(map[string]any)
	i.keepIndex = make(map[string]int)
	keepCheck := make(map[string]bool)
	for _, v := range i.KeepValues.First {
		if ok := keepCheck[v]; ok {
			return fmt.Errorf("duplicate column in keepValues.first: %v", v)
		}
		keepCheck[v] = true
		i.keepIndex[v] = keepFirst
	}

	for _, v := range i.KeepValues.Last {
		if ok := keepCheck[v]; ok {
			return fmt.Errorf("duplicate column in keepValues.last: %v", v)
		}
		keepCheck[v] = true
		i.keepIndex[v] = keepLast
	}

	for _, v := range i.KeepValues.All {
		if ok := keepCheck[v]; ok {
			return fmt.Errorf("duplicate column in keepValues.all: %v", v)
		}
		keepCheck[v] = true
		i.keepIndex[v] = keepAll
	}

	db, err := sqlx.Connect(i.Driver, i.Dsn)
	if err != nil {
		return err
	}

	db.DB.SetConnMaxIdleTime(i.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(i.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(i.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(i.ConnsMaxOpen)

	if len(i.OnPoll.File) > 0 {
		rawQuery, err := os.ReadFile(i.OnPoll.File)
		if err != nil {
			return fmt.Errorf("onPoll.file reading failed: %w", err)
		}

		i.OnPoll.Query = string(rawQuery)
	}

	if len(i.OnDelivery.File) > 0 {
		rawQuery, err := os.ReadFile(i.OnDelivery.File)
		if err != nil {
			return fmt.Errorf("onDelivery.file reading failed: %w", err)
		}

		i.OnDelivery.Query = string(rawQuery)
	}

	if len(i.OnInit.File) > 0 || len(i.OnInit.Query) > 0 {
		if len(i.OnInit.File) > 0 {
			rawQuery, err := os.ReadFile(i.OnInit.File)
			if err != nil {
				return fmt.Errorf("onInit.file reading failed: %w", err)
			}
	
			i.OnInit.Query = string(rawQuery)
		}

		if err := i.performInitQuery(); err != nil {
			return fmt.Errorf("onInit query failed: %w", err)
		}
	}

	return nil
}

func (i *Sql) Close() error {
	return i.db.Close()
}

func (i *Sql) Run() {
	
}

func (i *Sql) performInitQuery() error {
	ctx, cancel := context.WithTimeout(context.Background(), i.Timeout)
	defer cancel()

	rows, err := i.db.QueryContext(ctx, i.OnInit.Query)
	if err != nil {
		return err
	}
	defer rows.Close()

	first := true
	hasRows := rows.Next()
	for hasRows {
		fetchedRow := make(map[string]any)
		if err := sqlx.MapScan(rows, fetchedRow); err != nil {
			return err
		}
		hasRows = rows.Next()

		for k, v := range i.keepIndex {
			col, ok := fetchedRow[k]
			if !ok {
				continue
			}

			switch {
			case v == keepFirst && first:
				i.keepValues[k] = col
			case v == keepLast && !hasRows:
				i.keepValues[k] = col
			case v == keepAll:
				val, ok := i.keepValues[k]
				if !ok {
					i.keepValues[k] = []any{col}
				} else {
					i.keepValues[k] = append(val.([]any), col)
				}
			}
		}

		first = false
	}

	return nil
}

func init() {
	plugins.AddInput("sql", func() core.Input {
		return &Sql{
			ConnsMaxIdleTime: 10 * time.Minute,
			ConnsMaxLifetime: 10 * time.Minute,
			ConnsMaxOpen:     5,
			ConnsMaxIdle:     2,
			Transactional:    false,
			Timeout:          30 * time.Second,
		}
	})
}
