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
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/brunotm/replicant/server"
	"github.com/brunotm/replicant/transaction"
	"gopkg.in/yaml.v2"
)

// AddTransaction to the replicant manager
func AddTransaction(srv *server.Server) (handle server.Handle) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result
		var err error
		var buf []byte

		if buf, err = ioutil.ReadAll(r.Body); err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}

		config := transaction.Config{}

		switch r.Header.Get("Content-Type") {
		case "application/json":
			if err = json.Unmarshal(buf, &config); err != nil {
				httpError(w, err, http.StatusBadRequest)
				return
			}
		case "application/yaml":
			if err = yaml.Unmarshal(buf, &config); err != nil {
				httpError(w, err, http.StatusBadRequest)
				return
			}
		default:
			httpError(w, errors.New("unknown Content-Type"), http.StatusBadRequest)
			return
		}

		err = srv.Manager().AddFromConfig(config)
		if err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}

		result.Message = "transaction created"
		buf, err = json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write(buf)

	}
}

// GetTransaction fetches a named transaction definition from the replicant manager
func GetTransaction(srv *server.Server) (handle server.Handle) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result

		name := p.ByName("name")
		config, err := srv.Manager().GetTransactionConfig(name)

		if err != nil {
			httpError(w, err, http.StatusNotFound)
			return
		}

		result.Data = []transaction.Config{config}
		buf, err := json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}

// GetTransactions fetches all transaction definition from the replicant manager
func GetTransactions(srv *server.Server) (handle server.Handle) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result
		result.Data = srv.Manager().GetTransactionsConfig()
		buf, err := json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}

// RemoveTransaction removes a named transaction from the replicant manager
func RemoveTransaction(srv *server.Server) (handle server.Handle) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result

		name := p.ByName("name")
		err := srv.Manager().RemoveTransaction(name)

		if err != nil {
			httpError(w, err, http.StatusNotFound)
			return
		}

		result.Message = "removed"
		buf, err := json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}
