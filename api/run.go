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
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/brunotm/replicant/server"
	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/callback"
	"github.com/oklog/ulid"
	"gopkg.in/yaml.v2"
)

// RunTransaction runs a unmanaged ad-hoc transaction
func RunTransaction(srv *server.Server) (handle server.Handler) {
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
		case "", "application/json":
			if err = json.Unmarshal(buf, &config); err != nil {
				httpError(w, err, http.StatusBadRequest)
				return
			}
		case "application/yaml":
			if err = yaml.Unmarshal(buf, &config); err != nil {
				httpError(w, err, http.StatusBadRequest)
				return
			}
		}

		tx, err := srv.Manager().New(config)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
			return
		}

		entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
		u, _ := ulid.New(ulid.Timestamp(time.Now()), entropy)
		uuid := u.String()

		ctx := context.WithValue(context.Background(), "transaction_uuid", uuid)

		if config.CallBack != nil {
			listener, err := callback.GetListener(config.CallBack.Type)
			if err != nil {
				httpError(w, err, http.StatusBadRequest)
				return
			}

			ctx = context.WithValue(ctx, config.CallBack.Type, listener)
		}

		res := tx.Run(ctx)
		res.UUID = uuid
		result.Data = []transaction.Result{res}

		buf, err = json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}

// RunTransactionByName runs a managed ad-hoc transaction
func RunTransactionByName(srv *server.Server) (handle server.Handler) {
	return func(w http.ResponseWriter, r *http.Request, p server.Params) {
		defer r.Body.Close()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var result Result
		var err error
		var buf []byte

		name := p.ByName("name")

		res, err := srv.Manager().Run(context.Background(), name)
		if err != nil {
			httpError(w, err, http.StatusNotFound)
			return
		}

		result.Data = []transaction.Result{res}
		buf, err = json.Marshal(&result)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	}
}
