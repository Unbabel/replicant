// Package prometheus implements a result exporter for prometheus.
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
	"fmt"
	"net/http"

	"strings"

	"github.com/Unbabel/replicant/transaction"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config options
type Config struct {
	Path              string              `json:"path" yaml:"path"`
	Labels            []string            `json:"labels" yaml:"labels"`
	Gauges            bool                `json:"gauges" yaml:"gauges"`
	Summaries         bool                `json:"summaries" yaml:"summaries"`
	SummaryObjectives map[float64]float64 `json:"summary_objectives" yaml:"summary_objectives"`
}

// DefaultConfig for prometheus config
var DefaultConfig = Config{
	Path:              "/metrics",
	Gauges:            true,
	Summaries:         true,
	Labels:            []string{"transaction", "application", "environment", "component"},
	SummaryObjectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
}

// Emitter for prometheus
type Emitter struct {
	config         Config
	runs           prometheus.Counter
	failures       prometheus.Counter
	latencyGauge   *prometheus.GaugeVec
	retriesGauge   *prometheus.GaugeVec
	failuresGauge  *prometheus.GaugeVec
	latencySummary *prometheus.SummaryVec
}

// Close emitter
func (e *Emitter) Close() {}

// Emit results
func (e *Emitter) Emit(result transaction.Result) {

	e.runs.Add(1)
	switch result.Failed {
	case true:
		e.failures.Add(1)
		e.failuresGauge.With(result.Metadata).Set(1)
	case false:
		e.failuresGauge.With(result.Metadata).Set(0)
	}

	if e.config.Gauges {
		e.latencyGauge.With(result.Metadata).Set(result.DurationSeconds)
		e.retriesGauge.With(result.Metadata).Set(float64(result.RetryCount))
	}

	if e.config.Summaries {
		e.latencySummary.With(result.Metadata).Observe(result.DurationSeconds)
	}
}

// New creates a new transaction.Result emitter
func New(c Config, router *httprouter.Router) (emitter *Emitter, err error) {

	if !strings.HasPrefix(c.Path, "/") {
		return nil, fmt.Errorf("emitter/prometheus: url path must start with /")
	}

	emitter = &Emitter{}
	emitter.config = c

	emitter.runs = promauto.NewCounter(prometheus.CounterOpts{
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

	if c.Gauges {
		emitter.latencyGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "replicant",
			Subsystem: "test",
			Name:      "latency",
			Help:      "transaction latencies"},
			c.Labels)

		emitter.retriesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "replicant",
			Subsystem: "test",
			Name:      "retries",
			Help:      "transaction retries"},
			c.Labels)

		emitter.failuresGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "replicant",
			Subsystem: "test",
			Name:      "failures",
			Help:      "transaction failures"},
			c.Labels)
	}

	if c.Summaries {
		emitter.latencySummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "replicant",
			Subsystem:  "test",
			Name:       "latency_summary",
			Help:       "transaction latencies",
			Objectives: c.SummaryObjectives},
			c.Labels)
	}

	router.Handler(http.MethodGet, c.Path, promhttp.Handler())
	return emitter, nil
}
