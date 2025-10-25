package metrics

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/gekatateam/neptunus/metrics"
)

var (
	pluginDbConnectionsMax = func(d dbDescriptor) string {
		return fmt.Sprintf("plugin_db_connections_max{pipeline=%q,plugin_name=%q,driver=%q}", d.pipeline, d.pluginName, d.driver)
	}
	pluginDbConnectionsOpen = func(d dbDescriptor) string {
		return fmt.Sprintf("plugin_db_connections_open{pipeline=%q,plugin_name=%q,driver=%q}", d.pipeline, d.pluginName, d.driver)
	}
	pluginDbConnectionsInUse = func(d dbDescriptor) string {
		return fmt.Sprintf("plugin_db_connections_in_use{pipeline=%q,plugin_name=%q,driver=%q}", d.pipeline, d.pluginName, d.driver)
	}
	pluginDbConnectionsIdle = func(d dbDescriptor) string {
		return fmt.Sprintf("plugin_db_connections_idle{pipeline=%q,plugin_name=%q,driver=%q}", d.pipeline, d.pluginName, d.driver)
	}
	pluginDbConnectionsWaitedTotal = func(d dbDescriptor) string {
		return fmt.Sprintf("plugin_db_connections_waited_total{pipeline=%q,plugin_name=%q,driver=%q}", d.pipeline, d.pluginName, d.driver)
	}
	pluginDbConnectionsWaitedSecondsTotal = func(d dbDescriptor) string {
		return fmt.Sprintf("plugin_db_connections_waited_seconds_total{pipeline=%q,plugin_name=%q,driver=%q}", d.pipeline, d.pluginName, d.driver)
	}
)

var (
	dbMetricsRegister  = &sync.Once{}
	dbMetricsCollector = &dbCollector{
		dbs: make(map[dbDescriptor]*sqlx.DB),
		mu:  &sync.Mutex{},
	}
)

type dbDescriptor struct {
	pipeline   string
	pluginName string
	driver     string
}

type dbCollector struct {
	dbs map[dbDescriptor]*sqlx.DB
	mu  *sync.Mutex
}

func (c *dbCollector) append(d dbDescriptor, db *sqlx.DB) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dbs[d] = db
}

func (c *dbCollector) delete(d dbDescriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.dbs, d)
	metrics.PluginsSet.UnregisterMetric(pluginDbConnectionsMax(d))
	metrics.PluginsSet.UnregisterMetric(pluginDbConnectionsOpen(d))
	metrics.PluginsSet.UnregisterMetric(pluginDbConnectionsInUse(d))
	metrics.PluginsSet.UnregisterMetric(pluginDbConnectionsIdle(d))
	metrics.PluginsSet.UnregisterMetric(pluginDbConnectionsWaitedTotal(d))
	metrics.PluginsSet.UnregisterMetric(pluginDbConnectionsWaitedSecondsTotal(d))
}

func (c *dbCollector) Collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for d, db := range c.dbs {
		stats := db.Stats()
		metrics.PluginsSet.GetOrCreateGauge(pluginDbConnectionsMax(d), nil).Set(float64(stats.MaxOpenConnections))
		metrics.PluginsSet.GetOrCreateGauge(pluginDbConnectionsOpen(d), nil).Set(float64(stats.OpenConnections))
		metrics.PluginsSet.GetOrCreateGauge(pluginDbConnectionsInUse(d), nil).Set(float64(stats.InUse))
		metrics.PluginsSet.GetOrCreateGauge(pluginDbConnectionsIdle(d), nil).Set(float64(stats.Idle))
		metrics.PluginsSet.GetOrCreateCounter(pluginDbConnectionsWaitedTotal(d)).Set(uint64(stats.WaitCount))
		metrics.PluginsSet.GetOrCreateFloatCounter(pluginDbConnectionsWaitedSecondsTotal(d)).Set(stats.WaitDuration.Seconds())
	}
}

func RegisterDB(pipeline, pluginName, driver string, db *sqlx.DB) {
	dbMetricsRegister.Do(func() {
		metrics.GlobalCollectorsRunner.Append(dbMetricsCollector)
	})

	dbMetricsCollector.append(dbDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		driver:     driver,
	}, db)
}

func UnregisterDB(pipeline, pluginName, driver string) {
	dbMetricsCollector.delete(dbDescriptor{
		pipeline:   pipeline,
		pluginName: pluginName,
		driver:     driver,
	})
}
