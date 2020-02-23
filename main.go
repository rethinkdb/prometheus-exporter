package main

import (
	"github.com/rethinkdb/prometheus-exporter/cmd"
	"github.com/rs/zerolog/log"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Msg("unhandled error")
	}
}
