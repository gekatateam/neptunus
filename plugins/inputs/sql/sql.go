package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins"
	"github.com/gekatateam/neptunus/plugins/common/ider"
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
	WaitForDelivery  bool          `mapstructure:"wait_for_delivery"`

	Transactional  bool   `mapstructure:"transactional"`
	IsolationLevel string `mapstructure:"isolation_level"`
	ReadOnly       bool   `mapstructure:"read_only"`

	OnInit       QueryInfo         `mapstructure:"on_init"`
	OnPoll       QueryInfo         `mapstructure:"on_poll"`
	OnDone       QueryInfo         `mapstructure:"on_done"`
	KeepValues   KeepValues        `mapstructure:"keep_values"`
	LabelColumns map[string]string `mapstructure:"labelcolumns"`

	Ider *ider.Ider `mapstructure:",squash"`

	keepIndex  map[string]int
	keepValues map[string]any

	stopCh chan struct{}
	doneCh chan struct{}

	txLevel sql.IsolationLevel
	db      *sqlx.DB
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

	if err := i.Ider.Init(); err != nil {
		return err
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
	for _, v := range i.KeepValues.First {
		if _, ok := i.keepIndex[v]; ok {
			return fmt.Errorf("duplicate column in keepValues.first: %v", v)
		}
		i.keepIndex[v] = keepFirst
	}

	for _, v := range i.KeepValues.Last {
		if _, ok := i.keepIndex[v]; ok {
			return fmt.Errorf("duplicate column in keepValues.last: %v", v)
		}
		i.keepIndex[v] = keepLast
	}

	for _, v := range i.KeepValues.All {
		if _, ok := i.keepIndex[v]; ok {
			return fmt.Errorf("duplicate column in keepValues.all: %v", v)
		}
		i.keepIndex[v] = keepAll
	}

	if err := i.OnInit.Init(); err != nil {
		return fmt.Errorf("onInit: %w", err)
	}

	if err := i.OnPoll.Init(); err != nil {
		return fmt.Errorf("onPoll: %w", err)
	}

	if err := i.OnDone.Init(); err != nil {
		return fmt.Errorf("onDone: %w", err)
	}

	db, err := sqlx.Connect(i.Driver, i.Dsn)
	if err != nil {
		return err
	}
	i.db = db

	db.DB.SetConnMaxIdleTime(i.ConnsMaxIdleTime)
	db.DB.SetConnMaxLifetime(i.ConnsMaxLifetime)
	db.DB.SetMaxIdleConns(i.ConnsMaxIdle)
	db.DB.SetMaxOpenConns(i.ConnsMaxOpen)

	i.stopCh = make(chan struct{})
	i.doneCh = make(chan struct{})

	if len(i.OnInit.Query) > 0 {
		if err := i.init(); err != nil {
			return fmt.Errorf("onInit query failed: %w", err)
		}
	}

	return nil
}

func (i *Sql) Close() error {
	i.stopCh <- struct{}{}
	<- i.doneCh
	return i.db.Close()
}

func (i *Sql) Run() {
	if i.Interval > 0 {
		ticker := time.NewTicker(i.Interval)
		for {
			select {
			case <- ticker.C:
				i.poll()
			case <- i.stopCh:
				ticker.Stop()
				i.doneCh <- struct{}{}
				return
			}
		}
	} else {
		for {
			select {
			case <- i.stopCh:
				i.doneCh <- struct{}{}
				return
			default:
				i.poll()
			}
		}
	}
}

func (i *Sql) init() error {
	ctx, cancel := context.WithTimeout(context.Background(), i.Timeout)
	defer cancel()

	rows, err := i.db.QueryContext(ctx, i.OnInit.Query)
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
		i.keepColumns(fetchedRow, keepValues, first, !hasRows)
		first = false
	}

	i.keepValues = keepValues
	return nil
}

