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
	"context"
	"errors"
	"sync"

	"github.com/brunotm/replicant/internal/scheduler"
	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/callback"
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
	transactions Store
	results      sync.Map
}

// New creates a new manager
func New(s Store) (manager *Manager) {
	manager = &Manager{}
	manager.transactions = s
	manager.scheduler = scheduler.New()
	manager.scheduler.Start()
	return manager
}

// New creates a transaction for the given config
func (m *Manager) New(config transaction.Config) (tx transaction.Transaction, err error) {
	return transaction.New(config)
}

// Add adds a replicant transaction to the manager and scheduler if the scheduling
// spec is provided
func (m *Manager) Add(tx transaction.Transaction) (err error) {

	if tx == nil {
		return errors.New("can't add a nil transaction")
	}

	if ok := m.transactions.Has(tx.Config().Name); ok {
		return errors.New("transaction already exists")
	}

	config := tx.Config()

	var listener callback.Listener
	if config.CallBack != nil {
		listener, err = callback.GetListener(config.CallBack.Type)
		if err != nil {
			return err
		}
	}

	if config.Schedule != "" {
		err = m.scheduler.AddTaskFunc(config.Name, config.Schedule,
			func() {
				var result transaction.Result

				for x := 0; x <= config.RetryCount; x++ {
					ctx := context.Background()

					if config.CallBack != nil {
						ctx = context.WithValue(ctx, config.CallBack.Type, listener)
					}

					result = tx.Run(ctx)
					result.RetryCount = x

					if !result.Failed {
						break
					}
				}

				m.results.Store(config.Name, result)

				if m.emitters != nil {
					for x := 0; x < len(m.emitters); x++ {
						m.emitters[x].Emit(result)
					}
				}

			})

		if err != nil {
			return err
		}
	}

	m.transactions.Set(config.Name, tx.Config())
	return nil
}

// AddFromConfig is like Add but creates the transaction from a transaction.Config
func (m *Manager) AddFromConfig(config transaction.Config) (err error) {
	tx, err := transaction.New(config)
	if err != nil {
		return err
	}

	return m.Add(tx)
}

// RemoveTransaction from the manager
func (m *Manager) RemoveTransaction(name string) (err error) {

	if err = m.transactions.Delete(name); err != nil {
		return err
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
		return nil, err
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
		return result, errors.New("no results found for the specified transaction")
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
