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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Unbabel/replicant/server"
	"github.com/Unbabel/replicant/transaction"
	"github.com/Unbabel/replicant/transaction/callback"
)

// CallbackRequest handler
func CallbackRequest(srv *server.Server) (handle server.Handler) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var buf []byte
		var err error

		if buf, err = ioutil.ReadAll(r.Body); err != nil {
			httpError(w, fmt.Errorf("error reading request body: %w", err), http.StatusBadRequest)
			return
		}

		config := transaction.Config{}
		if err = json.Unmarshal(buf, &config); err != nil {
			httpError(w, fmt.Errorf("error deserializing json request body: %w", err), http.StatusBadRequest)
			return
		}

		if config.CallBack == nil {
			httpError(w, fmt.Errorf("no callback config found"), http.StatusBadRequest)
			return
		}

		listener, err := callback.GetListener(config.CallBack.Type)
		if err != nil {
			httpError(w, err, http.StatusBadRequest)
			return
		}

		handler, err := listener.Listen(context.WithValue(r.Context(), "transaction_uuid", p.ByName("uuid")))
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
			return
		}

		resp := <-handler.Response
		if resp.Error != nil {
			httpError(w, resp.Error, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp.Data)
	}
}
