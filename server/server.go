package server

/*
   Copyright 2019 Bruno Moura <brunotm@gmail.com>

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"context"
	"encoding/json"

	"net/http"
	"time"

	"github.com/brunotm/log"
	"github.com/brunotm/replicant/transaction/manager"
	"github.com/julienschmidt/httprouter"
)

// Config for replicant server
type Config struct {
	Addr              string
	WriteTimeout      time.Duration
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// Server is an replicant manager and api server
type Server struct {
	config  Config
	http    *http.Server
	router  *httprouter.Router
	manager *manager.Manager
}

// New creates a new replicant server
func New(config Config, s manager.Store) (server *Server, err error) {
	server = &Server{}
	server.config = config
	server.router = httprouter.New()
	server.http = &http.Server{}
	server.http.Addr = config.Addr

	if config.WriteTimeout != 0 {
		server.http.WriteTimeout = config.WriteTimeout
	}

	if config.ReadTimeout != 0 {
		server.http.ReadTimeout = config.ReadTimeout
	}

	if config.ReadHeaderTimeout != 0 {
		server.http.ReadHeaderTimeout = config.ReadHeaderTimeout
	}

	server.manager = manager.New(s)

	server.http.Handler = server.router
	return server, nil
}

// Start serving
func (s *Server) Start() (err error) {
	if err = s.http.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Router returns this server http router
func (s *Server) Router() (r *httprouter.Router) {
	return s.router
}

// Manager returns this server replicant manager
func (s *Server) Manager() (m *manager.Manager) {
	return s.manager
}

// ServeHTTP implements the http.Handler interface for testing and handler usage
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

// Close this server
func (s *Server) Close(ctx context.Context) (err error) {
	s.http.Shutdown(ctx)
	return s.manager.Close()
}

// AddHandler adds a handler for the given method and path
func (s *Server) AddHandler(method, path string, handler Handle) {
	s.router.Handle(method, path, logger(recovery(handler)))
}

// AddServerHandler adds a handler for the given method and path
func (s *Server) AddServerHandler(method, path string, handler Handler) {
	log.Info("adding handler").String("path", path).String("method", method).Log()
	s.router.Handle(method, path, logger(recovery(handler(s))))
}

// Handler is handler that has access to the server
type Handler func(*Server) httprouter.Handle

// Handle is a http handler
type Handle = httprouter.Handle

// Params from the URL
type Params = httprouter.Params

// recovery middleware
func recovery(h Handle) (n Handle) {
	return func(w http.ResponseWriter, r *http.Request, p Params) {

		defer func() {
			err := recover()
			if err != nil {
				jsonBody, _ := json.Marshal(map[string]interface{}{
					"message": "There was an internal server error",
					"error":   err,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}
		}()

		h(w, r, p)

	}
}

// logger middleware
func logger(h Handle) (n Handle) {
	return func(w http.ResponseWriter, r *http.Request, p Params) {
		start := time.Now()
		h(w, r, p)
		log.Info("api request").String("method", r.Method).
			String("uri", r.URL.String()).
			String("requester", r.RemoteAddr).
			Int("duration_ms", time.Since(start).Milliseconds()).
			Log()
	}
}
