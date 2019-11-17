package config

import (
	"time"

	"github.com/brunotm/replicant/driver/web"
	"github.com/brunotm/replicant/emitter/elasticsearch"
	"github.com/brunotm/replicant/emitter/prometheus"
	"github.com/brunotm/replicant/emitter/stdout"
	"github.com/brunotm/replicant/internal/webhook"
	"github.com/brunotm/replicant/server"
)

// Config for replicant
type Config struct {
	Debug     bool           `json:"debug" yaml:"debug"`
	LogLevel  string         `json:"log_level" yaml:"log_level"`
	APIPrefix string         `json:"api_prefix" yaml:"api_prefix"`
	StoreURI  string         `json:"store_path" yaml:"store_uri"`
	Server    server.Config  `json:"server" yaml:"server"`
	Drivers   DriverConfig   `json:"drivers" yaml:"drivers"`
	Emitters  EmitterConfig  `json:"emitters" yaml:"emitters"`
	Callbacks CallbackConfig `json:"callbacks" yaml:"callbacks"`
}

// DriverConfig optins
type DriverConfig struct {
	Web web.Config `json:"web" yaml:"web"`
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
	LogLevel:  "INFO",
	APIPrefix: "/api",
	StoreURI:  "memory:-",
	Server: server.Config{
		ListenAddress:     "0.0.0.0:8080",
		WriteTimeout:      time.Second * 600,
		ReadTimeout:       time.Second * 600,
		ReadHeaderTimeout: time.Second * 600,
	},

	Drivers: DriverConfig{
		Web: web.Config{
			ServerURL:    "http://127.0.0.1:9222",
			DNSDiscovery: true,
		},
	},

	Callbacks: CallbackConfig{
		Webhook: webhook.Config{
			AdvertiseURL: "http://0.0.0.0:8080",
			PathPrefix:   "/api/callback",
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
