package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	readOperation    = "read"
	writtenOperation = "written"
)

func (e *RethinkdbExporter) Describe(ch chan<- *prometheus.Desc) {
	e.metrics.clusterClientConnections.Describe(ch)
	e.metrics.clusterDocsPerSecond.Describe(ch)

	e.metrics.serverClientConnections.Describe(ch)
	e.metrics.serverQueriesPerSecond.Describe(ch)
	e.metrics.serverDocsPerSecond.Describe(ch)

	e.metrics.tableDocsPerSecond.Describe(ch)
	if e.metrics.tableRowsCount != nil {
		e.metrics.tableRowsCount.Describe(ch)
	}

	e.metrics.tableReplicaDocsPerSecond.Describe(ch)
	e.metrics.tableReplicaCacheBytes.Describe(ch)
	e.metrics.tableReplicaIO.Describe(ch)
	e.metrics.tableReplicaDataBytes.Describe(ch)

	e.metrics.scrapeLatency.Describe(ch)
	e.metrics.scrapeErrors.Describe(ch)
}

func (e *RethinkdbExporter) initMetrics() {
	e.metrics.clusterClientConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cluster_client_connections",
		Help: "Total number of connections from the cluster",
	})
	e.metrics.clusterDocsPerSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cluster_docs_per_second",
		Help: "Total number of reads and writes of documents per second from the cluster",
	}, []string{"operation"})

	e.metrics.serverClientConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "server_client_connections",
		Help: "Number of client connections to the server",
	}, []string{"server"})
	e.metrics.serverQueriesPerSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "server_queries_per_second",
		Help: "Number of queries per second from the server",
	}, []string{"server"})
	e.metrics.serverDocsPerSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "server_docs_per_second",
		Help: "Total number of reads and writes of documents per second from the server",
	}, []string{"server", "operation"})

	e.metrics.tableDocsPerSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "table_docs_per_second",
		Help: "Number of reads and writes of documents per second from the table",
	}, []string{"table", "operation"})

	if e.collectTableStats {
		e.metrics.tableRowsCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "table_rows_count",
			Help: "Approximate number of rows in the table",
		}, []string{"table"})
	}

	e.metrics.tableReplicaDocsPerSecond = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tablereplica_docs_per_second",
		Help: "Number of reads and writes of documents per second from the table replica",
	}, []string{"table", "server", "operation"})
	e.metrics.tableReplicaCacheBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tablereplica_cache_bytes",
		Help: "Table replica cache size in bytes",
	}, []string{"table", "server"})
	e.metrics.tableReplicaIO = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tablereplica_io",
		Help: "Table replica reads and writes of bytes per second",
	}, []string{"table", "server", "operation"})
	e.metrics.tableReplicaDataBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tablereplica_data_bytes",
		Help: "Table replica size in stored bytes",
	}, []string{"table", "server"})

	e.metrics.scrapeLatency = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "scrape_latency",
		Help: "Latency of collecting scrape",
	})
	e.metrics.scrapeErrors = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "scrape_errors",
		Help: "Number of errors while collecting scrape",
	})
}
