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

	OnInit struct {
		Query      string   `mapstructure:"query"`
		File       string   `mapstructure:"file"`
		KeepValues []string `mapstructure:"keep_values"`
		keepedValues map[string]any
	} `mapstructure:"on_init"`

	OnPoll struct {
		Query      string         `mapstructure:"query"`
		File       string         `mapstructure:"file"`
		KeepValues []string `mapstructure:"keep_values"`
		keepedValues map[string]any
	} `mapstructure:"on_poll"`

	OnDelivery struct {
		Query string `mapstructure:"query"`
		File  string `mapstructure:"file"`
	} `mapstructure:"on_delivery"`

	db *sqlx.DB
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

	i.OnInit.keepedValues = make(map[string]any, len(i.OnInit.KeepValues))
	i.OnPoll.keepedValues = make(map[string]any, len(i.OnPoll.KeepValues))

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
func (i *Sql) Run()

func (i *Sql) performInitQuery() error {
	ctx, cancel := context.WithTimeout(context.Background(), i.Timeout)
	defer cancel()

	rows, err := i.db.QueryContext(ctx, i.OnInit.Query)
	if err != nil {
		return err
	}
	defer rows.Close()

	fetchedRows := make([]map[string]any, 0)
	for rows.Next() {
		fetchedRow := make(map[string]any)
		if err := sqlx.MapScan(rows, fetchedRow); err != nil {
			return err
		}
		fetchedRows = append(fetchedRows, fetchedRow)
	}

	if len(fetchedRows) == 0 {
		return errors.New("init request returns no data")
	}

	i.Log.Debug(fmt.Sprintf("init request fetched %v rows", len(fetchedRows)))

	if len(fetchedRows) == 1 {
		for _, v := range i.OnInit.KeepValues {
			val, ok := fetchedRows[0][v]
			if !ok {
				return fmt.Errorf("init request not returns column %v", v)
			}
	
			i.OnInit.keepedValues[v] = val
		}
		return nil
	}

	rawKeepedRows := make(map[string][]any)
	for _, row := range fetchedRows {
		for k, v := range row {
			rawKeepedRows[k] = append(rawKeepedRows[k], v)
		}
	}

	for _, v := range i.OnInit.KeepValues {
		val, ok := rawKeepedRows[v]
		if !ok {
			return fmt.Errorf("init request not returns column %v", v)
		}

		i.OnInit.keepedValues[v] = val
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
