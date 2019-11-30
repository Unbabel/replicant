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
	"fmt"
	"net/http"

	"github.com/brunotm/replicant/server"
	"github.com/brunotm/replicant/transaction"
)

// GetResult of managed replicant transactions by name
func GetResult(srv *server.Server) (handle server.Handler) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result

		name := p.ByName("name")
		res, err := srv.Manager().GetResult(name)
		if err != nil {
			httpError(w, err, http.StatusNotFound)
			return
		}

		result.Data = []transaction.Result{res}
		buf, err := json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}

// GetResults of all managed replicant transactions
func GetResults(srv *server.Server) (handle server.Handler) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result
		result.Data = srv.Manager().GetResults()
		buf, err := json.Marshal(&result)
		if err != nil {
			httpError(w, fmt.Errorf("error serializing results: %w", err), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}
