package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/brunotm/log"
	"github.com/brunotm/replicant/api"
	goDriver "github.com/brunotm/replicant/driver/go"
	webDriver "github.com/brunotm/replicant/driver/web"
	"github.com/brunotm/replicant/emitter/elasticsearch"
	"github.com/brunotm/replicant/emitter/prometheus"
	"github.com/brunotm/replicant/emitter/stdout"
	"github.com/brunotm/replicant/internal/webhook"
	"github.com/brunotm/replicant/server"
	"github.com/brunotm/replicant/store"
	_ "github.com/brunotm/replicant/store/leveldb"
	_ "github.com/brunotm/replicant/store/memory"
	"github.com/brunotm/replicant/transaction/callback"
	"github.com/brunotm/replicant/transaction/manager"
)

var (
	defaultConfigFile = "replicant.yaml"
	writeConfig       = flag.Bool("write", false, "write default configuration")
	configFile        = flag.String("config", "", "replicant configuration file")
)

// bloody and dirty
func main() {
	flag.Parse()

	var err error
	cfg := DefaultConfig

	// Write default config to file
	if *writeConfig {
		if *configFile != "" {
			defaultConfigFile = *configFile
		}

		err = writeConfigFile(cfg, defaultConfigFile)
		if err != nil {
			log.Error("could not write config file").
				String("file", defaultConfigFile).
				String("error", err.Error()).Log()
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Load config from file if specified
	if *configFile != "" {
		cfg, err = readConfigFile(*configFile)
		if err != nil {
			log.Error("could not read config file").
				String("path", *configFile).
				String("error", err.Error()).Log()
			os.Exit(1)
		}
	}

	// Initialize the transaction store
	log.Info("initializing store").String("uri", cfg.StoreURI).Log()
	st, err := store.New(cfg.StoreURI)
	if err != nil {
		log.Error("could not initialize store").
			String("error", err.Error()).Log()
		os.Exit(1)
	}

	// Create a new transaction manager and service
	m := manager.New(st, goDriver.New(), webDriver.New(cfg.Drivers.Web))

	srv, err := server.New(cfg.Server, m)
	if err != nil {
		log.Error("failed to create replicant server").
			String("error", err.Error()).Log()
		os.Exit(1)
	}

	// Register the webhook asynchronous response listener
	err = callback.Register("webhook", webhook.New(cfg.Callbacks.Webhook, srv.Router()))
	if err != nil {
		log.Error("failed to register webhook response handler").
			String("error", err.Error()).Log()
		os.Exit(1)
	}
	log.Info("registered response handler").
		String("type", "webhook").
		String("advertise_url",
			fmt.Sprintf("%s%s", cfg.Callbacks.Webhook.AdvertiseURL, cfg.Callbacks.Webhook.PathPrefix)).Log()

	// Register result emitters with the manager service
	var e manager.Emitter
	srv.Manager().AddEmitter(stdout.New(cfg.Emitters.Stdout))

	e, err = prometheus.New(cfg.Emitters.Prometheus, srv.Router())
	if err != nil {
		log.Error("failed to create prometheus emitter").
			String("error", err.Error()).Log()
		os.Exit(1)
	}
	srv.Manager().AddEmitter(e)

	if len(cfg.Emitters.Elasticsearch.Urls) > 0 && cfg.Emitters.Elasticsearch.Index != "" {
		e, err = elasticsearch.New(cfg.Emitters.Elasticsearch)
		if err != nil {
			log.Error("failed to create prometheus emitter").
				String("error", err.Error()).Log()
			os.Exit(1)
		}
		srv.Manager().AddEmitter(e)
	}

	// Enable profiling endpoints
	if cfg.Debug {
		log.Info("adding debug api routes for runtime profiling data").Log()
		api.AddDebugRoutes(srv)
	}

	// Register all api endpoints and start the service
	api.AddAllRoutes(cfg.APIPrefix, srv)
	go srv.Start()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	log.Info("replicant server started").
		String("address", cfg.Server.ListenAddress).Log()
	<-signalCh

	if err = srv.Close(context.Background()); err != nil {
		log.Info("replicant server stopped").String("error", err.Error()).Log()
		os.Exit(1)
	}

	log.Info("replicant server stopped").Log()

}
