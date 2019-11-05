package main

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/brunotm/replicant/emitter/elasticsearch"
	"github.com/brunotm/replicant/emitter/prometheus"
	"github.com/brunotm/replicant/emitter/stdout"
	"github.com/brunotm/replicant/internal/webhook"
	"github.com/brunotm/replicant/server"
	"gopkg.in/yaml.v2"
)

// Config for replicant
type Config struct {
	Debug     bool           `json:"debug" yaml:"debug"`
	LogLevel  string         `json:"log_level" yaml:"log_level"`
	APIPrefix string         `json:"api_prefix" yaml:"api_prefix"`
	StoreURI  string         `json:"store_path" yaml:"store_uri"`
	Server    server.Config  `json:"server" yaml:"server"`
	Emitters  EmitterConfig  `json:"emitters" yaml:"emitters"`
	Callbacks CallbackConfig `json:"callbacks" yaml:"callbacks"`
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
	Server: server.Config{
		ListenAddress:     "0.0.0.0:8080",
		WriteTimeout:      time.Second * 600,
		ReadTimeout:       time.Second * 600,
		ReadHeaderTimeout: time.Second * 600,
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
			Path:      "/metrics",
			Gauges:    true,
			Summaries: true,
			Labels:    []string{"application", "environment", "component"},
		},

		Elasticsearch: elasticsearch.Config{},
	},
}

func writeConfigFile(c Config, f string) (err error) {
	b, err := yaml.Marshal(&c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(*configFile, b, 0644)
	if err != nil {
		return err
	}

	return nil
}

func readConfigFile(f string) (c Config, err error) {
	file, err := os.Open(f)
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return c, err
	}
	file.Close()

	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return c, err
	}

	return c, nil
}
