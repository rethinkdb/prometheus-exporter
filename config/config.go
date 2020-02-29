package config

// Config defines the exporter's parameters
type Config struct {
	// Web defines http-server for prometheus protocol
	Web struct {
		// ListenAddress is http listen endpoint
		ListenAddress string `mapstructure:"listen_address"`
		// TelemetryPath is http url path for metrics
		TelemetryPath string `mapstructure:"telemetry_path"`
	} `mapstructure:"web"`

	// Stats defines collecting stats parameters
	Stats struct {
		// TableDocsEstimates tells the exporter to get table rows count estimates
		TableDocsEstimates bool `mapstructure:"table_docs_estimates"`
	} `mapstructure:"stats"`

	// DB defines rethinkdb-connection parameters
	DB struct {
		// RethinkdbAddresses list of endpoints of the rethinkdb nodes to connect
		RethinkdbAddresses []string `mapstructure:"rethinkdb_addresses"`

		// Username to auth in the rethinkdb
		Username string `mapstructure:"username"`
		// Password to auth in the rethinkdb
		Password string `mapstructure:"password"`

		// EnableTLS enables encryption on the connection
		EnableTLS bool `mapstructure:"enable_tls"`
		// CAFile locates path of the CA file
		CAFile string `mapstructure:"ca_file"`
		// CertificateFile locates path of the client certificate file
		CertificateFile string `mapstructure:"certificate_file"`
		// KeyFile locates path of the key file to the client certificate
		KeyFile string `mapstructure:"key_file"`

		// ConnectionPoolSize defines size of the connection pool to the rethinkdb
		ConnectionPoolSize int `mapstructure:"connection_pool_size"`
	} `mapstructure:"db"`

	// Log defines exporter's logging
	Log struct {
		// JSONOutput enables output in json-format, use for structured logging
		JSONOutput bool `mapstructure:"json_output"`
		// Debug enables more logs for debugging
		Debug bool `mapstructure:"debug"`
	} `mapstructure:"log"`
}