func (i *Sql) poll() {
	now := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), i.Timeout)
	defer cancel()

	var querier sqlx.ExtContext = i.db
	if i.Transactional {
		tx, err := i.db.BeginTxx(ctx, &sql.TxOptions{Isolation: i.txLevel, ReadOnly: i.ReadOnly})
		if err != nil {
			i.Log.Error("tx begin failed", 
				"error", err,
			)
			i.Observe(metrics.EventFailed, time.Since(now))
			return
		}
		defer tx.Rollback()
		querier = tx
	}

	query, args, err := i.bindNamed(i.OnPoll.Query, i.keepValues, querier)
	if err != nil {
		i.Log.Error("onPoll query binding failed", 
			"error", err,
		)
		i.Observe(metrics.EventFailed, time.Since(now))
		return
	}

	rows, err := querier.QueryContext(ctx, query, args...)
	if err != nil {
		i.Log.Error("onPoll query exec failed", 
			"error", err,
		)
		i.Observe(metrics.EventFailed, time.Since(now))
		return
	}
	defer rows.Close()

	batchWg := &sync.WaitGroup{}
	keepValues := make(map[string]any)
	first := true
	hasRows := rows.Next()
	for hasRows {
		fetchedRow := make(map[string]any)
		if err := sqlx.MapScan(rows, fetchedRow); err != nil {
			i.Log.Error("row scan failed", 
				"error", err,
			)
			i.Observe(metrics.EventFailed, time.Since(now))
			return
		}

		hasRows = rows.Next()
		i.keepColumns(fetchedRow, keepValues, first, !hasRows)
		first = false

		e := core.NewEventWithData("sql."+i.Driver, fetchedRow)
		for k, v := range i.LabelColumns {
			if valRaw, ok := fetchedRow[v]; ok {
				if val, ok := valRaw.(string); ok {
					e.SetLabel(k, val)
				}
			}
		}

		if i.WaitForDelivery {
			batchWg.Add(1)
			e.SetHook(batchWg.Done)
		}

		i.Ider.Apply(e)
		i.Out <- e
		i.Log.Debug("event accepted",
			slog.Group("event",
				"id", e.Id,
				"key", e.RoutingKey,
			),
		)
		i.Observe(metrics.EventAccepted, time.Since(now))
		now = time.Now()
	}

	if first { // if at least one row returns, flag will be set to false
		i.Log.Debug("onPoll query returns no rows")
		return
	}

	// copy keys from init/previous query for usage in done stage
	for k, v := range i.keepValues {
		if _, ok := keepValues[k]; !ok {
			keepValues[k] = v
		}
	}

	batchWg.Wait()

	if len(i.OnDone.Query) > 0 {
		query, args, err := i.bindNamed(i.OnDone.Query, keepValues, querier)
		if err != nil {
			i.Log.Error("onDone query binding failed", 
				"error", err,
			)
			return
		}

		_, err = querier.ExecContext(ctx, query, args...)
		if err != nil {
			i.Log.Error("onDone query exec failed", 
				"error", err,
			)
			return
		}
	}

	if tx, ok := querier.(*sqlx.Tx); ok {
		if err := tx.Commit(); err != nil {
			i.Log.Error("tx commit failed", 
				"error", err,
			)
			return
		}
	}

	// if all stages passed successfully
	// replace previously keeped values with actual data
	i.keepValues = keepValues
}

func (i *Sql) bindNamed(query string, args map[string]any, querier sqlx.ExtContext) (string, []any, error) {
	q, a, err := sqlx.Named(query, args)
	if err != nil {
		return "", nil, fmt.Errorf("sqlx.Named: %w", err)
	}

	q, a, err = sqlx.In(q, a...)
	if err != nil {
		return "", nil, fmt.Errorf("sqlx.In: %w", err)
	}

	return querier.Rebind(q), a, nil
}

func (i *Sql) keepColumns(from, to map [string]any, first, last bool) {
	for k, v := range i.keepIndex {
		col, ok := from[k]
		if !ok {
			continue
		}

		switch {
		case v == keepFirst && first:
			to[k] = col
		case v == keepLast && last:
			to[k] = col
		case v == keepAll:
			val, ok := to[k]
			if !ok {
				to[k] = []any{col}
			} else {
				to[k] = append(val.([]any), col)
			}
		}
	}
}

func init() {
	plugins.AddInput("sql", func() core.Input {
		return &Sql{
			ConnsMaxIdleTime: 10 * time.Minute,
			ConnsMaxLifetime: 10 * time.Minute,
			ConnsMaxOpen:     2,
			ConnsMaxIdle:     1,
			Transactional:    false,
			IsolationLevel:   "Default",
			Timeout:          30 * time.Second,
			Interval:         0,
			WaitForDelivery:  true,

			Ider: &ider.Ider{},
		}
	})
}
