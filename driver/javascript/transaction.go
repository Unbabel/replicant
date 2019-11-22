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
	result.Name = t.config.Name
	result.Driver = "javascript"
	result.Time = time.Now()
	result.Metadata = t.config.Metadata

	// copy the vm to avoid problems such as cancellation and GC
	// Use a channel to read the values of execution from the running
	// goroutine needed due to the otto cancelation mechanism which needs a panic
	vm := t.vm.Copy()

	// vmResp type
	type vmResp struct {
		value otto.Value
		err   error
	}
	respCh := make(chan vmResp, 1)

	// Run in a service goroutine and recover in case
	// of cancellation
	go func() {
		defer func() {
			recover()
		}()

		var r vmResp
		r.value, r.err = vm.Run(`Run()`)
		respCh <- r
	}()

	// Handle the results of execution and cancellation
	var r vmResp
	timer := time.NewTimer(t.timeout)
	select {
	case r = <-respCh:
		timer.Stop()
		result.DurationSeconds = time.Since(result.Time).Seconds()
		if r.err != nil {
			result.Failed = true
			result.Error = r.err
			return result
		}

		rs, err := r.value.ToString()
		if err != nil {
			result.Message = "failed to load response from javascript vm"
			result.Failed = true
			result.Error = err
			return result
		}

		err = json.Unmarshal([]byte(rs), &result)
		if err != nil {
			result.Message = "failed to deserialize result from javascript vm"
			result.Failed = true
			result.Error = err
			return result
		}

	case <-timer.C:
		timer.Stop()
		vm.Interrupt <- func() {
			panic("stop")
		}
		result.DurationSeconds = time.Since(result.Time).Seconds()
		result.Error = fmt.Errorf("timed out running transaction, timeout: %.2f seconds", t.timeout.Seconds())
	}

	return result

}
