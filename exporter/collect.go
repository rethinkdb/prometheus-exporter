package exporter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func (e *RethinkdbExporter) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()

	ctx := context.TODO() // TODO: add scrape timeout
	errcount := e.collectRethinkStats(ctx, ch)

	elapsed := time.Since(start)
	ch <- prometheus.MustNewConstMetric(e.metrics.scrapeErrors, prometheus.GaugeValue, float64(errcount))
	ch <- prometheus.MustNewConstMetric(e.metrics.scrapeLatency, prometheus.GaugeValue, elapsed.Seconds())

	log.Debug().Dur("duration", elapsed).Msg("collect finished")
}

func (e *RethinkdbExporter) collectRethinkStats(ctx context.Context, ch chan<- prometheus.Metric) int {
	errcount := 0

	cur, err := r.DB(r.SystemDatabase).Table(r.StatsSystemTable).Run(e.rconn, r.RunOpts{Context: ctx})
	if err != nil {
		log.Error().Err(err).Msg("failed to query system stats table")
		errcount++
		return errcount
	}
	defer func() {
		err := cur.Close()
		if err != nil {
			log.Warn().Err(err).Msg("error while closing cursor")
		}
	}()

	if cur.Err() != nil {
		log.Error().Err(cur.Err()).Msg("query error from cursor")
		errcount++
		return errcount
	}

	wg := &errgroup.Group{}
	var stat stat
	for cur.Next(&stat) {
		if cur.Err() != nil {
			log.Error().Err(cur.Err()).Msg("query error from cursor")
			errcount++
			return errcount
		}

		err = e.processStat(ctx, stat, wg, ch)
		if err != nil {
			log.Warn().Err(err).Msg("error while processing stat")
			errcount++
		}
	}
	err = wg.Wait()
	if err != nil {
		log.Warn().Err(err).Msg("error while processing stat")
		errcount++
	}

	return errcount
}

type stat struct {
	ID            []string      `rethinkdb:"id"`
	Server        string        `rethinkdb:"server"`
	Database      string        `rethinkdb:"db"`
	Table         string        `rethinkdb:"table"`
	QueryEngine   queryEngine   `rethinkdb:"query_engine"`
	StorageEngine storageEngine `rethinkdb:"storage_engine"`
}

type queryEngine struct {
	ClientConnections float64 `rethinkdb:"client_connections"`
	QPS               float64 `rethinkdb:"queries_per_sec"`
	ReadDocsPerSec    float64 `rethinkdb:"read_docs_per_sec"`
	WrittenDocsPerSec float64 `rethinkdb:"written_docs_per_sec"`
}

type storageEngine struct {
	Cache struct {
		InUseBytes float64 `rethinkdb:"in_use_bytes"`
	} `rethinkdb:"cache"`
	Disk struct {
		ReadBytesPerSec    float64 `rethinkdb:"read_bytes_per_sec"`
		WrittenBytesPerSec float64 `rethinkdb:"written_bytes_per_sec"`
		SpaceUsage         struct {
			DataBytes float64 `rethinkdb:"data_bytes"`
		} `rethinkdb:"space_usage"`
	} `rethinkdb:"disk"`
}

type info struct {
	DocCountEstimates []float64 `rethinkdb:"doc_count_estimates"`
}

func (e *RethinkdbExporter) processStat(ctx context.Context, stat stat, wg *errgroup.Group, ch chan<- prometheus.Metric) error {
	if len(stat.ID) == 0 {
		return errors.New("unexpected empty stat id")
	}
	switch stat.ID[0] {
	case "cluster":
		e.processClusterStat(stat, ch)
	case "server":
		e.processServerStat(stat, ch)
	case "table":
		e.processTableStat(ctx, stat, wg, ch)
	case "table_server":
		e.processTableServerStat(stat, ch)
	default:
		return fmt.Errorf("unexpected stat id: '%v'", stat.ID[0])
	}
	return nil
}

func (e *RethinkdbExporter) processClusterStat(stat stat, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.metrics.clusterClientConnections, prometheus.GaugeValue, stat.QueryEngine.ClientConnections)

	ch <- prometheus.MustNewConstMetric(e.metrics.clusterDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.ReadDocsPerSec, readOperation)
	ch <- prometheus.MustNewConstMetric(e.metrics.clusterDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.WrittenDocsPerSec, writtenOperation)
}

func (e *RethinkdbExporter) processServerStat(stat stat, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.metrics.serverClientConnections, prometheus.GaugeValue, stat.QueryEngine.ClientConnections, stat.Server)

	ch <- prometheus.MustNewConstMetric(e.metrics.serverDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.ReadDocsPerSec, stat.Server, readOperation)
	ch <- prometheus.MustNewConstMetric(e.metrics.serverDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.WrittenDocsPerSec, stat.Server, writtenOperation)

	ch <- prometheus.MustNewConstMetric(e.metrics.serverQueriesPerSecond, prometheus.GaugeValue, stat.QueryEngine.ReadDocsPerSec, stat.Server)
}

func (e *RethinkdbExporter) processTableStat(ctx context.Context, stat stat, wg *errgroup.Group, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.metrics.tableDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.ReadDocsPerSec, stat.Database, stat.Table, readOperation)
	ch <- prometheus.MustNewConstMetric(e.metrics.tableDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.WrittenDocsPerSec, stat.Database, stat.Table, writtenOperation)

	if e.metrics.tableRowsCount != nil {
		dbName := stat.Database
		tableName := stat.Table

		wg.Go(func() error {
			var info info
			err := r.DB(dbName).Table(tableName).Info().ReadOne(&info, e.rconn, r.RunOpts{Context: ctx})
			if err != nil {
				log.Warn().Err(err).Str("db", dbName).Str("table", tableName).Msg("failed to get table info")
				return err
			}

			sum := 0.0
			for _, e := range info.DocCountEstimates {
				sum += float64(e)
			}

			ch <- prometheus.MustNewConstMetric(e.metrics.tableRowsCount, prometheus.GaugeValue, sum, dbName, tableName)
			return nil
		})
	}
}

func (e *RethinkdbExporter) processTableServerStat(stat stat, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.metrics.tableReplicaDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.ReadDocsPerSec, stat.Database, stat.Table, stat.Server, readOperation)
	ch <- prometheus.MustNewConstMetric(e.metrics.tableReplicaDocsPerSecond, prometheus.GaugeValue, stat.QueryEngine.WrittenDocsPerSec, stat.Database, stat.Table, stat.Server, writtenOperation)

	ch <- prometheus.MustNewConstMetric(e.metrics.tableReplicaCacheBytes, prometheus.GaugeValue, stat.StorageEngine.Cache.InUseBytes, stat.Database, stat.Table, stat.Server)

	ch <- prometheus.MustNewConstMetric(e.metrics.tableReplicaIO, prometheus.GaugeValue, stat.StorageEngine.Disk.ReadBytesPerSec, stat.Database, stat.Table, stat.Server, readOperation)
	ch <- prometheus.MustNewConstMetric(e.metrics.tableReplicaIO, prometheus.GaugeValue, stat.StorageEngine.Disk.WrittenBytesPerSec, stat.Database, stat.Table, stat.Server, writtenOperation)

	ch <- prometheus.MustNewConstMetric(e.metrics.tableReplicaDataBytes, prometheus.GaugeValue, stat.StorageEngine.Disk.SpaceUsage.DataBytes, stat.Database, stat.Table, stat.Server)
}
