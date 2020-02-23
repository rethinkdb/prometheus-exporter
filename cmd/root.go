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

	_ = viper.BindPFlag("Log.Debug", rootCmd.PersistentFlags().Lookup("log.debug"))
	_ = viper.BindEnv("Log.Debug", "LOG_DEBUG")
	_ = viper.BindPFlag("Log.JSONOutput", rootCmd.PersistentFlags().Lookup("log.json-output"))
	_ = viper.BindEnv("Log.JSONOutput", "LOG_JSON_OUTPUT")

	_ = viper.BindPFlag("DB.RethinkdbAddresses", rootCmd.PersistentFlags().Lookup("db.address"))
	_ = viper.BindEnv("DB.RethinkdbAddresses", "DB_ADDRESSES")
	_ = viper.BindPFlag("DB.Username", rootCmd.PersistentFlags().Lookup("db.username"))
	_ = viper.BindEnv("DB.Username", "DB_USERNAME")
	_ = viper.BindPFlag("DB.Password", rootCmd.PersistentFlags().Lookup("db.password"))
	_ = viper.BindEnv("DB.Password", "DB_PASSWORD")
	_ = viper.BindPFlag("DB.EnableTLS", rootCmd.PersistentFlags().Lookup("db.enable-tls"))
	_ = viper.BindEnv("DB.EnableTLS", "DB_ENABLE_TLS")
	_ = viper.BindPFlag("DB.CAFile", rootCmd.PersistentFlags().Lookup("db.ca"))
	_ = viper.BindEnv("DB.CAFile", "DB_CA")
	_ = viper.BindPFlag("DB.CertificateFile", rootCmd.PersistentFlags().Lookup("db.cert"))
	_ = viper.BindEnv("DB.CertificateFile", "DB_CERT")
	_ = viper.BindPFlag("DB.KeyFile", rootCmd.PersistentFlags().Lookup("db.key"))
	_ = viper.BindEnv("DB.KeyFile", "DB_KEY")
	_ = viper.BindPFlag("DB.ConnectionPoolSize", rootCmd.PersistentFlags().Lookup("db.pool-size"))
	_ = viper.BindEnv("DB.ConnectionPoolSize", "DB_POOL_SIZE")
	_ = viper.BindPFlag("Web.ListenAddress", rootCmd.PersistentFlags().Lookup("web.listen-address"))
	_ = viper.BindEnv("Web.ListenAddress", "WEB_LISTEN_ADDRESS")
	_ = viper.BindPFlag("Web.TelemetryPath", rootCmd.PersistentFlags().Lookup("web.telemetry-path"))
	_ = viper.BindEnv("Web.TelemetryPath", "WEB_TELEMETRY_PATH")
	_ = viper.BindPFlag("Stats.TableDocsEstimates", rootCmd.PersistentFlags().Lookup("stats.table-estimates"))
	_ = viper.BindEnv("Stats.TableDocsEstimates", "STATS_TABLE_ESTIMATES")

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
	//decoderOption := func(cfg *mapstructure.DecoderConfig) {
	//	cfg.TagName = "json"
	//	cfg.DecodeHook = mapstructure.ComposeDecodeHookFunc(
	//		cfg.DecodeHook,
	//		func(cfgt reflect.Type, strt reflect.Type, data interface{}) (interface{}, error) {
	//			if cfgt.Kind() != reflect.Map { // любой конфиг в файле на верхнем уровне это key-value
	//				return data, nil
	//			}
	//			if strt.Kind() != reflect.Struct { // любой конфиг в коде на верхнем это структура
	//				return data, nil
	//			}
	//			raw, err := json.Marshal(data) // кодируем map[string]interface{} в json
	//			if err != nil {
	//				return nil, err
	//			}
	//			out := reflect.New(strt).Interface() // создаём экземпляр типа конфига
	//			err = json.Unmarshal(raw, out)       // декодируем json в тип конфига
	//			return out, err                      // mapstructure decoder увидит, что типы конфига совпадают и сделает просто reflect.Set()
	//		},
	//	)
	//}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal().Err(err).Msg("failed to read config file")
		}
	}
	//log.Info().Interface("viper", viper.AllSettings()).Msg("viper")
	//if err := viper.Unmarshal(&cfg, decoderOption); err != nil {
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}
	//log.Info().Interface("cfg", cfg).Msg("cfg")
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
