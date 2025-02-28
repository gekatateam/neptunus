package metrics

import (
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dbMetricsRegister  = &sync.Once{}
	dbMetricsCollector = &dbCollector{
		dbs: make(map[dbDescriptor]*sqlx.DB),
		mu:  &sync.Mutex{},
	}

	dbConnectionsMax                *prometheus.Desc
	dbConnectionsOpen               *prometheus.Desc
	dbConnectionsInUse              *prometheus.Desc
	dbConnectionsIdle               *prometheus.Desc
	dbConnectionsWaitedTotal        *prometheus.Desc
	dbConnectionsWaitedSecondsTotal *prometheus.Desc
)

func init() {
	dbConnectionsMax = prometheus.NewDesc(
		"plugin_db_connections_max",
		"Pipeline plugin DB pool maximum number of open connections.",
		[]string{"pipeline", "plugin_name", "driver"},
		nil,
	)
	dbConnectionsOpen = prometheus.NewDesc(
		"plugin_db_connections_open",
		"Pipeline plugin DB pool number of established connections both in use and idle.",
		[]string{"pipeline", "plugin_name", "driver"},
		nil,
	)
	dbConnectionsInUse = prometheus.NewDesc(
		"plugin_db_connections_in_use",
		"Pipeline plugin DB pool number of connections currently in use.",
		[]string{"pipeline", "plugin_name", "driver"},
		nil,
	)
	dbConnectionsIdle = prometheus.NewDesc(
		"plugin_db_connections_idle",
		"Pipeline plugin DB pool number of idle connections.",
		[]string{"pipeline", "plugin_name", "driver"},
		nil,
	)
	dbConnectionsWaitedTotal = prometheus.NewDesc(
		"plugin_db_connections_waited_total",
		"Pipeline plugin DB pool total number of connections waited for.",
		[]string{"pipeline", "plugin_name", "driver"},
		nil,
	)
	dbConnectionsWaitedSecondsTotal = prometheus.NewDesc(
		"plugin_db_connections_waited_seconds_total",
		"Pipeline plugin DB pool total time blocked waiting for a new connection.",
		[]string{"pipeline", "plugin_name", "driver"},
		nil,
	)
}

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
}

func (c *dbCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- dbConnectionsMax
	ch <- dbConnectionsOpen
	ch <- dbConnectionsInUse
	ch <- dbConnectionsIdle
	ch <- dbConnectionsWaitedTotal
	ch <- dbConnectionsWaitedSecondsTotal
}

func (c *dbCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for desc, db := range c.dbs {
		stats := db.Stats()

		ch <- prometheus.MustNewConstMetric(
			dbConnectionsMax,
			prometheus.GaugeValue,
			float64(stats.MaxOpenConnections),
			desc.pipeline, desc.pluginName, desc.driver,
		)
		ch <- prometheus.MustNewConstMetric(
			dbConnectionsOpen,
			prometheus.GaugeValue,
			float64(stats.OpenConnections),
			desc.pipeline, desc.pluginName, desc.driver,
		)
		ch <- prometheus.MustNewConstMetric(
			dbConnectionsInUse,
			prometheus.GaugeValue,
			float64(stats.InUse),
			desc.pipeline, desc.pluginName, desc.driver,
		)
		ch <- prometheus.MustNewConstMetric(
			dbConnectionsIdle,
			prometheus.GaugeValue,
			float64(stats.Idle),
			desc.pipeline, desc.pluginName, desc.driver,
		)
		ch <- prometheus.MustNewConstMetric(
			dbConnectionsWaitedTotal,
			prometheus.CounterValue,
			float64(stats.WaitCount),
			desc.pipeline, desc.pluginName, desc.driver,
		)
		ch <- prometheus.MustNewConstMetric(
			dbConnectionsWaitedSecondsTotal,
			prometheus.CounterValue,
			stats.WaitDuration.Seconds(),
			desc.pipeline, desc.pluginName, desc.driver,
		)
	}
}

func RegisterDB(pipeline, pluginName, driver string, db *sqlx.DB) {
	dbMetricsRegister.Do(func() {
		prometheus.MustRegister(dbMetricsCollector)
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
