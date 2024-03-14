package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
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
	Timeout          time.Duration `mapstructure:"timeout"`
	Interval         time.Duration `mapstructure:"interval"`

	Transactional  bool   `mapstructure:"transactional"`
	IsolationLevel string `mapstructure:"isolation_level"`
	ReadOnly       bool   `mapstructure:"read_only"`

	OnInit       QueryInfo         `mapstructure:"on_init"`
	OnPoll       QueryInfo         `mapstructure:"on_poll"`
	OnDelivery   QueryInfo         `mapstructure:"on_delivery"`
	KeepValues   KeepValues        `mapstructure:"keep_values"`
	LabelColumns map[string]string `mapstructure:"labelcolumns"`

	keepIndex  map[string]int
	keepValues map[string]any

	txLevel sql.IsolationLevel

	db *sqlx.DB
}

type QueryInfo struct {
	Query string `mapstructure:"query"`
	File  string `mapstructure:"file"`
}

func (q *QueryInfo) Init() error {
	if len(q.File) > 0 {
		rawQuery, err := os.ReadFile(q.File)
		if err != nil {
			return fmt.Errorf("file reading failed: %w", err)
		}
		q.Query = string(rawQuery)
	}
	return nil
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

	if i.Transactional {
		switch i.IsolationLevel {
		case "Default":
			i.txLevel = sql.LevelDefault
		case "ReadUncommitted":
			i.txLevel = sql.LevelReadUncommitted
		case "ReadCommitted":
			i.txLevel = sql.LevelReadCommitted
		case "WriteCommitted":
			i.txLevel = sql.LevelWriteCommitted
		case "RepeatableRead":
			i.txLevel = sql.LevelRepeatableRead
		case "Snapshot":
			i.txLevel = sql.LevelSnapshot
		case "Serializable":
			i.txLevel = sql.LevelSerializable
		case "Linearizable":
			i.txLevel = sql.LevelLinearizable
		default:
			return fmt.Errorf("unknown tx isolation level: %v", i.IsolationLevel)
		}
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

	if err := i.OnPoll.Init(); err != nil {
		return fmt.Errorf("onPoll: %w", err)
	}

	if err := i.OnDelivery.Init(); err != nil {
		return fmt.Errorf("onDelivery: %w", err)
	}

	if err := i.OnInit.Init(); err != nil {
		return fmt.Errorf("onInit: %w", err)
	}

	db, err := sqlx.Connect(i.Driver, i.Dsn)
	if err != nil {
		return err
	}

	db.DB.SetConnMaxIdleTime(i.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(i.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(i.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(i.ConnsMaxOpen)

	if len(i.OnInit.Query) > 0 {
		if err := i.performOnInitQuery(); err != nil {
			return fmt.Errorf("onInit query failed: %w", err)
		}
	}

	return nil
}

func (i *Sql) Close() error {




	return i.db.Close()
}

func (i *Sql) Run() {
	for {
		now := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), i.Timeout)

		var querier sqlx.ExtContext = i.db
		if i.Transactional {
			tx, err := i.db.BeginTxx(ctx, &sql.TxOptions{Isolation: i.txLevel, ReadOnly: i.ReadOnly})
			if err != nil {
				i.Log.Error("tx begin error", 
					"error", err,
				)
				i.Observe(metrics.EventFailed, time.Since(now))

				cancel()
				continue
			}

			querier = tx
		}

	}
}

func (i *Sql) performOnInitQuery() error {
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
			if !ok { // it is okay if query does not returns some values
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

func (i *Sql) performOnPollQuery(ctx context.Context, querier sqlx.ExtContext) error {
	rows, err := sqlx.NamedQueryContext(ctx, querier, i.OnDelivery.Query, i.keepValues)
	if err != nil {
		return err
	}
	defer rows.Close()

	keepValues := make(map[string]any)
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
				keepValues[k] = col
			case v == keepLast && !hasRows:
				keepValues[k] = col
			case v == keepAll:
				val, ok := keepValues[k]
				if !ok {
					keepValues[k] = []any{col}
				} else {
					keepValues[k] = append(val.([]any), col)
				}
			}
		}
		first = false

		e := core.NewEventWithData("sql."+i.Driver, fetchedRow)
		for k, v := range i.LabelColumns {
			if valRaw, ok := fetchedRow[v]; ok {
				if val, ok := valRaw.(string); ok {
					e.SetLabel(k, val)
				}
			}
		}

		i.Out <- e
	}

	for k, v := range keepValues {
		i.keepValues[k] = v
	}

	return nil
}

// NOT LIKE THIS
func (i *Sql) performOnDeliveryQuery(ctx context.Context, tx *sqlx.Tx) error {
	var err error

	if tx != nil {
		_, err = sqlx.NamedQueryContext(ctx, tx, i.OnDelivery.Query, i.keepValues)
	} else {
		_, err = sqlx.NamedQueryContext(ctx, i.db, i.OnDelivery.Query, i.keepValues)
	}

	return err
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
