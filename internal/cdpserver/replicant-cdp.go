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

	"github.com/Unbabel/replicant/driver/web"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
	"github.com/julienschmidt/httprouter"
)

// Server is a replicant-cdp server for replicant
type Server struct {
	m        sync.RWMutex
	stop     chan struct{}
	cmd      *exec.Cmd
	args     string
	interval time.Duration
	driver   *web.Driver
	router   *httprouter.Router
	http     *http.Server
}

// New creates a new replicant-cdp server
func New(addr, args string, interval time.Duration) (s *Server, err error) {
	s = &Server{}
	s.args = args
	s.interval = interval
	s.stop = make(chan struct{})
	s.driver = web.New(web.Config{ServerURL: "", DNSDiscovery: false, Proxied: false})
	s.router = httprouter.New()
	s.http = &http.Server{}
	s.http.Addr = addr
	s.http.Handler = s.router

	// Handle web transactions
	s.router.Handle(http.MethodPost, "/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

		// override cpd_address for config
		if config.Inputs == nil {
			config.Inputs = make(map[string]interface{})
		}
		config.Inputs["cdp_address"] = "http://127.0.0.1:9222"

		if err = json.Unmarshal(buf, &config); err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}

		tx, err := s.driver.New(config)
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

// Start the replicant-cdp server, calling start
func (s *Server) Start() (err error) {
	// start serving requests
	go s.http.ListenAndServe()

	// start chrome management proccess
	return s.monitor()
}

// Close stops the replicant-cdp
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
		return fmt.Errorf("replicant-cdp: error starting chrome process: %w", err)
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
				return fmt.Errorf("replicant-cdp: error stopping chrome process: %w", err)
			}

			// start chrome process and set process group id to avoid
			// leaving zombies upon termination
			s.cmd = exec.Command(arguments[0], arguments[1:]...)
			s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			if err := s.cmd.Start(); err != nil {
				return fmt.Errorf("replicant-cdp: error starting chrome process: %w", err)
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
