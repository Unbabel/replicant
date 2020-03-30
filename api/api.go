// Package api implements the replicant API handlers.
package api

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
	"encoding/json"
	"net/http"

	"github.com/Unbabel/replicant/server"
	"github.com/Unbabel/replicant/transaction"
)

var (
	EndpointTransaction = "/api/v1/transaction"
	EndpointRun         = "/api/v1/run"
	EndpointResult      = "/api/v1/result"
)

// Result is the api calls result envelope
type Result struct {
	Error        string               `json:"error,omitempty"`
	Message      string               `json:"message,omitempty"`
	Results      []transaction.Result `json:"results,omitempty"`
	Transactions []transaction.Config `json:"transactions,omitempty"`
}

// AddAllRoutes api routes to the given server with the given prefix.
// The prefix can be empty.
func AddAllRoutes(server *server.Server) {
	server.AddServerHandler(http.MethodPost, EndpointTransaction, AddTransaction)
	server.AddServerHandler(http.MethodGet, EndpointTransaction, GetTransactions)
	server.AddServerHandler(http.MethodGet, EndpointTransaction+"/:name", GetTransaction)
	server.AddServerHandler(http.MethodDelete, EndpointTransaction+"/:name", RemoveTransaction)
	server.AddServerHandler(http.MethodPost, EndpointRun, RunTransaction)
	server.AddServerHandler(http.MethodPost, EndpointRun+`/:name`, RunTransactionByName)
	server.AddServerHandler(http.MethodGet, EndpointResult, GetResults)
	server.AddServerHandler(http.MethodGet, EndpointResult+"/:name", GetResult)
	server.AddServerHandler(http.MethodPost, "/api/v1/callback/:uuid", CallbackRequest)
}

// httpError wraps http status codes and error messages as json responses
func httpError(w http.ResponseWriter, err error, code int) {
	var result Result
	result.Error = err.Error()
	res, _ := json.Marshal(&result)

	w.WriteHeader(code)
	w.Write(res)
}

// wrapHTTPHandler from net/http for usage with the server package
func wrapHTTPHandler(h http.Handler) (handle server.Handler) {
	return func(w http.ResponseWriter, r *http.Request, _ server.Params) {
		h.ServeHTTP(w, r)
	}
}

// wrapHTTPHandlerFunc from net/http for usage with the server package
func wrapHTTPHandlerFunc(h http.HandlerFunc) (handle server.Handler) {
	return func(w http.ResponseWriter, r *http.Request, _ server.Params) {
		h(w, r)
	}
}
