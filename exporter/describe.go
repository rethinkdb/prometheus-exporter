package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	readOperation    = "read"
	writtenOperation = "written"
)

// Describe sends metrics descriptions to the prometheus chan
func (e *RethinkdbExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.metrics.clusterClientConnections
	ch <- e.metrics.clusterDocsPerSecond

	ch <- e.metrics.serverClientConnections
	ch <- e.metrics.serverQueriesPerSecond
	ch <- e.metrics.serverDocsPerSecond

	ch <- e.metrics.tableDocsPerSecond
	if e.metrics.tableRowsCount != nil {
		ch <- e.metrics.tableRowsCount
	}

	ch <- e.metrics.tableReplicaDocsPerSecond
	ch <- e.metrics.tableReplicaCacheBytes
	ch <- e.metrics.tableReplicaIO
	ch <- e.metrics.tableReplicaDataBytes

	ch <- e.metrics.scrapeLatency
	ch <- e.metrics.scrapeErrors
}

func (e *RethinkdbExporter) initMetrics() {
	e.metrics.clusterClientConnections = prometheus.NewDesc(
		"cluster_client_connections",
		"Total number of connections from the cluster",
		nil, nil,
	)
	e.metrics.clusterDocsPerSecond = prometheus.NewDesc(
		"cluster_docs_per_second",
		"Total number of reads and writes of documents per second from the cluster",
		[]string{"operation"}, nil)

	e.metrics.serverClientConnections = prometheus.NewDesc(
		"server_client_connections",
		"Number of client connections to the server",
		[]string{"server"}, nil)
	e.metrics.serverQueriesPerSecond = prometheus.NewDesc(
		"server_queries_per_second",
		"Number of queries per second from the server",
		[]string{"server"}, nil)
	e.metrics.serverDocsPerSecond = prometheus.NewDesc(
		"server_docs_per_second",
		"Total number of reads and writes of documents per second from the server",
		[]string{"server", "operation"}, nil)

	e.metrics.tableDocsPerSecond = prometheus.NewDesc(
		"table_docs_per_second",
		"Number of reads and writes of documents per second from the table",
		[]string{"db", "table", "operation"}, nil)

	if e.collectTableStats {
		e.metrics.tableRowsCount = prometheus.NewDesc(
			"table_rows_count",
			"Approximate number of rows in the table",
			[]string{"db", "table"}, nil)
	}

	e.metrics.tableReplicaDocsPerSecond = prometheus.NewDesc(
		"tablereplica_docs_per_second",
		"Number of reads and writes of documents per second from the table replica",
		[]string{"db", "table", "server", "operation"}, nil)
	e.metrics.tableReplicaCacheBytes = prometheus.NewDesc(
		"tablereplica_cache_bytes",
		"Table replica cache size in bytes",
		[]string{"db", "table", "server"}, nil)
	e.metrics.tableReplicaIO = prometheus.NewDesc(
		"tablereplica_io",
		"Table replica reads and writes of bytes per second",
		[]string{"db", "table", "server", "operation"}, nil)
	e.metrics.tableReplicaDataBytes = prometheus.NewDesc(
		"tablereplica_data_bytes",
		"Table replica size in stored bytes",
		[]string{"db", "table", "server"}, nil)

	e.metrics.scrapeLatency = prometheus.NewDesc(
		"scrape_latency",
		"Latency of collecting scrape",
		nil, nil)
	e.metrics.scrapeErrors = prometheus.NewDesc(
		"scrape_errors",
		"Number of errors while collecting scrape",
		nil, nil)
}
