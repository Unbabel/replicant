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

	"github.com/brunotm/replicant/transaction"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Emitter for elasticsearch
type Emitter struct {
	config         Config
	tests          prometheus.Counter
	failures       prometheus.Counter
	stateGauge     *prometheus.GaugeVec
	retriesGauge   *prometheus.GaugeVec
	latencyGauge   *prometheus.GaugeVec
	stateSummary   *prometheus.SummaryVec
	retriesSummary *prometheus.SummaryVec
	latencySummary *prometheus.SummaryVec
}

// Config options
type Config struct {
	Path              string              `json:"path" yaml:"path"`
	Labels            []string            `json:"labels" yaml:"labels"`
	Gauges            bool                `json:"gauges" yaml:"gauges"`
	Summaries         bool                `json:"summaries" yaml:"summaries"`
	SummaryObjectives map[float64]float64 `json:"summary_objectives" yaml:"summary_objectives"`
}

// Close and flush pending data
func (e *Emitter) Close() {}

// Emit results
func (e *Emitter) Emit(result transaction.Result) {
	if e.config.Gauges {
		e.latencyGauge.With(result.Metadata).Set(result.DurationSeconds)
		e.retriesGauge.With(result.Metadata).Set(float64(result.RetryCount))

		switch result.Failed {
		case true:
			e.stateGauge.With(result.Metadata).Set(1)
		case false:
			e.stateGauge.With(result.Metadata).Set(0)
		}
	}

	if e.config.Summaries {
		e.latencySummary.With(result.Metadata).Observe(result.DurationSeconds)
		e.retriesSummary.With(result.Metadata).Observe(float64(result.RetryCount))

		switch result.Failed {
		case true:
			e.stateSummary.With(result.Metadata).Observe(1)
		case false:
			e.stateSummary.With(result.Metadata).Observe(0)
		}
	}

	e.tests.Add(1)
	if result.Failed {
		e.failures.Add(1)
	}
}

// New creates a new transaction.Result emitter
func New(c Config, router *httprouter.Router) (emitter *Emitter, err error) {

	if !strings.HasPrefix(c.Path, "/") {
		return nil, fmt.Errorf("url path must start with /")
	}

	emitter = &Emitter{}
	emitter.config = c

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

	if c.Gauges {
		emitter.stateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "replicant",
			Subsystem: "current",
			Name:      "state",
			Help:      "transaction result state"},
			c.Labels)

		emitter.retriesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "replicant",
			Subsystem: "current",
			Name:      "retries",
			Help:      "transaction retries"},
			c.Labels)

		emitter.latencyGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "replicant",
			Subsystem: "current",
			Name:      "latency",
			Help:      "transaction latencies"},
			c.Labels)
	}

	if c.Summaries {
		emitter.stateSummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "replicant",
			Subsystem:  "summary",
			Name:       "state",
			Help:       "transaction result state",
			Objectives: c.SummaryObjectives},
			c.Labels)

		emitter.retriesSummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "replicant",
			Subsystem:  "summary",
			Name:       "retries",
			Help:       "transaction retries",
			Objectives: c.SummaryObjectives},
			c.Labels)

		emitter.latencySummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "replicant",
			Subsystem:  "summary",
			Name:       "latency",
			Help:       "transaction latencies",
			Objectives: c.SummaryObjectives},
			c.Labels)
	}

	router.Handler(http.MethodGet, c.Path, promhttp.Handler())
	return emitter, nil
}
