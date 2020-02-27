package gd

import (
	"context"
	"net/http"
	"testing"

	"github.com/Unbabel/replicant/internal/tmpl"
	"github.com/Unbabel/replicant/transaction"
)

func TestDriverTransaction(t *testing.T) {

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Query().Get("q") != "blade runner" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"reason": "query parameter q not found"}`))
			return
		}

		if r.Header.Get("X-Auth") != "Joi" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"reason": "request header X-Auth not found"}`))
			return
		}

		w.WriteHeader(200)
		w.Write([]byte(`{"reason": "test successful"}`))
	})

	server := &http.Server{}
	server.Addr = "localhost:8080"
	server.Handler = mux
	go server.ListenAndServe()
	defer server.Close()

	d, err := New()
	if err != nil {
		t.Fatalf("error creating driver: %s", err)
	}

	cfg, err := tmpl.Parse(config)
	if err != nil {
		t.Fatalf("error parsing template: %s", err)
	}

	txn, err := d.New(cfg)
	if err != nil {
		t.Fatalf("error creating transaction: %s\n, %#v", err, cfg)
	}

	ctx := context.WithValue(context.Background(), "transaction_uuid", "test-test-test")

	result := txn.Run(ctx)
	if result.Error != nil {
		t.Fatalf("error running transaction: %s", result.Error)
	}

	if result.Failed {
		t.Fatalf("transaction failed:\n%#v", result)
	}

	t.Logf("%#v", result)
}

// test transaction
var config transaction.Config = transaction.Config{
	Name:       "test-transaction",
	Driver:     "go",
	Schedule:   "@every 60s",
	Timeout:    "5s",
	RetryCount: 1,
	Inputs: map[string]interface{}{
		"url":   "http://localhost:8080/test",
		"text":  "blade runner",
		"xauth": "Joi",
	},
	Metadata: map[string]string{
		"transaction": "api-test",
		"application": "test",
		"environment": "test",
		"component":   "api",
	},
	Script: `
	package transaction
	import (
		"fmt"
		"net/http"
		"context"
		"encoding/json"
		"io/ioutil"
	)
	func Run(ctx context.Context) (m string, d string, err error) {
		req, err := http.NewRequest(http.MethodGet, "{{ index . "url" }}", nil)
		if err != nil {
			return "request build failed", "", err
		}
		req.Header.Add("X-Auth","{{ index . "xauth" }}")
		q := req.URL.Query()
		q.Add("q", "{{ index . "text" }}")
		req.URL.RawQuery = q.Encode()
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
		return "failed to send request", "", err
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
		return "failed to read response", "", err
		}
		result := struct{Reason string}{}
		err = json.Unmarshal(buf, &result)
		if err != nil {
		return "failed to unmarshal response", string(buf), err
		}
		if resp.StatusCode > 200 {
			return "status code > 200", result.Reason, fmt.Errorf(result.Reason)
		}
		return "test successful", result.Reason, nil
	}`,
}
