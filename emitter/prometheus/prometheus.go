package prometheus

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
	"net/http"

	"github.com/brunotm/replicant/replicant/transaction"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Emitter for elasticsearch
type Emitter struct {
	path     string
	tests    prometheus.Counter
	failures prometheus.Counter
	state    *prometheus.GaugeVec
	retries  *prometheus.GaugeVec
	latency  *prometheus.GaugeVec
}

// Close and flush pending data
func (e *Emitter) Close() {}

// Emit results
func (e *Emitter) Emit(result transaction.Result) {

	e.latency.With(result.Metadata).Set(result.DurationSeconds)
	e.retries.With(result.Metadata).Set(float64(result.RetryCount))

	switch result.Failed {
	case true:
		e.state.With(result.Metadata).Set(1)
	case false:
		e.state.With(result.Metadata).Set(0)
	}

	e.tests.Add(1)
	if result.Failed {
		e.failures.Add(1)
	}
}

// New creates a new transaction.Result emitter
func New(path string, router *httprouter.Router) (emitter *Emitter, err error) {
	emitter = &Emitter{}

	emitter.tests = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "replicant",
		Subsystem: "total",
		Name:      "runs",
		Help:      "total number of transactions runs",
	})

	emitter.failures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "replicant",
		Subsystem: "total",
		Name:      "failures",
		Help:      "total number of failed transactions",
	})

	emitter.state = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "replicant",
		Subsystem: "transaction",
		Name:      "state",
		Help:      "transaction result state"},
		[]string{"application", "environment", "component"})

	emitter.retries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "replicant",
		Subsystem: "transaction",
		Name:      "retries",
		Help:      "transaction retries"},
		[]string{"application", "environment", "component"})

	emitter.latency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "replicant",
		Subsystem: "transaction",
		Name:      "latency",
		Help:      "transaction latencies"},
		[]string{"application", "environment", "component"})

	router.Handler(http.MethodGet, path, promhttp.Handler())
	return emitter, nil
}
