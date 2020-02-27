package cmd

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/Unbabel/replicant/api"
	"github.com/Unbabel/replicant/emitter/prometheus"
	"github.com/Unbabel/replicant/emitter/stdout"
	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/internal/webhook"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/manager"
	"github.com/Unbabel/replicant/server"
	"github.com/Unbabel/replicant/store"
	"github.com/Unbabel/replicant/transaction/callback"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"

	// Load store and callback drivers
	_ "github.com/Unbabel/replicant/store/leveldb"
	_ "github.com/Unbabel/replicant/store/memory"
	_ "github.com/Unbabel/replicant/store/s3"
)

func init() {
	Server.Flags().String("listen-address", "0.0.0.0:8080", "Address to for server to listen on")
	Server.Flags().Duration("max-runtime", time.Minute*5, "Maximum individual test runtime")
	Server.Flags().String("store-uri", "memory:-", "store uri, currently supported: memory:-, leveldb:/<path>, s3://<user>:<password>@<bucket>/path?region=<region>")
	Server.Flags().String("executor-url", "http://localhost:8081", "Replicant executor url")
	Server.Flags().Bool("emit-stdout", true, "Emit json structured results to standard output")
	Server.Flags().Bool("emit-stdout-pretty", false, "Pretty print stdout json output")
	Server.Flags().Bool("emit-prometheus", true, "Expose a prometheus exporter for emitting result data at /metrics")
	Server.Flags().String("webhook-advertise-url", "http://localhost:8080", "URL to advertise when receiving webhook based async responses")
	Server.Flags().String("webhook-path-prefix", "/callback", "Path prefix to receive callbacks on, eg: /<path-prefix>/<transaction-uuid>")
	Server.Flags().Bool("debug", false, "Expose a debug profile endpoint at /debug/pprof")

}

// Server command
var Server = &cobra.Command{
	Use:   "server",
	Short: "Start the replicant server",
	Run: func(cmd *cobra.Command, args []string) {
		// Init httprouter
		router := httprouter.New()

		// Setup Store
		storeURI := cmdutil.GetFlagString(cmd, "store-uri")
		st, err := store.New(storeURI)
		if err != nil {
			log.Error("could not initialize store").String("error", err.Error()).Log()
			os.Exit(1)
		}

		// Setup manager
		executorURL := cmdutil.GetFlagString(cmd, "executor-url")
		m := manager.New(st, executorURL)

		emitStdout := cmdutil.GetFlagBool(cmd, "emit-stdout")
		if emitStdout {
			m.AddEmitter(stdout.New(stdout.Config{Pretty: cmdutil.GetFlagBool(cmd, "emit-stdout-pretty")}))
		}

		emitPrometheus := cmdutil.GetFlagBool(cmd, "emit-prometheus")
		if emitPrometheus {
			e, err := prometheus.New(prometheus.DefaultConfig, router)
			if err != nil {
				log.Error("failed to create server").String("error", err.Error()).Log()
				os.Exit(1)
			}
			m.AddEmitter(e)
		}

		// Setup webhook based callbacks
		webhookURL := cmdutil.GetFlagString(cmd, "webhook-advertise-url")
		webhookPrefix := cmdutil.GetFlagString(cmd, "webhook-path-prefix")

		err = callback.Register("webhook", webhook.New(
			webhook.Config{AdvertiseURL: webhookURL, PathPrefix: webhookPrefix}, router))
		if err != nil {
			log.Error("failed to create server").String("error", err.Error()).Log()
			os.Exit(1)
		}

		// Setup server
		address := cmdutil.GetFlagString(cmd, "listen-address")
		timeout := cmdutil.GetFlagDuration(cmd, "max-runtime")

		srv, err := server.New(server.Config{
			ListenAddress:     address,
			ReadTimeout:       timeout,
			WriteTimeout:      timeout,
			ReadHeaderTimeout: timeout},
			m, router)

		if err != nil {
			log.Error("failed to create replicant server").String("error", err.Error()).Log()
			os.Exit(1)
		}

		// Debugging endpoints
		if cmdutil.GetFlagBool(cmd, "debug") {
			log.Info("adding debug api routes for runtime profiling data").Log()
			api.AddDebugRoutes(srv)
		}

		// Register all api endpoints and start the service
		api.AddAllRoutes("/api", srv)
		go srv.Start()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)

		log.Info("replicant server started").
			String("address", address).Log()
		<-signalCh

		if err = srv.Close(context.Background()); err != nil {
			log.Info("replicant server stopped").String("error", err.Error()).Log()
			os.Exit(1)
		}

		log.Info("replicant server stopped").Log()
	},
}
