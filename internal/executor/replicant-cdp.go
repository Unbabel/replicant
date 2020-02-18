// Package cdpserver implements a replicant proxy for the chrome developer tools protocol
package cdpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Unbabel/replicant/driver"
	gd "github.com/Unbabel/replicant/driver/go"
	"github.com/Unbabel/replicant/driver/javascript"
	"github.com/Unbabel/replicant/driver/web"
	"github.com/Unbabel/replicant/internal/xz"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
	"github.com/julienschmidt/httprouter"
)

// Server is a replicant-executor server
type Server struct {
	m        sync.RWMutex
	stop     chan struct{}
	cmd      *exec.Cmd
	args     string
	interval time.Duration
	drivers  *xz.Map
	router   *httprouter.Router
	http     *http.Server
}

// New creates a new replicant-executor
func New(addr, args string, interval time.Duration) (s *Server, err error) {
	s = &Server{}
	s.args = args
	s.interval = interval
	s.stop = make(chan struct{})
	s.drivers = xz.NewMap()
	s.router = httprouter.New()
	s.http = &http.Server{}
	s.http.Addr = addr
	s.http.Handler = s.router

	var drv driver.Driver
	drv, err = web.New(web.Config{ServerURL: "http://127.0.0.1:9222"})
	if err != nil {
		return nil, err
	}
	s.drivers.Store(drv.Type, drv)

	drv, err = javascript.New()
	if err != nil {
		return nil, err
	}
	s.drivers.Store(drv.Type, drv)

	drv, err = gd.New()
	if err != nil {
		return nil, err
	}
	s.drivers.Store(drv.Type, drv)

	// Handler for transactions
	s.router.Handle(http.MethodPost, "/api/v1/run", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.m.RLock()
		defer s.m.RUnlock()
		defer r.Body.Close()

		var err error
		var buf []byte
		var config transaction.Config

		if buf, err = ioutil.ReadAll(r.Body); err != nil {
			httpError(w, fmt.Errorf("error reading request body: %w", err), http.StatusBadRequest)
			return
		}

		if err = json.Unmarshal(buf, &config); err != nil {
			httpError(w, fmt.Errorf("error deserializing json request body: %w", err), http.StatusBadRequest)
			return
		}

		if err = json.Unmarshal(buf, &config); err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}

		d, _ := s.drivers.Load(config.Driver)
		drv := d.(driver.Driver)

		tx, err := drv.New(config)
		if err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}

		log.Info("handling proxied transaction request").String("name", config.Name).Log()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		result := tx.Run(ctx)
		buf, err = json.Marshal(&result)
		if err != nil {
			httpError(w, fmt.Errorf("error serializing results: %w", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	})

	return s, err
}

// Start the replicant-executor server, calling start
func (s *Server) Start() (err error) {
	// start serving requests
	go s.http.ListenAndServe()

	// start chrome management proccess
	return s.monitor()
}

// Close stops the replicant-executor
func (s *Server) Close() (err error) {
	s.m.Lock()
	defer s.m.Unlock()

	// terminate the current chrome process after serving all current requests
	defer s.cmd.Process.Kill()

	close(s.stop)
	return s.http.Shutdown(context.Background())
}

// monitor the running chrome process to service transactions and recycle at every interval
func (s *Server) monitor() (err error) {
	arguments := strings.Split(s.args, " ")

	// start chrome process and set process group id to avoid
	// leaving zombies upon termination
	s.cmd = exec.Command(arguments[0], arguments[1:]...)
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("replicant-executor: error starting chrome process: %w", err)
	}

	log.Info("chrome process created").Int("pid", int64(s.cmd.Process.Pid)).Log()

	for {
		select {
		case <-time.After(s.interval):

			log.Info("recycling chrome process").Int("pid", int64(s.cmd.Process.Pid)).Log()
			s.m.Lock()

			// stop chrome process and its children
			err := syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
			if err != nil {
				return fmt.Errorf("replicant-executor: error stopping chrome process: %w", err)
			}

			// start chrome process and set process group id to avoid
			// leaving zombies upon termination
			s.cmd = exec.Command(arguments[0], arguments[1:]...)
			s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := s.cmd.Start(); err != nil {
				return fmt.Errorf("replicant-executor: error starting chrome process: %w", err)
			}

			s.m.Unlock()
			log.Info("chrome process created").Int("pid", int64(s.cmd.Process.Pid)).Log()

		case <-s.stop:
			return nil
		}
	}
}

// httpError wraps http status codes and error messages as json responses
func httpError(w http.ResponseWriter, err error, code int) {
	var result transaction.Result
	result.Error = err
	res, _ := json.Marshal(&result)

	w.WriteHeader(code)
	w.Write(res)

	log.Error("handling web transaction request").Error("error", err).Log()
}
