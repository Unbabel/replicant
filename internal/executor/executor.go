// Package executor implements the replicant transaction execution service
package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Unbabel/replicant/driver"
	gd "github.com/Unbabel/replicant/driver/go"
	"github.com/Unbabel/replicant/driver/javascript"
	"github.com/Unbabel/replicant/driver/web"
	"github.com/Unbabel/replicant/internal/xz"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
	"github.com/Unbabel/replicant/transaction/callback"
	"github.com/julienschmidt/httprouter"
)

// Server is a replicant-executor server
type Server struct {
	config  Config
	drivers *xz.Map
	router  *httprouter.Router
	http    *http.Server
}

// Config for executor
type Config struct {
	Web           web.Config
	ServerURL     string
	AdvertiseURL  string
	ListenAddress string
}

// New creates a new replicant-executor
func New(c Config) (s *Server, err error) {
	s = &Server{}
	s.config = c
	s.drivers = xz.NewMap()
	s.router = httprouter.New()
	s.http = &http.Server{}
	s.http.Addr = c.ListenAddress
	s.http.Handler = s.router

	var drv driver.Driver
	drv, err = web.New(c.Web)
	if err != nil {
		return nil, err
	}
	s.drivers.Store(drv.Type(), drv)

	drv, err = javascript.New()
	if err != nil {
		return nil, err
	}
	s.drivers.Store(drv.Type(), drv)

	drv, err = gd.New()
	if err != nil {
		return nil, err
	}
	s.drivers.Store(drv.Type(), drv)

	// Handler for transactions
	s.router.Handle(http.MethodPost, "/api/v1/run/:uuid", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

		d, _ := s.drivers.Load(config.Driver)
		drv := d.(driver.Driver)

		tx, err := drv.New(config)
		if err != nil {
			httpError(w, uuid, config, err, http.StatusBadRequest)
			return
		}

		log.Info("handling proxied transaction request").String("name", config.Name).Log()

		var ctx context.Context
		var cancel context.CancelFunc

		switch config.Timeout != "" {
		case true:
			timeout, err := time.ParseDuration(config.Timeout)
			if err != nil {
				httpError(w, uuid, config, fmt.Errorf("error parsing timeout: %w", err), http.StatusBadRequest)
				return
			}

			ctx, cancel = context.WithTimeout(context.Background(), timeout)
			defer cancel()

		case false:
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
		}

		// TODO: UUID should come from request
		ctx = context.WithValue(ctx, "transaction_uuid", uuid)

		// Inject callback proxy
		if config.CallBack != nil {
			listener := &callbackProxy{}
			listener.uuid = uuid
			listener.client = &http.Client{}
			listener.config = config
			listener.serverURL = c.ServerURL
			listener.advertiseURL = c.AdvertiseURL
			ctx = context.WithValue(ctx, config.CallBack.Type, listener)
		}

		start := time.Now()
		result := tx.Run(ctx)
		result.UUID = uuid
		result.Time = start
		result.DurationSeconds = time.Since(result.Time).Seconds()
		result.Metadata = config.Metadata

		buf, err = json.Marshal(&result)
		if err != nil {
			httpError(w, uuid, config, fmt.Errorf("error serializing results: %w", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	})

	return s, err
}

// Start the replicant-executor
func (s *Server) Start() (err error) {
	// start serving requests
	return s.http.ListenAndServe()
}

// Close stops the replicant-executor
func (s *Server) Close() (err error) {
	return s.http.Shutdown(context.Background())
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

// TODO: find a proper way of pipelining callbacks
type callbackProxy struct {
	uuid         string
	config       transaction.Config
	client       *http.Client
	serverURL    string
	advertiseURL string
}

func (c *callbackProxy) Listen(ctx context.Context) (h *callback.Handle, err error) {

	response := make(chan callback.Response, 1)

	h = &callback.Handle{}
	h.Response = response
	h.Address = c.advertiseURL + "/callback/" + c.uuid

	buf, err := json.Marshal(c.config)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost, c.serverURL+"/api/v1/callback/"+c.uuid, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	go func() {
		resp, err := c.client.Do(req)
		if err != nil {
			response <- callback.Response{Data: nil, Error: err}
			return
		}
		defer resp.Body.Close()

		buf, err := ioutil.ReadAll(resp.Body)
		response <- callback.Response{Data: buf, Error: err}
	}()

	return h, nil
}
