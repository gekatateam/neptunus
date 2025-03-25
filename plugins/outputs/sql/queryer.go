package sql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/plugins/common/batcher"
	"github.com/gekatateam/neptunus/plugins/common/retryer"
	csql "github.com/gekatateam/neptunus/plugins/common/sql"
)

type querier struct {
	*core.BaseOutput
	columns map[string]string

	tableName string
	query     string
	timeout   time.Duration

	db *sqlx.DB
	*batcher.Batcher[*core.Event]
	*retryer.Retryer

	input chan *core.Event
}

func (q *querier) Run() {
	q.Log.Info(fmt.Sprintf("queryer for table %v spawned", q.tableName))

	q.Batcher.Run(q.input, func(buf []*core.Event) {
		if len(buf) == 0 {
			return
		}

		now := time.Now()

		rawArgs := make([]map[string]any, len(buf))
		for i, e := range buf {
			arg := map[string]any{}
			for column, field := range q.columns {
				// if field does not exists, nil returns
				value, _ := e.GetField(field)
				arg[column] = value
			}
			rawArgs[i] = arg
		}

		err := q.Retryer.Do("exec query", q.Log, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
			defer cancel()

			// after init tests, error normally does not occur here
			query, args, err := csql.BindNamed(q.query, rawArgs, q.db)
			if err != nil {
				return fmt.Errorf("query binding failed: %w", err)
			}

			q.Log.Debug(fmt.Sprintf("binded query: %v", query))

			_, err = q.db.ExecContext(ctx, query, args...)
			return err
		})

		timePerEvent := time.Duration(int64(time.Since(now)) / int64(len(buf)))
		for _, e := range buf {
			q.Done <- e
			if err != nil {
				q.Log.Error("event query execution failed",
					"error", err,
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				q.Observe(metrics.EventFailed, timePerEvent)
			} else {
				q.Log.Debug("event query executed successfully",
					slog.Group("event",
						"id", e.Id,
						"key", e.RoutingKey,
					),
				)
				q.Observe(metrics.EventAccepted, timePerEvent)
			}
		}
	})

	q.Log.Info(fmt.Sprintf("queryer for table %v closed", q.tableName))
}

func (q *querier) Push(e *core.Event) {
	q.input <- e
}

func (q *querier) Close() error {
	close(q.input)
	return nil
}
