package config

type Config struct {
	Web struct {
		ListenAddress string
		TelemetryPath string
	}

	Stats struct {
		TableDocsEstimates bool
	}

	DB struct {
		RethinkdbAddresses []string

		Username string
		Password string

		EnableTLS       bool
		CAFile          string
		CertificateFile string
		KeyFile         string

		ConnectionPoolSize int
	}

	Log struct {
		JSONOutput bool
		Debug      bool
	}
}
