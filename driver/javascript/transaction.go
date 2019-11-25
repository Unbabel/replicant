package javascript

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

	"github.com/brunotm/replicant/transaction"
	"github.com/brunotm/replicant/transaction/callback"
	"github.com/robertkrimen/otto"
)

const maxTimeout = 10 * time.Minute

// Transaction is a pre-compiled replicant transaction for javascript based transactions
type Transaction struct {
	vm      *otto.Otto
	config  transaction.Config
	timeout time.Duration
}

// Config returns the transaction config
func (t *Transaction) Config() (config transaction.Config) {
	return t.config
}

// Run executes the transaction
func (t *Transaction) Run(ctx context.Context) (result transaction.Result) {
	u := ctx.Value("transaction_uuid")
	uuid, ok := u.(string)
	if !ok {
		result.Failed = true
		result.Error = fmt.Errorf("transaction_uuid not found in context")
	}

	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	// If dealing with async responses for this transaction, we must first get a Listener and Handle
	var err error
	var handle *callback.Handle
	if t.config.CallBack != nil {
		listener, ok := ctx.Value(t.config.CallBack.Type).(callback.Listener)
		if !ok {
			result.Failed = true
			result.Error = fmt.Errorf("callback not found or does not implement callback.Listener interface")
		}

		handle, err = listener.Listen(ctx)
		if err != nil {
			result.Failed = true
			result.Error = fmt.Errorf("could not handle callback: %s", err)
			return result
		}

	}

	result.Name = t.config.Name
	result.Driver = "javascript"
	result.Time = time.Now()
	result.Metadata = t.config.Metadata

	// copy the vm to avoid problems such as cancellation and GC
	// Use a channel to read the values of execution from the running
	// goroutine needed due to the otto cancelation mechanism which needs a panic
	vm := t.vm.Copy()
	vm.Interrupt = make(chan func(), 1)
	respCh := make(chan transaction.Result, 1)

	// Run in a service goroutine and recover in case
	// of cancellation
	go func() {
		defer func() {
			recover()
		}()

		switch t.Config().CallBack == nil {

		// no async callback response
		case true:
			value, err := vm.Run(fmt.Sprintf(`Run({UUID:"%s", CallbackAddress: ""})`, uuid))
			if err != nil {
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}

			rs, err := value.ToString()
			if err != nil {
				result.Message = "failed to load response from javascript vm"
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}

			var txRes jsResult
			err = json.Unmarshal([]byte(rs), &txRes)
			if err != nil {
				result.Message = "failed to deserialize result from javascript vm for transaction Run()"
				result.Data = rs
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}
			result.Data = txRes.Data
			result.Message = txRes.Message
			result.Failed = txRes.Failed
			respCh <- result
			return

		// async callback response
		case false:
			value, err := vm.Run(fmt.Sprintf(`Run({UUID:"%s", CallbackAddress: "%s"})`, uuid, handle.Address))
			if err != nil {
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}

			rs, err := value.ToString()
			if err != nil {
				result.Message = "failed to load response from javascript vm for transaction Run()"
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}

			var txRes jsResult
			err = json.Unmarshal([]byte(rs), &txRes)
			if err != nil {
				result.Message = "failed to deserialize result from javascript vm for transaction Run()"
				result.Data = rs
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}
			result.Data = txRes.Data
			result.Message = txRes.Message
			result.Failed = txRes.Failed

			// wait for callback response or timeout
			var hr callback.Response
			select {
			case hr = <-handle.Response:
				if hr.Error != nil {
					result.Failed = true
					result.Error = err
					respCh <- result
					return
				}
			case <-ctx.Done():
				return
			}

			data, _ := json.Marshal(&struct{ Data string }{string(hr.Data)})
			value, err = vm.Run(`Handle(` + string(data) + `)`)
			if err != nil {
				result.Message = "failed to run callback Handle()"
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}

			rs, err = value.ToString()
			if err != nil {
				result.Message = "failed to load response from javascript vm for callback Handle()"
				result.Failed = true
				result.Error = err
				result.Data = value.String()
				respCh <- result
				return
			}

			var cbRes jsResult
			err = json.Unmarshal([]byte(rs), &cbRes)
			if err != nil {
				result.Message = "failed to deserialize result from javascript vm for callback Handle()"
				result.Data = rs
				result.Failed = true
				result.Error = err
				respCh <- result
				return
			}
			result.Data = cbRes.Data
			result.Message = cbRes.Message
			result.Failed = cbRes.Failed
			respCh <- result
			return
		}
	}()

	// Handle the results of execution and cancellation
	select {
	case result = <-respCh:
	case <-ctx.Done():
		vm.Interrupt <- func() {
			panic("stop")
		}
		result.Error = fmt.Errorf("timed out running transaction after: %.2f seconds", t.timeout.Seconds())
	}

	result.DurationSeconds = time.Since(result.Time).Seconds()
	return result
}
