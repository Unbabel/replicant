package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/Unbabel/replicant/api"
	"github.com/Unbabel/replicant/config"
	goDriver "github.com/Unbabel/replicant/driver/go"
	jsDriver "github.com/Unbabel/replicant/driver/javascript"
	webDriver "github.com/Unbabel/replicant/driver/web"
	"github.com/Unbabel/replicant/emitter/elasticsearch"
	"github.com/Unbabel/replicant/emitter/prometheus"
	"github.com/Unbabel/replicant/emitter/stdout"
	"github.com/Unbabel/replicant/internal/webhook"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/manager"
	"github.com/Unbabel/replicant/server"
	"github.com/Unbabel/replicant/store"
	_ "github.com/Unbabel/replicant/store/leveldb"
	_ "github.com/Unbabel/replicant/store/memory"
	_ "github.com/Unbabel/replicant/store/s3"
	"github.com/Unbabel/replicant/transaction/callback"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v2"
)

var (
	defaultConfigFile = "replicant.yaml"
	writeConfig       = flag.Bool("write", false, "write default configuration")
	configFile        = flag.String("config", "", "replicant configuration file")

	Version   string
	GitCommit string
	BuildTime string
)

// bloody and dirty
func main() {
	flag.Parse()
	fmt.Printf("replicant version: %s, build date: %s\n\n", Version, BuildTime)

	var err error
	cfg := config.DefaultConfig

	// Write default config to file
	if *writeConfig {
		if *configFile != "" {
			defaultConfigFile = *configFile
		}

		err = writeConfigFile(cfg, defaultConfigFile)
		if err != nil {
			fmt.Printf("could not write config file: %s, error: %s\n", defaultConfigFile, err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Load config from file if specified
	if *configFile != "" {
		cfg, err = readConfigFile(*configFile)
		if err != nil {
			fmt.Printf("could not read config file: %s, error: %s\n", *configFile, err)
			os.Exit(1)
		}
	}

	if err = log.Init(cfg.LogLevel); err != nil {
		fmt.Printf("error initializing log: %s\n", err)
		os.Exit(1)
	}
	log.Info("logger initialized").String("level", cfg.LogLevel).Log()

	// Initialize the transaction store
	log.Info("initializing store").String("uri", cfg.StoreURI).Log()
	st, err := store.New(cfg.StoreURI)
	if err != nil {
		log.Error("could not initialize store").
			String("error", err.Error()).Log()
		os.Exit(1)
	}

	driverJS, err := jsDriver.New()
	if err != nil {
		log.Error("could not initialize javascript driver").
			String("error", err.Error()).Log()
		os.Exit(1)
	}

	// Init httprouter
	router := httprouter.New()

	// Register the webhook asynchronous response listener
	err = callback.Register("webhook", webhook.New(cfg.Callbacks.Webhook, router))
	if err != nil {
		log.Error("failed to register webhook response handler").
			String("error", err.Error()).Log()
		os.Exit(1)
	}
	log.Info("registered response handler").
		String("type", "webhook").
		String("advertise_url",
			fmt.Sprintf("%s%s", cfg.Callbacks.Webhook.AdvertiseURL, cfg.Callbacks.Webhook.PathPrefix)).Log()

	// Create a new transaction manager and service
	m := manager.New(st,
		driverJS,
		goDriver.New(),
		webDriver.New(cfg.Drivers.Web))

	// Register result emitters with the manager service
	var e manager.Emitter
	m.AddEmitter(stdout.New(cfg.Emitters.Stdout))

	e, err = prometheus.New(cfg.Emitters.Prometheus, router)
	if err != nil {
		log.Error("failed to create prometheus emitter").
			String("error", err.Error()).Log()
		os.Exit(1)
	}
	m.AddEmitter(e)

	if len(cfg.Emitters.Elasticsearch.Urls) > 0 && cfg.Emitters.Elasticsearch.Index != "" {
		e, err = elasticsearch.New(cfg.Emitters.Elasticsearch)
		if err != nil {
			log.Error("failed to create prometheus emitter").
				String("error", err.Error()).Log()
			os.Exit(1)
		}
		m.AddEmitter(e)
	}

	srv, err := server.New(cfg.Server, m, router)
	if err != nil {
		log.Error("failed to create replicant server").
			String("error", err.Error()).Log()
		os.Exit(1)
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

func writeConfigFile(c config.Config, f string) (err error) {
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

func readConfigFile(f string) (c config.Config, err error) {
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
