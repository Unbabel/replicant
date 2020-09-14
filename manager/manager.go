// Package manager implements a transaction manager for running and maintaining
// test transactions.
package manager

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/Unbabel/replicant/internal/scheduler"
	"github.com/Unbabel/replicant/internal/tmpl"
	"github.com/Unbabel/replicant/internal/xz"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/store"
	"github.com/Unbabel/replicant/transaction"
	"github.com/segmentio/ksuid"
)

const (
	// DefaultTransactionTimeout if not specified
	DefaultTransactionTimeout = "1m"

	// default timeout grace period
	defaultTimeoutGracePeriod = time.Second * 20
)

// Emitter is the interface for result emitters to external systems
type Emitter interface {
	Emit(result transaction.Result)
}

// EmitterFunc is a function type that implements Emitter
type EmitterFunc func(result transaction.Result)

// Emit results
func (e EmitterFunc) Emit(result transaction.Result) { e(result) }

// Manager is a manager for replicant transactions.
// It tracks execution, scheduling and result data.
type Manager struct {
	mtx          sync.Mutex
	client       *http.Client
	executorURL  string
	emitters     []Emitter
	scheduler    *scheduler.Scheduler
	transactions store.Store
	results      *xz.Map
}

// New creates a new manager
func New(s store.Store, executorURL string) (manager *Manager) {
	manager = &Manager{}
	manager.client = &http.Client{}
	manager.executorURL = executorURL + "/api/v1/run/"
	manager.transactions = s
	manager.results = xz.NewMap()
	manager.scheduler = scheduler.New()
	manager.scheduler.Start()

	// Reconfigure previously stored transactions
	s.Iter(func(name string, config transaction.Config) (proceed bool) {

		if config.Schedule == "" {
			log.Info("stored transaction has no schedule").
				String("name", name).String("driver", config.Driver).Log()
			return true
		}

		if err := manager.schedule(config); err != nil {
			log.Error("error scheduling transaction").
				String("name", name).Error("error", err).Log()
			return true
		}

		log.Info("loaded stored transaction").
			String("name", name).String("driver", config.Driver).
			String("schedule", config.Schedule).Log()

		return true
	})

	return manager
}

// Close the manager
func (m *Manager) Close() (err error) {
	m.scheduler.Stop()
	return m.transactions.Close()
}

// Run the given transaction
func (m *Manager) Run(c transaction.Config) (r transaction.Result) {
	var err error

	uuid := ksuid.New().String()
	start := time.Now()

	if c.Inputs != nil {
		if c, err = tmpl.Parse(c); err != nil {
			return wrapErrorResult(
				uuid, c, start, fmt.Errorf("manager: error parsing transaction template: %w", err))
		}
	}

	buf, err := json.Marshal(&c)
	if err != nil {
		return wrapErrorResult(
			uuid, c, start, fmt.Errorf("manager: error marshaling config: %w", err))
	}

	req, err := http.NewRequest(http.MethodPost, m.executorURL+uuid, bytes.NewReader(buf))
	if err != nil {
		return wrapErrorResult(
			uuid, c, start, fmt.Errorf("manager: error creating executor request: %w", err))
	}

	// Use the default timeout when unspecified
	if c.Timeout == "" {
		c.Timeout = DefaultTransactionTimeout
	}

	// Ensure we timeout on executor calls. Use the transaction timeout plus
	// a grace period in order to have a timeout greater than the execution time.
	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return wrapErrorResult(
			uuid, c, start, fmt.Errorf("manager: error parsing timeout from template: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout+defaultTimeoutGracePeriod)
	defer cancel()

	res, err := m.client.Do(req.WithContext(ctx))
	if err != nil {
		return wrapErrorResult(
			uuid, c, start, fmt.Errorf("manager: error sending executor request: %w", err))
	}
	defer res.Body.Close()

	buf, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return wrapErrorResult(
			uuid, c, start, fmt.Errorf("manager: error reading executor response: %w", err))
	}

	err = json.Unmarshal(buf, &r)
	if err != nil {
		return wrapErrorResult(
			uuid, c, start, fmt.Errorf("manager: error reading executor response: %w", err))
	}

	return r
}

func (m *Manager) schedule(config transaction.Config) (err error) {

	return m.scheduler.AddTaskFunc(config.Name, config.Schedule,
		func() {
			var result transaction.Result

			for x := 0; x <= config.RetryCount; x++ {
				result = m.Run(config)
				result.RetryCount = x
				if !result.Failed && result.Error == nil {
					break
				}

				log.Debug("transaction failed").String("name", result.Name).
					Error("error", result.Error).String("data", result.Data).
					String("message", result.Message).String("uuid", result.UUID).
					Int("retry", int64(result.RetryCount)).Log()
			}

			m.results.Store(config.Name, result)
			for x := 0; x < len(m.emitters); x++ {
				m.emitters[x].Emit(result)
			}
		})
}

// Add adds a replicant transaction to the manager and scheduler if the scheduling
// spec is provided
func (m *Manager) Add(config transaction.Config) (err error) {
	ok, err := m.transactions.Has(config.Name)
	if err != nil {
		return fmt.Errorf("manager: %w", err)
	}

	if ok {
		return fmt.Errorf("manager: transaction already exists")
	}

	if config.Schedule != "" {
		if err = m.schedule(config); err != nil {
			return err
		}
	}

	return m.transactions.Set(config.Name, config)
}

// Delete a transaction from the manager by name
func (m *Manager) Delete(name string) (err error) {

	if err = m.transactions.Delete(name); err != nil {
		return fmt.Errorf("manager: %w", err)
	}

	m.scheduler.RemoveTask(name)
	m.results.Delete(name)
	return nil
}

// AddEmitter adds the given Emitter to emit result data to external systems
func (m *Manager) AddEmitter(emitter Emitter) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.emitters = append(m.emitters, emitter)
}

// AddEmitterFunc is like SetEmitter, but it takes a EmitterFunc as input
func (m *Manager) AddEmitterFunc(emitter func(result transaction.Result)) {
	m.AddEmitter(EmitterFunc(emitter))
}

// Get a existing transaction from the manager by name
func (m *Manager) Get(name string) (config transaction.Config, err error) {
	return m.transactions.Get(name)
}

// GetAll transactions from the manager
func (m *Manager) GetAll() (configs []transaction.Config) {
	m.transactions.Iter(func(_ string, config transaction.Config) (proceed bool) {
		configs = append(configs, config)
		return true
	})

	return configs
}

// RunByName a managed transaction in a ad-hoc manner.
func (m *Manager) RunByName(name string) (result transaction.Result, err error) {
	config, err := m.Get(name)
	if err != nil {
		return result, err
	}

	result = m.Run(config)
	return result, nil
}

// GetResult fetches the latest result from a managed transaction
func (m *Manager) GetResult(name string) (result transaction.Result, err error) {
	v, ok := m.results.Load(name)
	if !ok {
		return result, fmt.Errorf("manager: no results found")
	}

	return v.(transaction.Result), nil
}

// GetResults fetches all latest results
func (m *Manager) GetResults() (results []transaction.Result) {
	m.results.Range(func(_ interface{}, v interface{}) (proceed bool) {
		results = append(results, v.(transaction.Result))
		return true
	})

	return results
}

func wrapErrorResult(uuid string, c transaction.Config, start time.Time, err error) (r transaction.Result) {
	r.Name = c.Name
	r.Driver = c.Driver
	r.Metadata = c.Metadata
	r.Time = start
	r.DurationSeconds = time.Since(start).Seconds()
	r.Failed = true
	r.Error = err
	return r
}
