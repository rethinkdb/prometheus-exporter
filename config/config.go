package config

type Config struct {
	Web struct {
		ListenAddress string `mapstructure:"listen_address"`
		TelemetryPath string `mapstructure:"telemetry_path"`
	} `mapstructure:"web"`

	Stats struct {
		TableDocsEstimates bool `mapstructure:"table_docs_estimates"`
	} `mapstructure:"stats"`

	DB struct {
		RethinkdbAddresses []string `mapstructure:"rethinkdb_addresses"`

		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`

		EnableTLS       bool   `mapstructure:"enable_tls"`
		CAFile          string `mapstructure:"ca_file"`
		CertificateFile string `mapstructure:"certificate_file"`
		KeyFile         string `mapstructure:"key_file"`

		ConnectionPoolSize int `mapstructure:"connection_pool_size"`
	} `mapstructure:"db"`

	Log struct {
		JSONOutput bool `mapstructure:"json_output"`
		Debug      bool `mapstructure:"debug"`
	} `mapstructure:"log"`
}
