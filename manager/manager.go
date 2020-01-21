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
	"fmt"
	"sync"
	"text/template"

	"github.com/brunotm/replicant/driver"
	"github.com/brunotm/replicant/internal/scheduler"
	"github.com/brunotm/replicant/internal/xz"
	"github.com/brunotm/replicant/log"
	"github.com/brunotm/replicant/store"
	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/callback"
	"github.com/segmentio/ksuid"
)

const (
	// DefaultTransactionTimeout if not specified
	DefaultTransactionTimeout = "5m"
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
	emitters     []Emitter
	scheduler    *scheduler.Scheduler
	transactions store.Store
	drivers      *xz.Map
	results      *xz.Map
}

// New creates a new manager
func New(s store.Store, d ...driver.Driver) (manager *Manager) {
	manager = &Manager{}
	manager.transactions = s
	manager.drivers = xz.NewMap()
	manager.results = xz.NewMap()
	manager.scheduler = scheduler.New()
	manager.scheduler.Start()

	// Register provided transaction drivers
	for _, driver := range d {
		manager.drivers.Store(driver.Type(), driver)
		log.Info("registered driver").String("driver", driver.Type()).Log()
	}

	// Reconfigure previously stored transactions
	s.Iter(func(name string, config transaction.Config) (proceed bool) {
		tx, err := manager.New(config)

		if err != nil {
			log.Error("error creating transaction").
				String("name", name).Error("error", err).Log()
			return true
		}

		if err = manager.schedule(tx); err != nil {
			log.Error("error scheduling transaction").
				String("name", name).Error("error", err).Log()
			return true
		}

		log.Info("added stored transaction").
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

// New creates a transaction for the given config
func (m *Manager) New(config transaction.Config) (tx transaction.Transaction, err error) {

	d, ok := m.drivers.Load(config.Driver)

	if !ok {
		return nil, fmt.Errorf("manager: transaction driver %s not registered", config.Driver)
	}

	if config.Inputs != nil {
		if config, err = parseTemplate(config); err != nil {
			return nil, fmt.Errorf("manager: error parsing transaction template: %w", err)
		}
	}

	if config.Timeout == "" {
		config.Timeout = DefaultTransactionTimeout
	}

	return d.(driver.Driver).New(config)
}

func (m *Manager) schedule(tx transaction.Transaction) (err error) {

	var listener callback.Listener
	config := tx.Config()

	if config.CallBack != nil {
		listener, err = callback.GetListener(config.CallBack.Type)
		if err != nil {
			return err
		}
	}

	return m.scheduler.AddTaskFunc(config.Name, config.Schedule,
		func() {
			var result transaction.Result
			u := ksuid.New()
			uuid := u.String()

			for x := 0; x <= config.RetryCount; x++ {

				ctx := context.WithValue(context.Background(), "transaction_uuid", uuid)
				if config.CallBack != nil {
					ctx = context.WithValue(ctx, config.CallBack.Type, listener)
				}

				result = tx.Run(ctx)
				result.RetryCount = x

				if !result.Failed {
					break
				}
			}

			result.UUID = uuid
			m.results.Store(config.Name, result)

			for x := 0; x < len(m.emitters); x++ {
				m.emitters[x].Emit(result)
			}

		})
}

// Add adds a replicant transaction to the manager and scheduler if the scheduling
// spec is provided
func (m *Manager) Add(tx transaction.Transaction) (err error) {

	if tx == nil {
		return fmt.Errorf("manager: invalid null transaction")
	}

	config := tx.Config()

	ok, err := m.transactions.Has(config.Name)
	if err != nil {
		return fmt.Errorf("manager: %w", err)
	}

	if ok {
		return fmt.Errorf("manager: transaction already exists")
	}

	if config.Schedule != "" {
		if err = m.schedule(tx); err != nil {
			return err
		}
	}

	m.transactions.Set(config.Name, tx.Config())
	return nil
}

// AddFromConfig is like Add but creates the transaction from a transaction.Config
func (m *Manager) AddFromConfig(config transaction.Config) (err error) {
	tx, err := m.New(config)
	if err != nil {
		return err
	}

	return m.Add(tx)
}

// RemoveTransaction from the manager
func (m *Manager) RemoveTransaction(name string) (err error) {

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

// GetTransaction fetches a existing transaction from the manager
func (m *Manager) GetTransaction(name string) (tx transaction.Transaction, err error) {
	config, err := m.transactions.Get(name)
	if err != nil {
		return nil, fmt.Errorf("manager: %w", err)
	}

	return m.New(config)

}

// GetTransactionConfig fetches the config from a managed transaction
func (m *Manager) GetTransactionConfig(name string) (config transaction.Config, err error) {
	return m.transactions.Get(name)
}

// Run a managed transaction in a ad-hoc manner.
func (m *Manager) Run(ctx context.Context, name string) (result transaction.Result, err error) {
	tx, err := m.GetTransaction(name)
	if err != nil {
		return result, err
	}

	return tx.Run(ctx), nil
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

// GetTransactionsConfig fetches all transactions definitions from the manager
func (m *Manager) GetTransactionsConfig() (configs []transaction.Config) {

	m.transactions.Iter(func(_ string, config transaction.Config) (proceed bool) {
		configs = append(configs, config)
		return true
	})

	return configs
}

func parseTemplate(config transaction.Config) (c transaction.Config, err error) {

	tpl, err := template.New(config.Name).Parse(config.Script)
	if err != nil {
		return config, err
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, config.Inputs)
	if err != nil {
		return config, err
	}

	config.Script = buf.String()
	return config, nil

}
