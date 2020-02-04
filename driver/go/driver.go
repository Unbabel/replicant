// Package gd implements a Go transaction driver.
package gd

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
	"fmt"
	"time"

	"github.com/Unbabel/replicant/transaction"
	"github.com/containous/yaegi/interp"
	"github.com/containous/yaegi/stdlib"
)

// Driver for Go language based transactions
type Driver struct{}

// New creates a new Go driver
func New() (d *Driver) {
	return &Driver{}
}

// Type returns this driver type
func (d *Driver) Type() (t string) {
	return "go"
}

// New creates a web transaction
func (d *Driver) New(config transaction.Config) (tx transaction.Transaction, err error) {
	txn := &Transaction{}

	if config.Timeout != "" {
		txn.timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("driver/go: error parsing timeout: %w", err)
		}
	}

	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)

	_, err = i.Eval(config.Script)
	if err != nil {
		return nil, fmt.Errorf("driver/go: error initializing transaction script: %w", err)
	}

	v, err := i.Eval("transaction.Run")
	if err != nil {
		return nil, fmt.Errorf("driver/go: error loading transaction program code: %w", err)
	}

	var ok bool
	txn.transaction, ok = v.Interface().(func(context.Context) (message, data string, err error))
	if !ok || txn.transaction == nil {
		return nil, fmt.Errorf(
			`driver/go: transaction.Run doesn't implement "func(context.Context) (message, data string, err error)" signature`)
	}

	if config.CallBack != nil {
		i := interp.New(interp.Options{})
		i.Use(stdlib.Symbols)

		_, err = i.Eval(config.CallBack.Script)
		if err != nil {
			return nil, fmt.Errorf("driver/go: error initializing callback script: %w", err)
		}

		var ok bool
		v, err := i.Eval("callback.Handle")
		if err != nil {
			return nil, fmt.Errorf("driver/go: error loading callback program code: %w", err)
		}

		txn.callbackHandler, ok = v.Interface().(func(context.Context, []byte) (message, data string, err error))
		if !ok {
			return nil, fmt.Errorf(
				`driver/go: callback.Handle doesn't implement "func(context.Context, []byte) (message, data string, err error)" signature`)
		}
	}

	txn.config = config
	return txn, nil
}
