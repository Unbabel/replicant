package web

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
	"time"

	"github.com/MontFerret/ferret/pkg/compiler"
	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	"github.com/MontFerret/ferret/pkg/runtime"
	"github.com/MontFerret/ferret/pkg/runtime/logging"
	"github.com/brunotm/replicant/transaction"
)

const (
	// Type of driver
	Type = "web"
)

func init() {
	transaction.Register(
		Type,
		func(config transaction.Config) (tx transaction.Transaction, err error) {
			return New(config)
		})
}

// Transaction is a pre-compiled replicant transaction for web applications
type Transaction struct {
	config  transaction.Config
	server  string
	timeout time.Duration
	program *runtime.Program
}

// New creates a web transaction
func New(config transaction.Config) (tx *Transaction, err error) {
	tx = &Transaction{}

	s, ok := config.Inputs["cdp_address"]
	if !ok {
		return nil, fmt.Errorf("cdp_address not specified in inputs")
	}

	server, ok := s.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected value for cdp_address")
	}

	tx.server = server

	if config.Timeout != "" {
		tx.timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, err
		}
	}

	tx.program, err = compiler.New().Compile(config.Script)
	if err != nil {
		return nil, err
	}

	tx.config = config
	return tx, nil
}

// Config returns the transaction config
func (t *Transaction) Config() (config transaction.Config) {
	return t.config
}

// Run executes the web transaction
func (t *Transaction) Run(ctx context.Context) (result transaction.Result) {

	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	ctx = drivers.WithContext(
		ctx, cdp.NewDriver(cdp.WithAddress(t.server)),
		drivers.AsDefault())

	result.Name = t.config.Name
	result.Type = "web"
	result.Time = time.Now()

	// runtime.WithLog runtime.WithLogFields runtime.WithLogLevel
	r, err := t.program.Run(ctx, runtime.WithLogLevel(logging.ErrorLevel))
	result.DurationSeconds = time.Since(result.Time).Seconds()

	if err != nil {
		result.Error = err
		result.Message = err.Error()
		result.Failed = true
	}

	result.Metadata = t.config.Metadata

	if len(r) == 0 {
		return result
	}

	if err = json.Unmarshal(r, &result); err != nil {
		result.Error = fmt.Errorf("%s, %s", result.Error, err)
		result.Data = string(r)
	}

	return result
}
