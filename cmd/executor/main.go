package main

import (
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/Unbabel/replicant/internal/cdpserver"
	"github.com/Unbabel/replicant/log"
)

var (
	address  = flag.String("address", "0.0.0.0:8080", "address to listen on")
	interval = flag.Duration("interval", time.Second*300, "interval to recycle chrome process")
	args     = flag.String("args", defaultArgs, "chrome command line")
	level    = flag.String("level", "INFO", "log level")

	defaultArgs = "/headless-shell/headless-shell --headless --no-zygote --no-sandbox --disable-gpu --disable-software-rasterizer --disable-dev-shm-usage --remote-debugging-address=127.0.0.1 --remote-debugging-port=9222 --incognito --disable-shared-workers --disable-remote-fonts --disable-background-networking --disable-crash-reporter --disable-default-apps --disable-domain-reliability --disable-extensions --disable-shared-workers --disable-setuid-sandbox"
	//--single-process --process-per-site
)

func main() {
	flag.Parse()
	log.Init(*level)

	s, err := cdpserver.New(*address, *args, *interval)
	if err != nil {
		log.Error("error creating replicant-cdp server").Error("error", err).Log()
		os.Exit(1)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	// listen for stop signals
	go func() {
		<-signalCh
		if err := s.Close(); err != nil {
			log.Error("error stopping replicant-cdp").Error("error", err).Log()
			os.Exit(1)
		}
	}()

	log.Info("starting replicant-cdp").Log()
	if err := s.Start(); err != nil {
		log.Error("error running replicant-cdp").Error("error", err).Log()
		os.Exit(1)
	}

	log.Info("replicant-cdp stopped").Log()

}
