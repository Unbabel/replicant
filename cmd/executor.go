package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Unbabel/replicant/internal/cmdutil"
	"github.com/Unbabel/replicant/internal/executor"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
)

func init() {
	Executor.Flags().String("listen-address", "0.0.0.0:8081", "Address to for executor to listen on")
	Executor.Flags().String("server-url", "http://127.0.0.1:8080", "Replicant server url")
	Executor.Flags().Duration("max-runtime", time.Minute*5, "Maximum individual test runtime")
	Executor.Flags().Bool("debug", false, "Expose a debug profile endpoint at /debug/pprof")
	Executor.Flags().String("webhook-advertise-url", "http://localhost:8080", "URL to advertise when receiving webhook based async responses")
	Executor.Flags().String("chrome-remote-url", "http://127.0.0.1:9222", "Chrome remote debugging protocol server. For using remote chrome process instead of a local managed process")
	Executor.Flags().Bool("chrome-enable-local", true, "Enable running a local chrome worker process for web transactions")
	Executor.Flags().String("chrome-local-command", "/headless-shell/headless-shell --headless --no-zygote --no-sandbox --disable-gpu --disable-software-rasterizer --disable-dev-shm-usage --remote-debugging-address=127.0.0.1 --remote-debugging-port=9222 --incognito --disable-shared-workers --disable-remote-fonts --disable-background-networking --disable-crash-reporter --disable-default-apps --disable-domain-reliability --disable-extensions --disable-shared-workers --disable-setuid-sandbox", "Command for launching chrome with arguments included")
	Executor.Flags().Duration("chrome-recycle-interval", time.Minute*5, "Chrome recycle interval for locally managed chrome process")
}

// Executor command
var Executor = &cobra.Command{
	Use:   "executor",
	Short: "Start the replicant transaction execution service",
	Run: func(cmd *cobra.Command, args []string) {

		config := executor.Config{}
		config.ServerURL = cmdutil.GetFlagString(cmd, "server-url")
		config.AdvertiseURL = cmdutil.GetFlagString(cmd, "webhook-advertise-url")

		// Setup chrome support for web applications
		config.Web.ServerURL = cmdutil.GetFlagString(cmd, "chrome-remote-url")

		if cmdutil.GetFlagBool(cmd, "chrome-enable-local") {
			arguments := strings.Split(cmdutil.GetFlagString(cmd, "chrome-local-command"), " ")
			config.Web.BinaryPath = arguments[:1][0]
			config.Web.BinaryArgs = arguments[1:]
			config.Web.RecycleInterval = cmdutil.GetFlagDuration(cmd, "chrome-recycle-interval")
		}

		server := &http.Server{}
		server.Addr = cmdutil.GetFlagString(cmd, "listen-address")
		server.ReadTimeout = cmdutil.GetFlagDuration(cmd, "max-runtime")
		server.WriteTimeout = cmdutil.GetFlagDuration(cmd, "max-runtime")
		server.ReadHeaderTimeout = cmdutil.GetFlagDuration(cmd, "max-runtime")
		router := httprouter.New()
		server.Handler = router

		e, err := executor.New(config)
		if err != nil {
			log.Error("error creating replicant-executor").Error("error", err).Log()
			os.Exit(1)
		}

		router.Handle(http.MethodPost, "/api/v1/run/:uuid", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			defer r.Body.Close()

			uuid := p.ByName("uuid")
			var err error
			var buf []byte
			var config transaction.Config

			if buf, err = ioutil.ReadAll(r.Body); err != nil {
				httpError(w, uuid, config, fmt.Errorf("error reading request body: %w", err), http.StatusBadRequest)
				return
			}

			if err = json.Unmarshal(buf, &config); err != nil {
				httpError(w, uuid, config, fmt.Errorf("error deserializing json request body: %w", err), http.StatusBadRequest)
				return
			}

			if err = json.Unmarshal(buf, &config); err != nil {
				httpError(w, uuid, config, err, http.StatusBadRequest)
				return
			}

			result, err := e.Run(uuid, config)
			if err != nil {
				httpError(w, uuid, config, err, http.StatusBadRequest)
				return
			}

			buf, err = json.Marshal(&result)
			if err != nil {
				httpError(w, uuid, config, fmt.Errorf("error serializing results: %w", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(buf)
		})

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)

		// listen for stop signals
		go func() {
			<-signalCh
			if err := server.Shutdown(context.Background()); err != nil {
				log.Error("error stopping replicant-executor").Error("error", err).Log()
				os.Exit(1)
			}
		}()

		log.Info("starting replicant-executor").Log()
		if err := server.ListenAndServe(); err != nil {
			log.Error("error running replicant-cdp").Error("error", err).Log()
			os.Exit(1)
		}

		log.Info("replicant-cdp stopped").Log()

	},
}

// httpError wraps http status codes and error messages as json responses
func httpError(w http.ResponseWriter, uuid string, config transaction.Config, err error, code int) {
	var result transaction.Result

	result.Name = config.Name
	result.Driver = config.Driver
	result.Metadata = config.Metadata
	result.Time = time.Now()
	result.DurationSeconds = 0
	result.Failed = true
	result.Error = err

	res, _ := json.Marshal(&result)

	w.WriteHeader(code)
	w.Write(res)

	log.Error("handling web transaction request").Error("error", err).Log()
}
