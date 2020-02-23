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
	e.metrics.scrapeErrors.Set(float64(errcount))
	e.metrics.scrapeErrors.Collect(ch)
	e.metrics.scrapeLatency.Set(elapsed.Seconds())
	e.metrics.scrapeLatency.Collect(ch)

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

		err = e.processStat(ctx, ch, stat, wg)
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

	e.collectMetrics(ch)

	return errcount
}

func (e *RethinkdbExporter) collectMetrics(ch chan<- prometheus.Metric) {
	e.metrics.clusterClientConnections.Collect(ch)
	e.metrics.clusterDocsPerSecond.Collect(ch)

	e.metrics.serverClientConnections.Collect(ch)
	e.metrics.serverDocsPerSecond.Collect(ch)
	e.metrics.serverQueriesPerSecond.Collect(ch)

	e.metrics.tableDocsPerSecond.Collect(ch)
	if e.metrics.tableRowsCount != nil {
		e.metrics.tableRowsCount.Collect(ch)
	}

	e.metrics.tableReplicaDocsPerSecond.Collect(ch)
	e.metrics.tableReplicaCacheBytes.Collect(ch)
	e.metrics.tableReplicaIO.Collect(ch)
	e.metrics.tableReplicaDataBytes.Collect(ch)
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
	ClientConnections int     `rethinkdb:"client_connections"`
	QPS               float64 `rethinkdb:"queries_per_sec"`
	ReadDocsPerSec    float64 `rethinkdb:"read_docs_per_sec"`
	WrittenDocsPerSec float64 `rethinkdb:"written_docs_per_sec"`
}

type storageEngine struct {
	Cache struct {
		InUseBytes int `rethinkdb:"in_use_bytes"`
	} `rethinkdb:"cache"`
	Disk struct {
		ReadBytesPerSec    int `rethinkdb:"read_bytes_per_sec"`
		WrittenBytesPerSec int `rethinkdb:"written_bytes_per_sec"`
		SpaceUsage         struct {
			DataBytes int `rethinkdb:"data_bytes"`
		} `rethinkdb:"space_usage"`
	} `rethinkdb:"disk"`
}

type info struct {
	DocCountEstimates []int `rethinkdb:"doc_count_estimates"`
}

func (e *RethinkdbExporter) processStat(ctx context.Context, ch chan<- prometheus.Metric, stat stat, wg *errgroup.Group) error {
	if len(stat.ID) == 0 {
		return errors.New("unexpected empty stat id")
	}
	switch stat.ID[0] {
	case "cluster":
		e.processClusterStat(ch, stat)
	case "server":
		e.processServerStat(ch, stat)
	case "table":
		e.processTableStat(ctx, ch, stat, wg)
	case "table_server":
		e.processTableServerStat(ch, stat)
	default:
		return fmt.Errorf("unexpected stat id: '%v'", stat.ID[0])
	}
	return nil
}

func (e *RethinkdbExporter) processClusterStat(ch chan<- prometheus.Metric, stat stat) {
	e.metrics.clusterClientConnections.Set(float64(stat.QueryEngine.ClientConnections))

	e.metrics.clusterDocsPerSecond.WithLabelValues(readOperation).Set(stat.QueryEngine.ReadDocsPerSec)
	e.metrics.clusterDocsPerSecond.WithLabelValues(writtenOperation).Set(stat.QueryEngine.WrittenDocsPerSec)
}

func (e *RethinkdbExporter) processServerStat(ch chan<- prometheus.Metric, stat stat) {
	e.metrics.serverClientConnections.WithLabelValues(stat.Server).Set(float64(stat.QueryEngine.ClientConnections))

	e.metrics.serverDocsPerSecond.WithLabelValues(stat.Server, readOperation).Set(stat.QueryEngine.ReadDocsPerSec)
	e.metrics.serverDocsPerSecond.WithLabelValues(stat.Server, writtenOperation).Set(stat.QueryEngine.WrittenDocsPerSec)

	e.metrics.serverQueriesPerSecond.WithLabelValues(stat.Server).Set(stat.QueryEngine.QPS)
}

func (e *RethinkdbExporter) processTableStat(ctx context.Context, ch chan<- prometheus.Metric, stat stat, wg *errgroup.Group) {
	e.metrics.tableDocsPerSecond.WithLabelValues(composeTableAndDB(stat), readOperation).Set(stat.QueryEngine.ReadDocsPerSec)
	e.metrics.tableDocsPerSecond.WithLabelValues(composeTableAndDB(stat), writtenOperation).Set(stat.QueryEngine.WrittenDocsPerSec)

	if e.metrics.tableRowsCount != nil {
		composedName := composeTableAndDB(stat)
		dbName := stat.Database
		tableName := stat.Table

		wg.Go(func() error {
			var info info
			err := r.DB(dbName).Table(tableName).Info().ReadOne(&info, e.rconn, r.RunOpts{Context: ctx})
			if err != nil {
				log.Warn().Err(err).Str("table", composedName).Msg("failed to get table info")
				return err
			}

			sum := 0.0
			for _, e := range info.DocCountEstimates {
				sum += float64(e)
			}

			e.metrics.tableRowsCount.WithLabelValues(composedName).Set(sum)
			return nil
		})
	}
}

func (e *RethinkdbExporter) processTableServerStat(ch chan<- prometheus.Metric, stat stat) {
	e.metrics.tableReplicaDocsPerSecond.WithLabelValues(composeTableAndDB(stat), stat.Server, readOperation).Set(stat.QueryEngine.ReadDocsPerSec)
	e.metrics.tableReplicaDocsPerSecond.WithLabelValues(composeTableAndDB(stat), stat.Server, writtenOperation).Set(stat.QueryEngine.WrittenDocsPerSec)

	e.metrics.tableReplicaCacheBytes.WithLabelValues(composeTableAndDB(stat), stat.Server).Set(float64(stat.StorageEngine.Cache.InUseBytes))

	e.metrics.tableReplicaIO.WithLabelValues(composeTableAndDB(stat), stat.Server, readOperation).Set(float64(stat.StorageEngine.Disk.ReadBytesPerSec))
	e.metrics.tableReplicaIO.WithLabelValues(composeTableAndDB(stat), stat.Server, writtenOperation).Set(float64(stat.StorageEngine.Disk.WrittenBytesPerSec))

	e.metrics.tableReplicaDataBytes.WithLabelValues(composeTableAndDB(stat), stat.Server).Set(float64(stat.StorageEngine.Disk.SpaceUsage.DataBytes))
}

func composeTableAndDB(stat stat) string {
	return fmt.Sprintf("%v.%v", stat.Database, stat.Table)
}
