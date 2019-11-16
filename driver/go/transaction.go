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
	"fmt"
	"time"

	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/callback"
)

// TxFunc is the Run function signature which will be called from the provided go code.
// Package name must be transaction, "transaction.Run".
type TxFunc func(context.Context) (message, data string, err error)

// CallBackHandler is the Handle function signature which will be called from the provided go code
// when dealing with callbacks. Package name must be callback, "callback.Handle"
type CallBackHandler func(context.Context, []byte) (message, data string, err error)

// Transaction is a pre-compiled replicant transaction for golang based custom transactions
type Transaction struct {
	config          transaction.Config
	timeout         time.Duration
	transaction     TxFunc
	callbackHandler CallBackHandler
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

	result.Name = t.config.Name
	result.Driver = "go"
	result.Metadata = t.config.Metadata

	var err error
	var handle *callback.Handle

	// If dealing with async responses for this transaction, we must first get a Listener and Handle
	if t.config.CallBack != nil {
		listener, ok := ctx.Value(t.config.CallBack.Type).(callback.Listener)
		if !ok {
			result.Failed = true
			result.Message = "callback not found or does not implement callback.Listener interface"
		}

		handle, err = listener.Listen(ctx)
		if err != nil {
			result.Failed = true
			result.Message = fmt.Sprintf("could not handle callback: %s", err)
			return result
		}

		ctx = context.WithValue(ctx, "callback_address", handle.Address)
	}

	result.Time = time.Now()
	m, d, err := t.transaction(ctx)
	result.DurationSeconds = time.Since(result.Time).Seconds()

	if err != nil {
		result.Error = err
		result.Message = m
		result.Failed = true
		return result
	}

	if t.config.CallBack == nil {
		result.Message = m
		result.Data = d
		return result
	}

	// Handle async responses, recalculate duration after response
	resp := <-handle.Response
	result.WithCallback = true
	result.DurationSeconds = time.Since(result.Time).Seconds()

	if resp.Error != nil {
		result.Error = err
		result.Data = string(resp.Data)
		result.Failed = true
		return result
	}

	m, d, err = t.callbackHandler(ctx, resp.Data)
	if err != nil {
		result.Failed = true
	}

	result.Error = err
	result.Message = m
	result.Data = d
	return result
}
