package cmd

import (
	"crypto/tls"

	"github.com/rethinkdb/prometheus-exporter/config"
	"github.com/rethinkdb/prometheus-exporter/dbconnector"
	"github.com/rethinkdb/prometheus-exporter/exporter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "prometheus-exporter",
	Short: "Rethinkdb statictics exporter to prometheus",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging(cfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var tlsConfig *tls.Config
		var err error
		if cfg.DB.EnableTLS {
			tlsConfig, err = dbconnector.PrepareTLSConfig(cfg.DB.CAFile, cfg.DB.CertificateFile, cfg.DB.KeyFile)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to read tls credentials")
			}
		}

		rconn := dbconnector.ConnectRethinkDB(
			cfg.DB.RethinkdbAddresses,
			cfg.DB.Username,
			cfg.DB.Password,
			tlsConfig,
			cfg.DB.ConnectionPoolSize,
		)

		exp, err := exporter.New(cfg.Web.ListenAddress, cfg.Web.TelemetryPath, rconn, cfg.Stats.TableDocsEstimates)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to init http exporter")
		}

		log.Info().Str("address", cfg.Web.ListenAddress).Msg("listening on address")
		err = exp.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to serve http exporter")
		}
	},
}

// Execute runs root command of cli of the exporter
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default to prometheus-exporter.yaml")
	rootCmd.PersistentFlags().Bool("log.debug", false, "Verbose debug logs")
	rootCmd.PersistentFlags().Bool("log.json-output", false, "Use JSON output for logs")

	rootCmd.PersistentFlags().StringSlice("db.address", []string{"localhost:28015"}, "Address of one or more nodes of rethinkdb")
	rootCmd.PersistentFlags().String("db.username", "", "Username of rethinkdb user")
	rootCmd.PersistentFlags().String("db.password", "", "Password of rethinkdb user")
	rootCmd.PersistentFlags().Bool("db.enable-tls", false, "Enable to use tls connection")
	rootCmd.PersistentFlags().String("db.ca", "", "Path to CA certificate file for tls connection")
	rootCmd.PersistentFlags().String("db.cert", "", "Path to certificate file for tls connection")
	rootCmd.PersistentFlags().String("db.key", "", "Path to key file for tls connection")
	rootCmd.PersistentFlags().Int("db.pool-size", 5, "Size of connection pool to rethinkdb")

	rootCmd.PersistentFlags().String("web.listen-address", "0.0.0.0:9055", "Address to listen on for web interface and telemetry")
	rootCmd.PersistentFlags().String("web.telemetry-path", "/metrics", "Path under which to expose metrics")

	rootCmd.PersistentFlags().Bool("stats.table-estimates", false, "Collect docs count estimates for each table")

	_ = viper.BindPFlag("log.debug", rootCmd.PersistentFlags().Lookup("log.debug"))
	_ = viper.BindEnv("log.debug", "LOG_DEBUG")
	_ = viper.BindPFlag("log.json_output", rootCmd.PersistentFlags().Lookup("log.json-output"))
	_ = viper.BindEnv("log.json_output", "LOG_JSON_OUTPUT")

	_ = viper.BindPFlag("db.rethinkdb_addresses", rootCmd.PersistentFlags().Lookup("db.address"))
	_ = viper.BindEnv("db.rethinkdb_addresses", "DB_ADDRESSES")
	_ = viper.BindPFlag("db.username", rootCmd.PersistentFlags().Lookup("db.username"))
	_ = viper.BindEnv("db.username", "DB_USERNAME")
	_ = viper.BindPFlag("db.password", rootCmd.PersistentFlags().Lookup("db.password"))
	_ = viper.BindEnv("db.password", "DB_PASSWORD")
	_ = viper.BindPFlag("db.enable_tls", rootCmd.PersistentFlags().Lookup("db.enable-tls"))
	_ = viper.BindEnv("db.enable_tls", "DB_ENABLE_TLS")
	_ = viper.BindPFlag("db.ca_file", rootCmd.PersistentFlags().Lookup("db.ca"))
	_ = viper.BindEnv("db.ca_file", "DB_CA")
	_ = viper.BindPFlag("db.certificate_file", rootCmd.PersistentFlags().Lookup("db.cert"))
	_ = viper.BindEnv("db.certificate_file", "DB_CERT")
	_ = viper.BindPFlag("db.key_file", rootCmd.PersistentFlags().Lookup("db.key"))
	_ = viper.BindEnv("db.key_file", "DB_KEY")
	_ = viper.BindPFlag("db.connection_pool_size", rootCmd.PersistentFlags().Lookup("db.pool-size"))
	_ = viper.BindEnv("db.connection_pool_size", "DB_POOL_SIZE")
	_ = viper.BindPFlag("web.listen_address", rootCmd.PersistentFlags().Lookup("web.listen-address"))
	_ = viper.BindEnv("web.listen_address", "WEB_LISTEN_ADDRESS")
	_ = viper.BindPFlag("web.telemetry_path", rootCmd.PersistentFlags().Lookup("web.telemetry-path"))
	_ = viper.BindEnv("web.TelemetryPath", "WEB_TELEMETRY_PATH")
	_ = viper.BindPFlag("stats.table_docs_estimates", rootCmd.PersistentFlags().Lookup("stats.table-estimates"))
	_ = viper.BindEnv("stats.table_docs_estimates", "STATS_TABLE_ESTIMATES")

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("prometheus-exporter")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal().Err(err).Msg("failed to read config file")
		}
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}
}

func initLogging(cfg config.Config) {
	if !cfg.Log.JSONOutput {
		log.Logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "2006-01-02T15:04:05 MST"
		}))
	}
	if cfg.Log.Debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Logger = log.With().Caller().Logger()
}
