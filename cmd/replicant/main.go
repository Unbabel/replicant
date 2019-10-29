package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/brunotm/log"
	"github.com/brunotm/replicant/api"
	_ "github.com/brunotm/replicant/driver/go"
	_ "github.com/brunotm/replicant/driver/web"
	"github.com/brunotm/replicant/emitter/elasticsearch"
	"github.com/brunotm/replicant/emitter/prometheus"
	"github.com/brunotm/replicant/emitter/stdout"
	"github.com/brunotm/replicant/internal/webhook"
	"github.com/brunotm/replicant/server"
	"github.com/brunotm/replicant/store/memory"
	"github.com/brunotm/replicant/transaction/callback"
)

// bloody and dirty
func main() {

	var ok bool
	var err error

	config := server.Config{}
	if config.Addr, ok = os.LookupEnv("LISTEN_ADDRESS"); !ok {
		config.Addr = "0.0.0.0:8080"
	}

	srv, err := server.New(config, memory.New())
	if err != nil {
		log.Error("failed to create replicant server").
			String("error", err.Error()).Log()
		os.Exit(1)
	}

	callbackURL, ok := os.LookupEnv("CALLBACK_URL")
	if !ok {
		callbackURL = fmt.Sprintf("http://%s", config.Addr)
	}

	err = callback.Register("webhook", webhook.New(callbackURL, "/v1/callbacks", srv.Router()))
	if err != nil {
		log.Error("failed to register webhook response handler").
			String("error", err.Error()).Log()
		os.Exit(1)
	}
	log.Info("registered response handler").
		String("type", "webhook").String("url", callbackURL).Log()

	emitters, _ := os.LookupEnv("EMITTER")
	setupStdoutEmitter(emitters, srv)
	setupElasticEmitter(emitters, srv)
	setupPrometheusEmitter(emitters, srv)

	prefix, _ := os.LookupEnv("API_PREFIX")
	api.AddAllRoutes(prefix, srv)
	go srv.Start()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	log.Info("replicant server started").
		String("address", config.Addr).Log()
	<-signalCh

	if err = srv.Close(context.Background()); err != nil {
		log.Info("replicant server stopped").String("error", err.Error()).Log()
		os.Exit(1)
	}

	log.Info("replicant server stopped").Log()

}

func setupStdoutEmitter(e string, s *server.Server) {
	if !strings.Contains(e, "stdout") {
		return
	}
	s.Manager().AddEmitterFunc(stdout.Emitter)
}

func setupPrometheusEmitter(e string, s *server.Server) {
	if !strings.Contains(e, "prometheus") {
		return
	}

	em, err := prometheus.New("/metrics", s.Router())
	if err != nil {
		log.Error("failed to initialize prometheus emitter").
			String("error", err.Error()).Log()
		os.Exit(1)
	}
	s.Manager().AddEmitter(em)
}

func setupElasticEmitter(e string, s *server.Server) {

	if !strings.Contains(e, "elasticsearch") {
		return
	}

	var err error
	var cfg elasticsearch.Config

	if urls, ok := os.LookupEnv("ELASTICSEARCH_URLS"); ok {
		cfg.Urls = strings.Split(urls, ",")
	}

	cfg.Username, _ = os.LookupEnv("ELASTICSEARCH_USERNAME")
	cfg.Password, _ = os.LookupEnv("ELASTICSEARCH_PASSWORD")
	cfg.Index, _ = os.LookupEnv("ELASTICSEARCH_INDEX")
	cfg.Username, _ = os.LookupEnv("ELASTICSEARCH_USERNAME")

	if mxpb, ok := os.LookupEnv("ELASTICSEARCH_MAX_PENDING_BYTES"); ok {
		cfg.MaxPendingBytes, err = strconv.ParseInt(mxpb, 10, 64)
		if err != nil {
			log.Error("failed to parse ELASTICSEARCH_MAX_PENDING_BYTES").
				String("error", err.Error()).Log()
			os.Exit(1)
		}
	}

	if mxpr, ok := os.LookupEnv("ELASTICSEARCH_MAX_PENDING_RESULTS"); ok {
		cfg.MaxPendingBytes, err = strconv.ParseInt(mxpr, 10, 64)
		if err != nil {
			log.Error("failed to parse ELASTICSEARCH_MAX_PENDING_RESULTS").
				String("error", err.Error()).Log()
			os.Exit(1)
		}
	}

	if mxpp, ok := os.LookupEnv("ELASTICSEARCH_MAX_PENDING_TIME"); ok {
		cfg.MaxPendingTime, err = time.ParseDuration(mxpp)
		if err != nil {
			log.Error("failed to parse ELASTICSEARCH_MAX_PENDING_TIME").
				String("error", err.Error()).Log()
			os.Exit(1)
		}
	}

	em, err := elasticsearch.New(cfg)
	if err != nil {
		log.Error("failed to initialize elasticsearch emitter").
			String("error", err.Error()).Log()
		os.Exit(1)
	}

	s.Manager().AddEmitter(em)
}
