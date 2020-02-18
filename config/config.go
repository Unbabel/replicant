// Package config implements a the common configuration and defaults for all
// replicant configuration
package config

import (
	"time"

	"github.com/Unbabel/replicant/emitter/elasticsearch"
	"github.com/Unbabel/replicant/emitter/prometheus"
	"github.com/Unbabel/replicant/emitter/stdout"
	"github.com/Unbabel/replicant/internal/webhook"
	"github.com/Unbabel/replicant/server"
)

// Config for replicant
type Config struct {
	Debug       bool           `json:"debug" yaml:"debug"`
	LogLevel    string         `json:"log_level" yaml:"log_level"`
	StoreURI    string         `json:"store_path" yaml:"store_uri"`
	ExecutorURL string         `json:"executor_url" yaml:"executor_url"`
	Server      server.Config  `json:"server" yaml:"server"`
	Emitters    EmitterConfig  `json:"emitters" yaml:"emitters"`
	Callbacks   CallbackConfig `json:"callbacks" yaml:"callbacks"`
}

// EmitterConfig options
type EmitterConfig struct {
	Stdout        stdout.Config        `json:"stdout" yaml:"stdout"`
	Prometheus    prometheus.Config    `json:"prometheus" yaml:"prometheus"`
	Elasticsearch elasticsearch.Config `json:"elasticsearch" yaml:"elasticsearch"`
}

// CallbackConfig options
type CallbackConfig struct {
	Webhook webhook.Config
}

// DefaultConfig for replicant
var DefaultConfig = Config{
	LogLevel: "INFO",
	StoreURI: "memory:-",
	Server: server.Config{
		ListenAddress:     "0.0.0.0:8080",
		WriteTimeout:      5 * time.Minute,
		ReadTimeout:       5 * time.Minute,
		ReadHeaderTimeout: 5 * time.Minute,
	},

	Callbacks: CallbackConfig{
		Webhook: webhook.Config{
			AdvertiseURL: "http://0.0.0.0:8080",
			PathPrefix:   "/callback",
		},
	},

	Emitters: EmitterConfig{
		Stdout: stdout.Config{Pretty: false},
		Prometheus: prometheus.Config{
			Path:              "/metrics",
			Gauges:            true,
			Summaries:         true,
			Labels:            []string{"transaction", "application", "environment", "component"},
			SummaryObjectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},

		Elasticsearch: elasticsearch.Config{},
	},
}
