package exporter

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/rs/zerolog/log"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type RethinkdbExporter struct {
	rconn r.QueryExecutor

	collectTableStats bool

	listenAddress string
	mux           *http.ServeMux

	metrics struct {
		clusterClientConnections *prometheus.Desc
		clusterDocsPerSecond     *prometheus.Desc

		serverClientConnections *prometheus.Desc
		serverQueriesPerSecond  *prometheus.Desc
		serverDocsPerSecond     *prometheus.Desc

		tableDocsPerSecond *prometheus.Desc
		tableRowsCount     *prometheus.Desc

		tableReplicaDocsPerSecond *prometheus.Desc
		tableReplicaCacheBytes    *prometheus.Desc
		tableReplicaIO            *prometheus.Desc
		tableReplicaDataBytes     *prometheus.Desc

		scrapeLatency *prometheus.Desc
		scrapeErrors  *prometheus.Desc
	}
}

type promHTTPLogger struct{}

func (l promHTTPLogger) Println(v ...interface{}) {
	log.Error().Msgf("msg: %v", fmt.Sprint(v...))
}

func New(
	listenAddress string,
	telemetryPath string,
	rconn r.QueryExecutor,
	collectTableStats bool,
) (*RethinkdbExporter, error) {
	exporter := &RethinkdbExporter{
		listenAddress:     listenAddress,
		collectTableStats: collectTableStats,
		rconn:             rconn,
	}

	exporter.initMetrics()

	prometheus.MustRegister(exporter)

	exporter.mux = http.NewServeMux()
	exporter.mux.Handle(telemetryPath,
		promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer,
			promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{
					ErrorLog: &promHTTPLogger{},
				},
			),
		),
	)
	exporter.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
             <head><title>RethinkDB Exporter</title></head>
             <body>
             <h1>RethinkDB Exporter</h1>
             <p><a href='` + telemetryPath + `'>Metrics</a></p>
             <h2>Build</h2>
             <pre>` + version.Info() + ` ` + version.BuildContext() + `</pre>
             </body>
             </html>`))
	})
	exporter.mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "OK")
	})
	exporter.mux.HandleFunc("/-/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "OK")
	})

	return exporter, nil
}

func (e *RethinkdbExporter) ListenAndServe() error {
	serv := http.Server{Addr: e.listenAddress, Handler: e.mux}
	return serv.ListenAndServe()
}
